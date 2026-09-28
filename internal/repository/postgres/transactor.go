package postgres

import (
	"context"
	"fmt"

	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txContextKey struct{}

// QueryExecutor abstracts executing SQL queries across a connection pool or a transaction
type QueryExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PostgresTransactor implements repository.Transactor
type PostgresTransactor struct {
	pool *pgxpool.Pool
}

// NewPostgresTransactor creates a new PostgresTransactor
func NewPostgresTransactor(pool *pgxpool.Pool) *PostgresTransactor {
	return &PostgresTransactor{pool: pool}
}

// WithinTransaction executes a closure inside an atomic PostgreSQL transaction
func (t *PostgresTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// If already in a transaction, reuse it
	if _, ok := ExtractTx(ctx); ok {
		return fn(ctx)
	}

	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to begin transaction: %w", err))
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txCtx := WithTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to commit transaction: %w", err))
	}

	return nil
}

// WithTx injects a pgx.Tx into the context
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txContextKey{}, tx)
}

// ExtractTx extracts an active pgx.Tx from the context if present
func ExtractTx(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(pgx.Tx)
	return tx, ok
}

// GetExecutor returns the active transaction from context or falls back to the default pool
func GetExecutor(ctx context.Context, fallback *pgxpool.Pool) QueryExecutor {
	if tx, ok := ExtractTx(ctx); ok {
		return tx
	}
	return fallback
}
