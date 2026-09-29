package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TenantService defines business contracts for tenant lifecycle, self-registration, and onboarding
type TenantService interface {
	RegisterTenant(ctx context.Context, req dto.TenantRegisterRequest) (*dto.TenantRegisterResponse, error)
	OnboardTenant(ctx context.Context, req dto.OnboardTenantRequest) (*dto.OnboardTenantResponse, error)
	GetTenantByID(ctx context.Context, id uuid.UUID) (*dto.TenantDetailResponse, error)
	ListTenants(ctx context.Context, page, pageSize int, status, search string) ([]dto.TenantResponse, int64, error)
	UpdateTenantStatus(ctx context.Context, id uuid.UUID, status string) error
	ChangeSubscription(ctx context.Context, tenantID, planID uuid.UUID) (*dto.TenantSubscriptionResponse, error)

	// Tenant Member Operations
	GetTenantProfile(ctx context.Context, tenantID uuid.UUID) (*dto.TenantResponse, error)
	GetTenantFinancialSummary(ctx context.Context, tenantID uuid.UUID) (*dto.TenantFinancialSummaryResponse, error)
	GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*dto.TenantSubscriptionResponse, error)
}

type tenantService struct {
	tenantRepo       repository.TenantRepository
	tenantUserRepo   repository.TenantUserRepository
	summaryRepo      repository.TenantFinancialSummaryRepository
	subscriptionRepo repository.PlatformSubscriptionRepository
	planRepo         repository.PlatformPlanRepository
	transactor       repository.Transactor
	hasher           auth.PasswordHasher
	jwtManager       auth.JWTManager
	metrics          *metrics.Metrics
	log              *slog.Logger
}

// NewTenantService creates a new TenantService instance
func NewTenantService(
	tenantRepo repository.TenantRepository,
	tenantUserRepo repository.TenantUserRepository,
	summaryRepo repository.TenantFinancialSummaryRepository,
	subscriptionRepo repository.PlatformSubscriptionRepository,
	planRepo repository.PlatformPlanRepository,
	transactor repository.Transactor,
	hasher auth.PasswordHasher,
	jwtManager auth.JWTManager,
	m *metrics.Metrics,
	log *slog.Logger,
) TenantService {
	return &tenantService{
		tenantRepo:       tenantRepo,
		tenantUserRepo:   tenantUserRepo,
		summaryRepo:      summaryRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		transactor:       transactor,
		hasher:           hasher,
		jwtManager:       jwtManager,
		metrics:          m,
		log:              log,
	}
}

