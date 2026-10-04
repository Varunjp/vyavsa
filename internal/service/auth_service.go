package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/mailer"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
)

// AuthService defines authentication and password recovery business logic
type AuthService interface {
	LoginPlatformAdmin(ctx context.Context, req dto.PlatformLoginRequest) (*dto.TokenResponse, error)
	LoginTenantUser(ctx context.Context, req dto.TenantLoginRequest) (*dto.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshTokenStr string) (*dto.TokenResponse, error)
	Logout(ctx context.Context, tokenID string, remainingTTL time.Duration) error
	RevokeRefreshToken(ctx context.Context, refreshTokenStr string) error
	GetProfile(ctx context.Context, claims *auth.CustomClaims) (*dto.UserProfile, error)
	ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest, clientIP string) error
	VerifyResetOTP(ctx context.Context, req dto.VerifyResetOTPRequest, clientIP string) (*dto.VerifyResetOTPResponse, error)
	ResetPassword(ctx context.Context, req dto.ResetPasswordRequest, clientIP string) error
}

type authService struct {
	platformAdminRepo repository.PlatformAdminRepository
	tenantUserRepo    repository.TenantUserRepository
	blacklistRepo     repository.TokenBlacklistRepository
	passwordResetRepo repository.PasswordResetRepository
	mailer            mailer.Mailer
	hasher            auth.PasswordHasher
	jwtManager        auth.JWTManager
	pwCfg             config.PasswordResetConfig
	metrics           *metrics.Metrics
	log               *slog.Logger
}

// NewAuthService creates a new instance of AuthService
func NewAuthService(
	platformAdminRepo repository.PlatformAdminRepository,
	tenantUserRepo repository.TenantUserRepository,
	blacklistRepo repository.TokenBlacklistRepository,
	passwordResetRepo repository.PasswordResetRepository,
	mail mailer.Mailer,
	hasher auth.PasswordHasher,
	jwtManager auth.JWTManager,
	pwCfg config.PasswordResetConfig,
	m *metrics.Metrics,
	log *slog.Logger,
) AuthService {
	if pwCfg.OTPLength <= 0 {
		pwCfg.OTPLength = 6
	}
	if pwCfg.OTPExpiry <= 0 {
		pwCfg.OTPExpiry = 5 * time.Minute
	}
	if pwCfg.TokenExpiry <= 0 {
		pwCfg.TokenExpiry = 10 * time.Minute
	}
	if pwCfg.MaxAttempts <= 0 {
		pwCfg.MaxAttempts = 5
	}
	if pwCfg.ResendCooldown <= 0 {
		pwCfg.ResendCooldown = 60 * time.Second
	}
	if pwCfg.MaxEmailRequests <= 0 {
		pwCfg.MaxEmailRequests = 3
	}
	if pwCfg.MaxIPRequests <= 0 {
		pwCfg.MaxIPRequests = 10
	}
	if pwCfg.RateLimitWindow <= 0 {
		pwCfg.RateLimitWindow = 15 * time.Minute
	}

	return &authService{
		platformAdminRepo: platformAdminRepo,
		tenantUserRepo:    tenantUserRepo,
		blacklistRepo:     blacklistRepo,
		passwordResetRepo: passwordResetRepo,
		mailer:            mail,
		hasher:            hasher,
		jwtManager:        jwtManager,
		pwCfg:             pwCfg,
		metrics:           m,
		log:               log,
	}
}

