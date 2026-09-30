package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
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

func (r *PlatformPlanTransactionPostgres) ListAll(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformPlanTransactionWithTenant, int64, error) {
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
		FROM platform_plan_transactions tx
		JOIN tenants t ON t.id = tx.tenant_id
		WHERE ($1 = '' OR tx.status = $1)
	`
	var total int64
	if err := exec.QueryRow(ctx, countQuery, status).Scan(&total); err != nil {
		return nil, 0, MapDBError(err, "failed to count platform plan transactions")
	}

	query := `
		SELECT tx.id, tx.tenant_id, t.name, t.email, tx.transaction_id, tx.payment_method,
		       tx.plan_id, COALESCE(tx.plan_name, p.plan_name, ''), tx.amount, tx.status,
		       COALESCE(tx.failure_reason, ''), tx.created_at, tx.updated_at
		FROM platform_plan_transactions tx
		JOIN tenants t ON t.id = tx.tenant_id
		LEFT JOIN platform_plans p ON p.id = tx.plan_id
		WHERE ($1 = '' OR tx.status = $1)
		ORDER BY tx.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := exec.Query(ctx, query, status, pageSize, offset)
	if err != nil {
		return nil, 0, MapDBError(err, "failed to list platform plan transactions")
	}
	defer rows.Close()

	items := make([]domain.PlatformPlanTransactionWithTenant, 0, pageSize)
	for rows.Next() {
		var item domain.PlatformPlanTransactionWithTenant
		if err := rows.Scan(
			&item.ID,
			&item.TenantID,
			&item.TenantName,
			&item.TenantEmail,
			&item.TransactionID,
			&item.PaymentMethod,
			&item.PlanID,
			&item.PlanName,
			&item.Amount,
			&item.Status,
			&item.FailureReason,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, 0, MapDBError(err, "failed to scan platform plan transaction")
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (r *PlatformPlanTransactionPostgres) GetMonthlyReceivedIncome(ctx context.Context, since time.Time) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM platform_plan_transactions
		WHERE status = 'completed' AND created_at >= $1
	`
	exec := GetExecutor(ctx, r.pool)
	var total decimal.Decimal
	if err := exec.QueryRow(ctx, query, since).Scan(&total); err != nil {
		return decimal.Zero, MapDBError(err, "failed to get monthly received income")
	}
	return total, nil
}

func (r *PlatformPlanTransactionPostgres) GetMonthlyRevenueTrend(ctx context.Context, since time.Time) ([]repository.MonthlyRevenueAggregate, error) {
	query := `
		SELECT to_char(date_trunc('month', created_at), 'YYYY-MM') AS month_key,
		       to_char(date_trunc('month', created_at), 'Mon YYYY') AS month_label,
		       COALESCE(SUM(amount), 0) AS revenue,
		       COUNT(*) AS tx_count
		FROM platform_plan_transactions
		WHERE status = 'completed' AND created_at >= $1
		GROUP BY date_trunc('month', created_at)
		ORDER BY date_trunc('month', created_at) ASC
	`
	exec := GetExecutor(ctx, r.pool)
	rows, err := exec.Query(ctx, query, since)
	if err != nil {
		return nil, MapDBError(err, "failed to query monthly revenue trend")
	}
	defer rows.Close()

	var results []repository.MonthlyRevenueAggregate
	for rows.Next() {
		var agg repository.MonthlyRevenueAggregate
		if err := rows.Scan(&agg.MonthKey, &agg.MonthLabel, &agg.Revenue, &agg.Count); err != nil {
			return nil, MapDBError(err, "failed to scan monthly revenue trend item")
		}
		results = append(results, agg)
	}
	return results, nil
}
