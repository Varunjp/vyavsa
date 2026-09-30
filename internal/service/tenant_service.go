package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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

	// Tenant Member & Admin Operations
	GetTenantProfile(ctx context.Context, tenantID uuid.UUID) (*dto.TenantResponse, error)
	UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, req dto.UpdateTenantSettingsRequest) (*dto.TenantResponse, error)
	GetTenantFinancialSummary(ctx context.Context, tenantID uuid.UUID) (*dto.TenantFinancialSummaryResponse, error)
	GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*dto.TenantSubscriptionResponse, error)
	ListAvailablePlans(ctx context.Context) ([]dto.PlanResponse, error)
	PurchasePlan(ctx context.Context, tenantID uuid.UUID, req dto.PurchasePlanRequest) (*dto.TenantSubscriptionResponse, *dto.PlanTransactionResponse, error)
	ListTenantTransactions(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]dto.PlanTransactionResponse, int64, error)

	// Platform Administrator Operations
	GetPlatformDashboardMetrics(ctx context.Context) (*dto.PlatformDashboardMetricsResponse, error)
	ListPlatformSubscriptions(ctx context.Context, page, pageSize int, status string) ([]dto.PlatformSubscriptionItemResponse, int64, error)
	ListPlatformTransactions(ctx context.Context, page, pageSize int, status string) ([]dto.PlatformTransactionItemResponse, int64, error)
	UpdateTenant(ctx context.Context, id uuid.UUID, req dto.UpdatePlatformTenantRequest) (*dto.TenantDetailResponse, error)
}

type tenantService struct {
	tenantRepo       repository.TenantRepository
	tenantUserRepo   repository.TenantUserRepository
	summaryRepo      repository.TenantFinancialSummaryRepository
	subscriptionRepo repository.PlatformSubscriptionRepository
	planRepo         repository.PlatformPlanRepository
	txnRepo          repository.PlatformPlanTransactionRepository
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
	txnRepo repository.PlatformPlanTransactionRepository,
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
		txnRepo:          txnRepo,
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

func (s *tenantService) UpdateTenantSettings(ctx context.Context, tenantID uuid.UUID, req dto.UpdateTenantSettingsRequest) (*dto.TenantResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, appErrors.NewValidation("business name is required", map[string]string{"name": "cannot be empty"})
	}
	if strings.TrimSpace(req.Email) == "" {
		return nil, appErrors.NewValidation("email is required", map[string]string{"email": "cannot be empty"})
	}

	tenant, err := s.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	if req.Email != tenant.Email {
		existing, err := s.tenantRepo.GetByEmail(ctx, req.Email)
		if err == nil && existing != nil && existing.ID != tenantID {
			return nil, appErrors.NewConflict(fmt.Sprintf("a business organization with email '%s' already exists", req.Email))
		}
	}

	tenant.Name = req.Name
	tenant.Email = req.Email
	tenant.Phone = req.Phone

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "tenant settings updated",
		slog.String("tenant_id", tenantID.String()),
		slog.String("name", tenant.Name),
		slog.String("email", tenant.Email),
	)

	resp := dto.ToTenantResponse(tenant)
	return &resp, nil
}

func (s *tenantService) GetTenantFinancialSummary(ctx context.Context, tenantID uuid.UUID) (*dto.TenantFinancialSummaryResponse, error) {
	if s.summaryRepo != nil {
		if synced, err := s.summaryRepo.SyncFromSourceRecords(ctx, tenantID); err == nil && synced != nil {
			resp := dto.ToFinancialSummaryResponse(synced)
			return &resp, nil
		}
	}

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

	now := time.Now().UTC()
	if sub.EndDate != nil && sub.EndDate.Before(now) {
		if sub.Status == "active" {
			sub.Status = "expired"
			_ = s.subscriptionRepo.Update(ctx, sub)
		}
	}

	resp := dto.ToSubscriptionResponse(sub)
	if s.planRepo != nil {
		plan, planErr := s.planRepo.GetByID(ctx, sub.CurrentPlanID)
		if planErr == nil && plan != nil {
			resp.Price = plan.Price
			resp.Note = plan.Note
		}
	}

	return &resp, nil
}

