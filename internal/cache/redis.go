package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/redis/go-redis/v9"
)

// Redis wraps redis.Client with lifecycle and health methods
type Redis struct {
	Client *redis.Client
	log    *slog.Logger
}

// NewRedis initializes a Redis client and tests connectivity
func NewRedis(ctx context.Context, cfg config.RedisConfig, log *slog.Logger) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	log.Info("connecting to redis",
		slog.String("addr", cfg.Addr()),
		slog.Int("db", cfg.DB),
	)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx).Err(); err != nil {
		log.Warn("failed to ping redis on startup, running in degraded cache mode",
			slog.String("error", err.Error()),
		)
		// We still return the client wrapper so callers can ping or retry without panic
		return &Redis{
			Client: client,
			log:    log,
		}, fmt.Errorf("redis ping failed: %w", err)
	}

	log.Info("connected to redis successfully")

	return &Redis{
		Client: client,
		log:    log,
	}, nil
}

// Ping checks if Redis is responsive
func (r *Redis) Ping(ctx context.Context) error {
	if r.Client == nil {
		return fmt.Errorf("redis client is nil")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return r.Client.Ping(pingCtx).Err()
}

// Close closes the Redis client connection
func (r *Redis) Close() error {
	if r.Client != nil {
		r.log.Info("closing redis client")
		return r.Client.Close()
	}
	return nil
}
