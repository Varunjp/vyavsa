package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// PlatformPlanRepository defines persistence contracts for SaaS subscription plans
type PlatformPlanRepository interface {
	Create(ctx context.Context, plan *domain.PlatformPlan) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformPlan, error)
	GetByName(ctx context.Context, name string) (*domain.PlatformPlan, error)
	List(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformPlan, int64, error)
	Update(ctx context.Context, plan *domain.PlatformPlan) error
	Archive(ctx context.Context, id uuid.UUID) error
}
