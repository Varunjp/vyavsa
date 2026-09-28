package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/database"
	"github.com/Varunjp/vyavsa/internal/handler/health"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/middleware"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// Server coordinates the HTTP server, routing, dependencies, and lifecycle
type Server struct {
	cfg     *config.Config
	log     *logger.Logger
	db      *database.Postgres
	redis   *cache.Redis
	metrics *metrics.Metrics
	router  *gin.Engine
	httpSrv *http.Server
}

// New creates and configures a new Server instance
func New(
	cfg *config.Config,
	log *logger.Logger,
	db *database.Postgres,
	redis *cache.Redis,
	m *metrics.Metrics,
) *Server {
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	s := &Server{
		cfg:     cfg,
		log:     log,
		db:      db,
		redis:   redis,
		metrics: m,
		router:  router,
	}

	s.setupMiddlewares()
	s.setupRoutes()

	s.httpSrv = &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return s
}

// Router returns the configured Gin engine (useful for handler and router testing)
func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) setupMiddlewares() {
	s.router.Use(
		middleware.RequestID(),
		middleware.Recovery(s.log),
		middleware.SecurityHeaders(s.cfg.App.Env == "production"),
		middleware.CORS(s.cfg.CORS.AllowedOrigins),
		middleware.RequestLogger(s.log, s.metrics),
	)
}

func (s *Server) setupRoutes() {
	healthHandler := health.NewHandler(s.db, s.redis)

	// Liveness check (process is up)
	s.router.GET("/health", healthHandler.Health)

	// Readiness check (dependencies available)
	s.router.GET("/ready", healthHandler.Ready)

	// Prometheus metrics endpoint
	if s.cfg.Metrics.Enabled && s.metrics != nil {
		s.router.GET(s.cfg.Metrics.Path, gin.WrapH(s.metrics.Handler()))
	}

	// Base API v1 group
	apiV1 := s.router.Group("/api/v1")
	{
		// Ping / heartbeat inside API v1
		apiV1.GET("/ping", func(c *gin.Context) {
			response.Success(c, gin.H{"pong": true}, "API v1 is active")
		})
	}

	// 404 handler returning standard JSON error
	s.router.NoRoute(func(c *gin.Context) {
		response.CustomError(c, http.StatusNotFound, "ROUTE_NOT_FOUND", fmt.Sprintf("path '%s' not found", c.Request.URL.Path))
	})
}

// Run starts the HTTP server and blocks until an interrupt signal is received for graceful shutdown
func (s *Server) Run() error {
	shutdownErr := make(chan error, 1)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit

		s.log.Info("shutdown signal received", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.App.ShutdownTimeout)
		defer cancel()

		var errs []error

		// Step 1: Stop accepting new HTTP requests and finish in-flight requests
		s.log.Info("stopping HTTP server")
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("http server shutdown: %w", err))
		}

		// Step 2: Close Redis client
		if s.redis != nil {
			if err := s.redis.Close(); err != nil {
				errs = append(errs, fmt.Errorf("redis close: %w", err))
			}
		}

		// Step 3: Close Postgres connection pool
		if s.db != nil {
			s.db.Close()
		}

		s.log.Info("all components stopped gracefully")

		if len(errs) > 0 {
			shutdownErr <- errors.Join(errs...)
		} else {
			shutdownErr <- nil
		}
	}()

	s.log.Info("starting HTTP server",
		slog.String("port", s.cfg.App.Port),
		slog.String("env", s.cfg.App.Env),
		slog.String("app_name", s.cfg.App.Name),
	)

	if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server failed: %w", err)
	}

	return <-shutdownErr
}
