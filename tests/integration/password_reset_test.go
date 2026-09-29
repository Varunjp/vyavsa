package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/config"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	authHandlerPkg "github.com/Varunjp/vyavsa/internal/handler/auth"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/middleware"
	"github.com/Varunjp/vyavsa/internal/service"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// In-Memory Test Repositories for Integration Testing

type memTenantUserRepo struct {
	users map[uuid.UUID]*domain.TenantUser
}

func newMemTenantUserRepo() *memTenantUserRepo {
	return &memTenantUserRepo{users: make(map[uuid.UUID]*domain.TenantUser)}
}

func (m *memTenantUserRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantUser, error) {
	u, ok := m.users[id]
	if !ok || u.TenantID != tenantID {
		return nil, appErrors.NewNotFound("tenant user not found")
	}
	return u, nil
}

func (m *memTenantUserRepo) GetByEmail(ctx context.Context, email string) (*domain.TenantUser, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, appErrors.NewNotFound("tenant user not found")
}

func (m *memTenantUserRepo) GetByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.TenantUser, error) {
	for _, u := range m.users {
		if u.TenantID == tenantID && u.Email == email {
			return u, nil
		}
	}
	return nil, appErrors.NewNotFound("tenant user not found")
}

func (m *memTenantUserRepo) GetAdminByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantUser, error) {
	for _, u := range m.users {
		if u.TenantID == tenantID && u.Role == "admin" {
			return u, nil
		}
	}
	return nil, appErrors.NewNotFound("admin not found")
}

func (m *memTenantUserRepo) Create(ctx context.Context, user *domain.TenantUser) error {
	m.users[user.ID] = user
	return nil
}

func (m *memTenantUserRepo) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	u, ok := m.users[id]
	if !ok || u.TenantID != tenantID {
		return appErrors.NewNotFound("tenant user not found")
	}
	u.Status = status
	return nil
}

func (m *memTenantUserRepo) UpdatePassword(ctx context.Context, tenantID, id uuid.UUID, passwordHash string) error {
	u, ok := m.users[id]
	if !ok || u.TenantID != tenantID {
		return appErrors.NewNotFound("tenant user not found")
	}
	u.PasswordHash = passwordHash
	return nil
}

func (m *memTenantUserRepo) Update(ctx context.Context, u *domain.TenantUser) error {
	m.users[u.ID] = u
	return nil
}

func (m *memTenantUserRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.users, id)
	return nil
}

func (m *memTenantUserRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, role, status string) ([]domain.TenantUser, int64, error) {
	var list []domain.TenantUser
	for _, u := range m.users {
		if u.TenantID == tenantID {
			list = append(list, *u)
		}
	}
	return list, int64(len(list)), nil
}

type memPlatformAdminRepo struct {
	admins map[uuid.UUID]*domain.PlatformAdmin
}

func newMemPlatformAdminRepo() *memPlatformAdminRepo {
	return &memPlatformAdminRepo{admins: make(map[uuid.UUID]*domain.PlatformAdmin)}
}

func (m *memPlatformAdminRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error) {
	admin, ok := m.admins[id]
	if !ok {
		return nil, appErrors.NewNotFound("admin not found")
	}
	return admin, nil
}

func (m *memPlatformAdminRepo) GetByIdentifier(ctx context.Context, identifier string) (*domain.PlatformAdmin, error) {
	for _, a := range m.admins {
		if a.Username == identifier || a.Email == identifier {
			return a, nil
		}
	}
	return nil, appErrors.NewNotFound("admin not found")
}

func (m *memPlatformAdminRepo) Create(ctx context.Context, admin *domain.PlatformAdmin) error {
	m.admins[admin.ID] = admin
	return nil
}

func (m *memPlatformAdminRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	admin, ok := m.admins[id]
	if !ok {
		return appErrors.NewNotFound("admin not found")
	}
	admin.Status = status
	return nil
}

func (m *memPlatformAdminRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	admin, ok := m.admins[id]
	if !ok {
		return appErrors.NewNotFound("admin not found")
	}
	admin.PasswordHash = passwordHash
	return nil
}

type memBlacklistRepo struct {
	revoked     map[string]bool
	userRevoked map[uuid.UUID]time.Time
}

func newMemBlacklistRepo() *memBlacklistRepo {
	return &memBlacklistRepo{
		revoked:     make(map[string]bool),
		userRevoked: make(map[uuid.UUID]time.Time),
	}
}

func (m *memBlacklistRepo) RevokeToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	m.revoked[tokenID] = true
	return nil
}

