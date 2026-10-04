package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
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

func (m *mockPlatformAdminRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	if m.admin != nil && m.admin.ID == id {
		m.admin.PasswordHash = passwordHash
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

func (m *mockTenantUserRepo) UpdatePassword(ctx context.Context, tenantID, id uuid.UUID, passwordHash string) error {
	if m.user != nil && m.user.TenantID == tenantID && m.user.ID == id {
		m.user.PasswordHash = passwordHash
		return nil
	}
	return appErrors.NewNotFound("user not found")
}

func (m *mockTenantUserRepo) GetAdminByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantUser, error) {
	if m.user != nil && m.user.TenantID == tenantID && m.user.Role == "admin" {
		return m.user, nil
	}
	return nil, appErrors.NewNotFound("admin not found")
}

func (m *mockTenantUserRepo) Update(ctx context.Context, user *domain.TenantUser) error {
	m.user = user
	return nil
}

func (m *mockTenantUserRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	m.user = nil
	return nil
}

func (m *mockTenantUserRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, role, status string) ([]domain.TenantUser, int64, error) {
	if m.user != nil {
		return []domain.TenantUser{*m.user}, 1, nil
	}
	return nil, 0, nil
}

// MockBlacklistRepo
type mockBlacklistRepo struct {
	revoked     map[string]bool
	userRevoked map[uuid.UUID]time.Time
}

func (m *mockBlacklistRepo) RevokeToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	m.revoked[tokenID] = true
	return nil
}

func (m *mockBlacklistRepo) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	return m.revoked[tokenID], nil
}

func (m *mockBlacklistRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, ttl time.Duration) error {
	m.userRevoked[userID] = time.Now()
	return nil
}

func (m *mockBlacklistRepo) IsUserTokenRevoked(ctx context.Context, userID uuid.UUID, issuedAt time.Time) (bool, error) {
	t, ok := m.userRevoked[userID]
	if !ok {
		return false, nil
	}
	return issuedAt.Before(t) || issuedAt.Equal(t), nil
}

// MockPasswordResetRepo
type mockPasswordResetRepo struct {
	otps       map[string]*domain.PasswordResetOTP
	tokens     map[string]*domain.PasswordResetTokenData
	cooldowns  map[string]bool
	rateLimits map[string]int
}

func newMockPasswordResetRepo() *mockPasswordResetRepo {
	return &mockPasswordResetRepo{
		otps:       make(map[string]*domain.PasswordResetOTP),
		tokens:     make(map[string]*domain.PasswordResetTokenData),
		cooldowns:  make(map[string]bool),
		rateLimits: make(map[string]int),
	}
}

func (m *mockPasswordResetRepo) CheckRateLimit(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, error) {
	m.rateLimits[key]++
	return m.rateLimits[key] <= maxRequests, nil
}

func (m *mockPasswordResetRepo) CheckCooldown(ctx context.Context, email string) (bool, error) {
	return m.cooldowns[email], nil
}

func (m *mockPasswordResetRepo) SetCooldown(ctx context.Context, email string, cooldown time.Duration) error {
	m.cooldowns[email] = true
	return nil
}

func (m *mockPasswordResetRepo) StoreOTP(ctx context.Context, email string, otpData *domain.PasswordResetOTP, ttl time.Duration) error {
	m.otps[email] = otpData
	return nil
}

func (m *mockPasswordResetRepo) GetOTP(ctx context.Context, email string) (*domain.PasswordResetOTP, error) {
	return m.otps[email], nil
}

func (m *mockPasswordResetRepo) IncrementOTPAttempts(ctx context.Context, email string) (int, error) {
	otp := m.otps[email]
	if otp == nil {
		return 0, fmt.Errorf("otp not found")
	}
	otp.Attempts++
	return otp.Attempts, nil
}

func (m *mockPasswordResetRepo) DeleteOTP(ctx context.Context, email string) error {
	delete(m.otps, email)
	return nil
}

func (m *mockPasswordResetRepo) StoreResetToken(ctx context.Context, tokenID string, tokenData *domain.PasswordResetTokenData, ttl time.Duration) error {
	m.tokens[tokenID] = tokenData
	return nil
}

func (m *mockPasswordResetRepo) GetResetToken(ctx context.Context, tokenID string) (*domain.PasswordResetTokenData, error) {
	return m.tokens[tokenID], nil
}

