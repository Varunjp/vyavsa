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

type validatedBankPayment struct {
	BankID   uuid.UUID
	BankName string
	Amount   decimal.Decimal
	Note     string
}

// TenantOperationsService orchestrates operational business logic for Tenant Admin and Tenant Users
type TenantOperationsService struct {
	userRepo      repository.TenantUserRepository
	empRepo       repository.TenantEmployeeRepository
	custRepo      repository.TenantCustomerRepository
	bankRepo      repository.TenantBankRepository
	lineSaleRepo  repository.LineSaleRepository
	countSaleRepo repository.CounterSaleRepository
	purchRepo     repository.TenantPurchaseRepository
	expRepo       repository.TenantExpenseRepository
	attRepo       repository.AttendanceRepository
	salaryRepo    repository.EmployeeSalaryRepository
	statsRepo     repository.TenantDailyStatsRepository
	summaryRepo   repository.TenantFinancialSummaryRepository
	transactor    repository.Transactor
	hasher        auth.PasswordHasher
	metrics       *metrics.Metrics
	location      *time.Location
	logger        *slog.Logger
}

func NewTenantOperationsService(
	userRepo repository.TenantUserRepository,
	empRepo repository.TenantEmployeeRepository,
	custRepo repository.TenantCustomerRepository,
	bankRepo repository.TenantBankRepository,
	lineSaleRepo repository.LineSaleRepository,
	countSaleRepo repository.CounterSaleRepository,
	purchRepo repository.TenantPurchaseRepository,
	expRepo repository.TenantExpenseRepository,
	attRepo repository.AttendanceRepository,
	salaryRepo repository.EmployeeSalaryRepository,
	statsRepo repository.TenantDailyStatsRepository,
	summaryRepo repository.TenantFinancialSummaryRepository,
	transactor repository.Transactor,
	hasher auth.PasswordHasher,
	logger *slog.Logger,
) *TenantOperationsService {
	return &TenantOperationsService{
		userRepo:      userRepo,
		empRepo:       empRepo,
		custRepo:      custRepo,
		bankRepo:      bankRepo,
		lineSaleRepo:  lineSaleRepo,
		countSaleRepo: countSaleRepo,
		purchRepo:     purchRepo,
		expRepo:       expRepo,
		attRepo:       attRepo,
		salaryRepo:    salaryRepo,
		statsRepo:     statsRepo,
		summaryRepo:   summaryRepo,
		transactor:    transactor,
		hasher:        hasher,
		logger:        logger,
	}
}

var defaultBusinessLocation *time.Location

// SetDefaultBusinessLocation sets the package-level business timezone location
func SetDefaultBusinessLocation(loc *time.Location) {
	defaultBusinessLocation = loc
}

func (s *TenantOperationsService) SetLocation(loc *time.Location) {
	s.location = loc
	SetDefaultBusinessLocation(loc)
}

func (s *TenantOperationsService) SetMetrics(m *metrics.Metrics) {
	s.metrics = m
}

// todayString returns today's date formatted as YYYY-MM-DD in the configured business timezone
func todayString() string {
	loc := defaultBusinessLocation
	if loc == nil {
		loc = time.Local
		if loc == nil {
			loc = time.UTC
		}
	}
	return time.Now().In(loc).Format("2006-01-02")
}

// checkCurrentDayRestriction ensures Tenant Users can only record or mutate today's transactions
func checkCurrentDayRestriction(role string, recordDate string) error {
	if role == auth.RoleTenantUser {
		today := todayString()
		if recordDate != "" && recordDate != today {
			return appErrors.NewForbidden("forbidden: tenant users can only process transactions for the current business day")
		}
	}
	return nil
}

func validationErr(field, msg string) *appErrors.AppError {
	return appErrors.NewValidation(msg, map[string]string{field: msg})
}

// ==========================================
// 1. Tenant User Management (Admin only)
// ==========================================

