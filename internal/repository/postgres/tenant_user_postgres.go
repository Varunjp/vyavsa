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

// TenantUserPostgres implements repository.TenantUserRepository using pgxpool
type TenantUserPostgres struct {
	pool *pgxpool.Pool
}

// NewTenantUserPostgres creates a new TenantUserPostgres repository
func NewTenantUserPostgres(pool *pgxpool.Pool) *TenantUserPostgres {
	return &TenantUserPostgres{pool: pool}
}

func (r *TenantUserPostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantUser, error) {
	query := `
		SELECT id, tenant_id, name, role, email, password_hash, status, created_at, updated_at
		FROM tenant_user
		WHERE tenant_id = $1 AND id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var user domain.TenantUser
	err := exec.QueryRow(ctx, query, tenantID, id).Scan(
		&user.ID,
		&user.TenantID,
		&user.Name,
		&user.Role,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("user not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant user: %w", err))
	}
	return &user, nil
}

func (r *TenantUserPostgres) GetByEmail(ctx context.Context, email string) (*domain.TenantUser, error) {
	query := `
		SELECT id, tenant_id, name, role, email, password_hash, status, created_at, updated_at
		FROM tenant_user
		WHERE LOWER(email) = LOWER($1)
		LIMIT 1
	`
	exec := GetExecutor(ctx, r.pool)
	var user domain.TenantUser
	err := exec.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.TenantID,
		&user.Name,
		&user.Role,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("user not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get user by email: %w", err))
	}
	return &user, nil
}

func (r *TenantUserPostgres) GetByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.TenantUser, error) {
	query := `
		SELECT id, tenant_id, name, role, email, password_hash, status, created_at, updated_at
		FROM tenant_user
		WHERE tenant_id = $1 AND LOWER(email) = LOWER($2)
	`
	exec := GetExecutor(ctx, r.pool)
	var user domain.TenantUser
	err := exec.QueryRow(ctx, query, tenantID, email).Scan(
		&user.ID,
		&user.TenantID,
		&user.Name,
		&user.Role,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("user not found within tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant user by email: %w", err))
	}
	return &user, nil
}

func (r *TenantUserPostgres) GetAdminByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantUser, error) {
	query := `
		SELECT id, tenant_id, name, role, email, password_hash, status, created_at, updated_at
		FROM tenant_user
		WHERE tenant_id = $1 AND role = 'admin'
		ORDER BY created_at ASC
		LIMIT 1
	`
	exec := GetExecutor(ctx, r.pool)
	var user domain.TenantUser
	err := exec.QueryRow(ctx, query, tenantID).Scan(
		&user.ID,
		&user.TenantID,
		&user.Name,
		&user.Role,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("admin user not found for tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant admin user: %w", err))
	}
	return &user, nil
}

func (r *TenantUserPostgres) Create(ctx context.Context, user *domain.TenantUser) error {
	query := `
		INSERT INTO tenant_user (tenant_id, name, role, email, password_hash, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		user.TenantID,
		user.Name,
		user.Role,
		user.Email,
		user.PasswordHash,
		user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create tenant user: %w", err))
	}
	return nil
}

func (r *TenantUserPostgres) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	query := `
		UPDATE tenant_user
		SET status = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
	`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, status, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to update tenant user status: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("tenant user not found")
	}
	return nil
}

func (r *TenantUserPostgres) UpdatePassword(ctx context.Context, tenantID, id uuid.UUID, passwordHash string) error {
	query := `
		UPDATE tenant_user
		SET password_hash = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = $3
	`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, passwordHash, tenantID, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to update tenant user password: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("tenant user not found")
	}
	return nil
}
