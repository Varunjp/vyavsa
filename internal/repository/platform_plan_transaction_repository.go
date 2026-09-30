package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// PlatformPlanTransactionRepository defines persistence contracts for SaaS plan payments and transactions
type PlatformPlanTransactionRepository interface {
	Create(ctx context.Context, tx *domain.PlatformPlanTransaction) error
	ListByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.PlatformPlanTransaction, int64, error)
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformPlanTransaction, error)
	GetByTransactionID(ctx context.Context, tenantID uuid.UUID, transactionID string) (*domain.PlatformPlanTransaction, error)
}
