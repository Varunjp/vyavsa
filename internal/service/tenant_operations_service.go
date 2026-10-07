package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
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
	userRepo         repository.TenantUserRepository
	empRepo          repository.TenantEmployeeRepository
	custRepo         repository.TenantCustomerRepository
	bankRepo         repository.TenantBankRepository
	lineSaleRepo     repository.LineSaleRepository
	countSaleRepo    repository.CounterSaleRepository
	purchRepo        repository.TenantPurchaseRepository
	expRepo          repository.TenantExpenseRepository
	attRepo          repository.AttendanceRepository
	advRepo          repository.EmployeeAdvanceRepository
	otRepo           repository.EmployeeOvertimeRepository
	salaryRepo       repository.EmployeeSalaryRepository
	statsRepo        repository.TenantDailyStatsRepository
	summaryRepo      repository.TenantFinancialSummaryRepository
	transactor       repository.Transactor
	hasher           auth.PasswordHasher
	metrics          *metrics.Metrics
	location         *time.Location
	logger           *slog.Logger
	dailyStatsWorker DailyStatsEnqueuer
}

// DailyStatsEnqueuer defines an interface for enqueueing daily statistics updates asynchronously
type DailyStatsEnqueuer interface {
	Enqueue(ctx context.Context, tenantID uuid.UUID, date string) error
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
	advRepo repository.EmployeeAdvanceRepository,
	otRepo repository.EmployeeOvertimeRepository,
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
		advRepo:       advRepo,
		otRepo:        otRepo,
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

// SetDailyStatsWorker attaches an asynchronous background worker for daily stats calculations
func (s *TenantOperationsService) SetDailyStatsWorker(w DailyStatsEnqueuer) {
	s.dailyStatsWorker = w
}

func (s *TenantOperationsService) enqueueDailyStats(ctx context.Context, tenantID uuid.UUID, date string) {
	if s.dailyStatsWorker != nil {
		if err := s.dailyStatsWorker.Enqueue(ctx, tenantID, date); err != nil {
			s.logger.WarnContext(ctx, "failed to enqueue daily stats update",
				slog.String("tenant_id", tenantID.String()),
				slog.String("date", date),
				slog.String("error", err.Error()),
			)
		}
		return
	}

	if s.statsRepo != nil {
		if _, err := s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, date); err != nil {
			s.logger.WarnContext(ctx, "failed to compute daily stats synchronously",
				slog.String("tenant_id", tenantID.String()),
				slog.String("date", date),
				slog.String("error", err.Error()),
			)
		}
	}
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
