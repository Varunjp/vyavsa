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
	"github.com/shopspring/decimal"
)

// TenantFinancialSummaryPostgres implements repository.TenantFinancialSummaryRepository using pgxpool
type TenantFinancialSummaryPostgres struct {
	pool *pgxpool.Pool
}

// NewTenantFinancialSummaryPostgres creates a new TenantFinancialSummaryPostgres repository
func NewTenantFinancialSummaryPostgres(pool *pgxpool.Pool) *TenantFinancialSummaryPostgres {
	return &TenantFinancialSummaryPostgres{pool: pool}
}

func (r *TenantFinancialSummaryPostgres) Create(ctx context.Context, summary *domain.TenantFinancialSummary) error {
	query := `
		INSERT INTO tenant_financial_summary (tenant_id, cash_balance, bank_balance, total_receivable, total_payable)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		summary.TenantID,
		summary.CashBalance,
		summary.BankBalance,
		summary.TotalReceivable,
		summary.TotalPayable,
	).Scan(&summary.ID, &summary.CreatedAt, &summary.UpdatedAt)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to create tenant financial summary: %w", err))
	}
	return nil
}

func (r *TenantFinancialSummaryPostgres) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	query := `
		SELECT id, tenant_id, cash_balance, bank_balance, total_receivable, total_payable, created_at, updated_at
		FROM tenant_financial_summary
		WHERE tenant_id = $1
	`
	exec := GetExecutor(ctx, r.pool)
	var s domain.TenantFinancialSummary
	err := exec.QueryRow(ctx, query, tenantID).Scan(
		&s.ID,
		&s.TenantID,
		&s.CashBalance,
		&s.BankBalance,
		&s.TotalReceivable,
		&s.TotalPayable,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("financial summary not found for tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant financial summary: %w", err))
	}
	return &s, nil
}

func (r *TenantFinancialSummaryPostgres) GetByTenantIDForUpdate(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	query := `
		SELECT id, tenant_id, cash_balance, bank_balance, total_receivable, total_payable, created_at, updated_at
		FROM tenant_financial_summary
		WHERE tenant_id = $1
		FOR UPDATE
	`
	exec := GetExecutor(ctx, r.pool)
	var s domain.TenantFinancialSummary
	err := exec.QueryRow(ctx, query, tenantID).Scan(
		&s.ID,
		&s.TenantID,
		&s.CashBalance,
		&s.BankBalance,
		&s.TotalReceivable,
		&s.TotalPayable,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("financial summary not found for tenant")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to get tenant financial summary for update: %w", err))
	}
	return &s, nil
}

func (r *TenantFinancialSummaryPostgres) Update(ctx context.Context, summary *domain.TenantFinancialSummary) error {
	query := `
		UPDATE tenant_financial_summary
		SET cash_balance = $1, bank_balance = $2, total_receivable = $3, total_payable = $4, updated_at = NOW()
		WHERE tenant_id = $5
		RETURNING id, updated_at
	`
	exec := GetExecutor(ctx, r.pool)
	err := exec.QueryRow(ctx, query,
		summary.CashBalance,
		summary.BankBalance,
		summary.TotalReceivable,
		summary.TotalPayable,
		summary.TenantID,
	).Scan(&summary.ID, &summary.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appErrors.NewNotFound("financial summary not found for tenant")
		}
		return appErrors.NewDatabase(fmt.Errorf("failed to update tenant financial summary: %w", err))
	}
	return nil
}

func (r *TenantFinancialSummaryPostgres) SyncFromSourceRecords(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	exec := GetExecutor(ctx, r.pool)

	query := `
		WITH derived AS (
			SELECT
				COALESCE((SELECT SUM(current_balance) FROM tenant_bank WHERE tenant_id = $1 AND status = 'active'), 0) AS bank_balance,
				COALESCE((SELECT SUM(current_balance) FROM tenant_customer WHERE tenant_id = $1 AND current_balance > 0), 0) +
				COALESCE((SELECT SUM(account) FROM counter_sale WHERE tenant_id = $1 AND account > 0), 0) AS total_receivable,
				COALESCE((SELECT SUM(total_pending) FROM tenant_purchase WHERE tenant_id = $1), 0) AS total_payable
		)
		INSERT INTO tenant_financial_summary (tenant_id, cash_balance, bank_balance, total_receivable, total_payable)
		SELECT $1, 0.00, d.bank_balance, d.total_receivable, d.total_payable
		FROM derived d
		ON CONFLICT (tenant_id) DO UPDATE
		SET bank_balance = EXCLUDED.bank_balance,
		    total_receivable = EXCLUDED.total_receivable,
		    total_payable = EXCLUDED.total_payable,
		    updated_at = NOW()
		RETURNING id, tenant_id, cash_balance, bank_balance, total_receivable, total_payable, created_at, updated_at
	`
	var s domain.TenantFinancialSummary
	err := exec.QueryRow(ctx, query, tenantID).Scan(
		&s.ID,
		&s.TenantID,
		&s.CashBalance,
		&s.BankBalance,
		&s.TotalReceivable,
		&s.TotalPayable,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to sync financial summary from source records: %w", err))
	}
	return &s, nil
}

// AdjustBalances applies atomic arithmetic updates to a tenant's financial summary.
// If the summary row does not exist yet, it is initialized with the given values.
func (r *TenantFinancialSummaryPostgres) AdjustBalances(ctx context.Context, tenantID uuid.UUID, cashDelta, bankDelta, receivableDelta, payableDelta decimal.Decimal) error {
	query := `
		INSERT INTO tenant_financial_summary (
			tenant_id, cash_balance, bank_balance, total_receivable, total_payable, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (tenant_id) DO UPDATE
		SET
			cash_balance = tenant_financial_summary.cash_balance + EXCLUDED.cash_balance,
			bank_balance = tenant_financial_summary.bank_balance + EXCLUDED.bank_balance,
			total_receivable = tenant_financial_summary.total_receivable + EXCLUDED.total_receivable,
			total_payable = tenant_financial_summary.total_payable + EXCLUDED.total_payable,
			updated_at = NOW()
	`
	exec := GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx, query, tenantID, cashDelta, bankDelta, receivableDelta, payableDelta)
	if err != nil {
		return appErrors.NewDatabase(fmt.Errorf("failed to adjust tenant financial summary atomically: %w", err))
	}
	return nil
}