// LoginPlatformAdmin authenticates a platform-level administrator
func (s *authService) LoginPlatformAdmin(ctx context.Context, req dto.PlatformLoginRequest) (*dto.TokenResponse, error) {
	identifier := req.GetIdentifier()
	if identifier == "" {
		return nil, appErrors.NewValidation("invalid request payload", map[string]string{
			"identifier": "username or email identifier is required",
		})
	}

	admin, err := s.platformAdminRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		s.log.WarnContext(ctx, "platform admin login failed: user not found", slog.String("identifier", identifier))
		return nil, appErrors.NewUnauthorized("invalid username or password")
	}

	if admin.Status != "active" {
		s.log.WarnContext(ctx, "platform admin login rejected: account inactive",
			slog.String("id", admin.ID.String()),
			slog.String("status", admin.Status),
		)
		return nil, appErrors.NewForbidden("account is inactive or suspended")
	}

	if err := s.hasher.Compare(admin.PasswordHash, req.Password); err != nil {
		s.log.WarnContext(ctx, "platform admin login failed: password mismatch", slog.String("id", admin.ID.String()))
		return nil, appErrors.NewUnauthorized("invalid username or password")
	}

	tokens, err := s.jwtManager.GenerateTokenPair(
		admin.ID,
		nil, // No tenant ID for platform admins
		admin.Email,
		auth.RolePlatformAdmin,
		auth.UserTypePlatformAdmin,
	)
	if err != nil {
		return nil, appErrors.NewInternal(fmt.Errorf("failed to issue tokens: %w", err))
	}

	s.log.InfoContext(ctx, "platform admin logged in successfully", slog.String("admin_id", admin.ID.String()))

	return &dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		User: dto.UserProfile{
			ID:       admin.ID,
			Name:     admin.Username,
			Email:    admin.Email,
			Role:     auth.RolePlatformAdmin,
			UserType: auth.UserTypePlatformAdmin,
			TenantID: nil,
		},
	}, nil
}

// LoginTenantUser authenticates a user belonging to a business tenant
func (s *authService) LoginTenantUser(ctx context.Context, req dto.TenantLoginRequest) (*dto.TokenResponse, error) {
	var user *dto.UserProfile
	var passwordHash string
	var status string

	if req.TenantID != nil {
		u, err := s.tenantUserRepo.GetByTenantAndEmail(ctx, *req.TenantID, req.Email)
		if err != nil {
			s.log.WarnContext(ctx, "tenant login failed: user not found in tenant",
				slog.String("tenant_id", req.TenantID.String()),
				slog.String("email", req.Email),
			)
			return nil, appErrors.NewUnauthorized("invalid email or password")
		}
		passwordHash = u.PasswordHash
		status = u.Status
		user = &dto.UserProfile{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			Role:     u.Role,
			UserType: auth.UserTypeTenantUser,
			TenantID: &u.TenantID,
		}
	} else {
		u, err := s.tenantUserRepo.GetByEmail(ctx, req.Email)
		if err != nil {
			s.log.WarnContext(ctx, "tenant login failed: user not found", slog.String("email", req.Email))
			return nil, appErrors.NewUnauthorized("invalid email or password")
		}
		passwordHash = u.PasswordHash
		status = u.Status
		user = &dto.UserProfile{
			ID:       u.ID,
			Name:     u.Name,
			Email:    u.Email,
			Role:     u.Role,
			UserType: auth.UserTypeTenantUser,
			TenantID: &u.TenantID,
		}
	}

	if status != "active" {
		s.log.WarnContext(ctx, "tenant login rejected: account inactive",
			slog.String("user_id", user.ID.String()),
			slog.String("status", status),
		)
		return nil, appErrors.NewForbidden("account is inactive or suspended")
	}

	if err := s.hasher.Compare(passwordHash, req.Password); err != nil {
		s.log.WarnContext(ctx, "tenant login failed: password mismatch", slog.String("user_id", user.ID.String()))
		return nil, appErrors.NewUnauthorized("invalid email or password")
	}

	tokens, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.TenantID,
		user.Email,
		user.Role,
		user.UserType,
	)
	if err != nil {
		return nil, appErrors.NewInternal(fmt.Errorf("failed to issue tokens: %w", err))
	}

	s.log.InfoContext(ctx, "tenant user logged in successfully",
		slog.String("user_id", user.ID.String()),
		slog.String("tenant_id", user.TenantID.String()),
		slog.String("role", user.Role),
	)

	return &dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		User:         *user,
	}, nil
}