func (s *TenantOperationsService) CreateTenantUser(ctx context.Context, tenantID uuid.UUID, req *dto.CreateTenantUserRequest) (*domain.TenantUser, error) {
	existing, _ := s.userRepo.GetByTenantAndEmail(ctx, tenantID, req.Email)
	if existing != nil {
		return nil, appErrors.NewConflict("user with this email already exists within organization")
	}

	hash, err := s.hasher.Hash(req.Password)
	if err != nil {
		return nil, appErrors.NewInternal(fmt.Errorf("failed to hash password during user creation: %w", err))
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	user := &domain.TenantUser{
		TenantID:     tenantID,
		Name:         req.Name,
		Role:         req.Role,
		Email:        req.Email,
		PasswordHash: hash,
		Status:       status,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *TenantOperationsService) UpdateTenantUser(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateTenantUserRequest) (*domain.TenantUser, error) {
	user, err := s.userRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Status != "" {
		user.Status = req.Status
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	if req.Password != "" {
		hash, err := s.hasher.Hash(req.Password)
		if err != nil {
			return nil, appErrors.NewInternal(fmt.Errorf("failed to hash new password during user update: %w", err))
		}
		if err := s.userRepo.UpdatePassword(ctx, tenantID, id, hash); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (s *TenantOperationsService) DeleteTenantUser(ctx context.Context, tenantID, currentUserID, id uuid.UUID) error {
	if currentUserID == id {
		return appErrors.NewBadRequest("cannot delete your own active administrator account")
	}
	return s.userRepo.Delete(ctx, tenantID, id)
}

func (s *TenantOperationsService) GetTenantUserByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantUser, error) {
	return s.userRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListTenantUsers(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, role, status string) ([]domain.TenantUser, int64, error) {
	return s.userRepo.List(ctx, tenantID, page, pageSize, search, role, status)
}

// ==========================================
// 2. Employee Management (Admin only)
// ==========================================

func (s *TenantOperationsService) CreateEmployee(ctx context.Context, tenantID uuid.UUID, req *dto.CreateEmployeeRequest) (*domain.TenantEmployee, error) {
	if req.Salary.IsNegative() {
		return nil, validationErr("salary", "salary cannot be negative")
	}
	if req.OTRate.IsNegative() {
		return nil, validationErr("ot_rate", "ot rate cannot be negative")
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	emp := &domain.TenantEmployee{
		TenantID: tenantID,
		Name:     req.Name,
		Phone:    req.Phone,
		Salary:   req.Salary,
		OTRate:   req.OTRate,
		Status:   status,
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.empRepo.Create(txCtx, emp); err != nil {
			return err
		}
		// Initialize salary balance record to zero
		return s.salaryRepo.UpsertBalance(txCtx, &domain.EmployeeSalary{
			TenantID:   tenantID,
			EmployeeID: emp.ID,
			Balance:    decimal.Zero,
		})
	})
	if err != nil {
		return nil, err
	}

	return emp, nil
}

func (s *TenantOperationsService) UpdateEmployee(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateEmployeeRequest) (*domain.TenantEmployee, error) {
	emp, err := s.empRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		emp.Name = req.Name
	}
	if req.Phone != "" {
		emp.Phone = req.Phone
	}
	if req.Salary != nil {
		if req.Salary.IsNegative() {
			return nil, validationErr("salary", "salary cannot be negative")
		}
		emp.Salary = *req.Salary
	}
	if req.OTRate != nil {
		if req.OTRate.IsNegative() {
			return nil, validationErr("ot_rate", "ot rate cannot be negative")
		}
		emp.OTRate = *req.OTRate
	}
	if req.Status != "" {
		emp.Status = req.Status
	}

	if err := s.empRepo.Update(ctx, emp); err != nil {
		return nil, err
	}

	return emp, nil
}

func (s *TenantOperationsService) DeleteEmployee(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.empRepo.Delete(ctx, tenantID, id)
}

func (s *TenantOperationsService) GetEmployeeByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantEmployee, error) {
	return s.empRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListEmployees(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantEmployee, int64, error) {
	return s.empRepo.List(ctx, tenantID, page, pageSize, search, status)
}

// ==========================================
// 3. Customer Management (Admin only)
// ==========================================

func (s *TenantOperationsService) CreateCustomer(ctx context.Context, tenantID uuid.UUID, req *dto.CreateCustomerRequest) (*domain.TenantCustomer, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}

	cust := &domain.TenantCustomer{
		TenantID:       tenantID,
		CustomerName:   req.CustomerName,
		Phone:          req.Phone,
		OpeningBalance: req.OpeningBalance,
		CurrentBalance: req.OpeningBalance,
		Status:         status,
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.custRepo.Create(txCtx, cust); err != nil {
			return err
		}
		// If opening balance > 0, reflect in total_receivable
		if req.OpeningBalance.GreaterThan(decimal.Zero) {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil {
				summary.TotalReceivable = summary.TotalReceivable.Add(req.OpeningBalance)
				if err := s.summaryRepo.Update(txCtx, summary); err != nil {
					return fmt.Errorf("failed to update financial summary for customer opening balance: %w", err)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return cust, nil
}

func (s *TenantOperationsService) UpdateCustomer(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateCustomerRequest) (*domain.TenantCustomer, error) {
	cust, err := s.custRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.CustomerName != "" {
		cust.CustomerName = req.CustomerName
	}
	if req.Phone != "" {
		cust.Phone = req.Phone
	}
	if req.Status != "" {
		cust.Status = req.Status
	}

	if err := s.custRepo.Update(ctx, cust); err != nil {
		return nil, err
	}

	return cust, nil
}

func (s *TenantOperationsService) DeleteCustomer(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.custRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		return nil
	})
}

func (s *TenantOperationsService) GetCustomerByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantCustomer, error) {
	return s.custRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListCustomers(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantCustomer, int64, error) {
	return s.custRepo.List(ctx, tenantID, page, pageSize, search, status)
}

func (s *TenantOperationsService) AdjustCustomerBalance(ctx context.Context, tenantID, customerID uuid.UUID, req *dto.AdjustCustomerBalanceRequest) (*domain.TenantCustomer, error) {
	if req.Reason == "" {
		return nil, validationErr("reason", "reason for balance adjustment is required")
	}
	if req.NewBalance == nil && req.AdjustmentAmount == nil {
		return nil, validationErr("balance", "either new_balance or adjustment_amount must be provided")
	}

	cust, err := s.custRepo.GetByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid customer for tenant")
	}

	prevBalance := cust.CurrentBalance
	var newBalance, delta decimal.Decimal

	if req.NewBalance != nil {
		newBalance = *req.NewBalance
		delta = newBalance.Sub(prevBalance)
	} else {
		delta = *req.AdjustmentAmount
		newBalance = prevBalance.Add(delta)
	}

	adjustment := &domain.CustomerBalanceAdjustment{
		TenantID:         tenantID,
		CustomerID:       customerID,
		CustomerName:     cust.CustomerName,
		PreviousBalance:  prevBalance,
		NewBalance:       newBalance,
		AdjustmentAmount: delta,
		Reason:           req.Reason,
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.custRepo.SetBalance(txCtx, tenantID, customerID, newBalance); err != nil {
			return err
		}

		if err := s.custRepo.RecordAdjustment(txCtx, adjustment); err != nil {
			return err
		}

		summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
		if err == nil {
			summary.TotalReceivable = summary.TotalReceivable.Add(delta)
			if summary.TotalReceivable.IsNegative() {
				summary.TotalReceivable = decimal.Zero
			}
			if err := s.summaryRepo.Update(txCtx, summary); err != nil {
				return fmt.Errorf("failed to update financial summary for customer balance adjustment: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	cust.CurrentBalance = newBalance
	return cust, nil
}

func (s *TenantOperationsService) ListCustomerAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, page, pageSize int) ([]domain.CustomerBalanceAdjustment, int64, error) {
	if _, err := s.custRepo.GetByID(ctx, tenantID, customerID); err != nil {
		return nil, 0, err
	}
	return s.custRepo.ListAdjustments(ctx, tenantID, customerID, page, pageSize)
}

func (s *TenantOperationsService) GetCustomerBalance(ctx context.Context, tenantID, customerID uuid.UUID) (*dto.CustomerBalanceResponse, error) {
	cust, err := s.custRepo.GetByID(ctx, tenantID, customerID)
	if err != nil {
		return nil, err
	}
	return &dto.CustomerBalanceResponse{
		CustomerID:     cust.ID,
		CustomerName:   cust.CustomerName,
		CurrentBalance: cust.CurrentBalance,
		Status:         cust.Status,
	}, nil
}

// ==========================================
// 4. Bank Management (Admin only)
// ==========================================

func (s *TenantOperationsService) CreateBank(ctx context.Context, tenantID uuid.UUID, req *dto.CreateBankRequest) (*domain.TenantBank, error) {
	status := req.Status
	if status == "" {
		status = "active"
	}

	currentBal := req.OpeningBalance
	if currentBal.IsNegative() {
		currentBal = decimal.Zero
	}

	bank := &domain.TenantBank{
		TenantID:       tenantID,
		BankName:       req.BankName,
		AccountNumber:  req.AccountNumber,
		IFSC:           req.IFSC,
		CurrentBalance: currentBal,
		Status:         status,
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.bankRepo.Create(txCtx, bank); err != nil {
			return err
		}
		if currentBal.GreaterThan(decimal.Zero) {
			tx := &domain.BankTransaction{
				TenantID:        tenantID,
				BankID:          bank.ID,
				Amount:          currentBal,
				TransactionType: "credit",
				Reason:          "Opening Balance",
			}
			if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
				return err
			}
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil {
				summary.BankBalance = summary.BankBalance.Add(currentBal)
				if err := s.summaryRepo.Update(txCtx, summary); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return bank, nil
}

func (s *TenantOperationsService) UpdateBank(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateBankRequest) (*domain.TenantBank, error) {
	bank, err := s.bankRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.BankName != "" {
		bank.BankName = req.BankName
	}
	if req.AccountNumber != "" {
		bank.AccountNumber = req.AccountNumber
	}
	if req.IFSC != "" {
		bank.IFSC = req.IFSC
	}
	if req.Status != "" {
		bank.Status = req.Status
	}

	if err := s.bankRepo.Update(ctx, bank); err != nil {
		return nil, err
	}

	return bank, nil
}

func (s *TenantOperationsService) DeleteBank(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.bankRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		return nil
	})
}

func (s *TenantOperationsService) GetBankByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantBank, error) {
	return s.bankRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListBanks(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantBank, int64, error) {
	return s.bankRepo.List(ctx, tenantID, page, pageSize, search, status)
}

func (s *TenantOperationsService) ListBankTransactions(ctx context.Context, tenantID, bankID uuid.UUID, page, pageSize int) ([]domain.BankTransaction, int64, error) {
	if _, err := s.bankRepo.GetByID(ctx, tenantID, bankID); err != nil {
		return nil, 0, err
	}
	return s.bankRepo.ListTransactions(ctx, tenantID, bankID, page, pageSize)
}

// ==========================================
// 5. Line Sale Operations
// ==========================================

func (s *TenantOperationsService) CreateLineSale(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreateLineSaleRequest) (*domain.LineSale, error) {
	if req.TotalAmount.IsNegative() {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("negative_total_amount")
		}
		return nil, validationErr("total_amount", "total amount cannot be negative")
	}

	// Verify customer belongs to tenant
	cust, err := s.custRepo.GetByID(ctx, tenantID, req.CustomerID)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("invalid_customer")
		}
		return nil, appErrors.NewBadRequest("invalid customer for tenant")
	}

	var cashReceived decimal.Decimal
	if req.CashAmount != nil {
		if req.CashAmount.IsNegative() {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("negative_cash_amount")
			}
			return nil, validationErr("cash_amount", "cash amount cannot be negative")
		}
		cashReceived = *req.CashAmount
	} else if req.TotalCashIn.IsNegative() {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("negative_cash_amount")
		}
		return nil, validationErr("total_cash_in", "cash amount cannot be negative")
	} else if req.TotalCashIn.GreaterThan(decimal.Zero) {
		cashReceived = req.TotalCashIn
	}

	// Extract raw bank splits
	var rawBankSplits []dto.BankPaymentSplitRequest
	if len(req.BankPayments) > 0 {
		rawBankSplits = req.BankPayments
	} else if len(req.Payments) > 0 {
		for _, p := range req.Payments {
			if p.Amount.IsNegative() || p.Amount.IsZero() {
				if s.metrics != nil {
					s.metrics.RecordSalesPaymentFailed("invalid_payment_amount")
				}
				return nil, validationErr("payment_amount", "payment amount must be greater than zero")
			}
			if p.PaymentMethod == "cash" {
				cashReceived = cashReceived.Add(p.Amount)
			} else if p.PaymentMethod == "bank" || p.PaymentMethod == "upi" || p.PaymentMethod == "online" || p.PaymentMethod == "cheque" || p.PaymentMethod == "other" {
				if p.BankID == nil || *p.BankID == uuid.Nil {
					if s.metrics != nil {
						s.metrics.RecordSalesPaymentFailed("missing_bank_id")
					}
					return nil, validationErr("bank_id", "bank account must be selected for bank payments")
				}
				rawBankSplits = append(rawBankSplits, dto.BankPaymentSplitRequest{
					BankID:   *p.BankID,
					BankName: p.BankName,
					Amount:   p.Amount,
					Note:     p.Note,
				})
			}
		}
	} else if req.BankAmount.GreaterThan(decimal.Zero) {
		if req.BankID == nil || *req.BankID == uuid.Nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("missing_bank_id")
			}
			return nil, validationErr("bank_id", "bank account must be selected when bank amount is specified")
		}
		rawBankSplits = append(rawBankSplits, dto.BankPaymentSplitRequest{
			BankID: *req.BankID,
			Amount: req.BankAmount,
		})
	}

	// Validate bank payments & tenant isolation
	seenBanks := make(map[uuid.UUID]bool)
	validatedBanks := make([]validatedBankPayment, 0, len(rawBankSplits))
	var bankReceived decimal.Decimal

	for _, bp := range rawBankSplits {
		if bp.BankID == uuid.Nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("empty_bank_id")
			}
			return nil, validationErr("bank_payments", "bank account must be selected for each bank payment")
		}
		if seenBanks[bp.BankID] {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("duplicate_bank")
			}
			return nil, validationErr("bank_payments", "duplicate bank account selected in payments")
		}
		seenBanks[bp.BankID] = true

		if bp.Amount.IsNegative() || bp.Amount.IsZero() {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("zero_or_negative_bank_amount")
			}
			return nil, validationErr("bank_payments", "bank payment amount must be greater than zero")
		}

		bank, err := s.bankRepo.GetByID(ctx, tenantID, bp.BankID)
		if err != nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("invalid_bank")
			}
			return nil, appErrors.NewBadRequest("invalid bank account specified for line sale payment")
		}
		if bank.Status != "active" {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("inactive_bank")
			}
			return nil, appErrors.NewBadRequest("selected bank account is inactive")
		}

		validatedBanks = append(validatedBanks, validatedBankPayment{
			BankID:   bank.ID,
			BankName: bank.BankName,
			Amount:   bp.Amount,
			Note:     bp.Note,
		})
		bankReceived = bankReceived.Add(bp.Amount)
	}

	collectedAmount := cashReceived.Add(bankReceived)
	if collectedAmount.GreaterThan(req.TotalAmount) {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("collected_exceeds_total")
		}
		return nil, validationErr("collected_amount", "collected amount cannot exceed sale total")
	}

	due := req.TotalAmount.Sub(collectedAmount)
	if due.IsNegative() {
		due = decimal.Zero
	}

	sale := &domain.LineSale{
		TenantID:        tenantID,
		CustomerID:      cust.ID,
		CustomerName:    cust.CustomerName,
		Route:           req.Route,
		Salesman:        req.Salesman,
		Note:            req.Note,
		TotalAmount:     req.TotalAmount,
		TotalCashIn:     cashReceived,
		BankAmount:      bankReceived,
		CollectedAmount: collectedAmount,
		Balance:         due,
	}

	payments := make([]domain.LineSalePayment, 0, len(validatedBanks)+1)
	if cashReceived.GreaterThan(decimal.Zero) {
		payments = append(payments, domain.LineSalePayment{
			TenantID:      tenantID,
			PaymentMethod: "cash",
			Amount:        cashReceived,
			Note:          "Cash payment",
		})
	}
	for _, vb := range validatedBanks {
		note := vb.Note
		if note == "" {
			note = "Bank payment"
		}
		payments = append(payments, domain.LineSalePayment{
			TenantID:      tenantID,
			PaymentMethod: "bank",
			BankID:        &vb.BankID,
			BankName:      vb.BankName,
			Amount:        vb.Amount,
			Note:          note,
		})
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.lineSaleRepo.Create(txCtx, sale, payments); err != nil {
			return err
		}

		for _, vb := range validatedBanks {
			if err := s.bankRepo.AdjustBalance(txCtx, tenantID, vb.BankID, vb.Amount); err != nil {
				return fmt.Errorf("failed to adjust bank balance for bank %s: %w", vb.BankName, err)
			}
			tx := &domain.BankTransaction{
				TenantID:        tenantID,
				BankID:          vb.BankID,
				Amount:          vb.Amount,
				TransactionType: "credit",
				Reason:          "Line Sale: " + cust.CustomerName,
				SaleType:        "line_sale",
				SaleID:          &sale.ID,
			}
			if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
				return fmt.Errorf("failed to record bank transaction for bank %s: %w", vb.BankName, err)
			}
		}

		// Adjust customer current balance by remaining credit balance
		if due.GreaterThan(decimal.Zero) {
			if err := s.custRepo.AdjustBalance(txCtx, tenantID, cust.ID, due); err != nil {
				return err
			}
		}

		// Update financial summary
		summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
		if err == nil {
			summary.CashBalance = summary.CashBalance.Add(cashReceived)
			summary.BankBalance = summary.BankBalance.Add(bankReceived)
			summary.TotalReceivable = summary.TotalReceivable.Add(due)
			if err := s.summaryRepo.Update(txCtx, summary); err != nil {
				return fmt.Errorf("failed to update financial summary for line sale: %w", err)
			}
		}

		// Sync tenant daily stats (best-effort: does not abort transaction on failure)
		if _, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString()); err != nil {
			s.logger.WarnContext(txCtx, "failed to sync daily stats after line sale creation", "error", err.Error())
		}

		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("transaction_error")
		}
		return nil, err
	}

	if s.metrics != nil {
		if cashReceived.GreaterThan(decimal.Zero) {
			s.metrics.RecordSalesPayment("cash", cashReceived.InexactFloat64())
		}
		for _, vb := range validatedBanks {
			s.metrics.RecordSalesPayment("bank", vb.Amount.InexactFloat64())
		}
	}

	return sale, nil
}

