package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/database"
	"github.com/Varunjp/vyavsa/internal/logger"
)

func main() {
	var (
		direction = flag.String("direction", "up", "Migration direction: up or down")
		path      = flag.String("path", "migrations", "Path to migrations directory")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	appLogger := logger.New(logger.Config{
		Environment: cfg.App.Env,
		ServiceName: cfg.App.Name + "-migrator",
		Level:       cfg.Log.Level,
		Format:      cfg.Log.Format,
	})

	ctx := context.Background()
	pg, err := database.NewPostgres(ctx, cfg.Database, appLogger.Logger)
	if err != nil {
		appLogger.Error("failed to connect to postgresql", "error", err)
		os.Exit(1)
	}
	defer pg.Close()

	migrator := database.NewMigrator(pg.Pool, *path, appLogger.Logger)

	switch *direction {
	case "up":
		if err := migrator.Up(ctx); err != nil {
			appLogger.Error("migration up failed", "error", err)
			os.Exit(1)
		}
		appLogger.Info("migration up completed successfully")
	case "down":
		if err := migrator.Down(ctx); err != nil {
			appLogger.Error("migration down failed", "error", err)
			os.Exit(1)
		}
		appLogger.Info("migration down completed successfully")
	default:
		appLogger.Error("invalid direction, must be 'up' or 'down'", "direction", *direction)
		os.Exit(1)
	}
}
