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
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Transactor
type mockTransactor struct{}

func (m *mockTransactor) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// Mock TenantRepo
type mockTenantRepo struct {
	tenants map[uuid.UUID]*domain.Tenant
}

func newMockTenantRepo() *mockTenantRepo {
	return &mockTenantRepo{tenants: make(map[uuid.UUID]*domain.Tenant)}
}

func (m *mockTenantRepo) Create(ctx context.Context, tenant *domain.Tenant) error {
	tenant.ID = uuid.New()
	tenant.CreatedAt = time.Now().UTC()
	tenant.UpdatedAt = time.Now().UTC()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t, ok := m.tenants[id]
	if !ok {
		return nil, appErrors.NewNotFound("tenant not found")
	}
	return t, nil
}

func (m *mockTenantRepo) GetByEmail(ctx context.Context, email string) (*domain.Tenant, error) {
	for _, t := range m.tenants {
		if t.Email == email {
			return t, nil
		}
	}
	return nil, appErrors.NewNotFound("tenant not found")
}

func (m *mockTenantRepo) List(ctx context.Context, page, pageSize int, status, search string) ([]domain.Tenant, int64, error) {
	var result []domain.Tenant
	for _, t := range m.tenants {
		result = append(result, *t)
	}
	return result, int64(len(result)), nil
}

func (m *mockTenantRepo) Update(ctx context.Context, tenant *domain.Tenant) error {
	_, ok := m.tenants[tenant.ID]
	if !ok {
		return appErrors.NewNotFound("tenant not found")
	}
	tenant.UpdatedAt = time.Now().UTC()
	m.tenants[tenant.ID] = tenant
	return nil
}

func (m *mockTenantRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	t, ok := m.tenants[id]
	if !ok {
		return appErrors.NewNotFound("tenant not found")
	}
	t.Status = status
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockTenantRepo) CountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	for _, t := range m.tenants {
		if status == "" || t.Status == status {
			count++
		}
	}
	return count, nil
}

func (m *mockTenantRepo) CountSince(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	for _, t := range m.tenants {
		if t.CreatedAt.After(since) || t.CreatedAt.Equal(since) {
			count++
		}
	}
	return count, nil
}

// Mock FinancialSummaryRepo
type mockFinancialSummaryRepo struct {
	summaries map[uuid.UUID]*domain.TenantFinancialSummary
}

func newMockFinancialSummaryRepo() *mockFinancialSummaryRepo {
	return &mockFinancialSummaryRepo{summaries: make(map[uuid.UUID]*domain.TenantFinancialSummary)}
}

func (m *mockFinancialSummaryRepo) Create(ctx context.Context, s *domain.TenantFinancialSummary) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now().UTC()
	s.UpdatedAt = time.Now().UTC()
	m.summaries[s.TenantID] = s
	return nil
}

func (m *mockFinancialSummaryRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	s, ok := m.summaries[tenantID]
	if !ok {
		return nil, appErrors.NewNotFound("summary not found")
	}
	return s, nil
}

func (m *mockFinancialSummaryRepo) GetByTenantIDForUpdate(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	return m.GetByTenantID(ctx, tenantID)
}

func (m *mockFinancialSummaryRepo) SyncFromSourceRecords(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	return m.GetByTenantID(ctx, tenantID)
}

func (m *mockFinancialSummaryRepo) Update(ctx context.Context, s *domain.TenantFinancialSummary) error {
	s.UpdatedAt = time.Now().UTC()
	m.summaries[s.TenantID] = s
	return nil
}

// Mock SubscriptionRepo
type mockSubscriptionRepo struct {
	subs map[uuid.UUID]*domain.PlatformSubscription
}

func newMockSubscriptionRepo() *mockSubscriptionRepo {
	return &mockSubscriptionRepo{subs: make(map[uuid.UUID]*domain.PlatformSubscription)}
}

func (m *mockSubscriptionRepo) Create(ctx context.Context, sub *domain.PlatformSubscription) error {
	sub.ID = uuid.New()
	sub.CreatedAt = time.Now().UTC()
	sub.UpdatedAt = time.Now().UTC()
	m.subs[sub.TenantID] = sub
	return nil
}