// RegisterTenant performs self-service registration for a new tenant organization with a default admin user using the same email
func (s *tenantService) RegisterTenant(ctx context.Context, req dto.TenantRegisterRequest) (*dto.TenantRegisterResponse, error) {
	adminName := req.AdminName
	if adminName == "" {
		adminName = req.Name
	}

	// 1. Resolve Subscription Plan
	var plan *domain.PlatformPlan
	var err error
	if req.PlanID != nil {
		plan, err = s.planRepo.GetByID(ctx, *req.PlanID)
		if err != nil {
			if appErrors.IsNotFound(err) {
				return nil, appErrors.NewValidation("invalid plan selection", map[string]string{"plan_id": "selected subscription plan does not exist"})
			}
			return nil, err
		}
	} else {
		// Default to free plan
		plan, err = s.planRepo.GetDefaultFreePlan(ctx)
		if err != nil {
			plan, err = s.planRepo.GetByName(ctx, "1 Month Free Trial")
			if err != nil {
				plan, err = s.planRepo.GetByName(ctx, "Free Starter")
				if err != nil {
					// Fallback to first available active plan
					plans, _, listErr := s.planRepo.List(ctx, 1, 1, "active")
					if listErr == nil && len(plans) > 0 {
						plan = &plans[0]
						err = nil
					} else {
						return nil, appErrors.NewInternal(fmt.Errorf("no default subscription plan available for registration"))
					}
				}
			}
		}
	}

	if plan.Status != "active" {
		return nil, appErrors.NewValidation("invalid plan selection", map[string]string{"plan_id": "selected subscription plan is not active"})
	}

	// 2. Check Email Uniqueness (Email is used for both Tenant Organization and Admin User)
	existingTenant, err := s.tenantRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingTenant != nil {
		return nil, appErrors.NewConflict(fmt.Sprintf("a business organization with email '%s' already exists", req.Email))
	} else if err != nil && !appErrors.IsNotFound(err) {
		return nil, err
	}

	existingUser, err := s.tenantUserRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, appErrors.NewConflict(fmt.Sprintf("a user with email '%s' already exists", req.Email))
	} else if err != nil && !appErrors.IsNotFound(err) {
		return nil, err
	}

	// 3. Hash Admin Password
	passwordHash, err := s.hasher.Hash(req.Password)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to hash admin password during registration", slog.String("error", err.Error()))
		return nil, appErrors.NewInternal(fmt.Errorf("failed to process admin credentials"))
	}

	// Prepare Entities
	tenant := &domain.Tenant{
		Name:   req.Name,
		Email:  req.Email,
		Phone:  req.Phone,
		Status: "active",
	}

	var adminUser *domain.TenantUser
	var summary *domain.TenantFinancialSummary
	var subscription *domain.PlatformSubscription

	endDate := time.Now().UTC().AddDate(0, 1, 0) // Default 30-day initial billing cycle

	// 4. Execute Atomic Multi-Step Transaction
	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 4a. Create Tenant
		if err := s.tenantRepo.Create(txCtx, tenant); err != nil {
			return fmt.Errorf("tenant insert failed: %w", err)
		}

		// 4b. Create Default Admin User with same email
		adminUser = &domain.TenantUser{
			TenantID:     tenant.ID,
			Name:         adminName,
			Role:         "admin",
			Email:        req.Email, // Same email as tenant organization
			PasswordHash: passwordHash,
			Status:       "active",
		}
		if err := s.tenantUserRepo.Create(txCtx, adminUser); err != nil {
			return fmt.Errorf("tenant admin user insert failed: %w", err)
		}

		// 4c. Create Initial Zeroed Financial Summary
		summary = &domain.TenantFinancialSummary{
			TenantID:        tenant.ID,
			CashBalance:     decimal.Zero,
			BankBalance:     decimal.Zero,
			TotalReceivable: decimal.Zero,
			TotalPayable:    decimal.Zero,
		}
		if err := s.summaryRepo.Create(txCtx, summary); err != nil {
			return fmt.Errorf("tenant financial summary insert failed: %w", err)
		}

		// 4d. Create Initial Subscription
		subscription = &domain.PlatformSubscription{
			TenantID:        tenant.ID,
			CurrentPlanID:   plan.ID,
			CurrentPlanName: plan.PlanName,
			Status:          "active",
			EndDate:         &endDate,
		}
		if err := s.subscriptionRepo.Create(txCtx, subscription); err != nil {
			return fmt.Errorf("tenant subscription insert failed: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.ErrorContext(ctx, "tenant self-registration transaction aborted", slog.String("error", err.Error()))
		return nil, err
	}

	// 5. Increment Metrics
	if s.metrics != nil {
		s.metrics.IncTenantsCreated()
	}

	s.log.InfoContext(ctx, "tenant self-registered successfully",
		slog.String("tenant_id", tenant.ID.String()),
		slog.String("tenant_name", tenant.Name),
		slog.String("admin_email", adminUser.Email),
		slog.String("plan_name", plan.PlanName),
	)

	// 6. Generate Initial Authentication Tokens if JWTManager is configured
	var tokenResp *dto.TokenResponse
	if s.jwtManager != nil {
		tokens, err := s.jwtManager.GenerateTokenPair(
			adminUser.ID,
			&tenant.ID,
			adminUser.Email,
			auth.RoleTenantAdmin,
			auth.UserTypeTenantUser,
		)
		if err == nil && tokens != nil {
			tokenResp = &dto.TokenResponse{
				AccessToken:  tokens.AccessToken,
				RefreshToken: tokens.RefreshToken,
				TokenType:    tokens.TokenType,
				ExpiresIn:    tokens.ExpiresIn,
				User: dto.UserProfile{
					ID:       adminUser.ID,
					Name:     adminUser.Name,
					Email:    adminUser.Email,
					Role:     adminUser.Role,
					UserType: auth.UserTypeTenantUser,
					TenantID: &tenant.ID,
				},
			}
		}
	}

	return &dto.TenantRegisterResponse{
		Tokens:           tokenResp,
		Tenant:           dto.ToTenantResponse(tenant),
		AdminUser:        dto.ToAdminUserResponse(adminUser),
		Subscription:     dto.ToSubscriptionResponse(subscription),
		FinancialSummary: dto.ToFinancialSummaryResponse(summary),
	}, nil
}

