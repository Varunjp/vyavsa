package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
)

// PlatformAdminRepository defines persistence contracts for platform administrators
type PlatformAdminRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error)
	GetByIdentifier(ctx context.Context, identifier string) (*domain.PlatformAdmin, error)
	Create(ctx context.Context, admin *domain.PlatformAdmin) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}