// RefreshToken validates an existing refresh token and issues a rotated new token pair
func (s *authService) RefreshToken(ctx context.Context, refreshTokenStr string) (*dto.TokenResponse, error) {
	claims, err := s.jwtManager.ValidateToken(refreshTokenStr)
	if err != nil {
		return nil, appErrors.NewUnauthorized("invalid or expired refresh token")
	}

	// Validate token type if present to prevent using access tokens as refresh tokens
	if claims.TokenType != "" && claims.TokenType != auth.TokenTypeRefresh {
		return nil, appErrors.NewUnauthorized("provided token is not a refresh token")
	}

	// Check if token was blacklisted
	if s.blacklistRepo != nil {
		revoked, err := s.blacklistRepo.IsTokenRevoked(ctx, claims.TokenID)
		if err != nil {
			s.log.WarnContext(ctx, "error checking token revocation status", slog.String("error", err.Error()))
		}
		if revoked {
			s.log.WarnContext(ctx, "attempted use of revoked refresh token", slog.String("token_id", claims.TokenID))
			return nil, appErrors.NewUnauthorized("refresh token has been revoked")
		}

		// Check user-wide revocation (e.g. after password reset)
		if claims.IssuedAt != nil {
			userRevoked, err := s.blacklistRepo.IsUserTokenRevoked(ctx, claims.UserID, claims.IssuedAt.Time)
			if err == nil && userRevoked {
				s.log.WarnContext(ctx, "attempted use of refresh token issued before password reset",
					slog.String("user_id", claims.UserID.String()),
				)
				return nil, appErrors.NewUnauthorized("refresh token has been revoked")
			}
		}
	}

	// Validate that the user exists and remains active BEFORE rotating/issuing new tokens
	profile, err := s.GetProfile(ctx, claims)
	if err != nil {
		return nil, err
	}

	// Issue new token pair
	tokens, err := s.jwtManager.GenerateTokenPair(
		profile.ID,
		profile.TenantID,
		profile.Email,
		profile.Role,
		profile.UserType,
	)
	if err != nil {
		return nil, appErrors.NewInternal(fmt.Errorf("failed to refresh tokens: %w", err))
	}

	// Rotate: blacklist the used refresh token only after new tokens are safely generated
	if s.blacklistRepo != nil && claims.TokenID != "" && claims.ExpiresAt != nil {
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining > 0 {
			if err := s.blacklistRepo.RevokeToken(ctx, claims.TokenID, remaining); err != nil {
				s.log.WarnContext(ctx, "failed to blacklist rotated refresh token", slog.String("error", err.Error()))
			}
		}
	}

	return &dto.TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		TokenType:    tokens.TokenType,
		ExpiresIn:    tokens.ExpiresIn,
		User:         *profile,
	}, nil
}

// Logout invalidates the active token in the Redis blacklist
func (s *authService) Logout(ctx context.Context, tokenID string, remainingTTL time.Duration) error {
	if s.blacklistRepo == nil || tokenID == "" {
		return nil
	}

	if remainingTTL <= 0 {
		remainingTTL = 15 * time.Minute
	}

	if err := s.blacklistRepo.RevokeToken(ctx, tokenID, remainingTTL); err != nil {
		s.log.WarnContext(ctx, "failed to revoke token in redis",
			slog.String("token_id", tokenID),
			slog.String("error", err.Error()),
		)
		return appErrors.NewInternal(fmt.Errorf("failed to complete logout: %w", err))
	}

	s.log.InfoContext(ctx, "user logged out and token invalidated", slog.String("token_id", tokenID))
	return nil
}

// RevokeRefreshToken validates and invalidates an existing refresh token in the Redis blacklist
func (s *authService) RevokeRefreshToken(ctx context.Context, refreshTokenStr string) error {
	if s.blacklistRepo == nil || refreshTokenStr == "" {
		return nil
	}

	claims, err := s.jwtManager.ValidateToken(refreshTokenStr)
	if err != nil || claims.TokenID == "" {
		return nil // Ignore invalid tokens during cleanup
	}

	var remaining time.Duration
	if claims.ExpiresAt != nil {
		remaining = time.Until(claims.ExpiresAt.Time)
	}
	if remaining <= 0 {
		remaining = 15 * time.Minute
	}

	if err := s.blacklistRepo.RevokeToken(ctx, claims.TokenID, remaining); err != nil {
		s.log.WarnContext(ctx, "failed to revoke refresh token in redis",
			slog.String("token_id", claims.TokenID),
			slog.String("error", err.Error()),
		)
		return appErrors.NewInternal(fmt.Errorf("failed to revoke refresh token: %w", err))
	}

	s.log.InfoContext(ctx, "refresh token revoked", slog.String("token_id", claims.TokenID))
	return nil
}

