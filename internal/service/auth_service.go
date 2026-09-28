package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
)

// AuthService defines authentication business logic
type AuthService interface {
	LoginPlatformAdmin(ctx context.Context, req dto.PlatformLoginRequest) (*dto.TokenResponse, error)
	LoginTenantUser(ctx context.Context, req dto.TenantLoginRequest) (*dto.TokenResponse, error)
	RefreshToken(ctx context.Context, refreshTokenStr string) (*dto.TokenResponse, error)
	Logout(ctx context.Context, tokenID string, remainingTTL time.Duration) error
	GetProfile(ctx context.Context, claims *auth.CustomClaims) (*dto.UserProfile, error)
}

type authService struct {
	platformAdminRepo repository.PlatformAdminRepository
	tenantUserRepo    repository.TenantUserRepository
	blacklistRepo     repository.TokenBlacklistRepository
	hasher            auth.PasswordHasher
	jwtManager        auth.JWTManager
	log               *slog.Logger
}

// NewAuthService creates a new instance of AuthService
func NewAuthService(
	platformAdminRepo repository.PlatformAdminRepository,
	tenantUserRepo repository.TenantUserRepository,
	blacklistRepo repository.TokenBlacklistRepository,
	hasher auth.PasswordHasher,
	jwtManager auth.JWTManager,
	log *slog.Logger,
) AuthService {
	return &authService{
		platformAdminRepo: platformAdminRepo,
		tenantUserRepo:    tenantUserRepo,
		blacklistRepo:     blacklistRepo,
		hasher:            hasher,
		jwtManager:        jwtManager,
		log:               log,
	}
}

// LoginPlatformAdmin authenticates a platform-level administrator
func (s *authService) LoginPlatformAdmin(ctx context.Context, req dto.PlatformLoginRequest) (*dto.TokenResponse, error) {
	admin, err := s.platformAdminRepo.GetByIdentifier(ctx, req.Identifier)
	if err != nil {
		s.log.WarnContext(ctx, "platform admin login failed: user not found", slog.String("identifier", req.Identifier))
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

		// Rotate: blacklist the used refresh token
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining > 0 {
			_ = s.blacklistRepo.RevokeToken(ctx, claims.TokenID, remaining)
		}
	}

	// Issue new token pair
	tokens, err := s.jwtManager.GenerateTokenPair(
		claims.UserID,
		claims.TenantID,
		claims.Email,
		claims.Role,
		claims.UserType,
	)
	if err != nil {
		return nil, appErrors.NewInternal(fmt.Errorf("failed to refresh tokens: %w", err))
	}

	profile, err := s.GetProfile(ctx, claims)
	if err != nil {
		return nil, err
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

// GetProfile retrieves the profile for an authenticated identity
func (s *authService) GetProfile(ctx context.Context, claims *auth.CustomClaims) (*dto.UserProfile, error) {
	if claims.UserType == auth.UserTypePlatformAdmin {
		admin, err := s.platformAdminRepo.GetByID(ctx, claims.UserID)
		if err != nil {
			return nil, err
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

	user, err := s.tenantUserRepo.GetByID(ctx, *claims.TenantID, claims.UserID)
	if err != nil {
		return nil, err
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