func (m *memBlacklistRepo) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	return m.revoked[tokenID], nil
}

func (m *memBlacklistRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, ttl time.Duration) error {
	m.userRevoked[userID] = time.Now()
	return nil
}

func (m *memBlacklistRepo) IsUserTokenRevoked(ctx context.Context, userID uuid.UUID, issuedAt time.Time) (bool, error) {
	revokedAt, ok := m.userRevoked[userID]
	if !ok {
		return false, nil
	}
	return issuedAt.Before(revokedAt) || issuedAt.Equal(revokedAt), nil
}

type memPasswordResetRepo struct {
	otps       map[string]*domain.PasswordResetOTP
	tokens     map[string]*domain.PasswordResetTokenData
	cooldowns  map[string]time.Time
	rateLimits map[string]int
}

func newMemPasswordResetRepo() *memPasswordResetRepo {
	return &memPasswordResetRepo{
		otps:       make(map[string]*domain.PasswordResetOTP),
		tokens:     make(map[string]*domain.PasswordResetTokenData),
		cooldowns:  make(map[string]time.Time),
		rateLimits: make(map[string]int),
	}
}

func (m *memPasswordResetRepo) CheckRateLimit(ctx context.Context, key string, maxRequests int, window time.Duration) (bool, error) {
	m.rateLimits[key]++
	return m.rateLimits[key] <= maxRequests, nil
}

func (m *memPasswordResetRepo) CheckCooldown(ctx context.Context, email string) (bool, error) {
	exp, ok := m.cooldowns[email]
	if !ok {
		return false, nil
	}
	return time.Now().Before(exp), nil
}

func (m *memPasswordResetRepo) SetCooldown(ctx context.Context, email string, cooldown time.Duration) error {
	m.cooldowns[email] = time.Now().Add(cooldown)
	return nil
}

func (m *memPasswordResetRepo) StoreOTP(ctx context.Context, email string, otpData *domain.PasswordResetOTP, ttl time.Duration) error {
	m.otps[email] = otpData
	return nil
}

func (m *memPasswordResetRepo) GetOTP(ctx context.Context, email string) (*domain.PasswordResetOTP, error) {
	otp, ok := m.otps[email]
	if !ok {
		return nil, nil
	}
	if time.Now().After(otp.ExpiresAt) {
		delete(m.otps, email)
		return nil, nil
	}
	return otp, nil
}

func (m *memPasswordResetRepo) IncrementOTPAttempts(ctx context.Context, email string) (int, error) {
	otp, ok := m.otps[email]
	if !ok {
		return 0, fmt.Errorf("otp not found")
	}
	otp.Attempts++
	return otp.Attempts, nil
}

func (m *memPasswordResetRepo) DeleteOTP(ctx context.Context, email string) error {
	delete(m.otps, email)
	return nil
}

func (m *memPasswordResetRepo) StoreResetToken(ctx context.Context, tokenID string, tokenData *domain.PasswordResetTokenData, ttl time.Duration) error {
	m.tokens[tokenID] = tokenData
	return nil
}

func (m *memPasswordResetRepo) GetResetToken(ctx context.Context, tokenID string) (*domain.PasswordResetTokenData, error) {
	t, ok := m.tokens[tokenID]
	if !ok {
		return nil, nil
	}
	if time.Now().After(t.ExpiresAt) {
		delete(m.tokens, tokenID)
		return nil, nil
	}
	return t, nil
}

func (m *memPasswordResetRepo) DeleteResetToken(ctx context.Context, tokenID string) error {
	delete(m.tokens, tokenID)
	return nil
}

type integrationMailer struct {
	sentEmails []string
	sentOTPs   []string
}

func (m *integrationMailer) SendPasswordResetOTP(ctx context.Context, toEmail, otp string, expiry time.Duration) error {
	m.sentEmails = append(m.sentEmails, toEmail)
	m.sentOTPs = append(m.sentOTPs, otp)
	return nil
}

