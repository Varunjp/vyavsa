package database

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migration represents a single SQL migration file
type Migration struct {
	Version  int64
	Name     string
	Filename string
	SQL      string
}

// Migrator handles database schema migrations
type Migrator struct {
	pool           *pgxpool.Pool
	migrationsPath string
	log            *slog.Logger
}

// NewMigrator creates a new Migrator instance
func NewMigrator(pool *pgxpool.Pool, migrationsPath string, log *slog.Logger) *Migrator {
	return &Migrator{
		pool:           pool,
		migrationsPath: migrationsPath,
		log:            log,
	}
}

// Up applies all pending up migrations
func (m *Migrator) Up(ctx context.Context) error {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	applied, err := m.getAppliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch applied migrations: %w", err)
	}

	migrations, err := m.loadMigrations(".up.sql")
	if err != nil {
		return fmt.Errorf("failed to load up migrations: %w", err)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	appliedCount := 0
	for _, mig := range migrations {
		if applied[mig.Version] {
			continue
		}

		m.log.Info("applying database migration",
			slog.Int64("version", mig.Version),
			slog.String("name", mig.Name),
		)

		if err := m.applyMigration(ctx, mig); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", mig.Version, mig.Name, err)
		}

		appliedCount++
		m.log.Info("database migration applied successfully",
			slog.Int64("version", mig.Version),
			slog.String("name", mig.Name),
		)
	}

	if appliedCount == 0 {
		m.log.Info("database schema is up to date, no migrations applied")
	} else {
		m.log.Info("all pending migrations applied", slog.Int("count", appliedCount))
	}

	return nil
}

// Down rolls back the latest applied migration
func (m *Migrator) Down(ctx context.Context) error {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}

	latestVersion, err := m.getLatestAppliedVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest applied migration: %w", err)
	}

	if latestVersion == 0 {
		m.log.Info("no applied migrations to roll back")
		return nil
	}

	migrations, err := m.loadMigrations(".down.sql")
	if err != nil {
		return fmt.Errorf("failed to load down migrations: %w", err)
	}

	var target *Migration
	for _, mig := range migrations {
		if mig.Version == latestVersion {
			target = &mig
			break
		}
	}

	if target == nil {
		return fmt.Errorf("no matching down migration found for version %d", latestVersion)
	}

	m.log.Info("rolling back database migration",
		slog.Int64("version", target.Version),
		slog.String("name", target.Name),
	)

	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin rollback transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, target.SQL); err != nil {
		return fmt.Errorf("failed to execute rollback SQL: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", target.Version); err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit rollback transaction: %w", err)
	}

	m.log.Info("migration rolled back successfully",
		slog.Int64("version", target.Version),
		slog.String("name", target.Name),
	)

	return nil
}

func (m *Migrator) ensureMigrationTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`
	_, err := m.pool.Exec(ctx, query)
	return err
}

func (m *Migrator) getAppliedVersions(ctx context.Context) (map[int64]bool, error) {
	rows, err := m.pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func (m *Migrator) getLatestAppliedVersion(ctx context.Context) (int64, error) {
	var version int64
	err := m.pool.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return version, nil
}

func (m *Migrator) applyMigration(ctx context.Context, mig Migration) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, mig.SQL); err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	_, err = tx.Exec(ctx, "INSERT INTO schema_migrations (version, name, applied_at) VALUES ($1, $2, $3)",
		mig.Version, mig.Name, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit(ctx)
}

func (m *Migrator) loadMigrations(suffix string) ([]Migration, error) {
	files, err := os.ReadDir(m.migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations dir '%s': %w", m.migrationsPath, err)
	}

	var migrations []Migration
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), suffix) {
			continue
		}

		parts := strings.SplitN(f.Name(), "_", 2)
		if len(parts) < 2 {
			continue
		}

		version, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			m.log.Warn("skipping migration file with invalid version prefix", slog.String("file", f.Name()), slog.String("error", err.Error()))
			continue
		}

		name := strings.TrimSuffix(parts[1], suffix)
		fullPath := filepath.Join(m.migrationsPath, f.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", fullPath, err)
		}

		migrations = append(migrations, Migration{
			Version:  version,
			Name:     name,
			Filename: f.Name(),
			SQL:      string(content),
		})
	}

	return migrations, nil
}
