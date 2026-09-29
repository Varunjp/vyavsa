package domain

import (
	"time"

	"github.com/google/uuid"
)

// PasswordResetOTP represents a stored recovery attempt in cache/Redis
type PasswordResetOTP struct {
	OTPHash   string     `json:"otp_hash"`
	Attempts  int        `json:"attempts"`
	UserID    uuid.UUID  `json:"user_id"`
	TenantID  *uuid.UUID `json:"tenant_id,omitempty"`
	UserType  string     `json:"user_type"`
	Email     string     `json:"email"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
}

// PasswordResetTokenData represents active reset token metadata in cache/Redis
type PasswordResetTokenData struct {
	TokenID   string     `json:"token_id"`
	UserID    uuid.UUID  `json:"user_id"`
	TenantID  *uuid.UUID `json:"tenant_id,omitempty"`
	UserType  string     `json:"user_type"`
	Email     string     `json:"email"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
}