func setupTestRouter(
	tenantRepo *memTenantUserRepo,
	adminRepo *memPlatformAdminRepo,
	blacklist *memBlacklistRepo,
	resetRepo *memPasswordResetRepo,
	mailer *integrationMailer,
	jwtManager auth.JWTManager,
	hasher auth.PasswordHasher,
	pwCfg config.PasswordResetConfig,
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	appMetrics := metrics.New()
	log := logger.Default().Logger

	authSvc := service.NewAuthService(
		adminRepo,
		tenantRepo,
		blacklist,
		resetRepo,
		mailer,
		hasher,
		jwtManager,
		pwCfg,
		appMetrics,
		log,
	)

	handler := authHandlerPkg.NewHandler(authSvc)

	// API v1 routes
	apiV1 := r.Group("/api/v1")
	{
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/tenant/login", handler.TenantLogin)
			authGroup.POST("/refresh", handler.RefreshToken)
			authGroup.POST("/forgot-password", handler.ForgotPassword)
			authGroup.POST("/verify-reset-otp", handler.VerifyResetOTP)
			authGroup.POST("/reset-password", handler.ResetPassword)
		}

		protected := apiV1.Group("")
		protected.Use(middleware.Authenticate(jwtManager, blacklist))
		{
			protected.GET("/auth/me", handler.GetMe)
		}
	}

	// Root /auth paths
	rootAuth := r.Group("/auth")
	{
		rootAuth.POST("/forgot-password", handler.ForgotPassword)
		rootAuth.POST("/verify-reset-otp", handler.VerifyResetOTP)
		rootAuth.POST("/reset-password", handler.ResetPassword)
	}

	return r
}

