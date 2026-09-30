package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/service"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// RequireActivePlan ensures that tenant data-mutating operations (POST, PUT, PATCH, DELETE)
// are only executed when the authenticated tenant possesses an active, unexpired plan.
// Read-only operations (GET) and subscription purchase/upgrade endpoints pass through unconditionally.
func RequireActivePlan(planService service.TenantPlanService, exemptPaths ...string) gin.HandlerFunc {
	exemptMap := make(map[string]bool, len(exemptPaths))
	for _, p := range exemptPaths {
		exemptMap[p] = true
	}

	return func(c *gin.Context) {
		// 1. Read-only HTTP operations are never blocked
		switch c.Request.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			// Mutating request: check if explicitly exempt (e.g. plan purchase/upgrade)
			reqPath := c.Request.URL.Path
			if exemptMap[reqPath] ||
				strings.HasSuffix(reqPath, "/subscription/purchase") ||
				strings.HasSuffix(reqPath, "/subscription/upgrade") {
				c.Next()
				return
			}
		default:
			c.Next()
			return
		}

		if planService == nil {
			c.Next()
			return
		}

		// 2. Extract authoritative tenant identity from context
		tenantID, err := auth.GetTenantID(c)
		if err != nil {
			response.Error(c, appErrors.NewForbidden("forbidden: tenant membership required"))
			c.Abort()
			return
		}

		// 3. Verify active plan via cache-first service
		hasActivePlan, err := planService.HasActivePlan(c.Request.Context(), tenantID)
		if err != nil {
			response.Error(c, appErrors.NewInternal(fmt.Errorf("unable to verify tenant plan status: %w", err)))
			c.Abort()
			return
		}

		if !hasActivePlan {
			response.Error(c, appErrors.NewForbidden("An active plan is required to perform this operation"))
			c.Abort()
			return
		}

		c.Next()
	}
}