func (s *TenantOperationsService) UpdateLineSale(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdateLineSaleRequest) (*domain.LineSale, error) {
	sale, err := s.lineSaleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, sale.CreatedAt.Format("2006-01-02")); err != nil {
		return nil, err
	}

	oldBalance := sale.Balance
	oldCash := sale.TotalCashIn

	if req.Route != "" {
		sale.Route = req.Route
	}
	if req.Salesman != "" {
		sale.Salesman = req.Salesman
	}
	if req.Note != "" {
		sale.Note = req.Note
	}
	if req.TotalAmount != nil {
		sale.TotalAmount = *req.TotalAmount
		sale.Balance = sale.TotalAmount.Sub(sale.TotalCashIn)
	}
	if req.TotalCashIn != nil {
		sale.TotalCashIn = *req.TotalCashIn
		sale.Balance = sale.TotalAmount.Sub(sale.TotalCashIn)
	}

	deltaBalance := sale.Balance.Sub(oldBalance)
	deltaCash := sale.TotalCashIn.Sub(oldCash)

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// 1. Adjust customer balance by delta in credit
		if !deltaBalance.IsZero() && s.custRepo != nil {
			if err := s.custRepo.AdjustBalance(txCtx, tenantID, sale.CustomerID, deltaBalance); err != nil {
				return fmt.Errorf("failed to adjust customer balance on line sale update: %w", err)
			}
		}

		// 2. Update line sale record
		if err := s.lineSaleRepo.Update(txCtx, sale); err != nil {
			return err
		}

		// 3. Update financial summary cash balance and sync authoritative balances
		if s.summaryRepo != nil {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil && summary != nil {
				summary.CashBalance = summary.CashBalance.Add(deltaCash)
				summary.TotalReceivable = summary.TotalReceivable.Add(deltaBalance)
				_ = s.summaryRepo.Update(txCtx, summary)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}

		// 4. Compute and sync daily stats
		if s.statsRepo != nil {
			_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, sale.CreatedAt.Format("2006-01-02"))
			if sale.CreatedAt.Format("2006-01-02") != todayString() {
				_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString())
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return sale, nil
}

func (s *TenantOperationsService) DeleteLineSale(ctx context.Context, tenantID, id uuid.UUID, role string) error {
	sale, err := s.lineSaleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if err := checkCurrentDayRestriction(role, sale.CreatedAt.Format("2006-01-02")); err != nil {
		return err
	}

	return s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		// Reverse balance on customer if unpaid
		if sale.Balance.GreaterThan(decimal.Zero) && s.custRepo != nil {
			if err := s.custRepo.AdjustBalance(txCtx, tenantID, sale.CustomerID, sale.Balance.Neg()); err != nil {
				return fmt.Errorf("failed to reverse customer balance on line sale deletion: %w", err)
			}
		}
		if err := s.lineSaleRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil && summary != nil {
				summary.CashBalance = summary.CashBalance.Sub(sale.TotalCashIn)
				summary.BankBalance = summary.BankBalance.Sub(sale.BankAmount)
				summary.TotalReceivable = summary.TotalReceivable.Sub(sale.Balance)
				_ = s.summaryRepo.Update(txCtx, summary)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		if s.statsRepo != nil {
			_, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, sale.CreatedAt.Format("2006-01-02"))
			if err != nil {
				s.logger.WarnContext(txCtx, "failed to sync daily stats after line sale deletion", "error", err.Error())
			}
			if sale.CreatedAt.Format("2006-01-02") != todayString() {
				_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString())
			}
		}
		return nil
	})
}