// GetProfile retrieves the profile for an authenticated identity
func (s *authService) GetProfile(ctx context.Context, claims *auth.CustomClaims) (*dto.UserProfile, error) {
	if claims.UserType == auth.UserTypePlatformAdmin {
		if s.platformAdminRepo == nil {
			return &dto.UserProfile{
				ID:       claims.UserID,
				Name:     claims.Email,
				Email:    claims.Email,
				Role:     auth.RolePlatformAdmin,
				UserType: auth.UserTypePlatformAdmin,
				TenantID: nil,
			}, nil
		}

		admin, err := s.platformAdminRepo.GetByID(ctx, claims.UserID)
		if err != nil {
			return nil, appErrors.NewUnauthorized("platform administrator not found")
		}
		if admin.Status != "active" {
			return nil, appErrors.NewForbidden("account is inactive or suspended")
		}
		return &dto.UserProfile{
			ID:       admin.ID,
			Name:     admin.Username,
			Email:    admin.Email,
			Role:     auth.RolePlatformAdmin,
			UserType: auth.UserTypePlatformAdmin,
			TenantID: nil,
		}, nil
	}

	if claims.TenantID == nil {
		return nil, appErrors.NewUnauthorized("missing tenant context")
	}

	if s.tenantUserRepo == nil {
		return &dto.UserProfile{
			ID:       claims.UserID,
			Name:     claims.Email,
			Email:    claims.Email,
			Role:     claims.Role,
			UserType: auth.UserTypeTenantUser,
			TenantID: claims.TenantID,
		}, nil
	}

	user, err := s.tenantUserRepo.GetByID(ctx, *claims.TenantID, claims.UserID)
	if err != nil {
		return nil, appErrors.NewUnauthorized("user not found within tenant")
	}
	if user.Status != "active" {
		return nil, appErrors.NewForbidden("account is inactive or suspended")
	}

	return &dto.UserProfile{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		Role:     user.Role,
		UserType: auth.UserTypeTenantUser,
		TenantID: &user.TenantID,
	}, nil
}

