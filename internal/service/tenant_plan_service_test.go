package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPlanCache struct {
	data          map[uuid.UUID]*domain.CachedTenantPlan
	ttls          map[uuid.UUID]time.Duration
	getCalled     int
	setCalled     int
	deleteCalled  int
	simulateError bool
}

func newMockPlanCache() *mockPlanCache {
	return &mockPlanCache{
		data: make(map[uuid.UUID]*domain.CachedTenantPlan),
		ttls: make(map[uuid.UUID]time.Duration),
	}
}

func (m *mockPlanCache) Get(ctx context.Context, tenantID uuid.UUID) (*domain.CachedTenantPlan, error) {
	m.getCalled++
	if m.simulateError {
		return nil, errors.New("redis connection refused")
	}
	plan, ok := m.data[tenantID]
	if !ok {
		return nil, nil // Cache miss
	}
	return plan, nil
}

func (m *mockPlanCache) Set(ctx context.Context, tenantID uuid.UUID, plan *domain.CachedTenantPlan, ttl time.Duration) error {
	m.setCalled++
	if m.simulateError {
		return errors.New("redis connection refused")
	}
	m.data[tenantID] = plan
	m.ttls[tenantID] = ttl
	return nil
}

func (m *mockPlanCache) Delete(ctx context.Context, tenantID uuid.UUID) error {
	m.deleteCalled++
	if m.simulateError {
		return errors.New("redis connection refused")
	}
	delete(m.data, tenantID)
	delete(m.ttls, tenantID)
	return nil
}

// Local mock Subscription Repo
type mockSubRepo struct {
	subs          map[uuid.UUID]*domain.PlatformSubscription
	getCalled     int
	simulateError bool
}

func newMockSubRepo() *mockSubRepo {
	return &mockSubRepo{subs: make(map[uuid.UUID]*domain.PlatformSubscription)}
}

func (m *mockSubRepo) Create(ctx context.Context, sub *domain.PlatformSubscription) error {
	m.subs[sub.TenantID] = sub
	return nil
}

func (m *mockSubRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.PlatformSubscription, error) {
	m.getCalled++
	if m.simulateError {
		return nil, errors.New("database connection failed")
	}
	sub, ok := m.subs[tenantID]
	if !ok {
		return nil, appErrors.NewNotFound("subscription not found")
	}
	return sub, nil
}

func (m *mockSubRepo) Update(ctx context.Context, sub *domain.PlatformSubscription) error {
	m.subs[sub.TenantID] = sub
	return nil
}

func (m *mockSubRepo) List(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformSubscriptionWithTenant, int64, error) {
	return nil, 0, nil
}

// Local mock Tenant Repo
type mockTenantRepoForPlan struct {
	tenants map[uuid.UUID]*domain.Tenant
}

func newMockTenantRepoForPlan() *mockTenantRepoForPlan {
	return &mockTenantRepoForPlan{tenants: make(map[uuid.UUID]*domain.Tenant)}
}

func (m *mockTenantRepoForPlan) Create(ctx context.Context, t *domain.Tenant) error {
	m.tenants[t.ID] = t
	return nil
}

func (m *mockTenantRepoForPlan) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return nil, appErrors.NewNotFound("tenant not found")
	}
	return t, nil
}

func (m *mockTenantRepoForPlan) GetByEmail(ctx context.Context, email string) (*domain.Tenant, error) {
	return nil, appErrors.NewNotFound("tenant not found")
}

func (m *mockTenantRepoForPlan) Update(ctx context.Context, t *domain.Tenant) error {
	m.tenants[t.ID] = t
	return nil
}

func (m *mockTenantRepoForPlan) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if t, ok := m.tenants[id]; ok {
		t.Status = status
	}
	return nil
}

func (m *mockTenantRepoForPlan) List(ctx context.Context, page, pageSize int, status, search string) ([]domain.Tenant, int64, error) {
	return nil, 0, nil
}