func (s *tenantService) ListAvailablePlans(ctx context.Context) ([]dto.PlanResponse, error) {
	plans, _, err := s.planRepo.List(ctx, 1, 50, "active")
	if err != nil {
		return nil, err
	}
	return dto.ToPlanListResponse(plans), nil
}

func (s *tenantService) PurchasePlan(ctx context.Context, tenantID uuid.UUID, req dto.PurchasePlanRequest) (*dto.TenantSubscriptionResponse, *dto.PlanTransactionResponse, error) {
	// 1. Verify Plan
	plan, err := s.planRepo.GetByID(ctx, req.PlanID)
	if err != nil {
		if appErrors.IsNotFound(err) {
			return nil, nil, appErrors.NewValidation("invalid plan selection", map[string]string{"plan_id": "selected subscription plan does not exist"})
		}
		return nil, nil, err
	}
	if plan.Status != "active" {
		return nil, nil, appErrors.NewValidation("invalid plan selection", map[string]string{"plan_id": "selected subscription plan is not active"})
	}

	// 2. Validate Payment Method
	switch req.PaymentMethod {
	case "card", "upi", "netbanking", "cash", "bank_transfer", "mock_gateway":
	default:
		return nil, nil, appErrors.NewValidation("invalid payment method", map[string]string{"payment_method": "payment method must be one of: card, upi, netbanking, cash, bank_transfer, mock_gateway"})
	}

	txnID := fmt.Sprintf("TXN-%d-%s", time.Now().Unix(), strings.ToUpper(uuid.New().String()[:8]))

	// 3. Simulated Gateway Failure Handling
	if req.SimulateFail {
		failedTxn := &domain.PlatformPlanTransaction{
			TenantID:      tenantID,
			TransactionID: txnID,
			PaymentMethod: req.PaymentMethod,
			PlanID:        plan.ID,
			PlanName:      plan.PlanName,
			Amount:        plan.Price,
			Status:        "failed",
			FailureReason: "Payment declined by payment gateway",
		}
		if s.txnRepo != nil {
			_ = s.txnRepo.Create(ctx, failedTxn)
		}
		resp := dto.ToPlanTransactionResponse(failedTxn)
		return nil, &resp, appErrors.NewBadRequest("payment declined by payment gateway")
	}

	// 4. Atomic Execution of Successful Purchase & Subscription Upgrade
	var subResp dto.TenantSubscriptionResponse
	var txnResp dto.PlanTransactionResponse

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Record successful transaction
		txn := &domain.PlatformPlanTransaction{
			TenantID:      tenantID,
			TransactionID: txnID,
			PaymentMethod: req.PaymentMethod,
			PlanID:        plan.ID,
			PlanName:      plan.PlanName,
			Amount:        plan.Price,
			Status:        "completed",
		}
		if s.txnRepo != nil {
			if err := s.txnRepo.Create(txCtx, txn); err != nil {
				return fmt.Errorf("failed to record plan transaction: %w", err)
			}
		}

		// Update or Create Subscription
		sub, err := s.subscriptionRepo.GetByTenantID(txCtx, tenantID)
		now := time.Now().UTC()
		var startDate time.Time = now
		var newEndDate time.Time

		if err != nil {
			if appErrors.IsNotFound(err) {
				newEndDate = now.AddDate(0, 1, 0)
				sub = &domain.PlatformSubscription{
					TenantID:        tenantID,
					CurrentPlanID:   plan.ID,
					CurrentPlanName: plan.PlanName,
					Status:          "active",
					StartDate:       &startDate,
					EndDate:         &newEndDate,
				}
				if err := s.subscriptionRepo.Create(txCtx, sub); err != nil {
					return fmt.Errorf("failed to create subscription: %w", err)
				}
			} else {
				return err
			}
		} else {
			if sub.EndDate != nil && sub.EndDate.After(now) && sub.Status == "active" {
				// Extend from current active end date
				newEndDate = sub.EndDate.AddDate(0, 1, 0)
				if sub.StartDate != nil {
					startDate = *sub.StartDate
				}
			} else {
				// Starting new period
				newEndDate = now.AddDate(0, 1, 0)
			}
			sub.CurrentPlanID = plan.ID
			sub.CurrentPlanName = plan.PlanName
			sub.Status = "active"
			sub.StartDate = &startDate
			sub.EndDate = &newEndDate
			if err := s.subscriptionRepo.Update(txCtx, sub); err != nil {
				return fmt.Errorf("failed to update subscription: %w", err)
			}
		}

		subResp = dto.ToSubscriptionResponse(sub)
		subResp.Price = plan.Price
		subResp.Note = plan.Note
		txnResp = dto.ToPlanTransactionResponse(txn)
		return nil
	})

	if err != nil {
		s.log.ErrorContext(ctx, "failed to purchase plan", slog.String("tenant_id", tenantID.String()), slog.String("error", err.Error()))
		return nil, nil, err
	}

	s.log.InfoContext(ctx, "subscription plan purchased successfully",
		slog.String("tenant_id", tenantID.String()),
		slog.String("plan_name", plan.PlanName),
		slog.String("transaction_id", txnID),
		slog.String("amount", plan.Price.String()),
	)

	return &subResp, &txnResp, nil
}

