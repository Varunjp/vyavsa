package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Varunjp/vyavsa/internal/domain"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlatformAdminPostgres implements repository.PlatformAdminRepository using pgxpool
type PlatformAdminPostgres struct {
	pool *pgxpool.Pool
}

// NewPlatformAdminPostgres creates a new PlatformAdminPostgres repository
func NewPlatformAdminPostgres(pool *pgxpool.Pool) *PlatformAdminPostgres {
	return &PlatformAdminPostgres{pool: pool}
}

func (r *PlatformAdminPostgres) GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error) {
	query := `
		SELECT id, username, email, COALESCE(phone, ''), status, password_hash, created_at, updated_at
		FROM platform_admin
		WHERE id = $1
	`
	var admin domain.PlatformAdmin
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&admin.ID,
		&admin.Username,
		&admin.Email,
		&admin.Phone,
		&admin.Status,
		&admin.PasswordHash,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("platform administrator not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get platform admin by id: %w", err))
	}

	return &admin, nil
}

func (r *PlatformAdminPostgres) GetByIdentifier(ctx context.Context, identifier string) (*domain.PlatformAdmin, error) {
	query := `
		SELECT id, username, email, COALESCE(phone, ''), status, password_hash, created_at, updated_at
		FROM platform_admin
		WHERE LOWER(username) = LOWER($1) OR LOWER(email) = LOWER($1)
	`
	var admin domain.PlatformAdmin
	err := r.pool.QueryRow(ctx, query, identifier).Scan(
		&admin.ID,
		&admin.Username,
		&admin.Email,
		&admin.Phone,
		&admin.Status,
		&admin.PasswordHash,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("platform administrator not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get platform admin by identifier: %w", err))
	}

	return &admin, nil
}

func (r *PlatformAdminPostgres) Create(ctx context.Context, admin *domain.PlatformAdmin) error {
	query := `
		INSERT INTO platform_admin (username, email, phone, status, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		admin.Username,
		admin.Email,
		admin.Phone,
		admin.Status,
		admin.PasswordHash,
	).Scan(&admin.ID, &admin.CreatedAt, &admin.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create platform admin: %w", err))
	}
	return nil
}

func (r *PlatformAdminPostgres) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE platform_admin
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	tag, err := r.pool.Exec(ctx, query, status, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to update platform admin status: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("platform administrator not found")
	}
	return nil
}