func (m *mockSubscriptionRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.PlatformSubscription, error) {
	sub, ok := m.subs[tenantID]
	if !ok {
		return nil, appErrors.NewNotFound("subscription not found")
	}
	return sub, nil
}

func (m *mockSubscriptionRepo) Update(ctx context.Context, sub *domain.PlatformSubscription) error {
	sub.UpdatedAt = time.Now().UTC()
	m.subs[sub.TenantID] = sub
	return nil
}

func (m *mockSubscriptionRepo) List(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformSubscriptionWithTenant, int64, error) {
	var items []domain.PlatformSubscriptionWithTenant
	for _, sub := range m.subs {
		if status == "" || sub.Status == status {
			items = append(items, domain.PlatformSubscriptionWithTenant{
				ID:              sub.ID,
				TenantID:        sub.TenantID,
				TenantName:      "Mock Tenant",
				TenantEmail:     "mock@vyavsa.test",
				CurrentPlanID:   sub.CurrentPlanID,
				CurrentPlanName: sub.CurrentPlanName,
				Status:          sub.Status,
				StartDate:       sub.StartDate,
				EndDate:         sub.EndDate,
				CreatedAt:       sub.CreatedAt,
				UpdatedAt:       sub.UpdatedAt,
			})
		}
	}
	return items, int64(len(items)), nil
}

// Mock PlanTxnRepo
type mockPlanTxnRepo struct {
	txns map[uuid.UUID]*domain.PlatformPlanTransaction
}

func newMockPlanTxnRepo() *mockPlanTxnRepo {
	return &mockPlanTxnRepo{txns: make(map[uuid.UUID]*domain.PlatformPlanTransaction)}
}

func (m *mockPlanTxnRepo) Create(ctx context.Context, tx *domain.PlatformPlanTransaction) error {
	tx.ID = uuid.New()
	tx.CreatedAt = time.Now().UTC()
	tx.UpdatedAt = time.Now().UTC()
	m.txns[tx.ID] = tx
	return nil
}

func (m *mockPlanTxnRepo) ListByTenantID(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.PlatformPlanTransaction, int64, error) {
	var res []domain.PlatformPlanTransaction
	for _, tx := range m.txns {
		if tx.TenantID == tenantID {
			res = append(res, *tx)
		}
	}
	return res, int64(len(res)), nil
}

func (m *mockPlanTxnRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformPlanTransaction, error) {
	tx, ok := m.txns[id]
	if !ok || tx.TenantID != tenantID {
		return nil, appErrors.NewNotFound("transaction not found")
	}
	return tx, nil
}

func (m *mockPlanTxnRepo) GetByTransactionID(ctx context.Context, tenantID uuid.UUID, transactionID string) (*domain.PlatformPlanTransaction, error) {
	for _, tx := range m.txns {
		if tx.TenantID == tenantID && tx.TransactionID == transactionID {
			return tx, nil
		}
	}
	return nil, appErrors.NewNotFound("transaction not found")
}

func (m *mockPlanTxnRepo) ListAll(ctx context.Context, page, pageSize int, status string) ([]domain.PlatformPlanTransactionWithTenant, int64, error) {
	var items []domain.PlatformPlanTransactionWithTenant
	for _, tx := range m.txns {
		if status == "" || tx.Status == status {
			items = append(items, domain.PlatformPlanTransactionWithTenant{
				ID:            tx.ID,
				TenantID:      tx.TenantID,
				TenantName:    "Mock Tenant",
				TenantEmail:   "mock@vyavsa.test",
				TransactionID: tx.TransactionID,
				PaymentMethod: tx.PaymentMethod,
				PlanID:        tx.PlanID,
				PlanName:      tx.PlanName,
				Amount:        tx.Amount,
				Status:        tx.Status,
				FailureReason: tx.FailureReason,
				CreatedAt:     tx.CreatedAt,
				UpdatedAt:     tx.UpdatedAt,
			})
		}
	}
	return items, int64(len(items)), nil
}

func (m *mockPlanTxnRepo) GetMonthlyReceivedIncome(ctx context.Context, since time.Time) (decimal.Decimal, error) {
	sum := decimal.Zero
	for _, tx := range m.txns {
		if tx.Status == "completed" && (tx.CreatedAt.After(since) || tx.CreatedAt.Equal(since)) {
			sum = sum.Add(tx.Amount)
		}
	}
	return sum, nil
}

