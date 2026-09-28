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

// TenantPostgres implements repository.TenantRepository using pgxpool
type TenantPostgres struct {
	pool *pgxpool.Pool
}

// NewTenantPostgres creates a new TenantPostgres repository
func NewTenantPostgres(pool *pgxpool.Pool) *TenantPostgres {
	return &TenantPostgres{pool: pool}
}

func (r *TenantPostgres) Create(ctx context.Context, tenant *domain.Tenant) error {
	query := `
		INSERT INTO tenants (name, email, phone, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		tenant.Name,
		tenant.Email,
		tenant.Phone,
		tenant.Status,
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create tenant: %w", err))
	}
	return nil
}

func (r *TenantPostgres) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	query := `
		SELECT id, name, email, COALESCE(phone, ''), status, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`
	exec := GetExecutor(ctx, r.pool)
	var t domain.Tenant
	err := exec.QueryRow(ctx, query, id).Scan(
		&t.ID,
		&t.Name,
		&t.Email,
		&t.Phone,
		&t.Status,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("tenant not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant by id: %w", err))
	}
	return &t, nil
}

func (r *TenantPostgres) GetByEmail(ctx context.Context, email string) (*domain.Tenant, error) {
	query := `
		SELECT id, name, email, COALESCE(phone, ''), status, created_at, updated_at
		FROM tenants
		WHERE LOWER(email) = LOWER($1)
	`
	exec := GetExecutor(ctx, r.pool)
	var t domain.Tenant
	err := exec.QueryRow(ctx, query, email).Scan(
		&t.ID,
		&t.Name,
		&t.Email,
		&t.Phone,
		&t.Status,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("tenant not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant by email: %w", err))
	}
	return &t, nil
}

func (r *TenantPostgres) List(ctx context.Context, page, pageSize int, status, search string) ([]domain.Tenant, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	exec := GetExecutor(ctx, r.pool)
	searchPattern := "%" + search + "%"

	countQuery := `
		SELECT COUNT(*)
		FROM tenants
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR name ILIKE $3 OR email ILIKE $3 OR phone ILIKE $3)
	`
	var total int64
	if err := exec.QueryRow(ctx, countQuery, status, search, searchPattern).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count tenants: %w", err))
	}

	query := `
		SELECT id, name, email, COALESCE(phone, ''), status, created_at, updated_at
		FROM tenants
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR name ILIKE $3 OR email ILIKE $3 OR phone ILIKE $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := exec.Query(ctx, query, status, search, searchPattern, pageSize, offset)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list tenants: %w", err))
	}
	defer rows.Close()

	tenants := make([]domain.Tenant, 0, pageSize)
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Email,
			&t.Phone,
			&t.Status,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan tenant: %w", err))
		}
		tenants = append(tenants, t)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("error iterating tenants: %w", err))
	}

	return tenants, total, nil
}

func (r *TenantPostgres) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `
		UPDATE tenants
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, status, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to update tenant status: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("tenant not found")
	}
	return nil
}