func (s *tenantService) OnboardTenant(ctx context.Context, req dto.OnboardTenantRequest) (*dto.OnboardTenantResponse, error) {
	// If admin email or name is omitted, default to tenant email and name
	adminEmail := req.AdminEmail
	if adminEmail == "" {
		adminEmail = req.Email
	}
	adminName := req.AdminName
	if adminName == "" {
		adminName = req.Name
	}

	// 1. Verify Plan Exists and is Active
	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		if appErrors.IsNotFound(err) {
			return nil, appErrors.NewValidation("invalid plan selection", map[string]string{"plan_id": "selected subscription plan does not exist"})
		}
		return nil, err
	}
	if plan.Status != "active" {
		return nil, appErrors.NewValidation("invalid plan selection", map[string]string{"plan_id": "selected subscription plan is not active"})
	}

	// 2. Check Tenant Email Uniqueness
	existingTenant, err := s.tenantRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingTenant != nil {
		return nil, appErrors.NewConflict(fmt.Sprintf("tenant with email '%s' already exists", req.Email))
	} else if err != nil && !appErrors.IsNotFound(err) {
		return nil, err
	}

	// 3. Check Tenant Admin User Email Uniqueness
	existingUser, err := s.tenantUserRepo.GetByEmail(ctx, adminEmail)
	if err == nil && existingUser != nil {
		return nil, appErrors.NewConflict(fmt.Sprintf("user with email '%s' already exists", adminEmail))
	} else if err != nil && !appErrors.IsNotFound(err) {
		return nil, err
	}

	// 4. Hash Tenant Admin Password
	passwordHash, err := s.hasher.Hash(req.AdminPassword)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to hash admin password during tenant onboarding", slog.String("error", err.Error()))
		return nil, appErrors.NewInternal(fmt.Errorf("failed to process admin credentials"))
	}

	// Prepare Entities
	tenant := &domain.Tenant{
		Name:   req.Name,
		Email:  req.Email,
		Phone:  req.Phone,
		Status: "active",
	}

	var adminUser *domain.TenantUser
	var summary *domain.TenantFinancialSummary
	var subscription *domain.PlatformSubscription

	endDate := time.Now().UTC().AddDate(0, 1, 0) // Default 30-day billing cycle

	// 5. Execute Atomic Multi-Step Transaction
	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Step 5a: Insert Tenant
		if err := s.tenantRepo.Create(txCtx, tenant); err != nil {
			return fmt.Errorf("tenant insert failed: %w", err)
		}

		// Step 5b: Insert Primary Tenant Admin User
		adminUser = &domain.TenantUser{
			TenantID:     tenant.ID,
			Name:         adminName,
			Role:         "admin",
			Email:        adminEmail,
			PasswordHash: passwordHash,
			Status:       "active",
		}
		if err := s.tenantUserRepo.Create(txCtx, adminUser); err != nil {
			return fmt.Errorf("tenant admin user insert failed: %w", err)
		}

		// Step 5c: Insert Initial Zeroed Financial Summary
		summary = &domain.TenantFinancialSummary{
			TenantID:        tenant.ID,
			CashBalance:     decimal.Zero,
			BankBalance:     decimal.Zero,
			TotalReceivable: decimal.Zero,
			TotalPayable:    decimal.Zero,
		}
		if err := s.summaryRepo.Create(txCtx, summary); err != nil {
			return fmt.Errorf("tenant financial summary insert failed: %w", err)
		}

		// Step 5d: Insert Initial Platform Subscription
		subscription = &domain.PlatformSubscription{
			TenantID:        tenant.ID,
			CurrentPlanID:   plan.ID,
			CurrentPlanName: plan.PlanName,
			Status:          "active",
			EndDate:         &endDate,
		}
		if err := s.subscriptionRepo.Create(txCtx, subscription); err != nil {
			return fmt.Errorf("tenant subscription insert failed: %w", err)
		}

		return nil
	})

	if err != nil {
		s.log.ErrorContext(ctx, "tenant onboarding transaction aborted", slog.String("error", err.Error()))
		return nil, err
	}

	// 6. Record Prometheus Metric
	if s.metrics != nil {
		s.metrics.IncTenantsCreated()
	}

	s.log.InfoContext(ctx, "tenant onboarded successfully",
		slog.String("tenant_id", tenant.ID.String()),
		slog.String("tenant_name", tenant.Name),
		slog.String("admin_email", adminUser.Email),
		slog.String("plan_name", plan.PlanName),
	)

	return &dto.OnboardTenantResponse{
		Tenant:           dto.ToTenantResponse(tenant),
		AdminUser:        dto.ToAdminUserResponse(adminUser),
		Subscription:     dto.ToSubscriptionResponse(subscription),
		FinancialSummary: dto.ToFinancialSummaryResponse(summary),
	}, nil
}

