package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/database"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "application fatal error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Load strongly typed configuration with fail-fast validation
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// 2. Initialize structured logger
	appLogger := logger.New(logger.Config{
		Environment: cfg.App.Env,
		ServiceName: cfg.App.Name,
		Level:       cfg.Log.Level,
		Format:      cfg.Log.Format,
	})

	appLogger.Info("starting vyavsa bill book saas backend",
		slog.String("version", "1.0.0"),
		slog.String("environment", cfg.App.Env),
		slog.String("port", cfg.App.Port),
	)

	ctx := context.Background()

	// 3. Initialize PostgreSQL connection pool
	pg, err := database.NewPostgres(ctx, cfg.Database, appLogger.Logger)
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}
	defer pg.Close()

	// 4. Initialize Redis cache client (allows degraded mode if unavailable)
	redisClient, err := cache.NewRedis(ctx, cfg.Redis, appLogger.Logger)
	if err != nil {
		appLogger.Warn("redis client initialization warning", slog.String("error", err.Error()))
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	// 5. Initialize Prometheus metrics & register DB pool collector
	appMetrics := metrics.New()
	if pg.Pool != nil {
		appMetrics.RegisterDBPoolMetrics(pg.Pool)
	}

	// 6. Run database migrations if enabled
	if cfg.Database.AutoMigrate {
		appLogger.Info("running automated database migrations")
		migrator := database.NewMigrator(pg.Pool, "migrations", appLogger.Logger)
		if err := migrator.Up(ctx); err != nil {
			return fmt.Errorf("failed to run database migrations: %w", err)
		}
	}

	// 7. Initialize and start HTTP server with graceful shutdown
	srv := server.New(cfg, appLogger, pg, redisClient, appMetrics)
	return srv.Run()
}
