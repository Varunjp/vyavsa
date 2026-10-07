package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRateLimitRepo struct {
	allowed   bool
	current   int64
	limit     int64
	remaining int64
	ttl       time.Duration
	err       error
	keysSeen  []string
}

func (m *mockRateLimitRepo) Allow(ctx context.Context, key string, limit int, window time.Duration) (*repository.RateLimitResult, error) {
	m.keysSeen = append(m.keysSeen, key)
	if m.err != nil {
		return nil, m.err
	}
	return &repository.RateLimitResult{
		Allowed:   m.allowed,
		Current:   m.current,
		Limit:     int64(limit),
		Remaining: m.remaining,
		TTL:       m.ttl,
		ResetAt:   time.Now().Add(m.ttl),
	}, nil
}

func (m *mockRateLimitRepo) Reset(ctx context.Context, key string) error {
	m.keysSeen = append(m.keysSeen, "reset:"+key)
	return nil
}

func TestRateLimiterService_DisabledLimiter(t *testing.T) {
	cfg := config.RateLimitConfig{Enabled: false}
	m := metrics.New()
	log := logger.Default().Logger
	repo := &mockRateLimitRepo{}

	svc := NewRateLimiterService(repo, cfg, m, log)
	res, err := svc.Check(context.Background(), "general", "127.0.0.1", 100, time.Minute, true)
	require.NoError(t, err)
	assert.True(t, res.Allowed)
	assert.Equal(t, int64(100), res.Limit)
	assert.Empty(t, repo.keysSeen, "Disabled limiter must not touch repository")
}

func TestRateLimiterService_AllowAndReject(t *testing.T) {
	cfg := config.RateLimitConfig{Enabled: true}
	m := metrics.New()
	log := logger.Default().Logger

	t.Run("Allowed request", func(t *testing.T) {
		repo := &mockRateLimitRepo{
			allowed:   true,
			current:   5,
			remaining: 95,
			ttl:       30 * time.Second,
		}
		svc := NewRateLimiterService(repo, cfg, m, log)
		res, err := svc.Check(context.Background(), "auth", "192.168.1.1", 100, time.Minute, true)
		require.NoError(t, err)
		assert.True(t, res.Allowed)
		assert.Equal(t, int64(5), res.Current)
		assert.Equal(t, int64(95), res.Remaining)
		assert.Equal(t, 30*time.Second, res.RetryAfter)
		assert.Contains(t, repo.keysSeen, "ratelimit:auth:192.168.1.1")
	})

	t.Run("Rejected request", func(t *testing.T) {
		repo := &mockRateLimitRepo{
			allowed:   false,
			current:   101,
			remaining: 0,
			ttl:       45 * time.Second,
		}
		svc := NewRateLimiterService(repo, cfg, m, log)
		res, err := svc.Check(context.Background(), "security", "10.0.0.1", 100, time.Minute, true)
		require.NoError(t, err)
		assert.False(t, res.Allowed)
		assert.Equal(t, int64(101), res.Current)
		assert.Equal(t, int64(0), res.Remaining)
		assert.Equal(t, 45*time.Second, res.RetryAfter)
		assert.Contains(t, repo.keysSeen, "ratelimit:security:10.0.0.1")
	})
}

func TestRateLimiterService_FailOpenFailClosed(t *testing.T) {
	cfg := config.RateLimitConfig{Enabled: true}
	m := metrics.New()
	log := logger.Default().Logger

	t.Run("Fail open on Redis error", func(t *testing.T) {
		repo := &mockRateLimitRepo{err: errors.New("redis connection refused")}
		svc := NewRateLimiterService(repo, cfg, m, log)
		res, err := svc.Check(context.Background(), "general", "127.0.0.1", 100, time.Minute, true)
		require.NoError(t, err)
		assert.True(t, res.Allowed, "When failOpen=true, request must be allowed on Redis error")
	})

	t.Run("Fail closed on Redis error", func(t *testing.T) {
		repo := &mockRateLimitRepo{err: errors.New("redis connection refused")}
		svc := NewRateLimiterService(repo, cfg, m, log)
		res, err := svc.Check(context.Background(), "auth", "127.0.0.1", 20, time.Minute, false)
		assert.Error(t, err)
		assert.Nil(t, res, "When failOpen=false, error must be returned")
	})
}

func TestRateLimiterService_NilRepo(t *testing.T) {
	cfg := config.RateLimitConfig{Enabled: true}
	m := metrics.New()
	log := logger.Default().Logger

	svc := NewRateLimiterService(nil, cfg, m, log)

	t.Run("Nil repo fails open when requested", func(t *testing.T) {
		res, err := svc.Check(context.Background(), "general", "127.0.0.1", 100, time.Minute, true)
		require.NoError(t, err)
		assert.True(t, res.Allowed)
	})

	t.Run("Nil repo fails closed when requested", func(t *testing.T) {
		res, err := svc.Check(context.Background(), "auth", "127.0.0.1", 20, time.Minute, false)
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("Reset with nil repo is safe", func(t *testing.T) {
		err := svc.Reset(context.Background(), "general", "127.0.0.1")
		assert.NoError(t, err)
	})
}
