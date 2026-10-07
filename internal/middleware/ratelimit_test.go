package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/repository"
	"github.com/Varunjp/vyavsa/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryRateLimitRepo struct {
	counts map[string]int64
	ttls   map[string]time.Duration
	err    error
}

func newMemoryRateLimitRepo() *memoryRateLimitRepo {
	return &memoryRateLimitRepo{
		counts: make(map[string]int64),
		ttls:   make(map[string]time.Duration),
	}
}

func (m *memoryRateLimitRepo) Allow(ctx context.Context, key string, limit int, window time.Duration) (*repository.RateLimitResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.counts[key]++
	current := m.counts[key]
	remaining := int64(limit) - current
	if remaining < 0 {
		remaining = 0
	}
	ttl := 45 * time.Second
	m.ttls[key] = ttl

	return &repository.RateLimitResult{
		Allowed:   current <= int64(limit),
		Current:   current,
		Limit:     int64(limit),
		Remaining: remaining,
		TTL:       ttl,
		ResetAt:   time.Now().Add(ttl),
	}, nil
}

func (m *memoryRateLimitRepo) Reset(ctx context.Context, key string) error {
	delete(m.counts, key)
	delete(m.ttls, key)
	return nil
}

func setupRateLimitTestRouter(svc service.RateLimiterService, m *metrics.Metrics, log *logger.Logger) (*gin.Engine, *RateLimiterMiddleware) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mw := NewRateLimiterMiddleware(svc, m, log)
	return r, mw
}