func (s *TenantOperationsService) GetLineSaleByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error) {
	return s.lineSaleRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListLineSales(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.LineSale, int64, error) {
	if role == auth.RoleTenantUser {
		date = todayString() // Force current-day for tenant users
	}
	return s.lineSaleRepo.List(ctx, tenantID, page, pageSize, date, customerID, search)
}

// ==========================================
// 6. Counter Sale Operations
// ==========================================

func (s *TenantOperationsService) CreateCounterSale(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreateCounterSaleRequest) (*domain.CounterSale, error) {
	if req.TotalAmount.IsNegative() {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("negative_total_amount")
		}
		return nil, validationErr("total_amount", "total amount cannot be negative")
	}

	var cashReceived decimal.Decimal
	if req.CashAmount != nil {
		if req.CashAmount.IsNegative() {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("negative_cash_amount")
			}
			return nil, validationErr("cash_amount", "cash amount cannot be negative")
		}
		cashReceived = *req.CashAmount
	} else if req.Cash.IsNegative() {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("negative_cash_amount")
		}
		return nil, validationErr("cash", "cash amount cannot be negative")
	} else if req.Cash.GreaterThan(decimal.Zero) {
		cashReceived = req.Cash
	}

	var rawBankSplits []dto.BankPaymentSplitRequest
	if len(req.BankPayments) > 0 {
		rawBankSplits = req.BankPayments
	} else if len(req.Payments) > 0 {
		for _, p := range req.Payments {
			if p.Amount.IsNegative() || p.Amount.IsZero() {
				if s.metrics != nil {
					s.metrics.RecordSalesPaymentFailed("invalid_payment_amount")
				}
				return nil, validationErr("payment_amount", "payment amount must be greater than zero")
			}
			if p.PaymentMethod == "cash" {
				cashReceived = cashReceived.Add(p.Amount)
			} else if p.PaymentMethod == "bank" || p.PaymentMethod == "online" || p.PaymentMethod == "upi" || p.PaymentMethod == "cheque" || p.PaymentMethod == "other" {
				if p.BankID == nil || *p.BankID == uuid.Nil {
					if s.metrics != nil {
						s.metrics.RecordSalesPaymentFailed("missing_bank_id")
					}
					return nil, validationErr("bank_id", "bank account must be selected for bank payments")
				}
				rawBankSplits = append(rawBankSplits, dto.BankPaymentSplitRequest{
					BankID:   *p.BankID,
					BankName: p.BankName,
					Amount:   p.Amount,
					Note:     p.Note,
				})
			}
		}
	} else if req.BankAmount.GreaterThan(decimal.Zero) {
		if req.BankID == nil || *req.BankID == uuid.Nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("missing_bank_id")
			}
			return nil, validationErr("bank_id", "bank account must be selected when bank amount is specified")
		}
		rawBankSplits = append(rawBankSplits, dto.BankPaymentSplitRequest{
			BankID: *req.BankID,
			Amount: req.BankAmount,
		})
	} else if (req.PaymentMethod == "bank" || req.PaymentMethod == "online" || req.PaymentMethod == "upi") && req.Account.GreaterThan(decimal.Zero) && req.BankID != nil {
		rawBankSplits = append(rawBankSplits, dto.BankPaymentSplitRequest{
			BankID: *req.BankID,
			Amount: req.Account,
		})
	}

	seenBanks := make(map[uuid.UUID]bool)
	validatedBanks := make([]validatedBankPayment, 0, len(rawBankSplits))
	var bankReceived decimal.Decimal

	for _, bp := range rawBankSplits {
		if bp.BankID == uuid.Nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("empty_bank_id")
			}
			return nil, validationErr("bank_payments", "bank account must be selected for each bank payment")
		}
		if seenBanks[bp.BankID] {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("duplicate_bank")
			}
			return nil, validationErr("bank_payments", "duplicate bank account selected in payments")
		}
		seenBanks[bp.BankID] = true

		if bp.Amount.IsNegative() || bp.Amount.IsZero() {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("zero_or_negative_bank_amount")
			}
			return nil, validationErr("bank_payments", "bank payment amount must be greater than zero")
		}

		bank, err := s.bankRepo.GetByID(ctx, tenantID, bp.BankID)
		if err != nil {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("invalid_bank")
			}
			return nil, appErrors.NewBadRequest("invalid bank account specified for counter sale")
		}
		if bank.Status != "active" {
			if s.metrics != nil {
				s.metrics.RecordSalesPaymentFailed("inactive_bank")
			}
			return nil, appErrors.NewBadRequest("selected bank account is not active")
		}

		validatedBanks = append(validatedBanks, validatedBankPayment{
			BankID:   bank.ID,
			BankName: bank.BankName,
			Amount:   bp.Amount,
			Note:     bp.Note,
		})
		bankReceived = bankReceived.Add(bp.Amount)
	}

	collectedAmount := cashReceived.Add(bankReceived)
	if collectedAmount.GreaterThan(req.TotalAmount) {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("collected_exceeds_total")
		}
		return nil, validationErr("collected_amount", "collected amount cannot exceed sale total")
	}

	due := req.TotalAmount.Sub(collectedAmount)
	if due.IsNegative() {
		due = decimal.Zero
	}

	var primaryBankID *uuid.UUID
	if len(validatedBanks) > 0 {
		primaryBankID = &validatedBanks[0].BankID
	}

	method := req.PaymentMethod
	if method == "" {
		if cashReceived.GreaterThan(decimal.Zero) && bankReceived.GreaterThan(decimal.Zero) {
			method = "split"
		} else if bankReceived.GreaterThan(decimal.Zero) {
			method = "bank"
		} else {
			method = "cash"
		}
	}

	sale := &domain.CounterSale{
		TenantID:        tenantID,
		Item:            req.Item,
		Price:           req.Price,
		TotalAmount:     req.TotalAmount,
		PaymentMethod:   method,
		Cash:            cashReceived,
		BankAmount:      bankReceived,
		BankID:          primaryBankID,
		CollectedAmount: collectedAmount,
		Account:         due,
	}

	payments := make([]domain.CounterSalePayment, 0, len(validatedBanks)+1)
	if cashReceived.GreaterThan(decimal.Zero) {
		payments = append(payments, domain.CounterSalePayment{
			TenantID:      tenantID,
			PaymentMethod: "cash",
			Amount:        cashReceived,
			Note:          "Cash payment",
		})
	}
	for _, vb := range validatedBanks {
		note := vb.Note
		if note == "" {
			note = "Bank payment"
		}
		payments = append(payments, domain.CounterSalePayment{
			TenantID:      tenantID,
			PaymentMethod: "bank",
			BankID:        &vb.BankID,
			BankName:      vb.BankName,
			Amount:        vb.Amount,
			Note:          note,
		})
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.countSaleRepo.Create(txCtx, sale, payments); err != nil {
			return err
		}

		for _, vb := range validatedBanks {
			if err := s.bankRepo.AdjustBalance(txCtx, tenantID, vb.BankID, vb.Amount); err != nil {
				return fmt.Errorf("failed to adjust bank balance for bank %s: %w", vb.BankName, err)
			}
			tx := &domain.BankTransaction{
				TenantID:        tenantID,
				BankID:          vb.BankID,
				Amount:          vb.Amount,
				TransactionType: "credit",
				Reason:          "Counter Sale: " + sale.Item,
				SaleType:        "counter_sale",
				SaleID:          &sale.ID,
			}
			if err := s.bankRepo.CreateTransaction(txCtx, tx); err != nil {
				return fmt.Errorf("failed to record bank transaction for bank %s: %w", vb.BankName, err)
			}
		}

		// Update financial summary
		summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
		if err == nil {
			summary.CashBalance = summary.CashBalance.Add(cashReceived)
			summary.BankBalance = summary.BankBalance.Add(bankReceived)
			summary.TotalReceivable = summary.TotalReceivable.Add(due)
			if err := s.summaryRepo.Update(txCtx, summary); err != nil {
				return fmt.Errorf("failed to update financial summary for counter sale: %w", err)
			}
		}

		if _, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString()); err != nil {
			s.logger.WarnContext(txCtx, "failed to sync daily stats after counter sale creation", "error", err.Error())
		}
		return nil
	})
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSalesPaymentFailed("transaction_error")
		}
		return nil, err
	}

	if s.metrics != nil {
		if cashReceived.GreaterThan(decimal.Zero) {
			s.metrics.RecordSalesPayment("cash", cashReceived.InexactFloat64())
		}
		for _, vb := range validatedBanks {
			s.metrics.RecordSalesPayment("bank", vb.Amount.InexactFloat64())
		}
	}

	return sale, nil
}

func (s *TenantOperationsService) UpdateCounterSale(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdateCounterSaleRequest) (*domain.CounterSale, error) {
	sale, err := s.countSaleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, sale.CreatedAt.Format("2006-01-02")); err != nil {
		return nil, err
	}

	oldCash := sale.Cash
	oldBank := sale.BankAmount

	if req.Item != "" {
		sale.Item = req.Item
	}
	if req.Price != nil {
		sale.Price = *req.Price
	}
	if req.TotalAmount != nil {
		sale.TotalAmount = *req.TotalAmount
	}
	if req.PaymentMethod != "" {
		sale.PaymentMethod = req.PaymentMethod
	}
	if req.Cash != nil {
		sale.Cash = *req.Cash
	}
	if req.BankAmount != nil {
		sale.BankAmount = *req.BankAmount
	}
	if req.Account != nil {
		sale.Account = *req.Account
	}

	deltaCash := sale.Cash.Sub(oldCash)
	deltaBank := sale.BankAmount.Sub(oldBank)

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.countSaleRepo.Update(txCtx, sale); err != nil {
			return err
		}

		if s.summaryRepo != nil {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil && summary != nil {
				summary.CashBalance = summary.CashBalance.Add(deltaCash)
				summary.BankBalance = summary.BankBalance.Add(deltaBank)
				_ = s.summaryRepo.Update(txCtx, summary)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}

		if s.statsRepo != nil {
			_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, sale.CreatedAt.Format("2006-01-02"))
			if sale.CreatedAt.Format("2006-01-02") != todayString() {
				_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString())
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return sale, nil
}

