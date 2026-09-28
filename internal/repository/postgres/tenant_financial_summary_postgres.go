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
