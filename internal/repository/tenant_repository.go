package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// TenantRepository defines persistence contracts for tenant entities
type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetByEmail(ctx context.Context, email string) (*domain.Tenant, error)
	List(ctx context.Context, page, pageSize int, status, search string) ([]domain.Tenant, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}
