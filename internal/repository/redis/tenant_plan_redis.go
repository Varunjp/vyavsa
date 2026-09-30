package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	tenantPlanKeyPrefix = "tenant:plan:"
)

// TenantPlanCacheRedis implements repository.TenantPlanCacheRepository using Redis
type TenantPlanCacheRedis struct {
	cache *cache.Redis
}

// NewTenantPlanCacheRedis creates a new TenantPlanCacheRedis repository
func NewTenantPlanCacheRedis(c *cache.Redis) *TenantPlanCacheRedis {
	return &TenantPlanCacheRedis{cache: c}
}

func (r *TenantPlanCacheRedis) tenantPlanKey(tenantID uuid.UUID) string {
	return tenantPlanKeyPrefix + tenantID.String()
}

// Get retrieves cached tenant plan details from Redis.
// Returns (nil, nil) on a cache miss (redis.Nil).
func (r *TenantPlanCacheRedis) Get(ctx context.Context, tenantID uuid.UUID) (*domain.CachedTenantPlan, error) {
	if r.cache == nil || r.cache.Client == nil {
		return nil, fmt.Errorf("redis client is not initialized")
	}

	key := r.tenantPlanKey(tenantID)
	val, err := r.cache.Client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("redis get failed for key %s: %w", key, err)
	}

	var plan domain.CachedTenantPlan
	if err := json.Unmarshal([]byte(val), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached tenant plan: %w", err)
	}

	return &plan, nil
}

// Set saves tenant plan status to Redis with the specified TTL
func (r *TenantPlanCacheRedis) Set(ctx context.Context, tenantID uuid.UUID, plan *domain.CachedTenantPlan, ttl time.Duration) error {
	if r.cache == nil || r.cache.Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}

	key := r.tenantPlanKey(tenantID)
	data, err := json.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal tenant plan for caching: %w", err)
	}

	if err := r.cache.Client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed for key %s: %w", key, err)
	}

	return nil
}

// Delete invalidates the cached tenant plan in Redis
func (r *TenantPlanCacheRedis) Delete(ctx context.Context, tenantID uuid.UUID) error {
	if r.cache == nil || r.cache.Client == nil {
		return nil
	}

	key := r.tenantPlanKey(tenantID)
	if err := r.cache.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del failed for key %s: %w", key, err)
	}

	return nil
}