func (m *mockTenantRepoForPlan) CountByStatus(ctx context.Context, status string) (int64, error) {
	return 0, nil
}

func (m *mockTenantRepoForPlan) CountSince(ctx context.Context, since time.Time) (int64, error) {
	return 0, nil
}

func TestTenantPlanService_HasActivePlan(t *testing.T) {
	ctx := context.Background()
	log := logger.Default().Logger
	m := metrics.New()

	t.Run("Cache Hit -> DB is NOT queried", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		subRepo := newMockSubRepo()
		tenantRepo := newMockTenantRepoForPlan()

		planID := uuid.New()
		cache.data[tenantID] = &domain.CachedTenantPlan{
			Active: true,
			PlanID: &planID,
			Status: "active",
		}

		svc := NewTenantPlanService(subRepo, tenantRepo, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.True(t, active)
		assert.Equal(t, 1, cache.getCalled)
		assert.Equal(t, 0, subRepo.getCalled, "PostgreSQL must NOT be queried on cache hit")
	})

	t.Run("Cached inactive plan rejects mutation without querying DB", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		subRepo := newMockSubRepo()
		tenantRepo := newMockTenantRepoForPlan()

		cache.data[tenantID] = &domain.CachedTenantPlan{
			Active: false,
			Status: "expired",
		}

		svc := NewTenantPlanService(subRepo, tenantRepo, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.False(t, active)
		assert.Equal(t, 1, cache.getCalled)
		assert.Equal(t, 0, subRepo.getCalled, "PostgreSQL must NOT be queried on cache hit")
	})

	t.Run("Cache Miss -> DB is queried and result stored in Redis with 24h TTL", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		subRepo := newMockSubRepo()
		tenantRepo := newMockTenantRepoForPlan()

		planID := uuid.New()
		future := time.Now().UTC().Add(30 * 24 * time.Hour)
		subRepo.subs[tenantID] = &domain.PlatformSubscription{
			ID:            uuid.New(),
			TenantID:      tenantID,
			CurrentPlanID: planID,
			Status:        "active",
			EndDate:       &future,
		}
		tenantRepo.tenants[tenantID] = &domain.Tenant{ID: tenantID, Status: "active"}

		svc := NewTenantPlanService(subRepo, tenantRepo, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.True(t, active)
		assert.Equal(t, 1, cache.getCalled)
		assert.Equal(t, 1, subRepo.getCalled, "PostgreSQL must be queried on cache miss")
		assert.Equal(t, 1, cache.setCalled, "Result must be stored in Redis")

		// Verify 24h TTL
		assert.Equal(t, 24*time.Hour, cache.ttls[tenantID])
		require.NotNil(t, cache.data[tenantID])
		assert.True(t, cache.data[tenantID].Active)
		assert.Equal(t, "active", cache.data[tenantID].Status)
		assert.Equal(t, planID, *cache.data[tenantID].PlanID)
	})

	t.Run("No subscription in DB -> Returns false and caches negative result with 24h TTL", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		subRepo := newMockSubRepo()
		tenantRepo := newMockTenantRepoForPlan()

		svc := NewTenantPlanService(subRepo, tenantRepo, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.False(t, active)
		assert.Equal(t, 1, subRepo.getCalled)
		assert.Equal(t, 1, cache.setCalled)
		assert.Equal(t, 24*time.Hour, cache.ttls[tenantID])
		assert.False(t, cache.data[tenantID].Active)
		assert.Equal(t, "none", cache.data[tenantID].Status)
	})

	t.Run("Expired subscription in DB -> Returns false and caches expired status", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		subRepo := newMockSubRepo()
		tenantRepo := newMockTenantRepoForPlan()

		planID := uuid.New()
		past := time.Now().UTC().Add(-24 * time.Hour)
		subRepo.subs[tenantID] = &domain.PlatformSubscription{
			ID:            uuid.New(),
			TenantID:      tenantID,
			CurrentPlanID: planID,
			Status:        "active",
			EndDate:       &past,
		}

		svc := NewTenantPlanService(subRepo, tenantRepo, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.False(t, active)
		assert.Equal(t, 24*time.Hour, cache.ttls[tenantID])
		assert.False(t, cache.data[tenantID].Active)
		assert.Equal(t, "expired", cache.data[tenantID].Status)
	})

	t.Run("Cancelled subscription in DB -> Returns false", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		subRepo := newMockSubRepo()
		tenantRepo := newMockTenantRepoForPlan()

		planID := uuid.New()
		future := time.Now().UTC().Add(30 * 24 * time.Hour)
		subRepo.subs[tenantID] = &domain.PlatformSubscription{
			ID:            uuid.New(),
			TenantID:      tenantID,
			CurrentPlanID: planID,
			Status:        "cancelled",
			EndDate:       &future,
		}

		svc := NewTenantPlanService(subRepo, tenantRepo, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.False(t, active)
		assert.False(t, cache.data[tenantID].Active)
		assert.Equal(t, "cancelled", cache.data[tenantID].Status)
	})

	t.Run("Suspended tenant organization -> Returns false", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		subRepo := newMockSubRepo()
		tenantRepo := newMockTenantRepoForPlan()

		planID := uuid.New()
		future := time.Now().UTC().Add(30 * 24 * time.Hour)
		subRepo.subs[tenantID] = &domain.PlatformSubscription{
			ID:            uuid.New(),
			TenantID:      tenantID,
			CurrentPlanID: planID,
			Status:        "active",
			EndDate:       &future,
		}
		tenantRepo.tenants[tenantID] = &domain.Tenant{ID: tenantID, Status: "suspended"}

		svc := NewTenantPlanService(subRepo, tenantRepo, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.False(t, active)
		assert.False(t, cache.data[tenantID].Active)
	})

	t.Run("Redis Failure Handling -> Safely falls back to PostgreSQL without bypassing security", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		cache.simulateError = true // Redis is down!

		subRepo := newMockSubRepo()
		planID := uuid.New()
		future := time.Now().UTC().Add(30 * 24 * time.Hour)
		subRepo.subs[tenantID] = &domain.PlatformSubscription{
			ID:            uuid.New(),
			TenantID:      tenantID,
			CurrentPlanID: planID,
			Status:        "active",
			EndDate:       &future,
		}

		svc := NewTenantPlanService(subRepo, nil, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.True(t, active, "Active plan in DB must be validated even when Redis is down")
		assert.Equal(t, 1, subRepo.getCalled, "PostgreSQL must be queried when Redis fails")
	})

	t.Run("Redis Failure with Inactive DB Subscription -> Rejects mutation (does not bypass)", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		cache.simulateError = true // Redis is down!

		subRepo := newMockSubRepo()
		// No subscription in subRepo

		svc := NewTenantPlanService(subRepo, nil, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		require.NoError(t, err)
		assert.False(t, active, "Must NOT bypass security when Redis fails and tenant has no plan")
	})

	t.Run("Both Redis and Database Fail -> Returns database error (does not bypass)", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		cache.simulateError = true

		subRepo := newMockSubRepo()
		subRepo.simulateError = true // DB is also down!

		svc := NewTenantPlanService(subRepo, nil, cache, m, log)
		active, err := svc.HasActivePlan(ctx, tenantID)

		assert.Error(t, err)
		assert.False(t, active)
	})

	t.Run("InvalidateTenantPlanCache removes cached key from Redis", func(t *testing.T) {
		tenantID := uuid.New()
		cache := newMockPlanCache()
		cache.data[tenantID] = &domain.CachedTenantPlan{Active: true, Status: "active"}

		svc := NewTenantPlanService(nil, nil, cache, m, log)
		err := svc.InvalidateTenantPlanCache(ctx, tenantID)

		require.NoError(t, err)
		assert.Equal(t, 1, cache.deleteCalled)
		assert.Nil(t, cache.data[tenantID])
	})
}