// ForgotPassword handles initiating a password recovery flow
func (s *authService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest, clientIP string) error {
	if s.metrics != nil {
		s.metrics.IncPasswordResetRequests()
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	// Abuse Protection & Rate Limiting
	if s.passwordResetRepo != nil {
		// 1. IP rate limit
		if clientIP != "" {
			ipKey := "password_reset:rate_limit:ip:" + clientIP
			allowed, err := s.passwordResetRepo.CheckRateLimit(ctx, ipKey, s.pwCfg.MaxIPRequests, s.pwCfg.RateLimitWindow)
			if err != nil {
				s.log.WarnContext(ctx, "failed to check IP rate limit in redis", slog.String("error", err.Error()))
			} else if !allowed {
				if s.metrics != nil {
					s.metrics.IncPasswordResetRateLimited()
				}
				s.log.WarnContext(ctx, "forgot-password request rejected: IP rate limit exceeded", slog.String("ip", clientIP))
				return appErrors.NewTooManyRequests("too many password reset requests from this IP, please try again later")
			}
		}

		// 2. Email rate limit
		emailKey := "password_reset:rate_limit:email:" + email
		allowed, err := s.passwordResetRepo.CheckRateLimit(ctx, emailKey, s.pwCfg.MaxEmailRequests, s.pwCfg.RateLimitWindow)
		if err != nil {
			s.log.WarnContext(ctx, "failed to check email rate limit in redis", slog.String("error", err.Error()))
		} else if !allowed {
			if s.metrics != nil {
				s.metrics.IncPasswordResetRateLimited()
			}
			s.log.WarnContext(ctx, "forgot-password request rejected: email rate limit exceeded", slog.String("email", email))
			return appErrors.NewTooManyRequests("too many password reset requests for this email, please try again later")
		}

		// 3. Cooldown check
		inCooldown, err := s.passwordResetRepo.CheckCooldown(ctx, email)
		if err != nil {
			s.log.WarnContext(ctx, "failed to check cooldown in redis", slog.String("error", err.Error()))
		} else if inCooldown {
			if s.metrics != nil {
				s.metrics.IncPasswordResetRateLimited()
			}
			s.log.WarnContext(ctx, "forgot-password request rejected: cooldown active", slog.String("email", email))
			return appErrors.NewTooManyRequests("a password reset code was recently requested, please wait before requesting another")
		}
	}

	// Account lookup - anti-enumeration: do not reveal non-existence
	var userID uuid.UUID
	var tenantID *uuid.UUID
	var userType string
	accountFound := false

	// Check tenant user first
	if s.tenantUserRepo != nil {
		u, err := s.tenantUserRepo.GetByEmail(ctx, email)
		if err == nil && u != nil {
			if u.Status == "active" {
				userID = u.ID
				tenantID = &u.TenantID
				userType = auth.UserTypeTenantUser
				accountFound = true
			} else {
				s.log.WarnContext(ctx, "forgot-password requested for inactive tenant user", slog.String("email", email))
			}
		}
	}

	// Check platform admin if not found in tenant users
	if !accountFound && s.platformAdminRepo != nil {
		admin, err := s.platformAdminRepo.GetByIdentifier(ctx, email)
		if err == nil && admin != nil {
			if admin.Status == "active" {
				userID = admin.ID
				tenantID = nil
				userType = auth.UserTypePlatformAdmin
				accountFound = true
			} else {
				s.log.WarnContext(ctx, "forgot-password requested for inactive platform admin", slog.String("email", email))
			}
		}
	}

	// If account does not exist or is inactive, return nil (generic message preserves anti-enumeration)
	if !accountFound {
		s.log.InfoContext(ctx, "forgot-password requested for unknown or inactive email", slog.String("email", email))
		return nil
	}

	// Generate cryptographically secure numeric OTP
	otp, err := generateNumericOTP(s.pwCfg.OTPLength)
	if err != nil {
		return appErrors.NewInternal(fmt.Errorf("failed to generate secure OTP: %w", err))
	}

	// Store hashed OTP in Redis
	otpHash := hashOTP(otp)
	now := time.Now().UTC()
	otpData := &domain.PasswordResetOTP{
		OTPHash:   otpHash,
		Attempts:  0,
		UserID:    userID,
		TenantID:  tenantID,
		UserType:  userType,
		Email:     email,
		CreatedAt: now,
		ExpiresAt: now.Add(s.pwCfg.OTPExpiry),
	}

	if s.passwordResetRepo != nil {
		if err := s.passwordResetRepo.StoreOTP(ctx, email, otpData, s.pwCfg.OTPExpiry); err != nil {
			s.log.ErrorContext(ctx, "failed to store OTP in repository", slog.String("error", err.Error()))
			return appErrors.NewInternal(fmt.Errorf("failed to persist recovery OTP: %w", err))
		}
		// Set resend cooldown
		_ = s.passwordResetRepo.SetCooldown(ctx, email, s.pwCfg.ResendCooldown)
	}

	// Dispatch email
	if s.mailer != nil {
		if err := s.mailer.SendPasswordResetOTP(ctx, email, otp, s.pwCfg.OTPExpiry); err != nil {
			s.log.ErrorContext(ctx, "failed to dispatch password recovery email", slog.String("error", err.Error()))
			return appErrors.NewInternal(fmt.Errorf("failed to send recovery email: %w", err))
		}
	}

	if s.metrics != nil {
		s.metrics.IncPasswordResetOTPSent()
	}

	s.log.InfoContext(ctx, "password reset OTP generated and dispatched",
		slog.String("email", email),
		slog.String("user_type", userType),
	)

	return nil
}

// VerifyResetOTP validates the OTP and issues a short-lived password reset token
func (s *authService) VerifyResetOTP(ctx context.Context, req dto.VerifyResetOTPRequest, clientIP string) (*dto.VerifyResetOTPResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if s.passwordResetRepo != nil && clientIP != "" {
		ipKey := "password_reset:rate_limit:verify_ip:" + clientIP
		allowed, _ := s.passwordResetRepo.CheckRateLimit(ctx, ipKey, 20, s.pwCfg.RateLimitWindow)
		if !allowed {
			if s.metrics != nil {
				s.metrics.IncPasswordResetRateLimited()
			}
			return nil, appErrors.NewTooManyRequests("too many verification attempts, please try again later")
		}
	}

	if s.passwordResetRepo == nil {
		return nil, appErrors.NewInternal(fmt.Errorf("password recovery repository not available"))
	}

	otpData, err := s.passwordResetRepo.GetOTP(ctx, email)
	if err != nil {
		return nil, appErrors.NewInternal(err)
	}
	if otpData == nil {
		if s.metrics != nil {
			s.metrics.IncPasswordResetOTPVerifyFailed()
		}
		return nil, appErrors.NewBadRequest("invalid or expired OTP")
	}

	// Check attempt limit
	if otpData.Attempts >= s.pwCfg.MaxAttempts {
		_ = s.passwordResetRepo.DeleteOTP(ctx, email)
		if s.metrics != nil {
			s.metrics.IncPasswordResetOTPVerifyFailed()
		}
		return nil, appErrors.NewBadRequest("maximum verification attempts exceeded, please request a new OTP")
	}

	// Increment attempt counter
	attempts, err := s.passwordResetRepo.IncrementOTPAttempts(ctx, email)
	if err != nil {
		s.log.WarnContext(ctx, "failed to increment OTP attempts", slog.String("error", err.Error()))
	}

	// Constant-time comparison between submitted OTP and stored hash
	submittedHash := hashOTP(req.OTP)
	if subtle.ConstantTimeCompare([]byte(submittedHash), []byte(otpData.OTPHash)) != 1 {
		if s.metrics != nil {
			s.metrics.IncPasswordResetOTPVerifyFailed()
		}
		remaining := s.pwCfg.MaxAttempts - attempts
		if remaining < 0 {
			remaining = 0
		}
		if attempts >= s.pwCfg.MaxAttempts {
			_ = s.passwordResetRepo.DeleteOTP(ctx, email)
			return nil, appErrors.NewBadRequest("maximum verification attempts exceeded, please request a new OTP")
		}
		return nil, appErrors.NewBadRequest(fmt.Sprintf("invalid OTP, %d attempts remaining", remaining))
	}

	// Invalidate OTP immediately upon success (single-use)
	_ = s.passwordResetRepo.DeleteOTP(ctx, email)

	// Issue dedicated single-use password reset token
	resetToken, tokenID, err := s.jwtManager.GeneratePasswordResetToken(
		otpData.UserID,
		otpData.TenantID,
		otpData.Email,
		otpData.UserType,
		s.pwCfg.TokenExpiry,
	)
	if err != nil {
		return nil, appErrors.NewInternal(fmt.Errorf("failed to issue password reset token: %w", err))
	}

	// Record reset token in repository for single-use guarantee
	now := time.Now().UTC()
	tokenData := &domain.PasswordResetTokenData{
		TokenID:   tokenID,
		UserID:    otpData.UserID,
		TenantID:  otpData.TenantID,
		UserType:  otpData.UserType,
		Email:     otpData.Email,
		CreatedAt: now,
		ExpiresAt: now.Add(s.pwCfg.TokenExpiry),
	}
	if err := s.passwordResetRepo.StoreResetToken(ctx, tokenID, tokenData, s.pwCfg.TokenExpiry); err != nil {
		return nil, appErrors.NewInternal(fmt.Errorf("failed to record reset token: %w", err))
	}

	if s.metrics != nil {
		s.metrics.IncPasswordResetOTPVerifySuccess()
	}

	s.log.InfoContext(ctx, "OTP verified successfully, issued reset token",
		slog.String("email", email),
		slog.String("token_id", tokenID),
	)

	return &dto.VerifyResetOTPResponse{
		ResetToken: resetToken,
	}, nil
}

// ResetPassword validates the reset token, updates the password hash, and revokes all active sessions/tokens
func (s *authService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest, clientIP string) error {
	if len(req.NewPassword) < 6 || len(req.NewPassword) > 72 {
		return appErrors.NewBadRequest("password must be between 6 and 72 characters")
	}

	if s.passwordResetRepo != nil && clientIP != "" {
		ipKey := "password_reset:rate_limit:reset_ip:" + clientIP
		allowed, _ := s.passwordResetRepo.CheckRateLimit(ctx, ipKey, 10, s.pwCfg.RateLimitWindow)
		if !allowed {
			if s.metrics != nil {
				s.metrics.IncPasswordResetRateLimited()
			}
			return appErrors.NewTooManyRequests("too many password reset attempts, please try again later")
		}
	}

	// Validate JWT
	claims, err := s.jwtManager.ValidatePasswordResetToken(req.ResetToken)
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncPasswordResetFailed()
		}
		return appErrors.NewBadRequest("invalid or expired reset token")
	}

	// Verify token hasn't been consumed yet
	if s.passwordResetRepo != nil {
		storedToken, err := s.passwordResetRepo.GetResetToken(ctx, claims.TokenID)
		if err != nil {
			return appErrors.NewInternal(err)
		}
		if storedToken == nil {
			if s.metrics != nil {
				s.metrics.IncPasswordResetFailed()
			}
			return appErrors.NewBadRequest("reset token has already been used or has expired")
		}

		// Invalidate reset token immediately (single-use)
		_ = s.passwordResetRepo.DeleteResetToken(ctx, claims.TokenID)
	}

	// Hash new password using existing hasher
	passwordHash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		if s.metrics != nil {
			s.metrics.IncPasswordResetFailed()
		}
		return appErrors.NewInternal(fmt.Errorf("failed to hash new password: %w", err))
	}

	// Update password in database with multi-tenant isolation
	if claims.UserType == auth.UserTypeTenantUser {
		if claims.TenantID == nil {
			if s.metrics != nil {
				s.metrics.IncPasswordResetFailed()
			}
			return appErrors.NewBadRequest("missing tenant context in reset token")
		}
		if s.tenantUserRepo == nil {
			return appErrors.NewInternal(fmt.Errorf("tenant user repository not available"))
		}
		if err := s.tenantUserRepo.UpdatePassword(ctx, *claims.TenantID, claims.UserID, passwordHash); err != nil {
			if s.metrics != nil {
				s.metrics.IncPasswordResetFailed()
			}
			return err
		}
	} else if claims.UserType == auth.UserTypePlatformAdmin {
		if s.platformAdminRepo == nil {
			return appErrors.NewInternal(fmt.Errorf("platform admin repository not available"))
		}
		if err := s.platformAdminRepo.UpdatePassword(ctx, claims.UserID, passwordHash); err != nil {
			if s.metrics != nil {
				s.metrics.IncPasswordResetFailed()
			}
			return err
		}
	} else {
		if s.metrics != nil {
			s.metrics.IncPasswordResetFailed()
		}
		return appErrors.NewBadRequest("invalid user type in reset token")
	}

	// Authentication invalidation: revoke all existing tokens/sessions for this user
	if s.blacklistRepo != nil {
		// Invalidate any tokens issued before now for the next 7 days (standard refresh token lifetime)
		_ = s.blacklistRepo.RevokeAllUserTokens(ctx, claims.UserID, 7*24*time.Hour)
	}

	if s.metrics != nil {
		s.metrics.IncPasswordResetSuccess()
	}

	s.log.InfoContext(ctx, "password successfully reset and active sessions invalidated",
		slog.String("user_id", claims.UserID.String()),
		slog.String("user_type", claims.UserType),
	)

	return nil
}

// Helpers

func generateNumericOTP(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	const digits = "0123456789"
	b := make([]byte, length)
	max := big.NewInt(int64(len(digits)))
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate random digit: %w", err)
		}
		b[i] = digits[num.Int64()]
	}
	return string(b), nil
}

func hashOTP(otp string) string {
	h := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(h[:])
}
