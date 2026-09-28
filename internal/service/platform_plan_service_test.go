package service

import (
	"context"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/logger"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPlanRepo struct {
	plans map[uuid.UUID]*domain.PlatformPlan
}

func newMockPlanRepo() *mockPlanRepo {
	return &mockPlanRepo{
		plans: make(map[uuid.UUID]*domain.PlatformPlan),
	}
}

func (m *mockPlanRepo) Create(ctx context.Context, plan *domain.PlatformPlan) error {
	plan.ID = uuid.New()
	plan.CreatedAt = time.Now().UTC()
	plan.UpdatedAt = time.Now().UTC()
	m.plans[plan.ID] = plan
	return nil
}

func (m *mockPlanRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformPlan, error) {
	p, ok := m.plans[id]
	if !ok {
		return nil, appErrors.NewNotFound("plan not found")
	}
	return p, nil
}

func (m *mockPlanRepo) GetByName(ctx context.Context, name string) (*domain.PlatformPlan, error) {
	for _, p := range m.plans {
		if p.PlanName == name {
			return p, nil
		}
	}
	return nil, appErrors.NewNotFound("plan not found")
}

func (m *mockPlanRepo) GetDefaultFreePlan(ctx context.Context) (*domain.PlatformPlan, error) {
	for _, p := range m.plans {
		if p.Status == "active" && p.Price.IsZero() {
			return p, nil
		}
	}
	return nil, appErrors.NewNotFound("default free plan not found")
}

func (m *mockPlanRepo) List(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformPlan, int64, error) {
	var result []domain.PlatformPlan
	for _, p := range m.plans {
		if status == "" || p.Status == status {
			result = append(result, *p)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockPlanRepo) Update(ctx context.Context, plan *domain.PlatformPlan) error {
	if _, ok := m.plans[plan.ID]; !ok {
		return appErrors.NewNotFound("plan not found")
	}
	plan.UpdatedAt = time.Now().UTC()
	m.plans[plan.ID] = plan
	return nil
}

func (m *mockPlanRepo) Archive(ctx context.Context, id uuid.UUID) error {
	p, ok := m.plans[id]
	if !ok {
		return appErrors.NewNotFound("plan not found")
	}
	p.Status = "archived"
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func TestPlatformPlanService(t *testing.T) {
	ctx := context.Background()
	log := logger.Default().Logger

	t.Run("CreatePlan success", func(t *testing.T) {
		repo := newMockPlanRepo()
		svc := NewPlatformPlanService(repo, log)

		req := dto.CreatePlanRequest{
			PlanName: "Pro Tier",
			Note:     "Professional subscription",
			Price:    decimal.NewFromFloat(799.00),
			Status:   "active",
		}

		resp, err := svc.CreatePlan(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "Pro Tier", resp.PlanName)
		assert.Equal(t, decimal.NewFromFloat(799.00), resp.Price)
		assert.Equal(t, "active", resp.Status)
		assert.NotEqual(t, uuid.Nil, resp.ID)
	})

	t.Run("CreatePlan fails with negative price", func(t *testing.T) {
		repo := newMockPlanRepo()
		svc := NewPlatformPlanService(repo, log)

		req := dto.CreatePlanRequest{
			PlanName: "Negative Plan",
			Price:    decimal.NewFromFloat(-10.00),
		}

		_, err := svc.CreatePlan(ctx, req)
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.CodeValidation, appErr.Code)
	})

	t.Run("CreatePlan fails on duplicate plan name", func(t *testing.T) {
		repo := newMockPlanRepo()
		svc := NewPlatformPlanService(repo, log)

		req := dto.CreatePlanRequest{
			PlanName: "Duplicate Plan",
			Price:    decimal.NewFromFloat(100.00),
		}

		_, err := svc.CreatePlan(ctx, req)
		require.NoError(t, err)

		_, err = svc.CreatePlan(ctx, req)
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.CodeConflict, appErr.Code)
	})

	t.Run("GetPlanByID returns existing plan", func(t *testing.T) {
		repo := newMockPlanRepo()
		svc := NewPlatformPlanService(repo, log)

		created, err := svc.CreatePlan(ctx, dto.CreatePlanRequest{
			PlanName: "Starter",
			Price:    decimal.Zero,
		})
		require.NoError(t, err)

		fetched, err := svc.GetPlanByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, "Starter", fetched.PlanName)
	})

	t.Run("ArchivePlan archives existing plan", func(t *testing.T) {
		repo := newMockPlanRepo()
		svc := NewPlatformPlanService(repo, log)

		created, err := svc.CreatePlan(ctx, dto.CreatePlanRequest{
			PlanName: "To Archive",
			Price:    decimal.NewFromFloat(50),
		})
		require.NoError(t, err)

		err = svc.ArchivePlan(ctx, created.ID)
		require.NoError(t, err)

		plan, err := svc.GetPlanByID(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "archived", plan.Status)
	})
}