func (s *TenantOperationsService) DeleteCounterSale(ctx context.Context, tenantID, id uuid.UUID, role string) error {
	sale, err := s.countSaleRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if err := checkCurrentDayRestriction(role, sale.CreatedAt.Format("2006-01-02")); err != nil {
		return err
	}

	return s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.countSaleRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil && summary != nil {
				summary.CashBalance = summary.CashBalance.Sub(sale.Cash)
				summary.BankBalance = summary.BankBalance.Sub(sale.BankAmount)
				_ = s.summaryRepo.Update(txCtx, summary)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		if s.statsRepo != nil {
			_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, sale.CreatedAt.Format("2006-01-02"))
			if sale.CreatedAt.Format("2006-01-02") != todayString() {
				_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString())
			}
		}
		return nil
	})
}

func (s *TenantOperationsService) GetCounterSaleByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.CounterSale, error) {
	return s.countSaleRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListCounterSales(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, search string) ([]domain.CounterSale, int64, error) {
	if role == auth.RoleTenantUser {
		date = todayString()
	}
	return s.countSaleRepo.List(ctx, tenantID, page, pageSize, date, search)
}

// ==========================================
// 7. Purchase Operations
// ==========================================

func (s *TenantOperationsService) CreatePurchase(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreatePurchaseRequest) (*domain.TenantPurchase, error) {
	if req.TotalAmount.IsNegative() {
		return nil, validationErr("total_amount", "total amount cannot be negative")
	}

	pending := req.TotalAmount.Sub(req.TotalPaid)
	if pending.IsNegative() {
		pending = decimal.Zero
	}

	purchase := &domain.TenantPurchase{
		TenantID:     tenantID,
		Item:         req.Item,
		Quantity:     req.Quantity,
		TotalAmount:  req.TotalAmount,
		TotalPaid:    req.TotalPaid,
		TotalPending: pending,
	}

	if req.CustomerID != nil && *req.CustomerID != uuid.Nil {
		cust, err := s.custRepo.GetByID(ctx, tenantID, *req.CustomerID)
		if err != nil {
			return nil, appErrors.NewBadRequest("invalid customer for tenant")
		}
		if cust.Status != "active" {
			return nil, appErrors.NewBadRequest("customer is not active")
		}
		purchase.CustomerID = &cust.ID
		purchase.CustomerName = cust.CustomerName
	}

	payments := make([]domain.TenantPurchasePayment, len(req.Payments))
	var cashPaid, bankPaid decimal.Decimal
	for i, p := range req.Payments {
		if p.PaymentMethod == "cash" {
			cashPaid = cashPaid.Add(p.Amount)
		} else {
			bankPaid = bankPaid.Add(p.Amount)
		}
		if p.BankID != nil && *p.BankID != uuid.Nil {
			bank, err := s.bankRepo.GetByID(ctx, tenantID, *p.BankID)
			if err != nil {
				return nil, appErrors.NewBadRequest("invalid bank account specified for purchase")
			}
			p.BankName = bank.BankName
		}
		payments[i] = domain.TenantPurchasePayment{
			TenantID:      tenantID,
			PaymentMethod: p.PaymentMethod,
			BankID:        p.BankID,
			BankName:      p.BankName,
			Amount:        p.Amount,
			Note:          p.Note,
		}
	}

	if len(payments) == 0 && req.TotalPaid.GreaterThan(decimal.Zero) {
		cashPaid = req.TotalPaid
		payments = append(payments, domain.TenantPurchasePayment{
			TenantID:      tenantID,
			PaymentMethod: "cash",
			Amount:        req.TotalPaid,
		})
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.purchRepo.Create(txCtx, purchase, payments); err != nil {
			return err
		}

		// Financial summary: paid reduces cash/bank, pending adds to payable
		summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
		if err == nil {
			summary.CashBalance = summary.CashBalance.Sub(cashPaid)
			summary.BankBalance = summary.BankBalance.Sub(bankPaid)
			summary.TotalPayable = summary.TotalPayable.Add(pending)
			if err := s.summaryRepo.Update(txCtx, summary); err != nil {
				return fmt.Errorf("failed to update financial summary for purchase: %w", err)
			}
		}

		if _, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString()); err != nil {
			s.logger.WarnContext(txCtx, "failed to sync daily stats after purchase creation", "error", err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return purchase, nil
}

func (s *TenantOperationsService) UpdatePurchase(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdatePurchaseRequest) (*domain.TenantPurchase, error) {
	purchase, err := s.purchRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, purchase.CreatedAt.Format("2006-01-02")); err != nil {
		return nil, err
	}

	oldPaid := purchase.TotalPaid
	oldPending := purchase.TotalPending

	if req.Item != "" {
		purchase.Item = req.Item
	}
	if req.Quantity != nil {
		purchase.Quantity = *req.Quantity
	}
	if req.TotalAmount != nil {
		purchase.TotalAmount = *req.TotalAmount
		purchase.TotalPending = purchase.TotalAmount.Sub(purchase.TotalPaid)
	}
	if req.TotalPaid != nil {
		purchase.TotalPaid = *req.TotalPaid
		purchase.TotalPending = purchase.TotalAmount.Sub(purchase.TotalPaid)
	}

	if req.CustomerID != nil {
		if *req.CustomerID == uuid.Nil {
			purchase.CustomerID = nil
			purchase.CustomerName = ""
		} else {
			cust, err := s.custRepo.GetByID(ctx, tenantID, *req.CustomerID)
			if err != nil {
				return nil, appErrors.NewBadRequest("invalid customer for tenant")
			}
			if cust.Status != "active" {
				return nil, appErrors.NewBadRequest("customer is not active")
			}
			purchase.CustomerID = &cust.ID
			purchase.CustomerName = cust.CustomerName
		}
	}

	deltaPaid := purchase.TotalPaid.Sub(oldPaid)
	deltaPending := purchase.TotalPending.Sub(oldPending)

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.purchRepo.Update(txCtx, purchase); err != nil {
			return err
		}

		if s.summaryRepo != nil {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil && summary != nil {
				summary.CashBalance = summary.CashBalance.Sub(deltaPaid)
				summary.TotalPayable = summary.TotalPayable.Add(deltaPending)
				_ = s.summaryRepo.Update(txCtx, summary)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}

		if s.statsRepo != nil {
			_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, purchase.CreatedAt.Format("2006-01-02"))
			if purchase.CreatedAt.Format("2006-01-02") != todayString() {
				_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString())
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return purchase, nil
}

func (s *TenantOperationsService) DeletePurchase(ctx context.Context, tenantID, id uuid.UUID, role string) error {
	purchase, err := s.purchRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if err := checkCurrentDayRestriction(role, purchase.CreatedAt.Format("2006-01-02")); err != nil {
		return err
	}

	return s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.purchRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil && summary != nil {
				summary.CashBalance = summary.CashBalance.Add(purchase.TotalPaid)
				summary.TotalPayable = summary.TotalPayable.Sub(purchase.TotalPending)
				_ = s.summaryRepo.Update(txCtx, summary)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		if s.statsRepo != nil {
			_, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, purchase.CreatedAt.Format("2006-01-02"))
			if err != nil {
				s.logger.WarnContext(txCtx, "failed to sync daily stats after purchase deletion", "error", err.Error())
			}
			if purchase.CreatedAt.Format("2006-01-02") != todayString() {
				_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString())
			}
		}
		return nil
	})
}