func TestRateLimiterMiddleware_AllowAndExceed(t *testing.T) {
	repo := newMemoryRateLimitRepo()
	cfg := config.RateLimitConfig{
		Enabled:         true,
		GeneralRequests: 2,
		GeneralWindow:   time.Minute,
		FailOpen:        true,
	}
	appMetrics := metrics.New()
	appLog := logger.Default()
	svc := service.NewRateLimiterService(repo, cfg, appMetrics, appLog.Logger)

	router, mw := setupRateLimitTestRouter(svc, appMetrics, appLog)
	router.GET("/test-general", mw.RequireGeneralLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 1. First request below limit -> allowed
	req1, _ := http.NewRequest(http.MethodGet, "/test-general", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Equal(t, "2", w1.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "1", w1.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w1.Header().Get("X-RateLimit-Reset"))

	// 2. Second request reaches limit -> allowed
	req2, _ := http.NewRequest(http.MethodGet, "/test-general", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Equal(t, "2", w2.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "0", w2.Header().Get("X-RateLimit-Remaining"))

	// 3. Third request exceeds limit -> 429 Too Many Requests
	req3, _ := http.NewRequest(http.MethodGet, "/test-general", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.NotEmpty(t, w3.Header().Get("Retry-After"))
	assert.Equal(t, "0", w3.Header().Get("X-RateLimit-Remaining"))

	var errResp map[string]any
	err := json.Unmarshal(w3.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.False(t, errResp["success"].(bool))
	errObj := errResp["error"].(map[string]any)
	assert.Equal(t, "TOO_MANY_REQUESTS", errObj["code"])
	assert.Contains(t, errObj["message"], "rate limit exceeded")
}

func TestRateLimiterMiddleware_TenantIsolation(t *testing.T) {
	repo := newMemoryRateLimitRepo()
	cfg := config.RateLimitConfig{
		Enabled:        true,
		TenantRequests: 2,
		TenantWindow:   time.Minute,
		FailOpen:       true,
	}
	appMetrics := metrics.New()
	appLog := logger.Default()
	svc := service.NewRateLimiterService(repo, cfg, appMetrics, appLog.Logger)

	router, mw := setupRateLimitTestRouter(svc, appMetrics, appLog)

	tenantA := uuid.New()
	tenantB := uuid.New()
	userA1 := uuid.New()
	userA2 := uuid.New()

	router.GET("/tenant/orders", mw.RequireTenantLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	injectClaims := func(c *gin.Context, tenantID, userID uuid.UUID) {
		auth.SetClaims(c, &auth.CustomClaims{
			UserID:   userID,
			TenantID: &tenantID,
			Role:     "tenant_user",
		})
	}

	// Helper to send request with custom claims
	sendWithTenant := func(tenantID, userID uuid.UUID) int {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/tenant/orders", nil)
		injectClaims(c, tenantID, userID)

		handler := mw.RequireTenantLimit()
		handler(c)
		if !c.IsAborted() {
			return http.StatusOK
		}
		return w.Code
	}

	// Tenant A - User 1: 1st request -> 200
	assert.Equal(t, http.StatusOK, sendWithTenant(tenantA, userA1))

	// Tenant A - User 2: 2nd request (reaches tenant quota) -> 200
	assert.Equal(t, http.StatusOK, sendWithTenant(tenantA, userA2))

	// Tenant A - User 1: 3rd request (exceeds tenant quota) -> 429
	// Confirms that different users within Tenant A cannot bypass limits!
	assert.Equal(t, http.StatusTooManyRequests, sendWithTenant(tenantA, userA1))

	// Tenant B: 1st request -> 200 (Tenant B is completely isolated from Tenant A)
	assert.Equal(t, http.StatusOK, sendWithTenant(tenantB, uuid.New()))
}

func TestRateLimiterMiddleware_AuthAndSecurityTiers(t *testing.T) {
	repo := newMemoryRateLimitRepo()
	cfg := config.RateLimitConfig{
		Enabled:          true,
		AuthRequests:     2,
		AuthWindow:       time.Minute,
		SecurityRequests: 1,
		SecurityWindow:   time.Minute,
		AuthFailOpen:     false,
	}
	appMetrics := metrics.New()
	appLog := logger.Default()
	svc := service.NewRateLimiterService(repo, cfg, appMetrics, appLog.Logger)

	router, mw := setupRateLimitTestRouter(svc, appMetrics, appLog)

	router.POST("/auth/login", mw.RequireAuthLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"login": true})
	})
	router.POST("/auth/forgot-password", mw.RequireSecurityLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"otp": true})
	})

	// Auth endpoint limit is 2
	reqLogin1, _ := http.NewRequest(http.MethodPost, "/auth/login", nil)
	wLogin1 := httptest.NewRecorder()
	router.ServeHTTP(wLogin1, reqLogin1)
	assert.Equal(t, http.StatusOK, wLogin1.Code)

	reqLogin2, _ := http.NewRequest(http.MethodPost, "/auth/login", nil)
	wLogin2 := httptest.NewRecorder()
	router.ServeHTTP(wLogin2, reqLogin2)
	assert.Equal(t, http.StatusOK, wLogin2.Code)

	reqLogin3, _ := http.NewRequest(http.MethodPost, "/auth/login", nil)
	wLogin3 := httptest.NewRecorder()
	router.ServeHTTP(wLogin3, reqLogin3)
	assert.Equal(t, http.StatusTooManyRequests, wLogin3.Code)

	// Security endpoint limit is 1
	reqSec1, _ := http.NewRequest(http.MethodPost, "/auth/forgot-password", nil)
	wSec1 := httptest.NewRecorder()
	router.ServeHTTP(wSec1, reqSec1)
	assert.Equal(t, http.StatusOK, wSec1.Code)

	reqSec2, _ := http.NewRequest(http.MethodPost, "/auth/forgot-password", nil)
	wSec2 := httptest.NewRecorder()
	router.ServeHTTP(wSec2, reqSec2)
	assert.Equal(t, http.StatusTooManyRequests, wSec2.Code)
}

func TestRateLimiterMiddleware_Disabled(t *testing.T) {
	repo := newMemoryRateLimitRepo()
	cfg := config.RateLimitConfig{
		Enabled:         false,
		GeneralRequests: 1,
	}
	appMetrics := metrics.New()
	appLog := logger.Default()
	svc := service.NewRateLimiterService(repo, cfg, appMetrics, appLog.Logger)

	router, mw := setupRateLimitTestRouter(svc, appMetrics, appLog)
	router.GET("/disabled", mw.RequireGeneralLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/disabled", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimiterMiddleware_FailClosed(t *testing.T) {
	repo := newMemoryRateLimitRepo()
	repo.err = errors.New("redis timeout")
	cfg := config.RateLimitConfig{
		Enabled:          true,
		SecurityRequests: 5,
		SecurityWindow:   time.Minute,
		AuthFailOpen:     false, // strict security: fail closed
	}
	appMetrics := metrics.New()
	appLog := logger.Default()
	svc := service.NewRateLimiterService(repo, cfg, appMetrics, appLog.Logger)

	router, mw := setupRateLimitTestRouter(svc, appMetrics, appLog)
	router.POST("/auth/reset", mw.RequireSecurityLimit(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req, _ := http.NewRequest(http.MethodPost, "/auth/reset", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	var errResp map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	errObj := errResp["error"].(map[string]any)
	assert.Equal(t, "SERVICE_UNAVAILABLE", errObj["code"])
}
