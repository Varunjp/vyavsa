package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/repository"
	"github.com/redis/go-redis/v9"
)

// rateLimitLuaScript executes an atomic fixed-window counter with automatic expiration.
// KEYS[1]: rate limit key
// ARGV[1]: window in seconds
// ARGV[2]: maximum allowed requests
// Returns: {current_count, remaining_ttl_in_seconds}
const rateLimitLuaScript = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('TTL', KEYS[1])
if ttl == -1 then
    redis.call('EXPIRE', KEYS[1], ARGV[1])
    ttl = tonumber(ARGV[1])
end
return {current, ttl}
`

// RateLimitRedis implements repository.RateLimitRepository using Redis
type RateLimitRedis struct {
	cache  *cache.Redis
	script *redis.Script
}

// NewRateLimitRedis creates a new RateLimitRedis repository
func NewRateLimitRedis(c *cache.Redis) *RateLimitRedis {
	return &RateLimitRedis{
		cache:  c,
		script: redis.NewScript(rateLimitLuaScript),
	}
}

// Allow atomically increments the counter and checks against the limit.
func (r *RateLimitRedis) Allow(ctx context.Context, key string, limit int, window time.Duration) (*repository.RateLimitResult, error) {
	windowSeconds := int(window.Seconds())
	if windowSeconds < 1 {
		windowSeconds = 1
	}

	// Degraded mode: If Redis is nil or uninitialized, permit requests gracefully
	if r.cache == nil || r.cache.Client == nil {
		now := time.Now()
		return &repository.RateLimitResult{
			Allowed:   true,
			Current:   1,
			Limit:     int64(limit),
			Remaining: int64(limit) - 1,
			TTL:       window,
			ResetAt:   now.Add(window),
		}, nil
	}

	res, err := r.script.Run(ctx, r.cache.Client, []string{key}, windowSeconds, limit).Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to execute redis rate limit script: %w", err)
	}

	if len(res) < 2 {
		return nil, fmt.Errorf("invalid rate limit script response length: %d", len(res))
	}

	current, ok1 := res[0].(int64)
	ttlSec, ok2 := res[1].(int64)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unexpected type in rate limit script response: %T, %T", res[0], res[1])
	}

	if ttlSec < 0 {
		ttlSec = int64(windowSeconds)
	}

	ttl := time.Duration(ttlSec) * time.Second
	resetAt := time.Now().Add(ttl)

	remaining := int64(limit) - current
	if remaining < 0 {
		remaining = 0
	}

	allowed := current <= int64(limit)

	return &repository.RateLimitResult{
		Allowed:   allowed,
		Current:   current,
		Limit:     int64(limit),
		Remaining: remaining,
		TTL:       ttl,
		ResetAt:   resetAt,
	}, nil
}

// Reset clears the counter for the specified key
func (r *RateLimitRedis) Reset(ctx context.Context, key string) error {
	if r.cache == nil || r.cache.Client == nil {
		return nil
	}
	return r.cache.Client.Del(ctx, key).Err()
}
