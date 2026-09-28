package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Postgres wraps the pgxpool.Pool instance with metrics and lifecycle methods
type Postgres struct {
	Pool *pgxpool.Pool
	log  *slog.Logger
}

// NewPostgres initializes a new PostgreSQL connection pool
func NewPostgres(ctx context.Context, cfg config.DatabaseConfig, log *slog.Logger) (*Postgres, error) {
	connStr := cfg.ConnectionString()

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	log.Info("connecting to postgresql",
		slog.String("host", cfg.Host),
		slog.String("port", cfg.Port),
		slog.String("database", cfg.Name),
		slog.Int("max_conns", int(cfg.MaxConns)),
		slog.Int("min_conns", int(cfg.MinConns)),
	)

	// Attempt connection with timeout
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres connection pool: %w", err)
	}

	// Verify connectivity
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	log.Info("connected to postgresql successfully")

	return &Postgres{
		Pool: pool,
		log:  log,
	}, nil
}

// Ping checks if the PostgreSQL database is reachable
func (p *Postgres) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return p.Pool.Ping(pingCtx)
}

// Stats returns connection pool statistics
func (p *Postgres) Stats() *pgxpool.Stat {
	return p.Pool.Stat()
}

// Close gracefully closes the connection pool
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.log.Info("closing postgres connection pool")
		p.Pool.Close()
	}
}
