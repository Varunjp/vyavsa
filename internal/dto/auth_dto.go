package dto

import "github.com/google/uuid"

// PlatformLoginRequest represents platform admin authentication payload
type PlatformLoginRequest struct {
	Identifier string `json:"identifier" binding:"required"` // Username or email
	Password   string `json:"password" binding:"required,min=6"`
}

// TenantLoginRequest represents tenant user authentication payload
type TenantLoginRequest struct {
	Email    string     `json:"email" binding:"required,email"`
	Password string     `json:"password" binding:"required,min=6"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"` // Optional if user belongs to single tenant
}

// RefreshTokenRequest represents token refresh payload
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// UserProfile represents the public identity of an authenticated user
type UserProfile struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	Email    string     `json:"email"`
	Role     string     `json:"role"`
	UserType string     `json:"user_type"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty"`
}

// TokenResponse represents standard authentication response payload
type TokenResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int64       `json:"expires_in"` // in seconds
	User         UserProfile `json:"user"`
}
