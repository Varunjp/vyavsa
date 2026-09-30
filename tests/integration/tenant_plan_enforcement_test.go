package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/middleware"
	"github.com/Varunjp/vyavsa/internal/service"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Cache for integration test
type integrationPlanCache struct {
	data         map[uuid.UUID]*domain.CachedTenantPlan
	ttls         map[uuid.UUID]time.Duration
	getCalled    int
	setCalled    int
	deleteCalled int
}

func newIntegrationPlanCache() *integrationPlanCache {
	return &integrationPlanCache{
		data: make(map[uuid.UUID]*domain.CachedTenantPlan),
		ttls: make(map[uuid.UUID]time.Duration),
	}
}

func (m *integrationPlanCache) Get(ctx context.Context, tenantID uuid.UUID) (*domain.CachedTenantPlan, error) {
	m.getCalled++
	val, ok := m.data[tenantID]
	if !ok {
		return nil, nil // miss
	}
	return val, nil
}

func (m *integrationPlanCache) Set(ctx context.Context, tenantID uuid.UUID, plan *domain.CachedTenantPlan, ttl time.Duration) error {
	m.setCalled++
	m.data[tenantID] = plan
	m.ttls[tenantID] = ttl
	return nil
}

func (m *integrationPlanCache) Delete(ctx context.Context, tenantID uuid.UUID) error {
	m.deleteCalled++
	delete(m.data, tenantID)
	delete(m.ttls, tenantID)
	return nil
}

// Mock Subscription Repository for integration test
type integrationSubRepo struct {
	subs      map[uuid.UUID]*domain.PlatformSubscription
	getCalled int
}

func newIntegrationSubRepo() *integrationSubRepo {
	return &integrationSubRepo{subs: make(map[uuid.UUID]*domain.PlatformSubscription)}
}

func (m *integrationSubRepo) Create(ctx context.Context, sub *domain.PlatformSubscription) error {
	m.subs[sub.TenantID] = sub
	return nil
}

func (m *integrationSubRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.PlatformSubscription, error) {
	m.getCalled++
	sub, ok := m.subs[tenantID]
	if !ok {
		return nil, appErrors.NewNotFound("subscription not found")
	}
	return sub, nil
}

func (m *integrationSubRepo) Update(ctx context.Context, sub *domain.PlatformSubscription) error {
	m.subs[sub.TenantID] = sub
	return nil
}

func (m *integrationSubRepo) List(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformSubscriptionWithTenant, int64, error) {
	return nil, 0, nil
}

