package redis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/cache"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) *cache.Redis {
	cfg := config.RedisConfig{
		Host: "localhost",
		Port: "6379",
		DB:   1, // Use DB 1 for test isolation
	}
	log := logger.Default().Logger
	c, err := cache.NewRedis(context.Background(), cfg, log)
	if err != nil {
		t.Skip("local redis is not running, skipping live redis tests")
	}
	t.Cleanup(func() {
		_ = c.Client.FlushDB(context.Background()).Err()
		_ = c.Close()
	})
	return c
}

func TestRateLimitRedis_DegradedMode(t *testing.T) {
	ctx := context.Background()
	repo := NewRateLimitRedis(nil)

	res, err := repo.Allow(ctx, "test:degraded", 10, time.Minute)
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, int64(10), res.Limit)
	assert.Equal(t, int64(9), res.Remaining)

	err = repo.Reset(ctx, "test:degraded")
	assert.NoError(t, err)
}

func TestRateLimitRedis_AllowAndExceed(t *testing.T) {
	c := setupTestRedis(t)
	ctx := context.Background()
	repo := NewRateLimitRedis(c)
	testKey := fmt.Sprintf("test:ratelimit:%d", time.Now().UnixNano())

	limit := 3
	window := 2 * time.Second

	// 1. First request -> allowed (current = 1, remaining = 2)
	res1, err := repo.Allow(ctx, testKey, limit, window)
	require.NoError(t, err)
	assert.True(t, res1.Allowed)
	assert.Equal(t, int64(1), res1.Current)
	assert.Equal(t, int64(2), res1.Remaining)
	assert.True(t, res1.TTL > 0 && res1.TTL <= window)

	// 2. Second request -> allowed (current = 2, remaining = 1)
	res2, err := repo.Allow(ctx, testKey, limit, window)
	require.NoError(t, err)
	assert.True(t, res2.Allowed)
	assert.Equal(t, int64(2), res2.Current)
	assert.Equal(t, int64(1), res2.Remaining)

	// 3. Third request -> reaches limit -> allowed (current = 3, remaining = 0)
	res3, err := repo.Allow(ctx, testKey, limit, window)
	require.NoError(t, err)
	assert.True(t, res3.Allowed)
	assert.Equal(t, int64(3), res3.Current)
	assert.Equal(t, int64(0), res3.Remaining)

	// 4. Fourth request -> exceeds limit -> rejected (current = 4, remaining = 0)
	res4, err := repo.Allow(ctx, testKey, limit, window)
	require.NoError(t, err)
	assert.False(t, res4.Allowed)
	assert.Equal(t, int64(4), res4.Current)
	assert.Equal(t, int64(0), res4.Remaining)

	// 5. Reset clears key
	err = repo.Reset(ctx, testKey)
	require.NoError(t, err)

	resAfterReset, err := repo.Allow(ctx, testKey, limit, window)
	require.NoError(t, err)
	assert.True(t, resAfterReset.Allowed)
	assert.Equal(t, int64(1), resAfterReset.Current)
}

func TestRateLimitRedis_WindowExpiration(t *testing.T) {
	c := setupTestRedis(t)
	ctx := context.Background()
	repo := NewRateLimitRedis(c)
	testKey := fmt.Sprintf("test:expire:%d", time.Now().UnixNano())

	limit := 1
	window := 1 * time.Second

	res1, err := repo.Allow(ctx, testKey, limit, window)
	require.NoError(t, err)
	assert.True(t, res1.Allowed)

	res2, err := repo.Allow(ctx, testKey, limit, window)
	require.NoError(t, err)
	assert.False(t, res2.Allowed)

	// Wait for window to expire
	time.Sleep(1200 * time.Millisecond)

	res3, err := repo.Allow(ctx, testKey, limit, window)
	require.NoError(t, err)
	assert.True(t, res3.Allowed)
	assert.Equal(t, int64(1), res3.Current)
}

func TestRateLimitRedis_ConcurrencySafety(t *testing.T) {
	c := setupTestRedis(t)
	ctx := context.Background()
	repo := NewRateLimitRedis(c)
	testKey := fmt.Sprintf("test:concurrent:%d", time.Now().UnixNano())

	limit := 50
	window := 10 * time.Second
	totalRequests := 100

	var wg sync.WaitGroup
	var allowedCount int64
	var rejectedCount int64
	var mu sync.Mutex

	for i := 0; i < totalRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := repo.Allow(ctx, testKey, limit, window)
			if err != nil {
				return
			}
			mu.Lock()
			if res.Allowed {
				allowedCount++
			} else {
				rejectedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(limit), allowedCount, "Exactly limit requests must be allowed under concurrent load")
	assert.Equal(t, int64(totalRequests-limit), rejectedCount, "Remaining requests must be rejected")
}
