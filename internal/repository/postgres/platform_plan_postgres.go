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

// PlatformPlanPostgres implements repository.PlatformPlanRepository using pgxpool
type PlatformPlanPostgres struct {
	pool *pgxpool.Pool
}

// NewPlatformPlanPostgres creates a new PlatformPlanPostgres repository
func NewPlatformPlanPostgres(pool *pgxpool.Pool) *PlatformPlanPostgres {
	return &PlatformPlanPostgres{pool: pool}
}

func (r *PlatformPlanPostgres) Create(ctx context.Context, plan *domain.PlatformPlan) error {
	query := `
		INSERT INTO platform_plans (plan_name, note, price, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		plan.PlanName,
		plan.Note,
		plan.Price,
		plan.Status,
	).Scan(&plan.ID, &plan.CreatedAt, &plan.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create platform plan: %w", err))
	}
	return nil
}

func (r *PlatformPlanPostgres) GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformPlan, error) {
	query := `
		SELECT id, plan_name, COALESCE(note, ''), price, status, created_at, updated_at
		FROM platform_plans
		WHERE id = $1
	`
	exec := GetExecutor(ctx, r.pool)
	var plan domain.PlatformPlan
	err := exec.QueryRow(ctx, query, id).Scan(
		&plan.ID,
		&plan.PlanName,
		&plan.Note,
		&plan.Price,
		&plan.Status,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("subscription plan not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get subscription plan by id: %w", err))
	}
	return &plan, nil
}

func (r *PlatformPlanPostgres) GetByName(ctx context.Context, name string) (*domain.PlatformPlan, error) {
	query := `
		SELECT id, plan_name, COALESCE(note, ''), price, status, created_at, updated_at
		FROM platform_plans
		WHERE LOWER(plan_name) = LOWER($1)
	`
	exec := GetExecutor(ctx, r.pool)
	var plan domain.PlatformPlan
	err := exec.QueryRow(ctx, query, name).Scan(
		&plan.ID,
		&plan.PlanName,
		&plan.Note,
		&plan.Price,
		&plan.Status,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("subscription plan not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get subscription plan by name: %w", err))
	}
	return &plan, nil
}

func (r *PlatformPlanPostgres) GetDefaultFreePlan(ctx context.Context) (*domain.PlatformPlan, error) {
	query := `
		SELECT id, plan_name, COALESCE(note, ''), price, status, created_at, updated_at
		FROM platform_plans
		WHERE status = 'active' AND price = 0
		ORDER BY created_at ASC
		LIMIT 1
	`
	exec := GetExecutor(ctx, r.pool)
	var plan domain.PlatformPlan
	err := exec.QueryRow(ctx, query).Scan(
		&plan.ID,
		&plan.PlanName,
		&plan.Note,
		&plan.Price,
		&plan.Status,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("default free subscription plan not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get default free plan: %w", err))
	}
	return &plan, nil
}

func (r *PlatformPlanPostgres) List(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformPlan, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	exec := GetExecutor(ctx, r.pool)

	countQuery := `SELECT COUNT(*) FROM platform_plans WHERE ($1 = '' OR status = $1)`
	var total int64
	if err := exec.QueryRow(ctx, countQuery, status).Scan(&total); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to count platform plans: %w", err))
	}

	query := `
		SELECT id, plan_name, COALESCE(note, ''), price, status, created_at, updated_at
		FROM platform_plans
		WHERE ($1 = '' OR status = $1)
		ORDER BY price ASC, plan_name ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := exec.Query(ctx, query, status, pageSize, offset)
	if err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to list platform plans: %w", err))
	}
	defer rows.Close()

	plans := make([]domain.PlatformPlan, 0, pageSize)
	for rows.Next() {
		var p domain.PlatformPlan
		if err := rows.Scan(
			&p.ID,
			&p.PlanName,
			&p.Note,
			&p.Price,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, 0, appErrors.NewDatabase(fmt.Errorf("failed to scan platform plan: %w", err))
		}
		plans = append(plans, p)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, appErrors.NewDatabase(fmt.Errorf("error iterating platform plans: %w", err))
	}

	return plans, total, nil
}

func (r *PlatformPlanPostgres) Update(ctx context.Context, plan *domain.PlatformPlan) error {
	query := `
		UPDATE platform_plans
		SET plan_name = $1, note = $2, price = $3, status = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		plan.PlanName,
		plan.Note,
		plan.Price,
		plan.Status,
		plan.ID,
	).Scan(&plan.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("subscription plan not found")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update platform plan: %w", err))
	}
	return nil
}

func (r *PlatformPlanPostgres) Archive(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE platform_plans
		SET status = 'archived', updated_at = NOW()
		WHERE id = $1
	`
	exec := GetExecutor(ctx, r.pool)
	tag, err := exec.Exec(ctx, query, id)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to archive platform plan: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return appErrors.NewNotFound("subscription plan not found")
	}
	return nil
}
