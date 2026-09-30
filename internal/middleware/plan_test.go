package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockPlanServiceForMiddleware struct {
	hasActivePlan bool
	err           error
}

func (m *mockPlanServiceForMiddleware) HasActivePlan(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.hasActivePlan, nil
}

func (m *mockPlanServiceForMiddleware) InvalidateTenantPlanCache(ctx context.Context, tenantID uuid.UUID) error {
	return nil
}

func setupTestRouter(planService *mockPlanServiceForMiddleware) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	planMiddleware := RequireActivePlan(planService)

	r.Use(func(c *gin.Context) {
		// Mock auth context if header present
		if tIDStr := c.GetHeader("X-Test-Tenant-ID"); tIDStr != "" {
			tID := uuid.MustParse(tIDStr)
			auth.SetClaims(c, &auth.CustomClaims{
				UserID:   uuid.New(),
				TenantID: &tID,
				Role:     auth.RoleTenantAdmin,
				UserType: auth.UserTypeTenantUser,
			})
		}
		c.Next()
	})

	api := r.Group("/tenant")
	api.Use(planMiddleware)
	{
		api.GET("/profile", func(c *gin.Context) {
			response.Success(c, gin.H{"status": "ok"}, "profile viewed")
		})
		api.POST("/line-sales", func(c *gin.Context) {
			response.Created(c, gin.H{"created": true}, "line sale created")
		})
		api.PUT("/customers/:id", func(c *gin.Context) {
			response.Success(c, gin.H{"updated": true}, "customer updated")
		})
		api.DELETE("/banks/:id", func(c *gin.Context) {
			response.Success(c, gin.H{"deleted": true}, "bank deleted")
		})
		api.POST("/subscription/purchase", func(c *gin.Context) {
			response.Success(c, gin.H{"purchased": true}, "plan purchased")
		})
		api.POST("/subscription/upgrade", func(c *gin.Context) {
			response.Success(c, gin.H{"upgraded": true}, "plan upgraded")
		})
	}

	return r
}

func TestRequireActivePlanMiddleware(t *testing.T) {
	tenantID := uuid.New()

	t.Run("GET requests pass unconditionally without active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: false}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodGet, "/tenant/profile", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "profile viewed")
	})

	t.Run("POST /subscription/purchase passes even without active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: false}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodPost, "/tenant/subscription/purchase", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "plan purchased")
	})

	t.Run("POST /subscription/upgrade passes even without active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: false}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodPost, "/tenant/subscription/upgrade", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "plan upgraded")
	})

	t.Run("POST mutation allowed when tenant has active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: true}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodPost, "/tenant/line-sales", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Contains(t, w.Body.String(), "line sale created")
	})

	t.Run("PUT mutation allowed when tenant has active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: true}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodPut, "/tenant/customers/123", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "customer updated")
	})

	t.Run("DELETE mutation allowed when tenant has active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: true}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodDelete, "/tenant/banks/456", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "bank deleted")
	})

	t.Run("POST mutation rejected with 403 Forbidden when tenant has NO active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: false}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodPost, "/tenant/line-sales", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

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

	t.Run("PUT mutation rejected with 403 Forbidden when tenant has NO active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: false}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodPut, "/tenant/customers/123", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "An active plan is required to perform this operation")
	})

	t.Run("DELETE mutation rejected with 403 Forbidden when tenant has NO active plan", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: false}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodDelete, "/tenant/banks/456", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "An active plan is required to perform this operation")
	})

	t.Run("Missing tenant claims on mutation returns 403 Forbidden", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{hasActivePlan: true}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodPost, "/tenant/line-sales", nil)
		// No X-Test-Tenant-ID header

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "FORBIDDEN")
	})

	t.Run("Service verification error returns 500 Internal Server Error", func(t *testing.T) {
		planSvc := &mockPlanServiceForMiddleware{
			err: errors.New("database connection failed"),
		}
		r := setupTestRouter(planSvc)

		req, _ := http.NewRequest(http.MethodPost, "/tenant/line-sales", nil)
		req.Header.Set("X-Test-Tenant-ID", tenantID.String())

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "INTERNAL_ERROR")
	})
}