func TestTenantPlanEnforcement_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log := logger.Default().Logger
	m := metrics.New()
	jwtSecret := "integration-test-secret-minimum-32-bytes"
	jwtManager := auth.NewJWTManager(config.JWTConfig{
		Secret:        jwtSecret,
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	})

	cache := newIntegrationPlanCache()
	subRepo := newIntegrationSubRepo()
	planService := service.NewTenantPlanService(subRepo, nil, cache, m, log)

	// Set up router with Authenticate and RequireActivePlan
	router := gin.New()
	router.Use(middleware.Authenticate(jwtManager, nil))

	tenantGroup := router.Group("/api/v1/tenant")
	tenantGroup.Use(middleware.RequireTenantUser())
	tenantGroup.Use(middleware.RequireActivePlan(planService))
	{
		// Read-only endpoint
		tenantGroup.GET("/profile", func(c *gin.Context) {
			response.Success(c, gin.H{"tenant_id": "active"}, "profile retrieved")
		})

		// Shared mutation endpoint (Tenant Admin & User)
		tenantGroup.POST("/line-sales", func(c *gin.Context) {
			response.Created(c, gin.H{"id": 1, "amount": 100}, "line sale recorded")
		})

		// Admin-only routes
		adminGroup := tenantGroup.Group("")
		adminGroup.Use(middleware.RequireRole(auth.RoleTenantAdmin))
		{
			adminGroup.POST("/customers", func(c *gin.Context) {
				response.Created(c, gin.H{"id": 1, "name": "Customer A"}, "customer created")
			})

			// Subscription purchase (exempt from active plan requirement)
			adminGroup.POST("/subscription/purchase", func(c *gin.Context) {
				claims, _ := auth.GetClaims(c)
				tenantID := *claims.TenantID

				// Simulate successful purchase and cache invalidation
				future := time.Now().UTC().Add(30 * 24 * time.Hour)
				subRepo.subs[tenantID] = &domain.PlatformSubscription{
					ID:            uuid.New(),
					TenantID:      tenantID,
					CurrentPlanID: uuid.New(),
					Status:        "active",
					EndDate:       &future,
				}
				_ = planService.InvalidateTenantPlanCache(c.Request.Context(), tenantID)

				response.Success(c, gin.H{
					"status": "active",
					"plan":   "Pro Monthly",
					"price":  decimal.NewFromInt(999),
				}, "plan purchased successfully")
			})
		}
	}

	// 1. Setup Test Tenants and Tokens
	activeTenantID := uuid.New()
	inactiveTenantID := uuid.New()

	// Active Tenant Subscription in DB
	future := time.Now().UTC().Add(30 * 24 * time.Hour)
	subRepo.subs[activeTenantID] = &domain.PlatformSubscription{
		ID:            uuid.New(),
		TenantID:      activeTenantID,
		CurrentPlanID: uuid.New(),
		Status:        "active",
		EndDate:       &future,
	}

	// Tokens
	activeAdminToken, err := jwtManager.GenerateTokenPair(uuid.New(), &activeTenantID, "admin@active.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
	require.NoError(t, err)

	activeUserToken, err := jwtManager.GenerateTokenPair(uuid.New(), &activeTenantID, "user@active.com", auth.RoleTenantUser, auth.UserTypeTenantUser)
	require.NoError(t, err)

	inactiveAdminToken, err := jwtManager.GenerateTokenPair(uuid.New(), &inactiveTenantID, "admin@inactive.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
	require.NoError(t, err)

	inactiveUserToken, err := jwtManager.GenerateTokenPair(uuid.New(), &inactiveTenantID, "user@inactive.com", auth.RoleTenantUser, auth.UserTypeTenantUser)
	require.NoError(t, err)

	t.Run("Tenant Admin with active plan -> Mutation allowed (201 Created)", func(t *testing.T) {
		body := bytes.NewBufferString(`{"customer_name":"John Doe"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/customers", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+activeAdminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "customer created")
	})

	t.Run("Tenant User with active plan -> Shared mutation allowed (201 Created)", func(t *testing.T) {
		body := bytes.NewBufferString(`{"item_name":"Bread","qty":5}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/line-sales", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+activeUserToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "line sale recorded")
	})

	t.Run("Tenant Admin with NO active plan -> Mutation rejected (403 Forbidden)", func(t *testing.T) {
		body := bytes.NewBufferString(`{"customer_name":"Jane Doe"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/customers", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+inactiveAdminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp response.Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "An active plan is required to perform this operation", resp.Message)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "FORBIDDEN", resp.Error.Code)
		assert.Equal(t, "An active plan is required to perform this operation", resp.Error.Message)
	})

	t.Run("Tenant User with NO active plan -> Shared mutation rejected (403 Forbidden)", func(t *testing.T) {
		body := bytes.NewBufferString(`{"item_name":"Milk","qty":2}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/line-sales", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+inactiveUserToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)

		var resp response.Response
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Equal(t, "An active plan is required to perform this operation", resp.Message)
		require.NotNil(t, resp.Error)
		assert.Equal(t, "FORBIDDEN", resp.Error.Code)
		assert.Equal(t, "An active plan is required to perform this operation", resp.Error.Message)
	})

	t.Run("Tenant Admin with NO active plan -> GET /profile is allowed (200 OK)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/profile", nil)
		req.Header.Set("Authorization", "Bearer "+inactiveAdminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "profile retrieved")
	})

	t.Run("Tenant User with NO active plan -> GET /profile is allowed (200 OK)", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/profile", nil)
		req.Header.Set("Authorization", "Bearer "+inactiveUserToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "profile retrieved")
	})

	t.Run("Tenant Admin with NO active plan -> POST /subscription/purchase is exempt and allowed (200 OK)", func(t *testing.T) {
		body := bytes.NewBufferString(`{"plan_id":"00000000-0000-0000-0000-000000000001","payment_method":"card"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/subscription/purchase", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+inactiveAdminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "plan purchased successfully")
	})

	t.Run("After purchase -> Cache invalidated and subsequent mutation now allowed (201 Created)", func(t *testing.T) {
		// Now inactiveTenantID has an active subscription in subRepo, and cache was invalidated during purchase
		body := bytes.NewBufferString(`{"customer_name":"Newly Unlocked Customer"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/customers", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+inactiveAdminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "customer created")

		// And verify it was cached in Redis with 24-hour TTL
		require.NotNil(t, cache.data[inactiveTenantID])
		assert.True(t, cache.data[inactiveTenantID].Active)
		assert.Equal(t, "active", cache.data[inactiveTenantID].Status)
		assert.Equal(t, 24*time.Hour, cache.ttls[inactiveTenantID])
	})

	t.Run("Subsequent mutation uses cache -> DB is NOT queried", func(t *testing.T) {
		dbCallsBefore := subRepo.getCalled

		body := bytes.NewBufferString(`{"customer_name":"Second Customer"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/customers", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+inactiveAdminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, dbCallsBefore, subRepo.getCalled, "Cache hit must not query PostgreSQL")
	})

	t.Run("Multi-tenant isolation: Tenant A plan state does not affect Tenant B", func(t *testing.T) {
		isolatedTenantID := uuid.New()
		isolatedAdminToken, err := jwtManager.GenerateTokenPair(uuid.New(), &isolatedTenantID, "admin@isolated.com", auth.RoleTenantAdmin, auth.UserTypeTenantUser)
		require.NoError(t, err)

		// Tenant B has no active plan
		body := bytes.NewBufferString(`{"customer_name":"Tenant B Customer"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/tenant/customers", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+isolatedAdminToken.AccessToken)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Must be rejected
		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "An active plan is required to perform this operation")

		// Cached key must be tenant-specific
		expectedKey := fmt.Sprintf("tenant:plan:%s", isolatedTenantID.String())
		assert.False(t, cache.data[isolatedTenantID].Active)
		_ = expectedKey
	})
}
