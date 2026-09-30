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

// PlatformSubscriptionPostgres implements repository.PlatformSubscriptionRepository using pgxpool
type PlatformSubscriptionPostgres struct {
	pool *pgxpool.Pool
}

// NewPlatformSubscriptionPostgres creates a new PlatformSubscriptionPostgres repository
func NewPlatformSubscriptionPostgres(pool *pgxpool.Pool) *PlatformSubscriptionPostgres {
	return &PlatformSubscriptionPostgres{pool: pool}
}

func (r *PlatformSubscriptionPostgres) Create(ctx context.Context, sub *domain.PlatformSubscription) error {
	query := `
		INSERT INTO platform_subscriptions (tenant_id, current_plan_id, current_plan_name, status, start_date, end_date)
		VALUES ($1, $2, $3, $4, COALESCE($5, NOW()), $6)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		sub.TenantID,
		sub.CurrentPlanID,
		sub.CurrentPlanName,
		sub.Status,
		sub.StartDate,
		sub.EndDate,
	).Scan(&sub.ID, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create platform subscription: %w", err))
	}
	return nil
}

func (r *PlatformSubscriptionPostgres) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.PlatformSubscription, error) {
	query := `
		SELECT id, tenant_id, current_plan_id, current_plan_name, status,
		       COALESCE(start_date, created_at), end_date, created_at, updated_at
		FROM platform_subscriptions
		WHERE tenant_id = $1
	`
	exec := GetExecutor(ctx, r.pool)
	var sub domain.PlatformSubscription
	err := exec.QueryRow(ctx, query, tenantID).Scan(
		&sub.ID,
		&sub.TenantID,
		&sub.CurrentPlanID,
		&sub.CurrentPlanName,
		&sub.Status,
		&sub.StartDate,
		&sub.EndDate,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("subscription not found for tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get platform subscription: %w", err))
	}
	return &sub, nil
}

func (r *PlatformSubscriptionPostgres) Update(ctx context.Context, sub *domain.PlatformSubscription) error {
	query := `
		UPDATE platform_subscriptions
		SET current_plan_id = $1, current_plan_name = $2, status = $3,
		    start_date = COALESCE($4, start_date, NOW()), end_date = $5, updated_at = NOW()
		WHERE tenant_id = $6
		RETURNING id, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		sub.CurrentPlanID,
		sub.CurrentPlanName,
		sub.Status,
		sub.StartDate,
		sub.EndDate,
		sub.TenantID,
	).Scan(&sub.ID, &sub.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("subscription not found for tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update platform subscription: %w", err))
	}
	return nil
}

func (r *PlatformSubscriptionPostgres) List(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformSubscriptionWithTenant, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	exec := GetExecutor(ctx, r.pool)

	countQuery := `
		SELECT COUNT(*)
		FROM platform_subscriptions s
		JOIN tenants t ON t.id = s.tenant_id
		WHERE ($1 = '' OR s.status = $1)
	`
	var total int64
	if err := exec.QueryRow(ctx, countQuery, status).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count platform subscriptions: %w", err))
	}

	query := `
		SELECT s.id, s.tenant_id, t.name, t.email, s.current_plan_id, s.current_plan_name,
		       s.status, s.start_date, s.end_date, s.created_at, s.updated_at
		FROM platform_subscriptions s
		JOIN tenants t ON t.id = s.tenant_id
		WHERE ($1 = '' OR s.status = $1)
		ORDER BY s.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := exec.Query(ctx, query, status, pageSize, offset)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list platform subscriptions: %w", err))
	}
	defer rows.Close()

	items := make([]domain.PlatformSubscriptionWithTenant, 0, pageSize)
	for rows.Next() {
		var item domain.PlatformSubscriptionWithTenant
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.TenantName,
			&item.TenantEmail,
			&item.CurrentPlanID,
			&item.CurrentPlanName,
			&item.Status,
			&item.StartDate,
			&item.EndDate,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan platform subscription: %w", err))
		}
		items = append(items, item)
	}

	return items, total, nil
}