func (s *TenantOperationsService) GetPurchaseByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error) {
	return s.purchRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListPurchases(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, search string) ([]domain.TenantPurchase, int64, error) {
	if role == auth.RoleTenantUser {
		date = todayString()
	}
	return s.purchRepo.List(ctx, tenantID, page, pageSize, date, search)
}

// ==========================================
// 8. Expense Operations
// ==========================================

func (s *TenantOperationsService) CreateExpense(ctx context.Context, tenantID uuid.UUID, role string, req *dto.CreateExpenseRequest) (*domain.TenantExpense, error) {
	if req.TotalAmount.IsNegative() {
		return nil, validationErr("total_amount", "total amount cannot be negative")
	}

	expense := &domain.TenantExpense{
		TenantID:    tenantID,
		Item:        req.Item,
		TotalAmount: req.TotalAmount,
	}

	payments := make([]domain.TenantExpensePayment, len(req.Payments))
	var cashPaid, bankPaid decimal.Decimal
	for i, p := range req.Payments {
		if p.PaymentMethod == "cash" {
			cashPaid = cashPaid.Add(p.Amount)
		} else {
			bankPaid = bankPaid.Add(p.Amount)
		}
		if p.BankID != nil && *p.BankID != uuid.Nil {
			bank, err := s.bankRepo.GetByID(ctx, tenantID, *p.BankID)
			if err != nil {
				return nil, appErrors.NewBadRequest("invalid bank account specified for expense")
			}
			p.BankName = bank.BankName
		}
		payments[i] = domain.TenantExpensePayment{
			TenantID:      tenantID,
			PaymentMethod: p.PaymentMethod,
			BankID:        p.BankID,
			BankName:      p.BankName,
			Amount:        p.Amount,
			Note:          p.Note,
		}
	}

	if len(payments) == 0 && req.TotalAmount.GreaterThan(decimal.Zero) {
		pm := req.PaymentMethod
		if pm == "" {
			pm = "cash"
		}
		if pm == "cash" {
			cashPaid = req.TotalAmount
		} else {
			bankPaid = req.TotalAmount
		}
		payments = append(payments, domain.TenantExpensePayment{
			TenantID:      tenantID,
			PaymentMethod: pm,
			BankID:        req.BankID,
			BankName:      req.BankName,
			Amount:        req.TotalAmount,
		})
	}

	err := s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.expRepo.Create(txCtx, expense, payments); err != nil {
			return err
		}

		summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
		if err == nil {
			summary.CashBalance = summary.CashBalance.Sub(cashPaid)
			summary.BankBalance = summary.BankBalance.Sub(bankPaid)
			if err := s.summaryRepo.Update(txCtx, summary); err != nil {
				return fmt.Errorf("failed to update financial summary for expense: %w", err)
			}
		}

		if _, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString()); err != nil {
			s.logger.WarnContext(txCtx, "failed to sync daily stats after expense creation", "error", err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return expense, nil
}

func (s *TenantOperationsService) UpdateExpense(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdateExpenseRequest) (*domain.TenantExpense, error) {
	exp, err := s.expRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, exp.CreatedAt.Format("2006-01-02")); err != nil {
		return nil, err
	}

	oldAmount := exp.TotalAmount

	if req.Item != "" {
		exp.Item = req.Item
	}
	if req.TotalAmount != nil {
		exp.TotalAmount = *req.TotalAmount
	}

	deltaAmount := exp.TotalAmount.Sub(oldAmount)

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.expRepo.Update(txCtx, exp); err != nil {
			return err
		}

		if s.summaryRepo != nil {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil && summary != nil {
				summary.CashBalance = summary.CashBalance.Sub(deltaAmount)
				_ = s.summaryRepo.Update(txCtx, summary)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}

		if s.statsRepo != nil {
			_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, exp.CreatedAt.Format("2006-01-02"))
			if exp.CreatedAt.Format("2006-01-02") != todayString() {
				_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString())
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return exp, nil
}

func (s *TenantOperationsService) DeleteExpense(ctx context.Context, tenantID, id uuid.UUID, role string) error {
	exp, err := s.expRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}

	if err := checkCurrentDayRestriction(role, exp.CreatedAt.Format("2006-01-02")); err != nil {
		return err
	}

	return s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.expRepo.Delete(txCtx, tenantID, id); err != nil {
			return err
		}
		if s.summaryRepo != nil {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil && summary != nil {
				summary.CashBalance = summary.CashBalance.Add(exp.TotalAmount)
				_ = s.summaryRepo.Update(txCtx, summary)
			}
			_, _ = s.summaryRepo.SyncFromSourceRecords(txCtx, tenantID)
		}
		if s.statsRepo != nil {
			_, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, exp.CreatedAt.Format("2006-01-02"))
			if err != nil {
				s.logger.WarnContext(txCtx, "failed to sync daily stats after expense deletion", "error", err.Error())
			}
			if exp.CreatedAt.Format("2006-01-02") != todayString() {
				_, _ = s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString())
			}
		}
		return nil
	})
}

func (s *TenantOperationsService) GetExpenseByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantExpense, error) {
	return s.expRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListExpenses(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, search string) ([]domain.TenantExpense, int64, error) {
	if role == auth.RoleTenantUser {
		date = todayString()
	}
	return s.expRepo.List(ctx, tenantID, page, pageSize, date, search)
}

// ==========================================
// 9. Attendance & Overtime & Advances
// ==========================================

func (s *TenantOperationsService) RecordAttendance(ctx context.Context, tenantID uuid.UUID, role string, req *dto.RecordAttendanceRequest) (*domain.Attendance, error) {
	emp, err := s.empRepo.GetByID(ctx, tenantID, req.EmployeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	attDate := req.Date
	if attDate == "" {
		attDate = todayString()
	}

	if err := checkCurrentDayRestriction(role, attDate); err != nil {
		return nil, err
	}

	// Fetch existing record to calculate salary and advance deltas, ensuring idempotent updates
	existing, err := s.attRepo.GetByEmployeeAndDate(ctx, tenantID, req.EmployeeID, attDate)
	if err != nil {
		return nil, err
	}

	var oldDailySalary decimal.Decimal
	var oldAdvance decimal.Decimal
	if existing != nil {
		oldDailySalary = existing.DailySalary
		oldAdvance = existing.Advance
	}

	var newDailySalary decimal.Decimal
	if req.Status == "present" {
		newDailySalary = emp.Salary
	} else if req.Status == "half_day" {
		newDailySalary = emp.Salary.Div(decimal.NewFromInt(2))
	} else {
		newDailySalary = decimal.Zero // absent, leave
	}

	salaryDelta := newDailySalary.Sub(oldDailySalary)
	advanceDelta := req.Advance.Sub(oldAdvance)

	att := &domain.Attendance{
		TenantID:    tenantID,
		EmployeeID:  req.EmployeeID,
		Date:        attDate,
		Status:      req.Status,
		DailySalary: newDailySalary,
		OT:          req.OT,
		Advance:     req.Advance,
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.attRepo.Upsert(txCtx, att); err != nil {
			return err
		}

		// Credit/debit employee salary balance by salary delta (avoids crediting absent, handles status change/retry idempotently)
		if !salaryDelta.IsZero() {
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, salaryDelta); err != nil {
				return fmt.Errorf("failed to adjust employee salary balance for attendance: %w", err)
			}
		}

		// If advance delta changed, adjust cash balance and employee salary balance
		if !advanceDelta.IsZero() {
			summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
			if err == nil {
				summary.CashBalance = summary.CashBalance.Sub(advanceDelta)
				if err := s.summaryRepo.Update(txCtx, summary); err != nil {
					return fmt.Errorf("failed to update financial summary for attendance advance: %w", err)
				}
			}
			// Deduct advance from employee salary balance
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, advanceDelta.Neg()); err != nil {
				return fmt.Errorf("failed to adjust employee salary balance for advance: %w", err)
			}
		}

		if _, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, attDate); err != nil {
			s.logger.WarnContext(txCtx, "failed to sync daily stats after attendance record", "error", err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return att, nil
}

func (s *TenantOperationsService) RecordOvertime(ctx context.Context, tenantID uuid.UUID, role string, req *dto.RecordOvertimeRequest) (*domain.Attendance, error) {
	emp, err := s.empRepo.GetByID(ctx, tenantID, req.EmployeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	date := req.Date
	if date == "" {
		date = todayString()
	}

	if err := checkCurrentDayRestriction(role, date); err != nil {
		return nil, err
	}

	var updatedAtt *domain.Attendance
	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.attRepo.GetByEmployeeAndDate(txCtx, tenantID, req.EmployeeID, date)
		if err != nil {
			return err
		}

		if existing == nil {
			existing = &domain.Attendance{
				TenantID:   tenantID,
				EmployeeID: req.EmployeeID,
				Date:       date,
				Status:     "present",
				OT:         req.OT,
				Advance:    decimal.Zero,
			}
		} else {
			existing.OT = existing.OT.Add(req.OT)
		}

		if err := s.attRepo.Upsert(txCtx, existing); err != nil {
			return err
		}
		updatedAtt = existing

		// Add OT wage to employee salary balance: ot_hours * ot_rate
		if emp.OTRate.GreaterThan(decimal.Zero) {
			otWages := req.OT.Mul(emp.OTRate)
			if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, otWages); err != nil {
				return err
			}
		}

		if _, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, date); err != nil {
			s.logger.WarnContext(txCtx, "failed to sync daily stats after overtime record", "error", err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedAtt, nil
}

func (s *TenantOperationsService) RecordAdvance(ctx context.Context, tenantID uuid.UUID, role string, req *dto.RecordAdvanceRequest) (*domain.Attendance, error) {
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		return nil, validationErr("amount", "advance amount must be greater than zero")
	}

	_, err := s.empRepo.GetByID(ctx, tenantID, req.EmployeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	date := req.Date
	if date == "" {
		date = todayString()
	}

	if err := checkCurrentDayRestriction(role, date); err != nil {
		return nil, err
	}

	var updatedAtt *domain.Attendance
	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.attRepo.GetByEmployeeAndDate(txCtx, tenantID, req.EmployeeID, date)
		if err != nil {
			return err
		}

		if existing == nil {
			existing = &domain.Attendance{
				TenantID:   tenantID,
				EmployeeID: req.EmployeeID,
				Date:       date,
				Status:     "present",
				OT:         decimal.Zero,
				Advance:    req.Amount,
			}
		} else {
			existing.Advance = existing.Advance.Add(req.Amount)
		}

		if err := s.attRepo.Upsert(txCtx, existing); err != nil {
			return err
		}
		updatedAtt = existing

		// Update financial summary: deduct cash (or bank)
		summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
		if err == nil {
			if req.PaymentMethod == "bank" {
				summary.BankBalance = summary.BankBalance.Sub(req.Amount)
			} else {
				summary.CashBalance = summary.CashBalance.Sub(req.Amount)
			}
			if err := s.summaryRepo.Update(txCtx, summary); err != nil {
				return fmt.Errorf("failed to update financial summary for advance: %w", err)
			}
		}

		// Deduct advance from employee salary balance
		if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, req.EmployeeID, req.Amount.Neg()); err != nil {
			return fmt.Errorf("failed to adjust employee salary balance for advance: %w", err)
		}

		if _, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, date); err != nil {
			s.logger.WarnContext(txCtx, "failed to sync daily stats after advance record", "error", err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedAtt, nil
}

func (s *TenantOperationsService) UpdateAttendance(ctx context.Context, tenantID, id uuid.UUID, role string, req *dto.UpdateAttendanceRequest) (*domain.Attendance, error) {
	att, err := s.attRepo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}

	if err := checkCurrentDayRestriction(role, att.Date); err != nil {
		return nil, err
	}

	if req.Status != "" {
		att.Status = req.Status
	}
	if req.OT != nil {
		att.OT = *req.OT
	}
	if req.Advance != nil {
		att.Advance = *req.Advance
	}

	if err := s.attRepo.Update(ctx, att); err != nil {
		return nil, err
	}

	_, _ = s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, att.Date)
	return att, nil
}

func (s *TenantOperationsService) GetAttendanceByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Attendance, error) {
	return s.attRepo.GetByID(ctx, tenantID, id)
}