func TestPasswordRecoveryIntegration(t *testing.T) {
	hasher := auth.NewBcryptHasher(bcrypt.MinCost)
	jwtCfg := config.JWTConfig{
		Secret:        "integration-test-secret-at-least-32-chars-long",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	}
	jwtManager := auth.NewJWTManager(jwtCfg)

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

	initialPassword := "OriginalPass123!"
	initialHash, _ := hasher.Hash(initialPassword)

	tenantID := uuid.New()
	userID := uuid.New()
	tenantUser := &domain.TenantUser{
		ID:           userID,
		TenantID:     tenantID,
		Name:         "Alice Tenant",
		Role:         auth.RoleTenantAdmin,
		Email:        "alice@tenant.com",
		Status:       "active",
		PasswordHash: initialHash,
	}

	tenantRepo := newMemTenantUserRepo()
	tenantRepo.users[userID] = tenantUser

	adminRepo := newMemPlatformAdminRepo()
	blacklist := newMemBlacklistRepo()
	resetRepo := newMemPasswordResetRepo()
	mailer := &integrationMailer{}

	router := setupTestRouter(tenantRepo, adminRepo, blacklist, resetRepo, mailer, jwtManager, hasher, pwCfg)

	// 1. Forgot password with valid email
	t.Run("1. Forgot password with valid email returns generic response and sends OTP", func(t *testing.T) {
		body, _ := json.Marshal(dto.ForgotPasswordRequest{Email: "alice@tenant.com"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		assert.Equal(t, "If an account exists with this email, an OTP has been sent.", resp["message"])

		require.Len(t, mailer.sentEmails, 1)
		assert.Equal(t, "alice@tenant.com", mailer.sentEmails[0])
		assert.Len(t, mailer.sentOTPs[0], 6)
	})

	// 2. Forgot password with unknown email
	t.Run("2. Forgot password with unknown email returns generic response without revealing non-existence", func(t *testing.T) {
		// Clear cooldown for test isolation
		resetRepo.cooldowns["stranger@unknown.com"] = time.Time{}

		body, _ := json.Marshal(dto.ForgotPasswordRequest{Email: "stranger@unknown.com"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))
		assert.Equal(t, "If an account exists with this email, an OTP has been sent.", resp["message"])

		// No additional email should have been sent
		assert.Len(t, mailer.sentEmails, 1)
	})

	// 3. OTP verification with valid OTP
	var validResetToken string
	t.Run("3. OTP verification with valid OTP returns dedicated reset token", func(t *testing.T) {
		otp := mailer.sentOTPs[0]
		body, _ := json.Marshal(dto.VerifyResetOTPRequest{
			Email: "alice@tenant.com",
			OTP:   otp,
		})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/verify-reset-otp", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp["success"].(bool))

		data := resp["data"].(map[string]any)
		validResetToken = data["reset_token"].(string)
		assert.NotEmpty(t, validResetToken)

		// OTP must now be deleted
		assert.Nil(t, resetRepo.otps["alice@tenant.com"])
	})

	// 4. OTP verification with invalid OTP
	t.Run("4. OTP verification with invalid OTP returns 400 Bad Request", func(t *testing.T) {
		// Re-trigger OTP
		resetRepo.cooldowns["alice@tenant.com"] = time.Time{}
		resetRepo.rateLimits["password_reset:rate_limit:email:alice@tenant.com"] = 0

		body, _ := json.Marshal(dto.ForgotPasswordRequest{Email: "alice@tenant.com"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		// Submit wrong OTP
		verifyBody, _ := json.Marshal(dto.VerifyResetOTPRequest{
			Email: "alice@tenant.com",
			OTP:   "000000",
		})
		verifyReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/verify-reset-otp", bytes.NewReader(verifyBody))
		verifyReq.Header.Set("Content-Type", "application/json")

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, verifyReq)

		assert.Equal(t, http.StatusBadRequest, w2.Code)
		assert.Contains(t, w2.Body.String(), "invalid OTP")
	})

	// 5. OTP verification after expiration
	t.Run("5. OTP verification after expiration returns 400 Bad Request", func(t *testing.T) {
		// Set OTP expiry in the past
		if otpData, ok := resetRepo.otps["alice@tenant.com"]; ok {
			otpData.ExpiresAt = time.Now().Add(-1 * time.Minute)
		}

		verifyBody, _ := json.Marshal(dto.VerifyResetOTPRequest{
			Email: "alice@tenant.com",
			OTP:   "123456",
		})
		verifyReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/verify-reset-otp", bytes.NewReader(verifyBody))
		verifyReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, verifyReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid or expired OTP")
	})

	// 6. OTP verification after maximum attempts
	t.Run("6. OTP verification after maximum attempts invalidates OTP", func(t *testing.T) {
		// Fresh OTP
		resetRepo.cooldowns["alice@tenant.com"] = time.Time{}
		resetRepo.rateLimits["password_reset:rate_limit:email:alice@tenant.com"] = 0

		body, _ := json.Marshal(dto.ForgotPasswordRequest{Email: "alice@tenant.com"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		// 5 invalid attempts
		for i := 0; i < 5; i++ {
			vBody, _ := json.Marshal(dto.VerifyResetOTPRequest{
				Email: "alice@tenant.com",
				OTP:   "999999",
			})
			vReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/verify-reset-otp", bytes.NewReader(vBody))
			vReq.Header.Set("Content-Type", "application/json")
			wTrial := httptest.NewRecorder()
			router.ServeHTTP(wTrial, vReq)
		}

		// Subsequent attempt should fail because OTP was deleted
		vBody, _ := json.Marshal(dto.VerifyResetOTPRequest{
			Email: "alice@tenant.com",
			OTP:   "999999",
		})
		vReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/verify-reset-otp", bytes.NewReader(vBody))
		vReq.Header.Set("Content-Type", "application/json")
		wFinal := httptest.NewRecorder()
		router.ServeHTTP(wFinal, vReq)

		assert.Equal(t, http.StatusBadRequest, wFinal.Code)
		assert.Nil(t, resetRepo.otps["alice@tenant.com"])
	})

	// 7. Password reset with valid reset token
	t.Run("7. Password reset with valid reset token updates password hash", func(t *testing.T) {
		// Log in first to have an active token pair
		loginBody, _ := json.Marshal(dto.TenantLoginRequest{
			Email:    "alice@tenant.com",
			Password: initialPassword,
		})
		loginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/tenant/login", bytes.NewReader(loginBody))
		loginReq.Header.Set("Content-Type", "application/json")
		wLogin := httptest.NewRecorder()
		router.ServeHTTP(wLogin, loginReq)
		require.Equal(t, http.StatusOK, wLogin.Code)

		var loginData map[string]any
		_ = json.Unmarshal(wLogin.Body.Bytes(), &loginData)
		tokens := loginData["data"].(map[string]any)
		oldAccessToken := tokens["access_token"].(string)
		oldRefreshToken := tokens["refresh_token"].(string)

		// Ensure old token works before reset
		meReq, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		meReq.Header.Set("Authorization", "Bearer "+oldAccessToken)
		wMe := httptest.NewRecorder()
		router.ServeHTTP(wMe, meReq)
		assert.Equal(t, http.StatusOK, wMe.Code)

		// Reset password with valid token from test #3
		newPassword := "BrandNewPass456!"
		resetBody, _ := json.Marshal(dto.ResetPasswordRequest{
			ResetToken:  validResetToken,
			NewPassword: newPassword,
		})
		resetReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(resetBody))
		resetReq.Header.Set("Content-Type", "application/json")

		wReset := httptest.NewRecorder()
		router.ServeHTTP(wReset, resetReq)

		assert.Equal(t, http.StatusOK, wReset.Code)
		var resetResp map[string]any
		_ = json.Unmarshal(wReset.Body.Bytes(), &resetResp)
		assert.True(t, resetResp["success"].(bool))
		assert.Equal(t, "Password reset successfully.", resetResp["message"])

		// Verify new password works for login
		newLoginBody, _ := json.Marshal(dto.TenantLoginRequest{
			Email:    "alice@tenant.com",
			Password: newPassword,
		})
		newLoginReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/tenant/login", bytes.NewReader(newLoginBody))
		newLoginReq.Header.Set("Content-Type", "application/json")
		wNewLogin := httptest.NewRecorder()
		router.ServeHTTP(wNewLogin, newLoginReq)
		assert.Equal(t, http.StatusOK, wNewLogin.Code)

		// 10. Old tokens invalidation
		// Old access token must now be rejected by Authenticate middleware
		meReqAfter, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
		meReqAfter.Header.Set("Authorization", "Bearer "+oldAccessToken)
		wMeAfter := httptest.NewRecorder()
		router.ServeHTTP(wMeAfter, meReqAfter)
		assert.Equal(t, http.StatusUnauthorized, wMeAfter.Code)

		// Old refresh token must be rejected
		refBody, _ := json.Marshal(dto.RefreshTokenRequest{RefreshToken: oldRefreshToken})
		refReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refBody))
		refReq.Header.Set("Content-Type", "application/json")
		wRef := httptest.NewRecorder()
		router.ServeHTTP(wRef, refReq)
		assert.Equal(t, http.StatusUnauthorized, wRef.Code)
	})

	// 8. Password reset with expired reset token
	t.Run("8. Password reset with expired reset token returns 400", func(t *testing.T) {
		expiredToken, _, err := jwtManager.GeneratePasswordResetToken(userID, &tenantID, "alice@tenant.com", auth.UserTypeTenantUser, -5*time.Minute)
		require.NoError(t, err)

		resetBody, _ := json.Marshal(dto.ResetPasswordRequest{
			ResetToken:  expiredToken,
			NewPassword: "ValidNewPassword123!",
		})
		resetReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(resetBody))
		resetReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, resetReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 9. Password reset with reused reset token
	t.Run("9. Password reset with reused reset token returns 400", func(t *testing.T) {
		// validResetToken was already used in test #7
		resetBody, _ := json.Marshal(dto.ResetPasswordRequest{
			ResetToken:  validResetToken,
			NewPassword: "AttemptReusingToken123!",
		})
		resetReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(resetBody))
		resetReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, resetReq)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "already been used")
	})

	// 11. Tenant isolation
	t.Run("11. Tenant isolation prevents cross-tenant password updates", func(t *testing.T) {
		otherTenantID := uuid.New()
		otherUserID := uuid.New()
		otherUser := &domain.TenantUser{
			ID:           otherUserID,
			TenantID:     otherTenantID,
			Name:         "Bob Other",
			Role:         auth.RoleTenantAdmin,
			Email:        "bob@othertenant.com",
			Status:       "active",
			PasswordHash: initialHash,
		}
		tenantRepo.users[otherUserID] = otherUser

		// Generate token purporting to be for otherTenantID but crafted with mismatched context
		token, tokenID, err := jwtManager.GeneratePasswordResetToken(otherUserID, &tenantID, "bob@othertenant.com", auth.UserTypeTenantUser, 10*time.Minute)
		require.NoError(t, err)

		resetRepo.tokens[tokenID] = &domain.PasswordResetTokenData{
			TokenID:   tokenID,
			UserID:    otherUserID,
			TenantID:  &tenantID,
			UserType:  auth.UserTypeTenantUser,
			Email:     "bob@othertenant.com",
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		resetBody, _ := json.Marshal(dto.ResetPasswordRequest{
			ResetToken:  token,
			NewPassword: "CrossTenantAttack123!",
		})
		resetReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", bytes.NewReader(resetBody))
		resetReq.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, resetReq)

		// Must fail because user does not belong to tenantID
		assert.NotEqual(t, http.StatusOK, w.Code)

		// Bob's password in otherTenant should remain unchanged
		err = hasher.Compare(tenantRepo.users[otherUserID].PasswordHash, initialPassword)
		assert.NoError(t, err)
	})

	// 12. Rate-limit behavior
	t.Run("12. Rate limit behavior returns 429 Too Many Requests when threshold exceeded", func(t *testing.T) {
		resetRepo.rateLimits["password_reset:rate_limit:email:alice@tenant.com"] = 3 // Max 3
		resetRepo.cooldowns["alice@tenant.com"] = time.Time{}

		body, _ := json.Marshal(dto.ForgotPasswordRequest{Email: "alice@tenant.com"})
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Contains(t, w.Body.String(), "TOO_MANY_REQUESTS")
	})
}
