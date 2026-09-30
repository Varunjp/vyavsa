package repository

import (
	"context"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// TenantPlanCacheRepository defines caching operations for tenant subscription and plan status
type TenantPlanCacheRepository interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.CachedTenantPlan, error)
	Set(ctx context.Context, tenantID uuid.UUID, plan *domain.CachedTenantPlan, ttl time.Duration) error
	Delete(ctx context.Context, tenantID uuid.UUID) error
}
