package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// TenantFinancialSummaryRepository defines persistence contracts for authoritative tenant balances
type TenantFinancialSummaryRepository interface {
	Create(ctx context.Context, summary *domain.TenantFinancialSummary) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error)
	GetByTenantIDForUpdate(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error)
	Update(ctx context.Context, summary *domain.TenantFinancialSummary) error
}
