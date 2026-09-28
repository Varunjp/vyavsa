package service

import (
	"context"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/metrics"
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

func (m *mockTenantRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	t, ok := m.tenants[id]
	if !ok {
		return appErrors.NewNotFound("tenant not found")
	}
	t.Status = status
	t.UpdatedAt = time.Now().UTC()
	return nil
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

func TestTenantService_Onboarding(t *testing.T) {
	ctx := context.Background()
	log := logger.Default().Logger
	appMetrics := metrics.New()
	hasher := auth.NewBcryptHasher()

	t.Run("OnboardTenant successfully executes all steps atomically", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
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
			transactor,
			hasher,
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

	t.Run("OnboardTenant fails when plan does not exist", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		transactor := &mockTransactor{}

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, transactor, hasher, appMetrics, log)

		req := dto.OnboardTenantRequest{
			Name:          "Test Store",
			Email:         "test@store.com",
			AdminName:     "Admin",
			AdminEmail:    "admin@store.com",
			AdminPassword: "password123",
			PlanID:        uuid.New(), // Random non-existent plan
		}

		_, err := svc.OnboardTenant(ctx, req)
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.CodeValidation, appErr.Code)
	})

	t.Run("OnboardTenant fails when tenant email already exists", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		transactor := &mockTransactor{}

		plan := &domain.PlatformPlan{
			PlanName: "Free",
			Price:    decimal.Zero,
			Status:   "active",
		}
		require.NoError(t, planRepo.Create(ctx, plan))

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, transactor, hasher, appMetrics, log)

		req := dto.OnboardTenantRequest{
			Name:          "First Store",
			Email:         "common@store.com",
			AdminName:     "Admin 1",
			AdminEmail:    "admin1@store.com",
			AdminPassword: "password123",
			PlanID:        plan.ID,
		}
		_, err := svc.OnboardTenant(ctx, req)
		require.NoError(t, err)

		// Second store with same email
		req2 := dto.OnboardTenantRequest{
			Name:          "Second Store",
			Email:         "common@store.com",
			AdminName:     "Admin 2",
			AdminEmail:    "admin2@store.com",
			AdminPassword: "password123",
			PlanID:        plan.ID,
		}
		_, err = svc.OnboardTenant(ctx, req2)
		require.Error(t, err)
		var appErr *appErrors.AppError
		require.ErrorAs(t, err, &appErr)
		assert.Equal(t, appErrors.CodeConflict, appErr.Code)
	})

	t.Run("UpdateTenantStatus modifies status", func(t *testing.T) {
		tenantRepo := newMockTenantRepo()
		userRepo := newMockTenantUserRepoFull()
		summaryRepo := newMockFinancialSummaryRepo()
		subRepo := newMockSubscriptionRepo()
		planRepo := newMockPlanRepo()
		transactor := &mockTransactor{}

		plan := &domain.PlatformPlan{PlanName: "Free", Price: decimal.Zero, Status: "active"}
		require.NoError(t, planRepo.Create(ctx, plan))

		svc := NewTenantService(tenantRepo, userRepo, summaryRepo, subRepo, planRepo, transactor, hasher, appMetrics, log)

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
}
