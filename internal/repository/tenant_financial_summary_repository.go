package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TenantFinancialSummaryRepository defines persistence contracts for authoritative tenant balances
type TenantFinancialSummaryRepository interface {
	Create(ctx context.Context, summary *domain.TenantFinancialSummary) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error)
	GetByTenantIDForUpdate(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error)
	Update(ctx context.Context, summary *domain.TenantFinancialSummary) error
	AdjustBalances(ctx context.Context, tenantID uuid.UUID, cashDelta, bankDelta, receivableDelta, payableDelta decimal.Decimal) error
	SyncFromSourceRecords(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error)
}