func (s *tenantService) ListTenantTransactions(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]dto.PlanTransactionResponse, int64, error) {
	if s.txnRepo == nil {
		return []dto.PlanTransactionResponse{}, 0, nil
	}
	txns, total, err := s.txnRepo.ListByTenantID(ctx, tenantID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return dto.ToPlanTransactionListResponse(txns), total, nil
}

func (s *tenantService) GetPlatformDashboardMetrics(ctx context.Context) (*dto.PlatformDashboardMetricsResponse, error) {
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	sevenDaysAgo := now.AddDate(0, 0, -7)
	sixMonthsAgo := time.Date(now.Year(), now.Month()-5, 1, 0, 0, 0, 0, time.UTC)

	activeCount, err := s.tenantRepo.CountByStatus(ctx, "active")
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get active tenants count", slog.String("error", err.Error()))
		return nil, err
	}

	totalCount, err := s.tenantRepo.CountByStatus(ctx, "")
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get total tenants count", slog.String("error", err.Error()))
		return nil, err
	}

	recentRegs, err := s.tenantRepo.CountSince(ctx, sevenDaysAgo)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get recent registrations count", slog.String("error", err.Error()))
		return nil, err
	}

	monthlyIncome, err := s.txnRepo.GetMonthlyReceivedIncome(ctx, startOfMonth)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get monthly received income", slog.String("error", err.Error()))
		return nil, err
	}

	rawTrend, err := s.txnRepo.GetMonthlyRevenueTrend(ctx, sixMonthsAgo)
	if err != nil {
		s.log.ErrorContext(ctx, "failed to get monthly revenue trend", slog.String("error", err.Error()))
		return nil, err
	}

	trendMap := make(map[string]repository.MonthlyRevenueAggregate, len(rawTrend))
	for _, item := range rawTrend {
		trendMap[item.MonthKey] = item
	}

	revenueTrend := make([]dto.MonthlyRevenueItem, 6)
	for i := 0; i < 6; i++ {
		m := time.Date(now.Year(), now.Month()-time.Month(5-i), 1, 0, 0, 0, 0, time.UTC)
		key := m.Format("2006-01")
		label := m.Format("Jan 2006")
		if item, exists := trendMap[key]; exists {
			revenueTrend[i] = dto.MonthlyRevenueItem{
				Month:   label,
				Revenue: item.Revenue,
				Count:   item.Count,
			}
		} else {
			revenueTrend[i] = dto.MonthlyRevenueItem{
				Month:   label,
				Revenue: decimal.Zero,
				Count:   0,
			}
		}
	}

	return &dto.PlatformDashboardMetricsResponse{
		ActiveTenants:         activeCount,
		TotalTenants:          totalCount,
		MonthlyReceivedIncome: monthlyIncome,
		RecentRegistrations7d: recentRegs,
		RevenueTrend:          revenueTrend,
	}, nil
}

