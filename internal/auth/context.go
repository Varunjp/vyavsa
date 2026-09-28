package auth

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var (
	ErrUnauthenticated   = errors.New("unauthenticated: no valid identity in context")
	ErrTenantNotFound    = errors.New("tenant context missing for authenticated operation")
	ErrForbiddenPlatform = errors.New("forbidden: requires platform administrator privileges")
)

const (
	claimsContextKey = "auth_claims"
)

// SetClaims stores authenticated claims in both Gin and standard request context
func SetClaims(c *gin.Context, claims *CustomClaims) {
	c.Set(claimsContextKey, claims)
	ctx := context.WithValue(c.Request.Context(), claimsContextKey, claims)
	c.Request = c.Request.WithContext(ctx)
}

// GetClaims retrieves CustomClaims from Gin context
func GetClaims(c *gin.Context) (*CustomClaims, error) {
	val, exists := c.Get(claimsContextKey)
	if !exists {
		return nil, ErrUnauthenticated
	}

	claims, ok := val.(*CustomClaims)
	if !ok || claims == nil {
		return nil, ErrUnauthenticated
	}

	return claims, nil
}

// GetTenantID extracts the authoritative tenant ID from context
func GetTenantID(c *gin.Context) (uuid.UUID, error) {
	claims, err := GetClaims(c)
	if err != nil {
		return uuid.Nil, err
	}

	if claims.TenantID == nil || *claims.TenantID == uuid.Nil {
		return uuid.Nil, ErrTenantNotFound
	}

	return *claims.TenantID, nil
}

// GetUserID extracts the authenticated user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	claims, err := GetClaims(c)
	if err != nil {
		return uuid.Nil, err
	}
	return claims.UserID, nil
}

// IsPlatformAdmin returns true if the authenticated user is a platform admin
func IsPlatformAdmin(c *gin.Context) bool {
	claims, err := GetClaims(c)
	if err != nil {
		return false
	}
	return claims.UserType == UserTypePlatformAdmin
}