func (m *mockPlanTxnRepo) GetMonthlyRevenueTrend(ctx context.Context, since time.Time) ([]repository.MonthlyRevenueAggregate, error) {
	return []repository.MonthlyRevenueAggregate{}, nil
}

// Mock TenantUserRepo
type mockTenantUserRepoFull struct {
	users map[uuid.UUID]*domain.TenantUser
}

func newMockTenantUserRepoFull() *mockTenantUserRepoFull {
	return &mockTenantUserRepoFull{users: make(map[uuid.UUID]*domain.TenantUser)}
}

func (m *mockTenantUserRepoFull) Create(ctx context.Context, u *domain.TenantUser) error {
	u.ID = uuid.New()
	u.CreatedAt = time.Now().UTC()
	u.UpdatedAt = time.Now().UTC()
	m.users[u.ID] = u
	return nil
}

func (m *mockTenantUserRepoFull) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantUser, error) {
	u, ok := m.users[id]
	if !ok || u.TenantID != tenantID {
		return nil, appErrors.NewNotFound("user not found")
	}
	return u, nil
}

func (m *mockTenantUserRepoFull) GetByEmail(ctx context.Context, email string) (*domain.TenantUser, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, appErrors.NewNotFound("user not found")
}

func (m *mockTenantUserRepoFull) GetByTenantAndEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.TenantUser, error) {
	for _, u := range m.users {
		if u.TenantID == tenantID && u.Email == email {
			return u, nil
		}
	}
	return nil, appErrors.NewNotFound("user not found")
}

func (m *mockTenantUserRepoFull) GetAdminByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantUser, error) {
	for _, u := range m.users {
		if u.TenantID == tenantID && u.Role == "admin" {
			return u, nil
		}
	}
	return nil, appErrors.NewNotFound("admin not found")
}

func (m *mockTenantUserRepoFull) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status string) error {
	u, ok := m.users[id]
	if !ok || u.TenantID != tenantID {
		return appErrors.NewNotFound("user not found")
	}
	u.Status = status
	return nil
}

func (m *mockTenantUserRepoFull) UpdatePassword(ctx context.Context, tenantID, id uuid.UUID, passwordHash string) error {
	u, ok := m.users[id]
	if !ok || u.TenantID != tenantID {
		return appErrors.NewNotFound("user not found")
	}
	u.PasswordHash = passwordHash
	u.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockTenantUserRepoFull) Update(ctx context.Context, u *domain.TenantUser) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockTenantUserRepoFull) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.users, id)
	return nil
}

func (m *mockTenantUserRepoFull) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, role, status string) ([]domain.TenantUser, int64, error) {
	var list []domain.TenantUser
	for _, u := range m.users {
		if u.TenantID == tenantID {
			list = append(list, *u)
		}
	}
	return list, int64(len(list)), nil
}

