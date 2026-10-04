package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// UserType discriminator
const (
	UserTypePlatformAdmin = "platform_admin"
	UserTypeTenantUser    = "tenant_user"
)

// Role constants
const (
	RolePlatformAdmin = "platform_admin"
	RoleTenantAdmin   = "admin"
	RoleTenantUser    = "user"
)

// Token purpose constants
const (
	PurposePasswordReset = "password_reset"
)

// Token type constants for access and refresh segregation
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// CustomClaims represents the JWT payload structure
type CustomClaims struct {
	UserID    uuid.UUID  `json:"user_id"`
	TenantID  *uuid.UUID `json:"tenant_id,omitempty"` // Null for platform administrators
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	UserType  string     `json:"user_type"`
	TokenType string     `json:"token_type,omitempty"`
	TokenID   string     `json:"jti,omitempty"` // Unique token identifier for revocation
	Purpose   string     `json:"purpose,omitempty"`
	jwt.RegisteredClaims
}

// PasswordResetClaims represents dedicated claims for password reset tokens
type PasswordResetClaims struct {
	UserID   uuid.UUID  `json:"user_id"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
	Email    string     `json:"email"`
	UserType string     `json:"user_type"`
	Purpose  string     `json:"purpose"`
	TokenID  string     `json:"jti"`
	jwt.RegisteredClaims
}