func (s *tenantService) ListPlatformSubscriptions(ctx context.Context, page, pageSize int, status string) ([]dto.PlatformSubscriptionItemResponse, int64, error) {
	if s.subscriptionRepo == nil {
		return []dto.PlatformSubscriptionItemResponse{}, 0, nil
	}
	subs, total, err := s.subscriptionRepo.List(ctx, page, pageSize, status)
	if err != nil {
		return nil, 0, err
	}

	items := make([]dto.PlatformSubscriptionItemResponse, len(subs))
	for i, sub := range subs {
		items[i] = dto.PlatformSubscriptionItemResponse{
			ID:              sub.ID,
			TenantID:        sub.TenantID,
			TenantName:      sub.TenantName,
			TenantEmail:     sub.TenantEmail,
			CurrentPlanID:   sub.CurrentPlanID,
			CurrentPlanName: sub.CurrentPlanName,
			Status:          sub.Status,
			StartDate:       sub.StartDate,
			EndDate:         sub.EndDate,
			CreatedAt:       sub.CreatedAt,
			UpdatedAt:       sub.UpdatedAt,
		}
	}
	return items, total, nil
}

func (s *tenantService) ListPlatformTransactions(ctx context.Context, page, pageSize int, status string) ([]dto.PlatformTransactionItemResponse, int64, error) {
	if s.txnRepo == nil {
		return []dto.PlatformTransactionItemResponse{}, 0, nil
	}
	txns, total, err := s.txnRepo.ListAll(ctx, page, pageSize, status)
	if err != nil {
		return nil, 0, err
	}

	items := make([]dto.PlatformTransactionItemResponse, len(txns))
	for i, tx := range txns {
		items[i] = dto.PlatformTransactionItemResponse{
			ID:            tx.ID,
			TenantID:      tx.TenantID,
			TenantName:    tx.TenantName,
			TenantEmail:   tx.TenantEmail,
			TransactionID: tx.TransactionID,
			PaymentMethod: tx.PaymentMethod,
			PlanID:        tx.PlanID,
			PlanName:      tx.PlanName,
			Amount:        tx.Amount,
			Status:        tx.Status,
			FailureReason: tx.FailureReason,
			CreatedAt:     tx.CreatedAt,
			UpdatedAt:     tx.UpdatedAt,
		}
	}
	return items, total, nil
}

func (s *tenantService) UpdateTenant(ctx context.Context, id uuid.UUID, req dto.UpdatePlatformTenantRequest) (*dto.TenantDetailResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, appErrors.NewValidation("tenant name is required", map[string]string{"name": "cannot be empty"})
	}
	if strings.TrimSpace(req.Email) == "" {
		return nil, appErrors.NewValidation("tenant email is required", map[string]string{"email": "cannot be empty"})
	}

	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Email != tenant.Email {
		existing, err := s.tenantRepo.GetByEmail(ctx, req.Email)
		if err == nil && existing != nil && existing.ID != id {
			return nil, appErrors.NewConflict(fmt.Sprintf("a business organization with email '%s' already exists", req.Email))
		}
	}

	tenant.Name = strings.TrimSpace(req.Name)
	tenant.Email = strings.TrimSpace(req.Email)
	tenant.Phone = strings.TrimSpace(req.Phone)

	if req.Status != "" {
		validStatuses := map[string]bool{"active": true, "inactive": true, "suspended": true}
		if !validStatuses[req.Status] {
			return nil, appErrors.NewValidation("invalid tenant status", map[string]string{"status": "must be active, inactive, or suspended"})
		}
		tenant.Status = req.Status
	}

	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, err
	}

	if req.Status != "" {
		if err := s.tenantRepo.UpdateStatus(ctx, id, req.Status); err != nil {
			return nil, err
		}
	}

	s.log.InfoContext(ctx, "platform admin updated tenant",
		slog.String("tenant_id", id.String()),
		slog.String("name", tenant.Name),
		slog.String("email", tenant.Email),
		slog.String("status", tenant.Status),
	)

	return s.GetTenantByID(ctx, id)
}