func (s *tenantService) GetTenantByID(ctx context.Context, id uuid.UUID) (*dto.TenantDetailResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &dto.TenantDetailResponse{
		Tenant: dto.ToTenantResponse(tenant),
	}

	// Lookup admin user
	admin, err := s.tenantUserRepo.GetAdminByTenantID(ctx, id)
	if err == nil && admin != nil {
		adminResp := dto.ToAdminUserResponse(admin)
		resp.AdminUser = &adminResp
	}

	// Lookup subscription
	sub, err := s.subscriptionRepo.GetByTenantID(ctx, id)
	if err == nil && sub != nil {
		subResp := dto.ToSubscriptionResponse(sub)
		resp.Subscription = &subResp
	}

	// Lookup financial summary
	financial, err := s.summaryRepo.GetByTenantID(ctx, id)
	if err == nil && financial != nil {
		finResp := dto.ToFinancialSummaryResponse(financial)
		resp.FinancialSummary = &finResp
	}

	return resp, nil
}

func (s *tenantService) ListTenants(ctx context.Context, page, pageSize int, status, search string) ([]dto.TenantResponse, int64, error) {
	tenants, total, err := s.tenantRepo.List(ctx, page, pageSize, status, search)
	if err != nil {
		return nil, 0, err
	}

	return dto.ToTenantListResponse(tenants), total, nil
}

func (s *tenantService) UpdateTenantStatus(ctx context.Context, id uuid.UUID, status string) error {
	switch status {
	case "active", "inactive", "suspended":
	default:
		return appErrors.NewValidation("invalid tenant status", map[string]string{"status": "must be one of: active, inactive, suspended"})
	}

	if err := s.tenantRepo.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	s.log.InfoContext(ctx, "tenant status updated",
		slog.String("tenant_id", id.String()),
		slog.String("status", status),
	)

	return nil
}

func (s *tenantService) ChangeSubscription(ctx context.Context, tenantID, planID uuid.UUID) (*dto.TenantSubscriptionResponse, error) {
	// 1. Verify Plan
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		if appErrors.IsNotFound(err) {
			return nil, appErrors.NewValidation("invalid plan selection", map[string]string{"plan_id": "selected subscription plan does not exist"})
		}
		return nil, err
	}
	if plan.Status != "active" {
		return nil, appErrors.NewValidation("invalid plan selection", map[string]string{"plan_id": "selected subscription plan is not active"})
	}

	// 2. Fetch Current Subscription
	sub, err := s.subscriptionRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// 3. Update Subscription
	newEndDate := time.Now().UTC().AddDate(0, 1, 0)
	sub.CurrentPlanID = plan.ID
	sub.CurrentPlanName = plan.PlanName
	sub.Status = "active"
	sub.EndDate = &newEndDate

	if err := s.subscriptionRepo.Update(ctx, sub); err != nil {
		s.log.ErrorContext(ctx, "failed to update tenant subscription",
			slog.String("tenant_id", tenantID.String()),
			slog.String("plan_id", planID.String()),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	s.log.InfoContext(ctx, "tenant subscription changed",
		slog.String("tenant_id", tenantID.String()),
		slog.String("new_plan", plan.PlanName),
	)

	resp := dto.ToSubscriptionResponse(sub)
	return &resp, nil
}

func (s *tenantService) GetTenantProfile(ctx context.Context, tenantID uuid.UUID) (*dto.TenantResponse, error) {
	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp := dto.ToTenantResponse(tenant)
	return &resp, nil
}

func (s *tenantService) GetTenantFinancialSummary(ctx context.Context, tenantID uuid.UUID) (*dto.TenantFinancialSummaryResponse, error) {
	summary, err := s.summaryRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp := dto.ToFinancialSummaryResponse(summary)
	return &resp, nil
}

func (s *tenantService) GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*dto.TenantSubscriptionResponse, error) {
	sub, err := s.subscriptionRepo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	resp := dto.ToSubscriptionResponse(sub)
	return &resp, nil
}
