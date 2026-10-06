package repository

import (
	"context"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TenantEmployeeRepository defines persistence contracts for employees
type TenantEmployeeRepository interface {
	Create(ctx context.Context, emp *domain.TenantEmployee) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantEmployee, error)
	Update(ctx context.Context, emp *domain.TenantEmployee) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantEmployee, int64, error)
}

// TenantCustomerRepository defines persistence contracts for customers
type TenantCustomerRepository interface {
	Create(ctx context.Context, cust *domain.TenantCustomer) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantCustomer, error)
	GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantCustomer, error)
	Update(ctx context.Context, cust *domain.TenantCustomer) error
	AdjustBalance(ctx context.Context, tenantID, id uuid.UUID, delta decimal.Decimal) error
	SetBalance(ctx context.Context, tenantID, id uuid.UUID, balance decimal.Decimal) error
	RecordAdjustment(ctx context.Context, adj *domain.CustomerBalanceAdjustment) error
	ListAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, page, pageSize int) ([]domain.CustomerBalanceAdjustment, int64, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantCustomer, int64, error)
}

// TenantBankRepository defines persistence contracts for tenant bank accounts
type TenantBankRepository interface {
	Create(ctx context.Context, bank *domain.TenantBank) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantBank, error)
	Update(ctx context.Context, bank *domain.TenantBank) error
	AdjustBalance(ctx context.Context, tenantID, id uuid.UUID, delta decimal.Decimal) error
	CreateTransaction(ctx context.Context, tx *domain.BankTransaction) error
	ListTransactions(ctx context.Context, tenantID, bankID uuid.UUID, page, pageSize int) ([]domain.BankTransaction, int64, error)
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantBank, int64, error)
}

// LineSaleRepository defines persistence contracts for line sales
type LineSaleRepository interface {
	Create(ctx context.Context, sale *domain.LineSale, payments []domain.LineSalePayment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error)
	GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error)
	Update(ctx context.Context, sale *domain.LineSale) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.LineSale, int64, error)
	CreatePayment(ctx context.Context, payment *domain.LineSalePayment) error
	ListPaymentsByLineSaleID(ctx context.Context, tenantID, lineSaleID uuid.UUID) ([]domain.LineSalePayment, error)
}

// CounterSaleRepository defines persistence contracts for counter sales
type CounterSaleRepository interface {
	Create(ctx context.Context, sale *domain.CounterSale, payments []domain.CounterSalePayment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.CounterSale, error)
	GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.CounterSale, error)
	Update(ctx context.Context, sale *domain.CounterSale) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, search string) ([]domain.CounterSale, int64, error)
	CreatePayment(ctx context.Context, payment *domain.CounterSalePayment) error
	ListPaymentsByCounterSaleID(ctx context.Context, tenantID, counterSaleID uuid.UUID) ([]domain.CounterSalePayment, error)
}

// TenantPurchaseRepository defines persistence contracts for purchases
type TenantPurchaseRepository interface {
	Create(ctx context.Context, purchase *domain.TenantPurchase, payments []domain.TenantPurchasePayment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error)
	GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error)
	GetCustomerPurchasesForUpdate(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.TenantPurchase, error)
	Update(ctx context.Context, purchase *domain.TenantPurchase) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.TenantPurchase, int64, error)
	CreatePayment(ctx context.Context, payment *domain.TenantPurchasePayment) error
	ListPaymentsByPurchaseID(ctx context.Context, tenantID, purchaseID uuid.UUID) ([]domain.TenantPurchasePayment, error)
	ListPaymentsByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.TenantPurchasePayment, error)
	GetCustomerPayableSummary(ctx context.Context, tenantID, customerID uuid.UUID) (totalPurchases, totalPaid, outstandingPayable decimal.Decimal, err error)
	GetCustomerPayableSummariesBatch(ctx context.Context, tenantID uuid.UUID, customerIDs []uuid.UUID) (map[uuid.UUID]domain.CustomerPayableSummary, error)
}

// TenantExpenseRepository defines persistence contracts for expenses
type TenantExpenseRepository interface {
	Create(ctx context.Context, expense *domain.TenantExpense, payments []domain.TenantExpensePayment) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantExpense, error)
	Update(ctx context.Context, expense *domain.TenantExpense) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, search string) ([]domain.TenantExpense, int64, error)
}

// AttendanceRepository defines persistence contracts for attendance, overtime, and advances
type AttendanceRepository interface {
	Upsert(ctx context.Context, att *domain.Attendance) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Attendance, error)
	GetByEmployeeAndDate(ctx context.Context, tenantID, employeeID uuid.UUID, date string) (*domain.Attendance, error)
	GetTodaySalaryEarned(ctx context.Context, tenantID uuid.UUID, date string) (decimal.Decimal, error)
	Update(ctx context.Context, att *domain.Attendance) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, employeeID *uuid.UUID) ([]domain.Attendance, int64, error)
}

// EmployeeAdvanceRepository defines persistence contracts for employee advances
type EmployeeAdvanceRepository interface {
	Create(ctx context.Context, adv *domain.EmployeeAdvance) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmployeeAdvance, error)
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, employeeID *uuid.UUID, date string) ([]domain.EmployeeAdvance, int64, error)
	GetTotalAdvancesByEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (decimal.Decimal, error)
}

// EmployeeOvertimeRepository defines persistence contracts for employee overtime transactions
type EmployeeOvertimeRepository interface {
	Create(ctx context.Context, ot *domain.EmployeeOvertime) error
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.EmployeeOvertime, error)
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, employeeID *uuid.UUID, date string) ([]domain.EmployeeOvertime, int64, error)
	GetTotalOvertimeByEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (decimal.Decimal, error)
}

// EmployeeSalaryRepository defines persistence contracts for employee salary balances and payments
type EmployeeSalaryRepository interface {
	GetByEmployeeID(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalary, error)
	GetByEmployeeIDForUpdate(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalary, error)
	UpsertBalance(ctx context.Context, salary *domain.EmployeeSalary) error
	AdjustBalance(ctx context.Context, tenantID, employeeID uuid.UUID, delta decimal.Decimal) error
	RecalculateBalance(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalary, error)
	Delete(ctx context.Context, tenantID, employeeID uuid.UUID) error
	List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search string) ([]domain.EmployeeSalary, int64, error)
	ListPending(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.EmployeeSalary, int64, error)
	CreatePayment(ctx context.Context, payment *domain.EmployeeSalaryPayment) error
	ListPayments(ctx context.Context, tenantID uuid.UUID, employeeID *uuid.UUID, page, pageSize int) ([]domain.EmployeeSalaryPayment, int64, error)
	GetTotalPaymentsByEmployee(ctx context.Context, tenantID, employeeID uuid.UUID) (decimal.Decimal, error)
	GetStatement(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalaryStatement, error)
}

// TenantDailyStatsRepository defines persistence contracts for daily aggregates
type TenantDailyStatsRepository interface {
	GetByDate(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error)
	GetLatestAvailable(ctx context.Context, tenantID uuid.UUID, beforeDate string) (*domain.TenantDailyStats, error)
	Upsert(ctx context.Context, stats *domain.TenantDailyStats) error
	ComputeAndSyncDailyStats(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error)
}
