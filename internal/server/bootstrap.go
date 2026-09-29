package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/repository"
	postgresRepo "github.com/Varunjp/vyavsa/internal/repository/postgres"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BootstrapPlatformAdminWithPool initializes or synchronizes the platform administrator account using a pgxpool connection
func BootstrapPlatformAdminWithPool(ctx context.Context, pool *pgxpool.Pool, cfg config.BootstrapAdminConfig, log *slog.Logger) error {
	if pool == nil {
		log.WarnContext(ctx, "database connection pool is nil, skipping platform admin bootstrap")
		return nil
	}
	repo := postgresRepo.NewPlatformAdminPostgres(pool)
	hasher := auth.NewBcryptHasher()
	return BootstrapPlatformAdmin(ctx, repo, hasher, cfg, log)
}

// BootstrapPlatformAdmin creates or synchronizes the platform administrator account using credentials fetched from environment configuration
func BootstrapPlatformAdmin(ctx context.Context, repo repository.PlatformAdminRepository, hasher auth.PasswordHasher, cfg config.BootstrapAdminConfig, log *slog.Logger) error {
	if !cfg.Enabled {
		log.InfoContext(ctx, "platform admin bootstrap is disabled by configuration")
		return nil
	}

	if cfg.Email == "" || cfg.Password == "" {
		log.WarnContext(ctx, "platform admin bootstrap skipped: PLATFORM_ADMIN_EMAIL or PLATFORM_ADMIN_PASSWORD is empty")
		return nil
	}

	username := cfg.Username
	if username == "" {
		username = "platform_admin"
	}

	// 1. Check if platform administrator already exists by email or username
	existing, err := repo.GetByIdentifier(ctx, cfg.Email)
	if err != nil && !appErrors.IsNotFound(err) {
		return fmt.Errorf("failed to lookup platform admin by email: %w", err)
	}

	if existing == nil {
		existing, err = repo.GetByIdentifier(ctx, username)
		if err != nil && !appErrors.IsNotFound(err) {
			return fmt.Errorf("failed to lookup platform admin by username: %w", err)
		}
	}

	// 2. If administrator exists, ensure credentials and status match environment configuration
	if existing != nil {
		// Verify if password matches the configured environment credentials
		if err := hasher.Compare(existing.PasswordHash, cfg.Password); err != nil {
			newHash, hashErr := hasher.Hash(cfg.Password)
			if hashErr != nil {
				return fmt.Errorf("failed to hash bootstrap admin password: %w", hashErr)
			}

			if err := repo.UpdatePassword(ctx, existing.ID, newHash); err != nil {
				return fmt.Errorf("failed to update platform admin password: %w", err)
			}

			log.InfoContext(ctx, "platform admin password synchronized from environment credentials",
				slog.String("email", cfg.Email),
				slog.String("username", username),
			)
		}

		if existing.Status != "active" {
			if err := repo.UpdateStatus(ctx, existing.ID, "active"); err != nil {
				return fmt.Errorf("failed to activate platform admin account: %w", err)
			}
		}

		log.InfoContext(ctx, "platform administrator verified and ready",
			slog.String("admin_id", existing.ID.String()),
			slog.String("email", existing.Email),
			slog.String("username", existing.Username),
		)
		return nil
	}

	// 3. Administrator does not exist, create new record from environment credentials
	passwordHash, err := hasher.Hash(cfg.Password)
	if err != nil {
		return fmt.Errorf("failed to hash bootstrap admin password: %w", err)
	}

	admin := &domain.PlatformAdmin{
		Username:     username,
		Email:        cfg.Email,
		Phone:        cfg.Phone,
		Status:       "active",
		PasswordHash: passwordHash,
	}

	if err := repo.Create(ctx, admin); err != nil {
		return fmt.Errorf("failed to create bootstrap platform admin: %w", err)
	}

	log.InfoContext(ctx, "bootstrapped platform administrator user from environment credentials",
		slog.String("admin_id", admin.ID.String()),
		slog.String("email", admin.Email),
		slog.String("username", admin.Username),
	)

	return nil
}
