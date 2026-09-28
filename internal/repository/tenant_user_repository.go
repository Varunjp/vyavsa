package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// TenantUserRepository defines persistence contracts for tenant members
type TenantUserRepository interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantUser, error)
	GetByEmail(ctx context.Context, email string) (*domain.TenantUser, error)
	GetByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.TenantUser, error)
	GetAdminByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantUser, error)
	Create(ctx context.Context, user *domain.TenantUser) error
	UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error
}