func (s *TenantOperationsService) ListAttendance(ctx context.Context, tenantID uuid.UUID, role string, page, pageSize int, date string, employeeID *uuid.UUID) ([]domain.Attendance, int64, error) {
	if role == auth.RoleTenantUser && date == "" {
		date = todayString()
	}
	return s.attRepo.List(ctx, tenantID, page, pageSize, date, employeeID)
}

// ==========================================
// 10. Salary Operations (Admin only)
// ==========================================

func (s *TenantOperationsService) ListSalaries(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search string) ([]domain.EmployeeSalary, int64, error) {
	return s.salaryRepo.List(ctx, tenantID, page, pageSize, search)
}

func (s *TenantOperationsService) ListPendingSalaries(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.EmployeeSalary, int64, error) {
	return s.salaryRepo.ListPending(ctx, tenantID, page, pageSize)
}

func (s *TenantOperationsService) GetSalaryByEmployeeID(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalary, error) {
	return s.salaryRepo.GetByEmployeeID(ctx, tenantID, employeeID)
}

func (s *TenantOperationsService) UpdateSalaryBalance(ctx context.Context, tenantID, employeeID uuid.UUID, req *dto.UpdateSalaryBalanceRequest) (*domain.EmployeeSalary, error) {
	_, err := s.empRepo.GetByID(ctx, tenantID, employeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	salary := &domain.EmployeeSalary{
		TenantID:   tenantID,
		EmployeeID: employeeID,
		Balance:    req.Balance,
	}

	if err := s.salaryRepo.UpsertBalance(ctx, salary); err != nil {
		return nil, err
	}

	return s.salaryRepo.GetByEmployeeID(ctx, tenantID, employeeID)
}

func (s *TenantOperationsService) PaySalary(ctx context.Context, tenantID, employeeID uuid.UUID, req *dto.PaySalaryRequest) (*domain.EmployeeSalaryPayment, error) {
	if req.Amount.IsNegative() || req.Amount.IsZero() {
		return nil, validationErr("amount", "salary payment amount must be greater than zero")
	}

	emp, err := s.empRepo.GetByID(ctx, tenantID, employeeID)
	if err != nil {
		return nil, appErrors.NewBadRequest("invalid employee for tenant")
	}

	if req.BankID != nil && *req.BankID != uuid.Nil {
		bank, err := s.bankRepo.GetByID(ctx, tenantID, *req.BankID)
		if err != nil {
			return nil, appErrors.NewBadRequest("invalid bank account specified for salary disbursement")
		}
		req.BankName = bank.BankName
	}

	payment := &domain.EmployeeSalaryPayment{
		TenantID:      tenantID,
		EmployeeID:    employeeID,
		EmployeeName:  emp.Name,
		PaymentMethod: req.PaymentMethod,
		BankID:        req.BankID,
		BankName:      req.BankName,
		Amount:        req.Amount,
		Note:          req.Note,
	}

	err = s.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		if err := s.salaryRepo.CreatePayment(txCtx, payment); err != nil {
			return err
		}

		// Deduct payment amount from employee salary balance
		if err := s.salaryRepo.AdjustBalance(txCtx, tenantID, employeeID, req.Amount.Neg()); err != nil {
			return err
		}

		// Update financial summary
		summary, err := s.summaryRepo.GetByTenantIDForUpdate(txCtx, tenantID)
		if err == nil {
			if req.PaymentMethod == "cash" {
				summary.CashBalance = summary.CashBalance.Sub(req.Amount)
			} else {
				summary.BankBalance = summary.BankBalance.Sub(req.Amount)
			}
			if err := s.summaryRepo.Update(txCtx, summary); err != nil {
				return fmt.Errorf("failed to update financial summary for salary payment: %w", err)
			}
		}

		if _, err := s.statsRepo.ComputeAndSyncDailyStats(txCtx, tenantID, todayString()); err != nil {
			s.logger.WarnContext(txCtx, "failed to sync daily stats after salary payment", "error", err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return payment, nil
}

func (s *TenantOperationsService) ListSalaryPayments(ctx context.Context, tenantID uuid.UUID, employeeID *uuid.UUID, page, pageSize int) ([]domain.EmployeeSalaryPayment, int64, error) {
	return s.salaryRepo.ListPayments(ctx, tenantID, employeeID, page, pageSize)
}

func hasStatsActivity(s *domain.TenantDailyStats) bool {
	if s == nil {
		return false
	}
	return s.TotalSales.GreaterThan(decimal.Zero) ||
		s.PurchaseAmount.GreaterThan(decimal.Zero) ||
		s.ExpenseAmount.GreaterThan(decimal.Zero) ||
		s.AttendancePresent > 0 ||
		s.AttendanceAbsent > 0 ||
		s.AdvanceAmount.GreaterThan(decimal.Zero) ||
		s.WagesAmount.GreaterThan(decimal.Zero) ||
		s.AmountReceived.GreaterThan(decimal.Zero) ||
		s.AmountPaid.GreaterThan(decimal.Zero)
}

func calculateDaysOld(requestedDate, dataDate string) int {
	if requestedDate == dataDate || requestedDate == "" || dataDate == "" {
		return 0
	}
	reqTime, err1 := time.Parse("2006-01-02", requestedDate)
	dateTime, err2 := time.Parse("2006-01-02", dataDate)
	if err1 != nil || err2 != nil {
		return 0
	}
	days := int(reqTime.Sub(dateTime).Hours() / 24)
	if days < 1 {
		days = 1
	}
	return days
}

// ==========================================
// 11. Financial Summary & Daily Statistics
// ==========================================

func (s *TenantOperationsService) GetDailyStats(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	if s.statsRepo == nil {
		return &domain.TenantDailyStats{TenantID: tenantID}, nil
	}
	today := todayString()
	isExplicitHistorical := date != "" && date != today

	if isExplicitHistorical {
		stats, err := s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, date)
		if err != nil {
			return nil, err
		}
		if stats != nil {
			stats.RequestedDate = date
			stats.DataDate = date
			isCur := false
			stats.IsCurrent = &isCur
			stats.DaysOld = calculateDaysOld(today, date)
		}
		return stats, nil
	}

	// For current business day:
	todayStats, err := s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, today)
	if err != nil {
		return nil, err
	}

	if hasStatsActivity(todayStats) {
		todayStats.RequestedDate = today
		todayStats.DataDate = today
		isCur := true
		todayStats.IsCurrent = &isCur
		todayStats.DaysOld = 0
		return todayStats, nil
	}

	// If no activity today, look for latest available previous-day data
	prevStats, err := s.statsRepo.GetLatestAvailable(ctx, tenantID, today)
	if err == nil && prevStats != nil && prevStats.Date != "" {
		prevStats.RequestedDate = today
		prevStats.DataDate = prevStats.Date
		isCur := false
		prevStats.IsCurrent = &isCur
		prevStats.DaysOld = calculateDaysOld(today, prevStats.Date)
		return prevStats, nil
	}

	// Fallback to today's (zeroed) stats if no prior data exists
	todayStats.RequestedDate = today
	todayStats.DataDate = today
	isCur := true
	todayStats.IsCurrent = &isCur
	todayStats.DaysOld = 0
	return todayStats, nil
}

func (s *TenantOperationsService) GetFinancialMetrics(ctx context.Context, tenantID uuid.UUID) (*domain.FinancialMetrics, error) {
	if s.summaryRepo == nil {
		todayOverview, _ := s.GetTodayOverview(ctx, tenantID)
		return &domain.FinancialMetrics{
			TodayOverview: todayOverview,
		}, nil
	}
	var summary *domain.TenantFinancialSummary
	var err error
	if s.summaryRepo != nil {
		summary, err = s.summaryRepo.SyncFromSourceRecords(ctx, tenantID)
		if err != nil {
			summary, err = s.summaryRepo.GetByTenantID(ctx, tenantID)
		}
	}
	if err != nil || summary == nil {
		if err == nil || appErrors.IsNotFound(err) {
			summary = &domain.TenantFinancialSummary{
				TenantID:        tenantID,
				CashBalance:     decimal.Zero,
				BankBalance:     decimal.Zero,
				TotalReceivable: decimal.Zero,
				TotalPayable:    decimal.Zero,
			}
		} else {
			return nil, err
		}
	}

	// Calculate pending salaries directly from employee_salary table
	var totalPendingSalary decimal.Decimal
	if s.salaryRepo != nil {
		pendingSalaries, _, _ := s.salaryRepo.ListPending(ctx, tenantID, 1, 500)
		for _, ps := range pendingSalaries {
			if ps.Balance.GreaterThan(decimal.Zero) {
				totalPendingSalary = totalPendingSalary.Add(ps.Balance)
			}
		}
	}

	today := todayString()
	var todayStats *domain.TenantDailyStats
	if s.statsRepo != nil {
		todayStats, _ = s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, today)
	}

	requestedDate := today
	dataDate := today
	isCurrent := true
	daysOld := 0
	activeStats := todayStats

	if hasStatsActivity(todayStats) {
		if todayStats != nil {
			todayStats.RequestedDate = today
			todayStats.DataDate = today
			isCur := true
			todayStats.IsCurrent = &isCur
			todayStats.DaysOld = 0
		}
	} else if s.statsRepo != nil {
		prevStats, err := s.statsRepo.GetLatestAvailable(ctx, tenantID, today)
		if err == nil && prevStats != nil && prevStats.Date != "" {
			activeStats = prevStats
			dataDate = prevStats.Date
			isCurrent = false
			daysOld = calculateDaysOld(today, prevStats.Date)
			isCur := false
			prevStats.RequestedDate = today
			prevStats.DataDate = prevStats.Date
			prevStats.IsCurrent = &isCur
			prevStats.DaysOld = daysOld
		} else if todayStats != nil {
			todayStats.RequestedDate = today
			todayStats.DataDate = today
			isCur := true
			todayStats.IsCurrent = &isCur
			todayStats.DaysOld = 0
		}
	}

	// Net dues is total payable on purchases + pending employee salaries
	netDues := summary.TotalPayable.Add(totalPendingSalary)

	// Fetch individual bank balances for dashboard/reporting
	var activeBanks []domain.TenantBank
	if s.bankRepo != nil {
		activeBanks, _, _ = s.bankRepo.List(ctx, tenantID, 1, 100, "", "active")
	}

	// Calculate today's earned employee salary from attendance
	var todaySalaryEarned decimal.Decimal
	if s.attRepo != nil {
		todaySalaryEarned, _ = s.attRepo.GetTodaySalaryEarned(ctx, tenantID, today)
	}

	todayOverview, _ := s.GetTodayOverview(ctx, tenantID)

	return &domain.FinancialMetrics{
		CashBalance:         summary.CashBalance,
		BankBalance:         summary.BankBalance,
		TotalReceivable:     summary.TotalReceivable,
		TotalPayable:        summary.TotalPayable,
		NetDues:             netDues,
		NetReceivables:      summary.TotalReceivable,
		PendingSalary:       totalPendingSalary,
		TodayEmployeeSalary: todaySalaryEarned,
		BankBalances:        activeBanks,
		TodayStats:          activeStats,
		TodayOverview:       todayOverview,
		RequestedDate:       requestedDate,
		DataDate:            dataDate,
		IsCurrent:           isCurrent,
		DaysOld:             daysOld,
	}, nil
}