func TestTenantService_RegistrationAndOnboarding(t *testing.T) {
	ctx := context.Background()
	log := logger.Default().Logger
	appMetrics := metrics.New()
	hasher := auth.NewBcryptHasher()
	jwtManager := auth.NewJWTManager(config.JWTConfig{
		Secret:        "test-secret-at-least-32-bytes-long",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 24 * time.Hour,
	})

	t.Run("RegisterTenant registers new tenant with same email for admin user", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		transactor := &mockTransactor{}

		// Seed a Free Starter plan
		freePlan := &domain.PlatformPlan{
			PlanName: "Free Starter",
			Price:    decimal.Zero,
			Status:   "active",
		}
		require.NoError(t, planRepo.Create(ctx, freePlan))

		txnRepo := newMockPlanTxnRepo()

		svc := NewTenantService(
			tenantRepo,
			userRepo,
			summaryRepo,
			subRepo,
			planRepo,
			txnRepo,
			transactor,
			hasher,
			jwtManager,
			appMetrics,
			log,
		)

		req := dto.TenantRegisterRequest{
			Name:      "Balaji Kirana Store",
			Email:     "balaji@kirana.com",
			Password:  "Kirana@12345",
			Phone:     "+919876500000",
			AdminName: "Rohan Balaji",
		}

		resp, err := svc.RegisterTenant(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "Balaji Kirana Store", resp.Tenant.Name)
		assert.Equal(t, "balaji@kirana.com", resp.Tenant.Email)
		assert.Equal(t, "Rohan Balaji", resp.AdminUser.Name)
		assert.Equal(t, "balaji@kirana.com", resp.AdminUser.Email) // Same email verified!
		assert.Equal(t, "admin", resp.AdminUser.Role)
		assert.Equal(t, "Free Starter", resp.Subscription.CurrentPlanName)
		assert.True(t, resp.FinancialSummary.CashBalance.IsZero())
		assert.True(t, resp.FinancialSummary.BankBalance.IsZero())
		require.NotNil(t, resp.Tokens)
		assert.NotEmpty(t, resp.Tokens.AccessToken)
		assert.NotEmpty(t, resp.Tokens.RefreshToken)
		assert.Equal(t, "balaji@kirana.com", resp.Tokens.User.Email)
		assert.Equal(t, resp.Tenant.ID, *resp.Tokens.User.TenantID)
	})

	t.Run("RegisterTenant defaults admin_name to business name when omitted", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		txnRepo := newMockPlanTxnRepo()
		transactor := &mockTransactor{}

		freePlan := &domain.PlatformPlan{PlanName: "Free Starter", Price: decimal.Zero, Status: "active"}
		require.NoError(t, planRepo.Create(ctx, freePlan))

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, txnRepo, transactor, hasher, jwtManager, appMetrics, log)

		req := dto.TenantRegisterRequest{
			Name:     "Quick Mart",
			Email:    "info@quickmart.com",
			Password: "Password@123",
		}

		resp, err := svc.RegisterTenant(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "Quick Mart", resp.Tenant.Name)
		assert.Equal(t, "Quick Mart", resp.AdminUser.Name)
		assert.Equal(t, "info@quickmart.com", resp.AdminUser.Email)
	})

	t.Run("OnboardTenant successfully executes all steps atomically", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		txnRepo := newMockPlanTxnRepo()
		transactor := &mockTransactor{}

		// Seed a plan
		plan := &domain.PlatformPlan{
			PlanName: "Pro Tier",
			Price:    decimal.NewFromFloat(499),
			Status:   "active",
		}
		require.NoError(t, planRepo.Create(ctx, plan))

		svc := NewTenantService(
			tenantRepo,
			userRepo,
			summaryRepo,
			subRepo,
			planRepo,
			txnRepo,
			transactor,
			hasher,
			jwtManager,
			appMetrics,
			log,
		)

		req := dto.OnboardTenantRequest{
			Name:          "Apex Grocery",
			Email:         "contact@apexgrocery.com",
			Phone:         "+919876543200",
			AdminName:     "Rahul Sharma",
			AdminEmail:    "rahul@apexgrocery.com",
			AdminPassword: "SecurePassword@123",
			PlanID:        plan.ID,
		}

		resp, err := svc.OnboardTenant(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "Apex Grocery", resp.Tenant.Name)
		assert.Equal(t, "contact@apexgrocery.com", resp.Tenant.Email)
		assert.Equal(t, "Rahul Sharma", resp.AdminUser.Name)
		assert.Equal(t, "rahul@apexgrocery.com", resp.AdminUser.Email)
		assert.Equal(t, "admin", resp.AdminUser.Role)
		assert.Equal(t, "Pro Tier", resp.Subscription.CurrentPlanName)
		assert.True(t, resp.FinancialSummary.CashBalance.IsZero())
		assert.True(t, resp.FinancialSummary.BankBalance.IsZero())

		// Verify GetTenantByID
		detail, err := svc.GetTenantByID(ctx, resp.Tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, resp.Tenant.ID, detail.Tenant.ID)
		require.NotNil(t, detail.AdminUser)
		assert.Equal(t, "rahul@apexgrocery.com", detail.AdminUser.Email)
		require.NotNil(t, detail.Subscription)
		assert.Equal(t, "Pro Tier", detail.Subscription.CurrentPlanName)
		require.NotNil(t, detail.FinancialSummary)
		assert.True(t, detail.FinancialSummary.CashBalance.IsZero())
	})

	t.Run("OnboardTenant defaults admin email to tenant email when admin_email is omitted", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		txnRepo := newMockPlanTxnRepo()
		transactor := &mockTransactor{}

		plan := &domain.PlatformPlan{PlanName: "Pro Tier", Price: decimal.NewFromFloat(499), Status: "active"}
		require.NoError(t, planRepo.Create(ctx, plan))

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, txnRepo, transactor, hasher, jwtManager, appMetrics, log)

		req := dto.OnboardTenantRequest{
			Name:          "Single Email Mart",
			Email:         "single@mart.com",
			AdminPassword: "Password@123",
			PlanID:        plan.ID,
		}

		resp, err := svc.OnboardTenant(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "single@mart.com", resp.Tenant.Email)
		assert.Equal(t, "single@mart.com", resp.AdminUser.Email)
		assert.Equal(t, "Single Email Mart", resp.AdminUser.Name)
	})

	t.Run("OnboardTenant fails when plan does not exist", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		txnRepo := newMockPlanTxnRepo()
		transactor := &mockTransactor{}

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, txnRepo, transactor, hasher, jwtManager, appMetrics, log)

		req := dto.OnboardTenantRequest{
			Name:          "Test Store",
			Email:         "test@store.com",
			AdminName:     "Admin",
			AdminEmail:    "admin@store.com",
			AdminPassword: "password123",
			PlanID:        uuid.New(),
		}

		_, err := svc.OnboardTenant(ctx, req)
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.CodeValidation, appErr.Code)
	})

	t.Run("UpdateTenantStatus modifies status", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		txnRepo := newMockPlanTxnRepo()
		transactor := &mockTransactor{}

		plan := &domain.PlatformPlan{PlanName: "Free", Price: decimal.Zero, Status: "active"}
		require.NoError(t, planRepo.Create(ctx, plan))

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, txnRepo, transactor, hasher, jwtManager, appMetrics, log)

		resp, err := svc.OnboardTenant(ctx, dto.OnboardTenantRequest{
			Name:          "Status Store",
			Email:         "status@store.com",
			AdminName:     "Admin",
			AdminEmail:    "admin@statusstore.com",
			AdminPassword: "password123",
			PlanID:        plan.ID,
		})
		require.NoError(t, err)

		err = svc.UpdateTenantStatus(ctx, resp.Tenant.ID, "suspended")
		require.NoError(t, err)

		updated, err := svc.GetTenantByID(ctx, resp.Tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, "suspended", updated.Tenant.Status)
	})

	t.Run("UpdateTenantSettings updates tenant name, email, phone with validation and conflict checks", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		txnRepo := newMockPlanTxnRepo()
		transactor := &mockTransactor{}

		freePlan := &domain.PlatformPlan{PlanName: "Free", Price: decimal.Zero, Status: "active"}
		require.NoError(t, planRepo.Create(ctx, freePlan))

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, txnRepo, transactor, hasher, jwtManager, appMetrics, log)

		resp1, err := svc.RegisterTenant(ctx, dto.TenantRegisterRequest{
			Name:      "First Store",
			Email:     "first@store.com",
			Password:  "Secret@123",
			Phone:     "+919876543210",
			AdminName: "First Admin",
		})
		require.NoError(t, err)

		resp2, err := svc.RegisterTenant(ctx, dto.TenantRegisterRequest{
			Name:      "Second Store",
			Email:     "second@store.com",
			Password:  "Secret@123",
			Phone:     "+919876543211",
			AdminName: "Second Admin",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, resp2.Tenant.ID)

		// 1. Success update
		newName := "First Store Updated"
		newPhone := "+919999988888"
		updated, err := svc.UpdateTenantSettings(ctx, resp1.Tenant.ID, dto.UpdateTenantSettingsRequest{
			Name:  newName,
			Email: "first@store.com",
			Phone: newPhone,
		})
		require.NoError(t, err)
		assert.Equal(t, newName, updated.Name)
		assert.Equal(t, newPhone, updated.Phone)

		// Verify in repo
		fetched, err := svc.GetTenantByID(ctx, resp1.Tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, newName, fetched.Tenant.Name)
		assert.Equal(t, newPhone, fetched.Tenant.Phone)

		// 2. Conflict update: attempting to update email to second store's email
		_, err = svc.UpdateTenantSettings(ctx, resp1.Tenant.ID, dto.UpdateTenantSettingsRequest{
			Name:  newName,
			Email: "second@store.com",
		})
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.CodeConflict, appErr.Code)

		// 3. Validation error: empty name
		_, err = svc.UpdateTenantSettings(ctx, resp1.Tenant.ID, dto.UpdateTenantSettingsRequest{
			Name:  "",
			Email: "valid@store.com",
		})
		require.Error(t, err)
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.CodeValidation, appErr.Code)
	})

	t.Run("Subscription lifecycle: Active, Expired, ListPlans, and PurchasePlan", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		txnRepo := newMockPlanTxnRepo()
		transactor := &mockTransactor{}

		// Create plans
		freePlan := &domain.PlatformPlan{PlanName: "Free Tier", Price: decimal.Zero, Status: "active"}
		require.NoError(t, planRepo.Create(ctx, freePlan))

		silverPlan := &domain.PlatformPlan{PlanName: "Silver Tier", Price: decimal.NewFromFloat(299), Status: "active"}
		require.NoError(t, planRepo.Create(ctx, silverPlan))

		goldPlan := &domain.PlatformPlan{PlanName: "Gold Tier", Price: decimal.NewFromFloat(999), Status: "active"}
		require.NoError(t, planRepo.Create(ctx, goldPlan))

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, txnRepo, transactor, hasher, jwtManager, appMetrics, log)

		// Register tenant
		regResp, err := svc.RegisterTenant(ctx, dto.TenantRegisterRequest{
			Name:      "Sub Tester",
			Email:     "sub@tester.com",
			Password:  "Pass@12345",
			Phone:     "+919876543201",
			AdminName: "Sub Admin",
		})
		require.NoError(t, err)
		tenantID := regResp.Tenant.ID

		// 1. Check current subscription
		subResp, err := svc.GetTenantSubscription(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, "Free Tier", subResp.CurrentPlanName)
		assert.False(t, subResp.IsExpired)
		assert.True(t, subResp.Price.IsZero())

		// 2. List available plans
		plans, err := svc.ListAvailablePlans(ctx)
		require.NoError(t, err)
		assert.Len(t, plans, 3)

		// 3. Purchase / Upgrade to Silver Tier
		purchasedSub, txn, err := svc.PurchasePlan(ctx, tenantID, dto.PurchasePlanRequest{
			PlanID:        silverPlan.ID,
			PaymentMethod: "upi",
		})
		require.NoError(t, err)
		require.NotNil(t, purchasedSub)
		require.NotNil(t, txn)
		assert.Equal(t, "Silver Tier", purchasedSub.CurrentPlanName)
		assert.Equal(t, "active", purchasedSub.Status)
		assert.False(t, purchasedSub.IsExpired)
		assert.True(t, purchasedSub.DaysRemaining >= 29)
		assert.Equal(t, "completed", txn.Status)
		assert.Equal(t, "upi", txn.PaymentMethod)
		assert.Equal(t, "Silver Tier", txn.PlanName)
		assert.True(t, txn.Amount.Equal(decimal.NewFromFloat(299)))

		// 4. Verify transaction list
		txns, count, err := svc.ListTenantTransactions(ctx, tenantID, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
		assert.Len(t, txns, 1)
		assert.Equal(t, txn.TransactionID, txns[0].TransactionID)

		// 5. Test expired subscription handling
		// Artificially expire the subscription
		subEntity, err := subRepo.GetByTenantID(ctx, tenantID)
		require.NoError(t, err)
		pastDate := time.Now().UTC().AddDate(0, 0, -5)
		subEntity.EndDate = &pastDate
		require.NoError(t, subRepo.Update(ctx, subEntity))

		// Fetch subscription - should gracefully report expired
		expiredResp, err := svc.GetTenantSubscription(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, "expired", expiredResp.Status)
		assert.True(t, expiredResp.IsExpired)
		assert.Equal(t, 0, expiredResp.DaysRemaining)

		// 6. Renew / Upgrade to Gold from expired state
		renewedSub, renewTxn, err := svc.PurchasePlan(ctx, tenantID, dto.PurchasePlanRequest{
			PlanID:        goldPlan.ID,
			PaymentMethod: "card",
		})
		require.NoError(t, err)
		require.NotNil(t, renewedSub)
		require.NotNil(t, renewTxn)
		assert.Equal(t, "Gold Tier", renewedSub.CurrentPlanName)
		assert.Equal(t, "active", renewedSub.Status)
		assert.False(t, renewedSub.IsExpired)
		assert.True(t, renewedSub.DaysRemaining >= 29)

		// 7. Verify 2 transactions in history
		txns, count, err = svc.ListTenantTransactions(ctx, tenantID, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int64(2), count)
		assert.Len(t, txns, 2)
	})

	t.Run("Platform Admin Dashboard, Subscriptions, Transactions, and Tenant Update", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		tenantUserRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		txnRepo := newMockPlanTxnRepo()
		transactor := &mockTransactor{}

		svc := NewTenantService(
			tenantRepo,
			tenantUserRepo,
			summaryRepo,
			subRepo,
			planRepo,
			txnRepo,
			transactor,
			hasher,
			jwtManager,
			appMetrics,
			log,
		)

		// Create a test plan
		plan := &domain.PlatformPlan{
			ID:       uuid.New(),
			PlanName: "Pro Tier",
			Price:    decimal.NewFromInt(1999),
			Status:   "active",
		}
		require.NoError(t, planRepo.Create(ctx, plan))

		// Onboard a tenant
		onboardResp, err := svc.OnboardTenant(ctx, dto.OnboardTenantRequest{
			Name:          "Platform Test Org",
			Email:         "platform-test@vyavsa.com",
			Phone:         "+91 9999999999",
			AdminPassword: "password123",
			PlanID:        plan.ID,
		})
		require.NoError(t, err)
		tenantID := onboardResp.Tenant.ID

		// Record a completed transaction
		tx := &domain.PlatformPlanTransaction{
			TenantID:      tenantID,
			TransactionID: "TXN-PLAT-001",
			PaymentMethod: "card",
			PlanID:        plan.ID,
			PlanName:      plan.PlanName,
			Amount:        decimal.NewFromInt(1999),
			Status:        "completed",
		}
		require.NoError(t, txnRepo.Create(ctx, tx))

		// 1. Test GetPlatformDashboardMetrics
		metricsResp, err := svc.GetPlatformDashboardMetrics(ctx)
		require.NoError(t, err)
		require.NotNil(t, metricsResp)
		assert.Equal(t, int64(1), metricsResp.ActiveTenants)
		assert.Equal(t, int64(1), metricsResp.TotalTenants)
		assert.Equal(t, int64(1), metricsResp.RecentRegistrations7d)
		assert.True(t, metricsResp.MonthlyReceivedIncome.Equal(decimal.NewFromInt(1999)))
		assert.Len(t, metricsResp.RevenueTrend, 6)

		// 2. Test ListPlatformSubscriptions
		subs, subCount, err := svc.ListPlatformSubscriptions(ctx, 1, 10, "")
		require.NoError(t, err)
		assert.Equal(t, int64(1), subCount)
		require.Len(t, subs, 1)
		assert.Equal(t, tenantID, subs[0].TenantID)

		// 3. Test ListPlatformTransactions
		txns, txCount, err := svc.ListPlatformTransactions(ctx, 1, 10, "")
		require.NoError(t, err)
		assert.Equal(t, int64(1), txCount)
		require.Len(t, txns, 1)
		assert.Equal(t, "TXN-PLAT-001", txns[0].TransactionID)
		assert.True(t, txns[0].Amount.Equal(decimal.NewFromInt(1999)))

		// 4. Test UpdateTenant
		updateResp, err := svc.UpdateTenant(ctx, tenantID, dto.UpdatePlatformTenantRequest{
			Name:   "Platform Test Org Updated",
			Email:  "updated-org@vyavsa.com",
			Phone:  "+91 8888888888",
			Status: "inactive",
		})
		require.NoError(t, err)
		require.NotNil(t, updateResp)
		assert.Equal(t, "Platform Test Org Updated", updateResp.Tenant.Name)
		assert.Equal(t, "updated-org@vyavsa.com", updateResp.Tenant.Email)
		assert.Equal(t, "+91 8888888888", updateResp.Tenant.Phone)
		assert.Equal(t, "inactive", updateResp.Tenant.Status)
	})
}
