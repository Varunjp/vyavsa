package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// PlatformSubscriptionRepository defines persistence contracts for tenant subscriptions
type PlatformSubscriptionRepository interface {
	Create(ctx context.Context, sub *domain.PlatformSubscription) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.PlatformSubscription, error)
	Update(ctx context.Context, sub *domain.PlatformSubscription) error
}
