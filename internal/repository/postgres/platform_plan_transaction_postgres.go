package postgres

import (
	"context"
	"errors"

	"github.com/Varunjp/vyavsa/internal/domain"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlatformPlanTransactionPostgres implements repository.PlatformPlanTransactionRepository using pgxpool
type PlatformPlanTransactionPostgres struct {
	pool *pgxpool.Pool
}

// NewPlatformPlanTransactionPostgres creates a new PlatformPlanTransactionPostgres repository
func NewPlatformPlanTransactionPostgres(pool *pgxpool.Pool) *PlatformPlanTransactionPostgres {
	return &PlatformPlanTransactionPostgres{pool: pool}
}

func (r *PlatformPlanTransactionPostgres) Create(ctx context.Context, tx *domain.PlatformPlanTransaction) error {
	query := `
		INSERT INTO platform_plan_transactions (tenant_id, transaction_id, payment_method, plan_id, plan_name, amount, status, failure_reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		tx.TenantID,
		tx.TransactionID,
		tx.PaymentMethod,
		tx.PlanID,
		tx.PlanName,
		tx.Amount,
		tx.Status,
		tx.FailureReason,
	).Scan(&tx.ID, &tx.CreatedAt, &tx.UpdatedAt)
	if err != nil {
		return MapDBError(err, "failed to record plan transaction")
	}
	return nil
}

func (r *PlatformPlanTransactionPostgres) ListByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.PlatformPlanTransaction, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	exec := GetExecutor(ctx, r.pool)

	countQuery := `SELECT COUNT(*) FROM platform_plan_transactions WHERE tenant_id = $1`
	var total int64
	if err := exec.QueryRow(ctx, countQuery, tenantID).Scan(&total); err != nil {
		return nil, 0, MapDBError(err, "failed to count plan transactions")
	}

	query := `
		SELECT t.id, t.tenant_id, t.transaction_id, t.payment_method, t.plan_id,
		       COALESCE(t.plan_name, p.plan_name, ''), t.amount, t.status, COALESCE(t.failure_reason, ''),
		       t.created_at, t.updated_at
		FROM platform_plan_transactions t
		LEFT JOIN platform_plans p ON p.id = t.plan_id
		WHERE t.tenant_id = $1
		ORDER BY t.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := exec.Query(ctx, query, tenantID, pageSize, offset)
	if err != nil {
		return nil, 0, MapDBError(err, "failed to list plan transactions")
	}
	defer rows.Close()

	items := make([]domain.PlatformPlanTransaction, 0, pageSize)
	for rows.Next() {
		var t domain.PlatformPlanTransaction
		if err := rows.Scan(
			&t.ID,
			&t.TenantID,
			&t.TransactionID,
			&t.PaymentMethod,
			&t.PlanID,
			&t.PlanName,
			&t.Amount,
			&t.Status,
			&t.FailureReason,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, 0, MapDBError(err, "failed to scan plan transaction")
		}
		items = append(items, t)
	}

	return items, total, nil
}

func (r *PlatformPlanTransactionPostgres) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformPlanTransaction, error) {
	query := `
		SELECT t.id, t.tenant_id, t.transaction_id, t.payment_method, t.plan_id,
		       COALESCE(t.plan_name, p.plan_name, ''), t.amount, t.status, COALESCE(t.failure_reason, ''),
		       t.created_at, t.updated_at
		FROM platform_plan_transactions t
		LEFT JOIN platform_plans p ON p.id = t.plan_id
		WHERE t.tenant_id = $1 AND t.id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var t domain.PlatformPlanTransaction
	err := exec.QueryRow(ctx, query, tenantID, id).Scan(
		&t.ID,
		&t.TenantID,
		&t.TransactionID,
		&t.PaymentMethod,
		&t.PlanID,
		&t.PlanName,
		&t.Amount,
		&t.Status,
		&t.FailureReason,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("plan transaction not found")
		}
		return nil, MapDBError(err, "failed to get plan transaction by id")
	}
	return &t, nil
}

func (r *PlatformPlanTransactionPostgres) GetByTransactionID(ctx context.Context, tenantID uuid.UUID, transactionID string) (*domain.PlatformPlanTransaction, error) {
	query := `
		SELECT t.id, t.tenant_id, t.transaction_id, t.payment_method, t.plan_id,
		       COALESCE(t.plan_name, p.plan_name, ''), t.amount, t.status, COALESCE(t.failure_reason, ''),
		       t.created_at, t.updated_at
		FROM platform_plan_transactions t
		LEFT JOIN platform_plans p ON p.id = t.plan_id
		WHERE t.tenant_id = $1 AND t.transaction_id = $2
	`
	exec := GetExecutor(ctx, r.pool)
	var t domain.PlatformPlanTransaction
	err := exec.QueryRow(ctx, query, tenantID, transactionID).Scan(
		&t.ID,
		&t.TenantID,
		&t.TransactionID,
		&t.PaymentMethod,
		&t.PlanID,
		&t.PlanName,
		&t.Amount,
		&t.Status,
		&t.FailureReason,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("plan transaction not found")
		}
		return nil, MapDBError(err, "failed to get plan transaction by transaction id")
	}
	return &t, nil
}