func (m *mockPasswordResetRepo) DeleteResetToken(ctx context.Context, tokenID string) error {
	delete(m.tokens, tokenID)
	return nil
}

// MockMailer
type mockMailer struct {
	sentEmails []string
	sentOTPs   []string
}

func (m *mockMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	m.sentEmails = append(m.sentEmails, toEmail)
	m.sentOTPs = append(m.sentOTPs, otp)
	return nil
}

func TestAuthService(t *testing.T) {
	hasher := auth.NewBcryptHasher(bcrypt.MinCost)
	jwtManager := auth.NewJWTManager(config.JWTConfig{
		Secret:        "test-secret-must-be-at-least-32-bytes-long",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	})
	log := logger.Default().Logger
	appMetrics := metrics.New()
	pwCfg := config.PasswordResetConfig{
		OTPExpiry:        5 * time.Minute,
		TokenExpiry:      10 * time.Minute,
		OTPLength:        6,
		MaxAttempts:      5,
		ResendCooldown:   60 * time.Second,
		MaxEmailRequests: 3,
		MaxIPRequests:    10,
		RateLimitWindow:  15 * time.Minute,
	}

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
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			newMockPasswordResetRepo(),
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
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
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			newMockPasswordResetRepo(),
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
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
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			newMockPasswordResetRepo(),
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
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
		blacklist := &mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)}
		svc := NewAuthService(
			&mockPlatformAdminRepo{admin: admin},
			&mockTenantUserRepo{user: user},
			blacklist,
			newMockPasswordResetRepo(),
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
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

		// Cannot use access token as refresh token
		_, err = svc.RefreshToken(context.Background(), refreshResp.AccessToken)
		require.Error(t, err)
		appErr = appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeUnauthorized, appErr.Code)
	})

	t.Run("Platform admin refresh token rotation success", func(t *testing.T) {
		blacklist := &mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)}
		svc := NewAuthService(
			&mockPlatformAdminRepo{admin: admin},
			&mockTenantUserRepo{user: user},
			blacklist,
			newMockPasswordResetRepo(),
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		loginResp, err := svc.LoginPlatformAdmin(context.Background(), dto.PlatformLoginRequest{
			Identifier: "superadmin",
			Password:   password,
		})
		require.NoError(t, err)

		refreshResp, err := svc.RefreshToken(context.Background(), loginResp.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, refreshResp.AccessToken)
		assert.NotEmpty(t, refreshResp.RefreshToken)
		assert.Equal(t, auth.RolePlatformAdmin, refreshResp.User.Role)
	})

	t.Run("Refresh token rejected when user account becomes inactive", func(t *testing.T) {
		inactiveUser := &domain.TenantUser{
			ID:           uuid.New(),
			TenantID:     tenantID,
			Name:         "Inactive Store User",
			Role:         auth.RoleTenantUser,
			Email:        "inactive@store.com",
			Status:       "inactive",
			PasswordHash: passHash,
		}
		blacklist := &mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)}
		svc := NewAuthService(
			&mockPlatformAdminRepo{admin: admin},
			&mockTenantUserRepo{user: inactiveUser},
			blacklist,
			newMockPasswordResetRepo(),
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		tokens, err := jwtManager.GenerateTokenPair(inactiveUser.ID, &inactiveUser.TenantID, inactiveUser.Email, inactiveUser.Role, auth.UserTypeTenantUser)
		require.NoError(t, err)

		_, err = svc.RefreshToken(context.Background(), tokens.RefreshToken)
		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeForbidden, appErr.Code)
	})

	// ========================================================
	// PASSWORD RECOVERY / FORGOT-PASSWORD TESTS
	// ========================================================

	t.Run("ForgotPassword with valid tenant email sends OTP and sets cooldown", func(t *testing.T) {
		resetRepo := newMockPasswordResetRepo()
		mail := &mockMailer{}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			resetRepo,
			mail,
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "john@store.com"}, "127.0.0.1")
		require.NoError(t, err)

		require.Len(t, mail.sentEmails, 1)
		assert.Equal(t, "john@store.com", mail.sentEmails[0])
		assert.Len(t, mail.sentOTPs[0], 6)

		// Verify OTP was stored in repo
		storedOTP, err := resetRepo.GetOTP(context.Background(), "john@store.com")
		require.NoError(t, err)
		require.NotNil(t, storedOTP)
		assert.Equal(t, userID, storedOTP.UserID)
		assert.Equal(t, &tenantID, storedOTP.TenantID)
		assert.Equal(t, auth.UserTypeTenantUser, storedOTP.UserType)

		// Verify cooldown was set
		inCooldown, err := resetRepo.CheckCooldown(context.Background(), "john@store.com")
		require.NoError(t, err)
		assert.True(t, inCooldown)
	})

	t.Run("ForgotPassword with unknown email returns success (anti-enumeration)", func(t *testing.T) {
		resetRepo := newMockPasswordResetRepo()
		mail := &mockMailer{}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			resetRepo,
			mail,
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "nonexistent@store.com"}, "127.0.0.1")
		require.NoError(t, err)
		assert.Empty(t, mail.sentEmails)
	})

	t.Run("ForgotPassword rejects when in cooldown", func(t *testing.T) {
		resetRepo := newMockPasswordResetRepo()
		resetRepo.cooldowns["john@store.com"] = true
		mail := &mockMailer{}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			resetRepo,
			mail,
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "john@store.com"}, "127.0.0.1")
		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeTooManyRequests, appErr.Code)
	})

	t.Run("ForgotPassword rejects when email rate limit exceeded", func(t *testing.T) {
		resetRepo := newMockPasswordResetRepo()
		resetRepo.rateLimits["password_reset:rate_limit:email:john@store.com"] = 3 // Max is 3
		mail := &mockMailer{}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			resetRepo,
			mail,
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "john@store.com"}, "127.0.0.1")
		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeTooManyRequests, appErr.Code)
	})

	t.Run("VerifyResetOTP success returns dedicated reset token and deletes OTP", func(t *testing.T) {
		resetRepo := newMockPasswordResetRepo()
		mail := &mockMailer{}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			resetRepo,
			mail,
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		// 1. Request recovery
		err := svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "john@store.com"}, "127.0.0.1")
		require.NoError(t, err)
		require.Len(t, mail.sentOTPs, 1)
		otp := mail.sentOTPs[0]

		// 2. Verify OTP
		resp, err := svc.VerifyResetOTP(context.Background(), dto.VerifyResetOTPRequest{
			Email: "john@store.com",
			OTP:   otp,
		}, "127.0.0.1")
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.NotEmpty(t, resp.ResetToken)

		// 3. OTP must be consumed and deleted (single-use)
		storedOTP, err := resetRepo.GetOTP(context.Background(), "john@store.com")
		require.NoError(t, err)
		assert.Nil(t, storedOTP)

		// 4. Token must be valid password reset claims
		claims, err := jwtManager.ValidatePasswordResetToken(resp.ResetToken)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.UserID)
		assert.Equal(t, &tenantID, claims.TenantID)
		assert.Equal(t, auth.PurposePasswordReset, claims.Purpose)
	})

	t.Run("VerifyResetOTP with invalid OTP increments attempt counter and fails", func(t *testing.T) {
		resetRepo := newMockPasswordResetRepo()
		mail := &mockMailer{}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			resetRepo,
			mail,
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		_ = svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "john@store.com"}, "127.0.0.1")

		_, err := svc.VerifyResetOTP(context.Background(), dto.VerifyResetOTPRequest{
			Email: "john@store.com",
			OTP:   "000000",
		}, "127.0.0.1")
		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeBadRequest, appErr.Code)

		storedOTP, err := resetRepo.GetOTP(context.Background(), "john@store.com")
		require.NoError(t, err)
		require.NotNil(t, storedOTP)
		assert.Equal(t, 1, storedOTP.Attempts)
	})

	t.Run("VerifyResetOTP exceeding max attempts invalidates OTP", func(t *testing.T) {
		resetRepo := newMockPasswordResetRepo()
		mail := &mockMailer{}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			&mockTenantUserRepo{user: user},
			&mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)},
			resetRepo,
			mail,
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		_ = svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "john@store.com"}, "127.0.0.1")

		// Fail 5 times
		var lastErr error
		for i := 0; i < 5; i++ {
			_, lastErr = svc.VerifyResetOTP(context.Background(), dto.VerifyResetOTPRequest{
				Email: "john@store.com",
				OTP:   "000000",
			}, "127.0.0.1")
		}

		// 5th attempt must return maximum attempts exceeded
		require.Error(t, lastErr)
		assert.Contains(t, lastErr.Error(), "exceeded")

		// Subsequent attempt should fail with invalid/expired because OTP was deleted
		_, err := svc.VerifyResetOTP(context.Background(), dto.VerifyResetOTPRequest{
			Email: "john@store.com",
			OTP:   "000000",
		}, "127.0.0.1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired OTP")

		storedOTP, _ := resetRepo.GetOTP(context.Background(), "john@store.com")
		assert.Nil(t, storedOTP)
	})

	t.Run("ResetPassword updates password and revokes previous tokens", func(t *testing.T) {
		resetRepo := newMockPasswordResetRepo()
		blacklist := &mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)}
		mail := &mockMailer{}
		tenantRepo := &mockTenantUserRepo{user: user}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			tenantRepo,
			blacklist,
			resetRepo,
			mail,
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		// 1. Initial login to get a refresh token
		loginResp, err := svc.LoginTenantUser(context.Background(), dto.TenantLoginRequest{
			Email:    "john@store.com",
			Password: password,
		})
		require.NoError(t, err)

		// 2. Request OTP and verify to get reset token
		_ = svc.ForgotPassword(context.Background(), dto.ForgotPasswordRequest{Email: "john@store.com"}, "127.0.0.1")
		verifyResp, err := svc.VerifyResetOTP(context.Background(), dto.VerifyResetOTPRequest{
			Email: "john@store.com",
			OTP:   mail.sentOTPs[0],
		}, "127.0.0.1")
		require.NoError(t, err)

		// 3. Reset password
		newPass := "NewPass98765!"
		err = svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
			ResetToken:  verifyResp.ResetToken,
			NewPassword: newPass,
		}, "127.0.0.1")
		require.NoError(t, err)

		// 4. Verify user password updated in repo
		err = hasher.Compare(tenantRepo.user.PasswordHash, newPass)
		assert.NoError(t, err)

		// 5. Old password must fail login
		_, err = svc.LoginTenantUser(context.Background(), dto.TenantLoginRequest{
			Email:    "john@store.com",
			Password: password,
		})
		assert.Error(t, err)

		// 6. New password must succeed login
		newLoginResp, err := svc.LoginTenantUser(context.Background(), dto.TenantLoginRequest{
			Email:    "john@store.com",
			Password: newPass,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, newLoginResp.AccessToken)

		// 7. Old refresh token issued before reset must be rejected
		_, err = svc.RefreshToken(context.Background(), loginResp.RefreshToken)
		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeUnauthorized, appErr.Code)

		// 8. Reset token cannot be reused
		err = svc.ResetPassword(context.Background(), dto.ResetPasswordRequest{
			ResetToken:  verifyResp.ResetToken,
			NewPassword: "AnotherPassword123!",
		}, "127.0.0.1")
		require.Error(t, err)
	})

	t.Run("RevokeRefreshToken revokes active refresh token", func(t *testing.T) {
		blacklist := &mockBlacklistRepo{revoked: make(map[string]bool), userRevoked: make(map[uuid.UUID]time.Time)}
		user := &domain.TenantUser{
			ID:           uuid.New(),
			TenantID:     uuid.New(),
			Email:        "user@store.com",
			PasswordHash: "$2a$10$xyz",
			Role:         "owner",
			Status:       "active",
		}
		tenantRepo := &mockTenantUserRepo{user: user}
		svc := NewAuthService(
			&mockPlatformAdminRepo{},
			tenantRepo,
			blacklist,
			newMockPasswordResetRepo(),
			&mockMailer{},
			hasher,
			jwtManager,
			pwCfg,
			appMetrics,
			log,
		)

		tokens, err := jwtManager.GenerateTokenPair(user.ID, &user.TenantID, user.Email, user.Role, auth.UserTypeTenantUser)
		require.NoError(t, err)

		// Revoke refresh token
		err = svc.RevokeRefreshToken(context.Background(), tokens.RefreshToken)
		require.NoError(t, err)

		// Verify refresh token is rejected now
		_, err = svc.RefreshToken(context.Background(), tokens.RefreshToken)
		require.Error(t, err)
		appErr := appErrors.FromError(err)
		assert.Equal(t, appErrors.CodeUnauthorized, appErr.Code)
	})
}
