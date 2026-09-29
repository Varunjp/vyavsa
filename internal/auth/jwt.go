package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or malformed token")
	ErrExpiredToken = errors.New("token has expired")
)

// TokenPair contains generated access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"` // in seconds
	ExpiresAt    time.Time `json:"expires_at"`
	TokenID      string    `json:"token_id"`
}

// JWTManager handles signing and verifying JWT tokens
type JWTManager interface {
	GenerateTokenPair(userID uuid.UUID, tenantID *uuid.UUID, email, role, userType string) (*TokenPair, error)
	ValidateToken(tokenString string) (*CustomClaims, error)
	GeneratePasswordResetToken(userID uuid.UUID, tenantID *uuid.UUID, email, userType string, expiry time.Duration) (string, string, error)
	ValidatePasswordResetToken(tokenString string) (*PasswordResetClaims, error)
}

// HMACJWTManager implements JWTManager using HMAC-SHA256
type HMACJWTManager struct {
	secret        []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
	issuer        string
}

// NewJWTManager creates a new HMACJWTManager
func NewJWTManager(cfg config.JWTConfig) *HMACJWTManager {
	return &HMACJWTManager{
		secret:        []byte(cfg.Secret),
		accessExpiry:  cfg.AccessExpiry,
		refreshExpiry: cfg.RefreshExpiry,
		issuer:        "vyavsa-bill-book",
	}
}

// GenerateTokenPair issues both an access token and a refresh token
func (m *HMACJWTManager) GenerateTokenPair(
	userID uuid.UUID,
	tenantID *uuid.UUID,
	email, role, userType string,
) (*TokenPair, error) {
	now := time.Now().UTC()
	accessExpiry := now.Add(m.accessExpiry)
	refreshExpiry := now.Add(m.refreshExpiry)
	tokenID := uuid.NewString()

	// Access Token Claims
	accessClaims := CustomClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		UserType: userType,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			ID:        tokenID,
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(m.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh Token Claims (longer expiry, distinct ID)
	refreshTokenID := uuid.NewString()
	refreshClaims := CustomClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		UserType: userType,
		TokenID:  refreshTokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			ID:        refreshTokenID,
		},
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(m.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(m.accessExpiry.Seconds()),
		ExpiresAt:    accessExpiry,
		TokenID:      tokenID,
	}, nil
}

// GeneratePasswordResetToken generates a dedicated single-purpose token for password recovery
func (m *HMACJWTManager) GeneratePasswordResetToken(
	userID uuid.UUID,
	tenantID *uuid.UUID,
	email, userType string,
	expiry time.Duration,
) (string, string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(expiry)
	tokenID := uuid.NewString()

	claims := PasswordResetClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		UserType: userType,
		Purpose:  PurposePasswordReset,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        tokenID,
		},
	}

	tokenStr, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign password reset token: %w", err)
	}

	return tokenStr, tokenID, nil
}

// ValidateToken verifies and parses a standard access/refresh JWT string.
// Rejects tokens issued for specific sub-purposes like password reset.
func (m *HMACJWTManager) ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Password reset tokens or other single-purpose tokens must not grant API access
	if claims.Purpose != "" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidatePasswordResetToken verifies and parses a password reset token
func (m *HMACJWTManager) ValidatePasswordResetToken(tokenString string) (*PasswordResetClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &PasswordResetClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*PasswordResetClaims)
	if !ok || !token.Valid || claims.Purpose != PurposePasswordReset {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
