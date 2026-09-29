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
		return nil, MapDBError(err, "failed to get tenant user")
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
		return nil, MapDBError(err, "failed to get user by email")
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
		return nil, MapDBError(err, "failed to get tenant user by email")
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
		return nil, MapDBError(err, "failed to get tenant admin user")
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
		return MapDBError(err, "failed to create tenant user")
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
		return MapDBError(err, "failed to update tenant user status")
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
		return MapDBError(err, "failed to update tenant user password")
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("tenant user not found")
	}
	return nil
}

func (r *TenantUserPostgres) Update(ctx context.Context, user *domain.TenantUser) error {
	query := `
		UPDATE tenant_user
		SET name = $1, role = $2, status = $3, updated_at = NOW()
		WHERE tenant_id = $4 AND id = $5
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query, user.Name, user.Role, user.Status, user.TenantID, user.ID).Scan(&user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("tenant user not found")
		}
		return MapDBError(err, "failed to update tenant user")
	}
	return nil
}

func (r *TenantUserPostgres) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	query := `DELETE FROM tenant_user WHERE tenant_id = $1 AND id = $2`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, tenantID, id)
	if err != nil {
		return MapDBError(err, "failed to delete tenant user")
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("tenant user not found")
	}
	return nil
}

func (r *TenantUserPostgres) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, role, status string) ([]domain.TenantUser, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseWhere := "WHERE tenant_id = $1"
	args := []any{tenantID}
	argIdx := 2

	if role != "" {
		baseWhere += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, role)
		argIdx++
	}

	if status != "" {
		baseWhere += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	if search != "" {
		baseWhere += fmt.Sprintf(" AND (name ILIKE $%d OR email ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM tenant_user %s", baseWhere)
	exec := GetExecutor(ctx, r.pool)
	var total int64
	if err := exec.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, MapDBError(err, "failed to count tenant users")
	}

	listQuery := fmt.Sprintf(`
		SELECT id, tenant_id, name, role, email, password_hash, status, created_at, updated_at
		FROM tenant_user
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseWhere, argIdx, argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := exec.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, MapDBError(err, "failed to list tenant users")
	}
	defer rows.Close()

	users := make([]domain.TenantUser, 0)
	for rows.Next() {
		var u domain.TenantUser
		if err := rows.Scan(
			&u.ID,
			&u.TenantID,
			&u.Name,
			&u.Role,
			&u.Email,
			&u.PasswordHash,
			&u.Status,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, 0, MapDBError(err, "failed to scan tenant user")
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, MapDBError(err, "error iterating tenant users")
	}

	return users, total, nil
}
