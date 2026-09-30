package redis

import (
	"context"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantPlanRedis_NilCacheHandling(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	repo := NewTenantPlanCacheRedis(nil)

	t.Run("Get with nil cache returns error", func(t *testing.T) {
		res, err := repo.Get(ctx, tenantID)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("Set with nil cache returns error", func(t *testing.T) {
		plan := &domain.CachedTenantPlan{Active: true, Status: "active"}
		err := repo.Set(ctx, tenantID, plan, 24*time.Hour)
		assert.Error(t, err)
	})

	t.Run("Delete with nil cache returns nil (safe)", func(t *testing.T) {
		err := repo.Delete(ctx, tenantID)
		assert.NoError(t, err)
	})
}

func TestTenantPlanRedis_KeyNaming(t *testing.T) {
	tenantID := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	repo := NewTenantPlanCacheRedis(&cache.Redis{})
	expectedKey := "tenant:plan:12345678-1234-1234-1234-123456789abc"
	assert.Equal(t, expectedKey, repo.tenantPlanKey(tenantID))
}

func TestCachedTenantPlan_Serialization(t *testing.T) {
	planID := uuid.New()
	plan := &domain.CachedTenantPlan{
		Active: true,
		PlanID: &planID,
		Status: "active",
	}

	t.Run("Active plan JSON format", func(t *testing.T) {
		require.True(t, plan.Active)
		assert.Equal(t, "active", plan.Status)
		assert.Equal(t, planID, *plan.PlanID)
	})

	t.Run("Inactive plan omits empty plan_id", func(t *testing.T) {
		inactive := &domain.CachedTenantPlan{
			Active: false,
			Status: "expired",
		}
		assert.False(t, inactive.Active)
		assert.Nil(t, inactive.PlanID)
	})
}
