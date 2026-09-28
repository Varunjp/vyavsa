package middleware

import (
	"github.com/Varunjp/vyavsa/internal/auth"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// RequirePlatformAdmin ensures that the authenticated request originates from a platform administrator
func RequirePlatformAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.GetClaims(c)
		if err != nil {
			response.Error(c, appErrors.NewUnauthorized("authentication required"))
			c.Abort()
			return
		}

		if claims.UserType != auth.UserTypePlatformAdmin {
			response.Error(c, appErrors.NewForbidden("forbidden: platform administrator privileges required"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole ensures that the user has at least one of the specified roles
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	allowedMap := make(map[string]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		allowedMap[r] = true
	}

	return func(c *gin.Context) {
		claims, err := auth.GetClaims(c)
		if err != nil {
			response.Error(c, appErrors.NewUnauthorized("authentication required"))
			c.Abort()
			return
		}

		if !allowedMap[claims.Role] {
			response.Error(c, appErrors.NewForbidden("forbidden: insufficient role permissions"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireTenantUser ensures that the request originates from a member of an active tenant
func RequireTenantUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := auth.GetClaims(c)
		if err != nil {
			response.Error(c, appErrors.NewUnauthorized("authentication required"))
			c.Abort()
			return
		}

		if claims.UserType != auth.UserTypeTenantUser || claims.TenantID == nil {
			response.Error(c, appErrors.NewForbidden("forbidden: tenant membership required"))
			c.Abort()
			return
		}

		c.Next()
	}
}
