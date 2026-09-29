package dto

import "github.com/google/uuid"

// PlatformLoginRequest represents platform admin authentication payload
type PlatformLoginRequest struct {
	Identifier string `json:"identifier"`
	Email      string `json:"email"`
	Username   string `json:"username"`
	Password   string `json:"password" binding:"required,min=6"`
}

// GetIdentifier returns the normalized identifier (email or username)
func (r *PlatformLoginRequest) GetIdentifier() string {
	if r.Identifier != "" {
		return r.Identifier
	}
	if r.Email != "" {
		return r.Email
	}
	return r.Username
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

// ForgotPasswordRequest represents payload for initiating password reset
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// VerifyResetOTPRequest represents payload for verifying password recovery OTP
type VerifyResetOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required,len=6,numeric"`
}

// VerifyResetOTPResponse represents data payload returned after successful OTP verification
type VerifyResetOTPResponse struct {
	ResetToken string `json:"reset_token"`
}

// ResetPasswordRequest represents payload for setting new password with a reset token
type ResetPasswordRequest struct {
	ResetToken  string `json:"reset_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6,max=72"`
}