// GetTodayOverview aggregates the current day's key business and staff metrics
func (s *TenantOperationsService) GetTodayOverview(ctx context.Context, tenantID uuid.UUID) (*domain.TodayOverview, error) {
	today := todayString()

	var todayStats *domain.TenantDailyStats
	if s.statsRepo != nil {
		todayStats, _ = s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, today)
	}
	if todayStats == nil {
		todayStats = &domain.TenantDailyStats{
			TenantID:          tenantID,
			Date:              today,
			LineSaleAmount:    decimal.Zero,
			CounterSaleAmount: decimal.Zero,
			TotalSales:        decimal.Zero,
			PurchaseAmount:    decimal.Zero,
			ExpenseAmount:     decimal.Zero,
			AdvanceAmount:     decimal.Zero,
		}
	}

	// 1. Staff Attendance & Total Employee Count
	totalStaff := 0
	if s.empRepo != nil {
		_, activeCount, err := s.empRepo.List(ctx, tenantID, 1, 1, "", "active")
		if err == nil && activeCount > 0 {
			totalStaff = int(activeCount)
		} else {
			_, allCount, err := s.empRepo.List(ctx, tenantID, 1, 1, "", "")
			if err == nil {
				totalStaff = int(allCount)
			}
		}
	}

	recordedTotal := todayStats.AttendancePresent + todayStats.AttendanceAbsent
	if recordedTotal > totalStaff {
		totalStaff = recordedTotal
	}
	hasRecords := recordedTotal > 0

	attOverview := domain.TodayAttendanceOverview{
		Present:    todayStats.AttendancePresent,
		Total:      totalStaff,
		Absent:     todayStats.AttendanceAbsent,
		HasRecords: hasRecords,
	}

	// 2. Line Sale (today's transactions only)
	lineSale := todayStats.LineSaleAmount

	// 3. Counter Sale (today's transactions only)
	counterSale := todayStats.CounterSaleAmount

	// 4. Employee Total Advance (today's records only)
	empAdvance := todayStats.AdvanceAmount

	// 5. Current Item (from existing inventory/purchase data or fallback to counter sales)
	var currentItem *domain.CurrentItemOverview
	if s.purchRepo != nil {
		// First check if an inventory purchase occurred today
		todayPurchases, _, err := s.purchRepo.List(ctx, tenantID, 1, 1, today, "")
		if err == nil && len(todayPurchases) > 0 {
			p := todayPurchases[0]
			currentItem = &domain.CurrentItemOverview{
				Name:         p.Item,
				Quantity:     p.Quantity,
				TotalAmount:  p.TotalAmount,
				TotalPaid:    p.TotalPaid,
				TotalPending: p.TotalPending,
				Date:         p.CreatedAt.Format("2006-01-02"),
				IsToday:      true,
				Source:       "purchase",
			}
		} else {
			// Check latest available inventory procurement
			allPurchases, _, err := s.purchRepo.List(ctx, tenantID, 1, 1, "", "")
			if err == nil && len(allPurchases) > 0 {
				p := allPurchases[0]
				isToday := p.CreatedAt.Format("2006-01-02") == today
				currentItem = &domain.CurrentItemOverview{
					Name:         p.Item,
					Quantity:     p.Quantity,
					TotalAmount:  p.TotalAmount,
					TotalPaid:    p.TotalPaid,
					TotalPending: p.TotalPending,
					Date:         p.CreatedAt.Format("2006-01-02"),
					IsToday:      isToday,
					Source:       "purchase",
				}
			}
		}
	}

	// Fallback to counter sale item if no purchase item exists
	if currentItem == nil && s.countSaleRepo != nil {
		todaySales, _, err := s.countSaleRepo.List(ctx, tenantID, 1, 1, today, "")
		if err == nil && len(todaySales) > 0 {
			cs := todaySales[0]
			currentItem = &domain.CurrentItemOverview{
				Name:         cs.Item,
				Quantity:     1,
				TotalAmount:  cs.TotalAmount,
				TotalPaid:    cs.CollectedAmount,
				TotalPending: cs.Account,
				Date:         cs.CreatedAt.Format("2006-01-02"),
				IsToday:      true,
				Source:       "counter_sale",
			}
		} else {
			allSales, _, err := s.countSaleRepo.List(ctx, tenantID, 1, 1, "", "")
			if err == nil && len(allSales) > 0 {
				cs := allSales[0]
				isToday := cs.CreatedAt.Format("2006-01-02") == today
				currentItem = &domain.CurrentItemOverview{
					Name:         cs.Item,
					Quantity:     1,
					TotalAmount:  cs.TotalAmount,
					TotalPaid:    cs.CollectedAmount,
					TotalPending: cs.Account,
					Date:         cs.CreatedAt.Format("2006-01-02"),
					IsToday:      isToday,
					Source:       "counter_sale",
				}
			}
		}
	}

	var currentItemVal any = struct{}{}
	if currentItem != nil {
		currentItemVal = currentItem
	}

	return &domain.TodayOverview{
		Attendance:      attOverview,
		LineSale:        lineSale,
		CounterSale:     counterSale,
		EmployeeAdvance: empAdvance,
		CurrentItem:     currentItemVal,
		Date:            today,
	}, nil
}
