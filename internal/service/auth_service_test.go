package service

import (
	"context"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/logger"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// MockPlatformAdminRepo
type mockPlatformAdminRepo struct {
	admin *domain.PlatformAdmin
}

func (m *mockPlatformAdminRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error) {
	if m.admin != nil && m.admin.ID == id {
		return m.admin, nil
	}
	return nil, appErrors.NewNotFound("admin not found")
}

func (m *mockPlatformAdminRepo) GetByIdentifier(ctx context.Context, identifier string) (*domain.PlatformAdmin, error) {
	if m.admin != nil && (m.admin.Username == identifier || m.admin.Email == identifier) {
		return m.admin, nil
	}
	return nil, appErrors.NewNotFound("admin not found")
}

func (m *mockPlatformAdminRepo) Create(ctx context.Context, admin *domain.PlatformAdmin) error {
	m.admin = admin
	return nil
}

func (m *mockPlatformAdminRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if m.admin != nil && m.admin.ID == id {
		m.admin.Status = status
		return nil
	}
	return appErrors.NewNotFound("admin not found")
}

// MockTenantUserRepo
type mockTenantUserRepo struct {
	user *domain.TenantUser
}

func (m *mockTenantUserRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantUser, error) {
	if m.user != nil && m.user.TenantID == tenantID && m.user.ID == id {
		return m.user, nil
	}
	return nil, appErrors.NewNotFound("user not found")
}

func (m *mockTenantUserRepo) GetByEmail(ctx context.Context, email string) (*domain.TenantUser, error) {
	if m.user != nil && m.user.Email == email {
		return m.user, nil
	}
	return nil, appErrors.NewNotFound("user not found")
}

func (m *mockTenantUserRepo) GetByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.TenantUser, error) {
	if m.user != nil && m.user.TenantID == tenantID && m.user.Email == email {
		return m.user, nil
	}
	return nil, appErrors.NewNotFound("user not found")
}

func (m *mockTenantUserRepo) Create(ctx context.Context, user *domain.TenantUser) error {
	m.user = user
	return nil
}

func (m *mockTenantUserRepo) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	if m.user != nil && m.user.TenantID == tenantID && m.user.ID == id {
		m.user.Status = status
		return nil
	}
	return appErrors.NewNotFound("user not found")
}

// MockBlacklistRepo
type mockBlacklistRepo struct {
	revoked map[string]bool
}

func (m *mockBlacklistRepo) RevokeToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	m.revoked[tokenID] = true
	return nil
}

func (m *mockBlacklistRepo) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	return m.revoked[tokenID], nil
}

func TestAuthService(t *testing.T) {
	hasher := auth.NewBcryptHasher(bcrypt.MinCost)
	jwtManager := auth.NewJWTManager(config.JWTConfig{
		Secret:        "test-secret-must-be-at-least-32-bytes-long",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	})
	log := logger.Default().Logger

	password := "Pass12345!"
	passHash, _ := hasher.Hash(password)

	adminID := uuid.New()
	admin := &domain.PlatformAdmin{
		ID:           adminID,
		Username:     "superadmin",
		Email:        "admin@vyavsa.com",
		Status:       "active",
		PasswordHash: passHash,
	}

	tenantID := uuid.New()
	userID := uuid.New()
	user := &domain.TenantUser{
		ID:           userID,
		TenantID:     tenantID,
		Name:         "John Store",
		Role:         auth.RoleTenantAdmin,
		Email:        "john@store.com",
		Status:       "active",
		PasswordHash: passHash,
	}

	t.Run("Platform admin login success", func(t *testing.T) {
		svc := NewAuthService(
			&mockPlatformAdminRepo{admin: admin},
			&mockTenantUserRepo{},
			&mockBlacklistRepo{revoked: make(map[string]bool)},
			hasher,
			jwtManager,
			log,
		)

		resp, err := svc.LoginPlatformAdmin(context.Background(), dto.PlatformLoginRequest{
			Identifier: "superadmin",
			Password:   password,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, auth.RolePlatformAdmin, resp.User.Role)
		assert.Equal(t, auth.UserTypePlatformAdmin, resp.User.UserType)
	})

	t.Run("Platform admin login invalid password returns 401", func(t *testing.T) {
		svc := NewAuthService(
			&mockPlatformAdminRepo{admin: admin},
			&mockTenantUserRepo{},
			&mockBlacklistRepo{revoked: make(map[string]bool)},
			hasher,
			jwtManager,
			log,
		)

		_, err := svc.LoginPlatformAdmin(context.Background(), dto.PlatformLoginRequest{
			Identifier: "superadmin",
			Password:   "wrong-password",
		})

		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeUnauthorized, appErr.Code)
	})

	t.Run("Tenant user login success", func(t *testing.T) {
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool)},
			hasher,
			jwtManager,
			log,
		)

		resp, err := svc.LoginTenantUser(context.Background(), dto.TenantLoginRequest{
			Email:    "john@store.com",
			Password: password,
		})

		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, auth.RoleTenantAdmin, resp.User.Role)
		assert.Equal(t, &tenantID, resp.User.TenantID)
	})

	t.Run("Refresh token rotation and blacklist", func(t *testing.T) {
		blacklist := &mockBlacklistRepo{revoked: make(map[string]bool)}
		svc := NewAuthService(
			&mockPlatformAdminRepo{admin: admin},
			&mockTenantUserRepo{user: user},
			blacklist,
			hasher,
			jwtManager,
			log,
		)

		loginResp, err := svc.LoginTenantUser(context.Background(), dto.TenantLoginRequest{
			Email:    "john@store.com",
			Password: password,
		})
		require.NoError(t, err)

		// Refresh using token
		refreshResp, err := svc.RefreshToken(context.Background(), loginResp.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, refreshResp.AccessToken)

		// Old refresh token must be revoked and rejected on re-use
		_, err = svc.RefreshToken(context.Background(), loginResp.RefreshToken)
		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeUnauthorized, appErr.Code)
	})
}
