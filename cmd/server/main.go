package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	Healthhandler "github.com/Varunjp/vyavsa/internal/handler/health"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	log := logger.NewLogger(logger.Config{
		Environment: cfg.AppEnv,
		ServiceName: cfg.ServiceName,
		Level:       cfg.LogLevel,
	})

	log.Zerolog.Info().
		Msg("starting application")

	router := gin.New()

	router.Use(
		logger.RequestLogger(*log),
		logger.Recovery(*log),
	)
	health := Healthhandler.NewHealthHandler()
	router.GET("/health", health.Health)

	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	go func() {
		log.Zerolog.Info().Str("port", cfg.AppPort).Msg("HTTP server started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Zerolog.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	waitForShutdown(server, log)
}

func waitForShutdown(server *http.Server, log *logger.Logger) {
	stop := make(chan os.Signal, 1)

	signal.Notify(
		stop,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-stop

	log.Zerolog.Info().Msg("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Zerolog.Error().Err(err).Msg("server shutdown failed")

		return
	}

	log.Zerolog.Info().Msg("server stopped")
}
