package server

import (
	"context"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/logger"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAdminRepo struct {
	admins map[uuid.UUID]*domain.PlatformAdmin
}

func newMockAdminRepo() *mockAdminRepo {
	return &mockAdminRepo{
		admins: make(map[uuid.UUID]*domain.PlatformAdmin),
	}
}

func (m *mockAdminRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error) {
	admin, ok := m.admins[id]
	if !ok {
		return nil, appErrors.NewNotFound("admin not found")
	}
	return admin, nil
}

func (m *mockAdminRepo) GetByIdentifier(ctx context.Context, identifier string) (*domain.PlatformAdmin, error) {
	for _, a := range m.admins {
		if a.Email == identifier || a.Username == identifier {
			return a, nil
		}
	}
	return nil, appErrors.NewNotFound("admin not found")
}

func (m *mockAdminRepo) Create(ctx context.Context, admin *domain.PlatformAdmin) error {
	admin.ID = uuid.New()
	admin.CreatedAt = time.Now().UTC()
	admin.UpdatedAt = time.Now().UTC()
	m.admins[admin.ID] = admin
	return nil
}

func (m *mockAdminRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	admin, ok := m.admins[id]
	if !ok {
		return appErrors.NewNotFound("admin not found")
	}
	admin.Status = status
	admin.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockAdminRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	admin, ok := m.admins[id]
	if !ok {
		return appErrors.NewNotFound("admin not found")
	}
	admin.PasswordHash = passwordHash
	admin.UpdatedAt = time.Now().UTC()
	return nil
}

func TestBootstrapPlatformAdmin(t *testing.T) {
	ctx := context.Background()
	hasher := auth.NewBcryptHasher()
	appLog := logger.Default().Logger

	t.Run("creates new platform admin if not found", func(t *testing.T) {
		repo := newMockAdminRepo()
		cfg := config.BootstrapAdminConfig{
			Enabled:  true,
			Email:    "newadmin@vyavsa.com",
			Password: "SecurePassword@123",
			Username: "newadmin",
			Phone:    "+919876543210",
		}

		err := BootstrapPlatformAdmin(ctx, repo, hasher, cfg, appLog)
		require.NoError(t, err)

		admin, err := repo.GetByIdentifier(ctx, "newadmin@vyavsa.com")
		require.NoError(t, err)
		assert.Equal(t, "newadmin", admin.Username)
		assert.Equal(t, "newadmin@vyavsa.com", admin.Email)
		assert.Equal(t, "active", admin.Status)
		assert.NoError(t, hasher.Compare(admin.PasswordHash, "SecurePassword@123"))
	})

	t.Run("synchronizes password if admin exists with different password", func(t *testing.T) {
		repo := newMockAdminRepo()
		oldHash, err := hasher.Hash("OldPassword@123")
		require.NoError(t, err)

		existingAdmin := &domain.PlatformAdmin{
			ID:           uuid.New(),
			Username:     "platform_admin",
			Email:        "admin@vyavsa.com",
			Status:       "active",
			PasswordHash: oldHash,
		}
		repo.admins[existingAdmin.ID] = existingAdmin

		cfg := config.BootstrapAdminConfig{
			Enabled:  true,
			Email:    "admin@vyavsa.com",
			Password: "UpdatedPassword@456",
			Username: "platform_admin",
		}

		err = BootstrapPlatformAdmin(ctx, repo, hasher, cfg, appLog)
		require.NoError(t, err)

		admin, err := repo.GetByIdentifier(ctx, "admin@vyavsa.com")
		require.NoError(t, err)
		assert.NoError(t, hasher.Compare(admin.PasswordHash, "UpdatedPassword@456"))
		assert.Error(t, hasher.Compare(admin.PasswordHash, "OldPassword@123"))
	})

	t.Run("skips when bootstrap is disabled", func(t *testing.T) {
		repo := newMockAdminRepo()
		cfg := config.BootstrapAdminConfig{
			Enabled:  false,
			Email:    "disabled@vyavsa.com",
			Password: "Password@123",
		}

		err := BootstrapPlatformAdmin(ctx, repo, hasher, cfg, appLog)
		require.NoError(t, err)

		_, err = repo.GetByIdentifier(ctx, "disabled@vyavsa.com")
		assert.Error(t, err)
	})

	t.Run("skips when credentials are empty", func(t *testing.T) {
		repo := newMockAdminRepo()
		cfg := config.BootstrapAdminConfig{
			Enabled:  true,
			Email:    "",
			Password: "",
		}

		err := BootstrapPlatformAdmin(ctx, repo, hasher, cfg, appLog)
		require.NoError(t, err)
		assert.Empty(t, repo.admins)
	})
}
