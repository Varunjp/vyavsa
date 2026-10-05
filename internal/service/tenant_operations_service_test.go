package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/logger"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// In-memory mock repositories for operations testing
type mockOpEmployeeRepo struct {
	emps map[uuid.UUID]*domain.TenantEmployee
}

func newMockOpEmployeeRepo() *mockOpEmployeeRepo {
	return &mockOpEmployeeRepo{emps: make(map[uuid.UUID]*domain.TenantEmployee)}
}

func (m *mockOpEmployeeRepo) Create(ctx context.Context, emp *domain.TenantEmployee) error {
	emp.ID = uuid.New()
	emp.CreatedAt = time.Now().UTC()
	emp.UpdatedAt = time.Now().UTC()
	m.emps[emp.ID] = emp
	return nil
}

func (m *mockOpEmployeeRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantEmployee, error) {
	e, ok := m.emps[id]
	if !ok || e.TenantID != tenantID {
		return nil, appErrors.NewNotFound("employee not found")
	}
	return e, nil
}

func (m *mockOpEmployeeRepo) Update(ctx context.Context, emp *domain.TenantEmployee) error {
	if _, ok := m.emps[emp.ID]; !ok {
		return appErrors.NewNotFound("employee not found")
	}
	m.emps[emp.ID] = emp
	return nil
}

func (m *mockOpEmployeeRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	e, ok := m.emps[id]
	if !ok || e.TenantID != tenantID {
		return appErrors.NewNotFound("employee not found")
	}
	delete(m.emps, id)
	return nil
}

func (m *mockOpEmployeeRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantEmployee, int64, error) {
	var list []domain.TenantEmployee
	for _, e := range m.emps {
		if e.TenantID == tenantID {
			list = append(list, *e)
		}
	}
	return list, int64(len(list)), nil
}

// Mock Customer Repo
type mockOpCustomerRepo struct {
	custs       map[uuid.UUID]*domain.TenantCustomer
	adjustments []domain.CustomerBalanceAdjustment
}

func newMockOpCustomerRepo() *mockOpCustomerRepo {
	return &mockOpCustomerRepo{
		custs:       make(map[uuid.UUID]*domain.TenantCustomer),
		adjustments: make([]domain.CustomerBalanceAdjustment, 0),
	}
}

func (m *mockOpCustomerRepo) Create(ctx context.Context, cust *domain.TenantCustomer) error {
	cust.ID = uuid.New()
	cust.CreatedAt = time.Now().UTC()
	cust.UpdatedAt = time.Now().UTC()
	m.custs[cust.ID] = cust
	return nil
}

func (m *mockOpCustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantCustomer, error) {
	c, ok := m.custs[id]
	if !ok || c.TenantID != tenantID {
		return nil, appErrors.NewNotFound("customer not found")
	}
	return c, nil
}

func (m *mockOpCustomerRepo) Update(ctx context.Context, cust *domain.TenantCustomer) error {
	m.custs[cust.ID] = cust
	return nil
}

func (m *mockOpCustomerRepo) AdjustBalance(ctx context.Context, tenantID, id uuid.UUID, delta decimal.Decimal) error {
	c, ok := m.custs[id]
	if !ok || c.TenantID != tenantID {
		return appErrors.NewNotFound("customer not found")
	}
	c.CurrentBalance = c.CurrentBalance.Add(delta)
	return nil
}

func (m *mockOpCustomerRepo) SetBalance(ctx context.Context, tenantID, id uuid.UUID, newBalance decimal.Decimal) error {
	c, ok := m.custs[id]
	if !ok || c.TenantID != tenantID {
		return appErrors.NewNotFound("customer not found")
	}
	c.CurrentBalance = newBalance
	return nil
}

func (m *mockOpCustomerRepo) RecordAdjustment(ctx context.Context, adj *domain.CustomerBalanceAdjustment) error {
	adj.ID = uuid.New()
	adj.CreatedAt = time.Now().UTC()
	m.adjustments = append(m.adjustments, *adj)
	return nil
}

func (m *mockOpCustomerRepo) ListAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, page, pageSize int) ([]domain.CustomerBalanceAdjustment, int64, error) {
	var list []domain.CustomerBalanceAdjustment
	for _, a := range m.adjustments {
		if a.TenantID == tenantID && a.CustomerID == customerID {
			list = append(list, a)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockOpCustomerRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.custs, id)
	return nil
}

func (m *mockOpCustomerRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantCustomer, int64, error) {
	var list []domain.TenantCustomer
	for _, c := range m.custs {
		if c.TenantID == tenantID {
			list = append(list, *c)
		}
	}
	return list, int64(len(list)), nil
}

// Mock Bank Repo
type mockOpBankRepo struct {
	banks        map[uuid.UUID]*domain.TenantBank
	transactions []domain.BankTransaction
	failOnTx     bool
}

func newMockOpBankRepo() *mockOpBankRepo {
	return &mockOpBankRepo{
		banks:        make(map[uuid.UUID]*domain.TenantBank),
		transactions: make([]domain.BankTransaction, 0),
	}
}

func (m *mockOpBankRepo) Create(ctx context.Context, bank *domain.TenantBank) error {
	bank.ID = uuid.New()
	bank.CreatedAt = time.Now().UTC()
	bank.UpdatedAt = time.Now().UTC()
	m.banks[bank.ID] = bank
	return nil
}

func (m *mockOpBankRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantBank, error) {
	b, ok := m.banks[id]
	if !ok || b.TenantID != tenantID {
		return nil, appErrors.NewNotFound("bank not found")
	}
	return b, nil
}

func (m *mockOpBankRepo) Update(ctx context.Context, bank *domain.TenantBank) error {
	m.banks[bank.ID] = bank
	return nil
}

func (m *mockOpBankRepo) AdjustBalance(ctx context.Context, tenantID, id uuid.UUID, delta decimal.Decimal) error {
	b, ok := m.banks[id]
	if !ok || b.TenantID != tenantID {
		return appErrors.NewNotFound("bank not found")
	}
	b.CurrentBalance = b.CurrentBalance.Add(delta)
	return nil
}

func (m *mockOpBankRepo) CreateTransaction(ctx context.Context, tx *domain.BankTransaction) error {
	if m.failOnTx {
		return errors.New("simulated bank transaction persistence failure")
	}
	tx.ID = uuid.New()
	tx.CreatedAt = time.Now().UTC()
	m.transactions = append(m.transactions, *tx)
	return nil
}

func (m *mockOpBankRepo) ListTransactions(ctx context.Context, tenantID, bankID uuid.UUID, page, pageSize int) ([]domain.BankTransaction, int64, error) {
	var list []domain.BankTransaction
	for _, tx := range m.transactions {
		if tx.TenantID == tenantID && tx.BankID == bankID {
			list = append(list, tx)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockOpBankRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.banks, id)
	return nil
}

func (m *mockOpBankRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantBank, int64, error) {
	var list []domain.TenantBank
	for _, b := range m.banks {
		if b.TenantID == tenantID {
			list = append(list, *b)
		}
	}
	return list, int64(len(list)), nil
}

// Mock Line Sale Repo
type mockOpLineSaleRepo struct {
	sales map[uuid.UUID]*domain.LineSale
}

func newMockOpLineSaleRepo() *mockOpLineSaleRepo {
	return &mockOpLineSaleRepo{sales: make(map[uuid.UUID]*domain.LineSale)}
}

func (m *mockOpLineSaleRepo) Create(ctx context.Context, sale *domain.LineSale, payments []domain.LineSalePayment) error {
	sale.ID = uuid.New()
	sale.CreatedAt = time.Now().UTC()
	sale.UpdatedAt = time.Now().UTC()
	sale.Payments = payments
	m.sales[sale.ID] = sale
	return nil
}

func (m *mockOpLineSaleRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error) {
	s, ok := m.sales[id]
	if !ok || s.TenantID != tenantID {
		return nil, appErrors.NewNotFound("sale not found")
	}
	return s, nil
}

func (m *mockOpLineSaleRepo) Update(ctx context.Context, sale *domain.LineSale) error {
	m.sales[sale.ID] = sale
	return nil
}

func (m *mockOpLineSaleRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.sales, id)
	return nil
}

func (m *mockOpLineSaleRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.LineSale, int64, error) {
	var list []domain.LineSale
	for _, s := range m.sales {
		if s.TenantID == tenantID {
			list = append(list, *s)
		}
	}
	return list, int64(len(list)), nil
}

// Mock Counter Sale Repo
type mockOpCounterSaleRepo struct {
	sales map[uuid.UUID]*domain.CounterSale
}

func newMockOpCounterSaleRepo() *mockOpCounterSaleRepo {
	return &mockOpCounterSaleRepo{sales: make(map[uuid.UUID]*domain.CounterSale)}
}

func (m *mockOpCounterSaleRepo) Create(ctx context.Context, sale *domain.CounterSale, payments []domain.CounterSalePayment) error {
	sale.ID = uuid.New()
	sale.CreatedAt = time.Now().UTC()
	sale.UpdatedAt = time.Now().UTC()
	sale.Payments = payments
	m.sales[sale.ID] = sale
	return nil
}

func (m *mockOpCounterSaleRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.CounterSale, error) {
	s, ok := m.sales[id]
	if !ok || s.TenantID != tenantID {
		return nil, appErrors.NewNotFound("counter sale not found")
	}
	return s, nil
}

func (m *mockOpCounterSaleRepo) Update(ctx context.Context, sale *domain.CounterSale) error {
	m.sales[sale.ID] = sale
	return nil
}

func (m *mockOpCounterSaleRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.sales, id)
	return nil
}

func (m *mockOpCounterSaleRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, search string) ([]domain.CounterSale, int64, error) {
	var list []domain.CounterSale
	for _, s := range m.sales {
		if s.TenantID == tenantID {
			list = append(list, *s)
		}
	}
	return list, int64(len(list)), nil
}

// Mock Purchase Repo
type mockOpPurchaseRepo struct {
	purchases map[uuid.UUID]*domain.TenantPurchase
}

func newMockOpPurchaseRepo() *mockOpPurchaseRepo {
	return &mockOpPurchaseRepo{purchases: make(map[uuid.UUID]*domain.TenantPurchase)}
}

func (m *mockOpPurchaseRepo) Create(ctx context.Context, p *domain.TenantPurchase, payments []domain.TenantPurchasePayment) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = time.Now().UTC()
	p.Payments = payments
	m.purchases[p.ID] = p
	return nil
}

func (m *mockOpPurchaseRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error) {
	p, ok := m.purchases[id]
	if !ok || p.TenantID != tenantID {
		return nil, appErrors.NewNotFound("purchase not found")
	}
	return p, nil
}

func (m *mockOpPurchaseRepo) Update(ctx context.Context, p *domain.TenantPurchase) error {
	m.purchases[p.ID] = p
	return nil
}

func (m *mockOpPurchaseRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.purchases, id)
	return nil
}

func (m *mockOpPurchaseRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, search string) ([]domain.TenantPurchase, int64, error) {
	var list []domain.TenantPurchase
	for _, p := range m.purchases {
		if p.TenantID == tenantID {
			list = append(list, *p)
		}
	}
	return list, int64(len(list)), nil
}

// Mock Expense Repo
type mockOpExpenseRepo struct {
	expenses map[uuid.UUID]*domain.TenantExpense
}

func newMockOpExpenseRepo() *mockOpExpenseRepo {
	return &mockOpExpenseRepo{expenses: make(map[uuid.UUID]*domain.TenantExpense)}
}

func (m *mockOpExpenseRepo) Create(ctx context.Context, e *domain.TenantExpense, payments []domain.TenantExpensePayment) error {
	e.ID = uuid.New()
	e.CreatedAt = time.Now().UTC()
	e.UpdatedAt = time.Now().UTC()
	e.Payments = payments
	m.expenses[e.ID] = e
	return nil
}

func (m *mockOpExpenseRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantExpense, error) {
	e, ok := m.expenses[id]
	if !ok || e.TenantID != tenantID {
		return nil, appErrors.NewNotFound("expense not found")
	}
	return e, nil
}

func (m *mockOpExpenseRepo) Update(ctx context.Context, e *domain.TenantExpense) error {
	m.expenses[e.ID] = e
	return nil
}

func (m *mockOpExpenseRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	delete(m.expenses, id)
	return nil
}

func (m *mockOpExpenseRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, search string) ([]domain.TenantExpense, int64, error) {
	var list []domain.TenantExpense
	for _, e := range m.expenses {
		if e.TenantID == tenantID {
			list = append(list, *e)
		}
	}
	return list, int64(len(list)), nil
}

// Mock Attendance Repo
type mockOpAttendanceRepo struct {
	attendance map[string]*domain.Attendance
}

func newMockOpAttendanceRepo() *mockOpAttendanceRepo {
	return &mockOpAttendanceRepo{attendance: make(map[string]*domain.Attendance)}
}

func (m *mockOpAttendanceRepo) key(tenantID, employeeID uuid.UUID, date string) string {
	return tenantID.String() + ":" + employeeID.String() + ":" + date
}

func (m *mockOpAttendanceRepo) Upsert(ctx context.Context, att *domain.Attendance) error {
	if att.ID == uuid.Nil {
		att.ID = uuid.New()
		att.CreatedAt = time.Now().UTC()
	}
	att.UpdatedAt = time.Now().UTC()
	k := m.key(att.TenantID, att.EmployeeID, att.Date)
	m.attendance[k] = att
	return nil
}

func (m *mockOpAttendanceRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Attendance, error) {
	for _, a := range m.attendance {
		if a.TenantID == tenantID && a.ID == id {
			return a, nil
		}
	}
	return nil, appErrors.NewNotFound("attendance not found")
}

func (m *mockOpAttendanceRepo) GetByEmployeeAndDate(ctx context.Context, tenantID, employeeID uuid.UUID, date string) (*domain.Attendance, error) {
	k := m.key(tenantID, employeeID, date)
	return m.attendance[k], nil
}

func (m *mockOpAttendanceRepo) Update(ctx context.Context, att *domain.Attendance) error {
	k := m.key(att.TenantID, att.EmployeeID, att.Date)
	m.attendance[k] = att
	return nil
}

func (m *mockOpAttendanceRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	for k, a := range m.attendance {
		if a.TenantID == tenantID && a.ID == id {
			delete(m.attendance, k)
			return nil
		}
	}
	return appErrors.NewNotFound("attendance not found")
}

func (m *mockOpAttendanceRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, employeeID *uuid.UUID) ([]domain.Attendance, int64, error) {
	var list []domain.Attendance
	for _, a := range m.attendance {
		if a.TenantID == tenantID {
			if date != "" && a.Date != date {
				continue
			}
			if employeeID != nil && a.EmployeeID != *employeeID {
				continue
			}
			list = append(list, *a)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockOpAttendanceRepo) GetTodaySalaryEarned(ctx context.Context, tenantID uuid.UUID, date string) (decimal.Decimal, error) {
	var total decimal.Decimal
	for _, a := range m.attendance {
		if a.TenantID == tenantID && a.Date == date && a.Status == "present" {
			total = total.Add(a.DailySalary)
		}
	}
	return total, nil
}

// Mock Salary Repo
type mockOpSalaryRepo struct {
	salaries map[uuid.UUID]*domain.EmployeeSalary
	payments []domain.EmployeeSalaryPayment
}

func newMockOpSalaryRepo() *mockOpSalaryRepo {
	return &mockOpSalaryRepo{
		salaries: make(map[uuid.UUID]*domain.EmployeeSalary),
		payments: make([]domain.EmployeeSalaryPayment, 0),
	}
}

func (m *mockOpSalaryRepo) GetByEmployeeID(ctx context.Context, tenantID, employeeID uuid.UUID) (*domain.EmployeeSalary, error) {
	s, ok := m.salaries[employeeID]
	if !ok {
		return &domain.EmployeeSalary{
			TenantID:   tenantID,
			EmployeeID: employeeID,
			Balance:    decimal.Zero,
		}, nil
	}
	return s, nil
}

func (m *mockOpSalaryRepo) UpsertBalance(ctx context.Context, s *domain.EmployeeSalary) error {
	m.salaries[s.EmployeeID] = s
	return nil
}

func (m *mockOpSalaryRepo) AdjustBalance(ctx context.Context, tenantID, employeeID uuid.UUID, delta decimal.Decimal) error {
	s, ok := m.salaries[employeeID]
	if !ok {
		s = &domain.EmployeeSalary{
			TenantID:   tenantID,
			EmployeeID: employeeID,
			Balance:    delta,
		}
		m.salaries[employeeID] = s
		return nil
	}
	s.Balance = s.Balance.Add(delta)
	return nil
}

func (m *mockOpSalaryRepo) Delete(ctx context.Context, tenantID, employeeID uuid.UUID) error {
	delete(m.salaries, employeeID)
	return nil
}

func (m *mockOpSalaryRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search string) ([]domain.EmployeeSalary, int64, error) {
	var list []domain.EmployeeSalary
	for _, s := range m.salaries {
		if s.TenantID == tenantID {
			list = append(list, *s)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockOpSalaryRepo) ListPending(ctx context.Context, tenantID uuid.UUID, page, pageSize int) ([]domain.EmployeeSalary, int64, error) {
	var list []domain.EmployeeSalary
	for _, s := range m.salaries {
		if s.TenantID == tenantID && s.Balance.GreaterThan(decimal.Zero) {
			list = append(list, *s)
		}
	}
	return list, int64(len(list)), nil
}

func (m *mockOpSalaryRepo) CreatePayment(ctx context.Context, payment *domain.EmployeeSalaryPayment) error {
	payment.ID = uuid.New()
	payment.CreatedAt = time.Now().UTC()
	payment.UpdatedAt = time.Now().UTC()
	m.payments = append(m.payments, *payment)
	return nil
}

func (m *mockOpSalaryRepo) ListPayments(ctx context.Context, tenantID uuid.UUID, employeeID *uuid.UUID, page, pageSize int) ([]domain.EmployeeSalaryPayment, int64, error) {
	var list []domain.EmployeeSalaryPayment
	for _, p := range m.payments {
		if p.TenantID == tenantID {
			if employeeID != nil && p.EmployeeID != *employeeID {
				continue
			}
			list = append(list, p)
		}
	}
	return list, int64(len(list)), nil
}

// Mock Daily Stats Repo
type mockOpDailyStatsRepo struct {
	stats map[string]*domain.TenantDailyStats
}

func newMockOpDailyStatsRepo() *mockOpDailyStatsRepo {
	return &mockOpDailyStatsRepo{stats: make(map[string]*domain.TenantDailyStats)}
}

func (m *mockOpDailyStatsRepo) GetByDate(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	return m.stats[tenantID.String()+":"+date], nil
}

func (m *mockOpDailyStatsRepo) Upsert(ctx context.Context, stats *domain.TenantDailyStats) error {
	m.stats[stats.TenantID.String()+":"+stats.Date] = stats
	return nil
}

func (m *mockOpDailyStatsRepo) GetLatestAvailable(ctx context.Context, tenantID uuid.UUID, beforeDate string) (*domain.TenantDailyStats, error) {
	var latest *domain.TenantDailyStats
	for _, s := range m.stats {
		if s.TenantID == tenantID && s.Date < beforeDate {
			if latest == nil || s.Date > latest.Date {
				latest = s
			}
		}
	}
	return latest, nil
}

func (m *mockOpDailyStatsRepo) ComputeAndSyncDailyStats(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	if existing, ok := m.stats[tenantID.String()+":"+date]; ok {
		return existing, nil
	}
	s := &domain.TenantDailyStats{
		ID:       uuid.New(),
		TenantID: tenantID,
		Date:     date,
	}
	m.stats[tenantID.String()+":"+date] = s
	return s, nil
}

func setupTestOperationsServiceWithRepos() (*TenantOperationsService, uuid.UUID, *mockOpCustomerRepo, *mockOpBankRepo, *mockOpPurchaseRepo, *mockOpLineSaleRepo, *mockOpCounterSaleRepo, *mockFinancialSummaryRepo) {
	tenantID := uuid.New()

	userRepo := newMockTenantUserRepoFull()
	empRepo := newMockOpEmployeeRepo()
	custRepo := newMockOpCustomerRepo()
	bankRepo := newMockOpBankRepo()
	lineSaleRepo := newMockOpLineSaleRepo()
	countSaleRepo := newMockOpCounterSaleRepo()
	purchRepo := newMockOpPurchaseRepo()
	expRepo := newMockOpExpenseRepo()
	attRepo := newMockOpAttendanceRepo()
	salaryRepo := newMockOpSalaryRepo()
	statsRepo := newMockOpDailyStatsRepo()
	summaryRepo := newMockFinancialSummaryRepo()

	_ = summaryRepo.Create(context.Background(), &domain.TenantFinancialSummary{
		TenantID:        tenantID,
		CashBalance:     decimal.NewFromFloat(50000.00),
		BankBalance:     decimal.NewFromFloat(100000.00),
		TotalReceivable: decimal.Zero,
		TotalPayable:    decimal.Zero,
	})

	transactor := &mockTransactor{}
	hasher := auth.NewBcryptHasher()
	log := logger.Default().Logger

	svc := NewTenantOperationsService(
		userRepo,
		empRepo,
		custRepo,
		bankRepo,
		lineSaleRepo,
		countSaleRepo,
		purchRepo,
		expRepo,
		attRepo,
		salaryRepo,
		statsRepo,
		summaryRepo,
		transactor,
		hasher,
		log,
	)

	return svc, tenantID, custRepo, bankRepo, purchRepo, lineSaleRepo, countSaleRepo, summaryRepo
}

func setupTestOperationsService() (*TenantOperationsService, uuid.UUID) {
	svc, tenantID, _, _, _, _, _, _ := setupTestOperationsServiceWithRepos()
	return svc, tenantID
}

// ----------------------------------------------------
// Tests
// ----------------------------------------------------

func TestOperationsService_TenantUserCRUD(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	// 1. Create Tenant User
	user, err := svc.CreateTenantUser(ctx, tenantID, &dto.CreateTenantUserRequest{
		Name:     "Operations Staff",
		Email:    "ops@vyavsa.com",
		Password: "password123",
		Role:     "user",
		Status:   "active",
	})
	require.NoError(t, err)
	assert.Equal(t, "Operations Staff", user.Name)
	assert.Equal(t, "user", user.Role)

	// 2. Duplicate email should fail
	_, err = svc.CreateTenantUser(ctx, tenantID, &dto.CreateTenantUserRequest{
		Name:     "Duplicate Staff",
		Email:    "ops@vyavsa.com",
		Password: "password123",
		Role:     "user",
	})
	assert.Error(t, err)

	// 3. Update User
	updated, err := svc.UpdateTenantUser(ctx, tenantID, user.ID, &dto.UpdateTenantUserRequest{
		Name: "Senior Ops Staff",
	})
	require.NoError(t, err)
	assert.Equal(t, "Senior Ops Staff", updated.Name)

	// 4. List Users
	users, total, err := svc.ListTenantUsers(ctx, tenantID, 1, 10, "", "", "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)

	// 5. Delete User (Cannot delete self)
	err = svc.DeleteTenantUser(ctx, tenantID, user.ID, user.ID)
	assert.Error(t, err)

	adminID := uuid.New()
	err = svc.DeleteTenantUser(ctx, tenantID, adminID, user.ID)
	require.NoError(t, err)
}

func TestOperationsService_EmployeeAndSalaryFlow(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	// 1. Create Employee
	emp, err := svc.CreateEmployee(ctx, tenantID, &dto.CreateEmployeeRequest{
		Name:   "Ramesh Kumar",
		Phone:  "9876543210",
		Salary: decimal.NewFromFloat(25000.00),
		OTRate: decimal.NewFromFloat(150.00),
	})
	require.NoError(t, err)
	assert.Equal(t, "Ramesh Kumar", emp.Name)

	// 2. Record Attendance with Advance
	att, err := svc.RecordAttendance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: emp.ID,
		Date:       todayString(),
		Status:     "present",
		Advance:    decimal.NewFromFloat(2000.00),
	})
	require.NoError(t, err)
	assert.Equal(t, "present", att.Status)

	// 3. Record OT
	_, err = svc.RecordOvertime(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordOvertimeRequest{
		EmployeeID: emp.ID,
		Date:       todayString(),
		OT:         decimal.NewFromFloat(4.00), // 4 * 150 = 600
	})
	require.NoError(t, err)

	// Salary balance should reflect: 0 (initial) + 25000 (present daily salary) - 2000 (advance) + 600 (OT) = 23600
	salary, err := svc.GetSalaryByEmployeeID(ctx, tenantID, emp.ID)
	require.NoError(t, err)
	assert.True(t, salary.Balance.Equal(decimal.NewFromFloat(23600.00)))

	// Adjust salary to 20,000 for monthly wage
	_, err = svc.UpdateSalaryBalance(ctx, tenantID, emp.ID, &dto.UpdateSalaryBalanceRequest{
		Balance: decimal.NewFromFloat(20000.00),
	})
	require.NoError(t, err)

	// 4. Pay Salary
	payment, err := svc.PaySalary(ctx, tenantID, emp.ID, &dto.PaySalaryRequest{
		PaymentMethod: "cash",
		Amount:        decimal.NewFromFloat(15000.00),
		Note:          "September Salary Partial",
	})
	require.NoError(t, err)
	assert.Equal(t, "cash", payment.PaymentMethod)

	// Remaining balance: 20000 - 15000 = 5000
	salary, err = svc.GetSalaryByEmployeeID(ctx, tenantID, emp.ID)
	require.NoError(t, err)
	assert.True(t, salary.Balance.Equal(decimal.NewFromFloat(5000.00)))
}

func TestOperationsService_LineSaleCustomerBalance(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	// 1. Create Customer
	cust, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName:   "Sharma Electronics",
		OpeningBalance: decimal.NewFromFloat(1000.00),
	})
	require.NoError(t, err)

	// 2. Create Line Sale: Total 5000, Cash In 2000 => Balance (credit) 3000
	sale, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
		CustomerID:  cust.ID,
		Route:       "North Zone",
		TotalAmount: decimal.NewFromFloat(5000.00),
		TotalCashIn: decimal.NewFromFloat(2000.00),
	})
	require.NoError(t, err)
	assert.True(t, sale.Balance.Equal(decimal.NewFromFloat(3000.00)))

	// Customer balance: 1000 opening + 3000 credit = 4000
	updatedCust, err := svc.GetCustomerByID(ctx, tenantID, cust.ID)
	require.NoError(t, err)
	assert.True(t, updatedCust.CurrentBalance.Equal(decimal.NewFromFloat(4000.00)))
}

func TestOperationsService_CurrentDayRestrictionForTenantUser(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	emp, err := svc.CreateEmployee(ctx, tenantID, &dto.CreateEmployeeRequest{
		Name:   "Sunil",
		Salary: decimal.NewFromFloat(15000.00),
	})
	require.NoError(t, err)

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")

	// 1. Tenant User trying to record attendance for yesterday should be rejected
	_, err = svc.RecordAttendance(ctx, tenantID, auth.RoleTenantUser, &dto.RecordAttendanceRequest{
		EmployeeID: emp.ID,
		Date:       yesterday,
		Status:     "present",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden: tenant users can only process transactions for the current business day")

	// 2. Tenant Admin CAN record for yesterday
	_, err = svc.RecordAttendance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: emp.ID,
		Date:       yesterday,
		Status:     "present",
	})
	require.NoError(t, err)
}

func TestOperationsService_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	svc, tenantA := setupTestOperationsService()
	tenantB := uuid.New()

	// Create employee for Tenant A
	empA, err := svc.CreateEmployee(ctx, tenantA, &dto.CreateEmployeeRequest{
		Name:   "Tenant A Worker",
		Salary: decimal.NewFromFloat(18000.00),
	})
	require.NoError(t, err)

	// Tenant B tries to get Tenant A's employee -> Must be Not Found
	_, err = svc.GetEmployeeByID(ctx, tenantB, empA.ID)
	assert.Error(t, err)

	// Tenant B tries to record attendance for Tenant A's employee -> Must be Bad Request / Not Found
	_, err = svc.RecordAttendance(ctx, tenantB, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: empA.ID,
		Status:     "present",
	})
	assert.Error(t, err)
}

func TestOperationsService_FinancialMetrics(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	metrics, err := svc.GetFinancialMetrics(ctx, tenantID)
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.True(t, metrics.CashBalance.Equal(decimal.NewFromFloat(50000.00)))
	assert.True(t, metrics.BankBalance.Equal(decimal.NewFromFloat(100000.00)))
}

func TestOperationsService_StaleDataFallback(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	// 1. Initially no activity today, no yesterday -> is_current is true (empty/zeroed)
	metrics, err := svc.GetFinancialMetrics(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, metrics.IsCurrent)
	assert.Equal(t, 0, metrics.DaysOld)

	// 2. Simulate yesterday had activity
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	yesterdayStats := &domain.TenantDailyStats{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Date:       yesterday,
		TotalSales: decimal.NewFromFloat(25000.00),
	}
	_ = svc.statsRepo.Upsert(ctx, yesterdayStats)

	// Query metrics: today has no activity, so it should fall back to yesterday's stats
	metricsStale, err := svc.GetFinancialMetrics(ctx, tenantID)
	require.NoError(t, err)
	assert.False(t, metricsStale.IsCurrent)
	assert.Equal(t, yesterday, metricsStale.DataDate)
	assert.Equal(t, 1, metricsStale.DaysOld)
	assert.True(t, metricsStale.TodayStats.TotalSales.Equal(decimal.NewFromFloat(25000.00)))

	// 3. Now simulate today records a sale
	today := todayString()
	todayActive := &domain.TenantDailyStats{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Date:       today,
		TotalSales: decimal.NewFromFloat(12000.00),
	}
	_ = svc.statsRepo.Upsert(ctx, todayActive)

	metricsToday, err := svc.GetFinancialMetrics(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, metricsToday.IsCurrent)
	assert.Equal(t, today, metricsToday.DataDate)
	assert.Equal(t, 0, metricsToday.DaysOld)
	assert.True(t, metricsToday.TodayStats.TotalSales.Equal(decimal.NewFromFloat(12000.00)))
}

func TestOperationsService_SalesPaymentCollectionAndBankBalance(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	// Setup Customer
	cust, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName: "Super Store",
		Phone:        "9876500001",
	})
	require.NoError(t, err)

	// Setup Bank
	bankA, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "HDFC Current",
		AccountNumber:  "1234567890",
		OpeningBalance: decimal.NewFromFloat(10000.00),
	})
	require.NoError(t, err)
	assert.True(t, bankA.CurrentBalance.Equal(decimal.NewFromFloat(10000.00)))

	// 1. Cash-only Line Sale
	sale1, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
		CustomerID:  cust.ID,
		TotalAmount: decimal.NewFromFloat(1000.00),
		TotalCashIn: decimal.NewFromFloat(1000.00),
	})
	require.NoError(t, err)
	assert.True(t, sale1.TotalCashIn.Equal(decimal.NewFromFloat(1000.00)))
	assert.True(t, sale1.BankAmount.IsZero())
	assert.True(t, sale1.CollectedAmount.Equal(decimal.NewFromFloat(1000.00)))
	assert.True(t, sale1.Balance.IsZero())

	// 2. Bank-only Line Sale
	sale2, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
		CustomerID:  cust.ID,
		TotalAmount: decimal.NewFromFloat(2000.00),
		BankAmount:  decimal.NewFromFloat(2000.00),
		BankID:      &bankA.ID,
	})
	require.NoError(t, err)
	assert.True(t, sale2.BankAmount.Equal(decimal.NewFromFloat(2000.00)))
	assert.True(t, sale2.CollectedAmount.Equal(decimal.NewFromFloat(2000.00)))
	assert.True(t, sale2.Balance.IsZero())

	// Verify Bank A balance updated: 10000 + 2000 = 12000
	updatedBankA, err := svc.GetBankByID(ctx, tenantID, bankA.ID)
	require.NoError(t, err)
	assert.True(t, updatedBankA.CurrentBalance.Equal(decimal.NewFromFloat(12000.00)))

	// Verify Bank Transaction recorded
	txs, totalTxs, err := svc.ListBankTransactions(ctx, tenantID, bankA.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), totalTxs) // 1 opening balance + 1 sale deposit
	assert.Equal(t, "credit", txs[0].TransactionType)
	assert.True(t, txs[0].Amount.Equal(decimal.NewFromFloat(10000.00)))
	assert.Equal(t, "credit", txs[1].TransactionType)
	assert.True(t, txs[1].Amount.Equal(decimal.NewFromFloat(2000.00)))
	assert.Equal(t, &sale2.ID, txs[1].SaleID)

	// 3. Split Cash + Bank Line Sale with Due
	sale3, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
		CustomerID:  cust.ID,
		TotalAmount: decimal.NewFromFloat(5000.00),
		TotalCashIn: decimal.NewFromFloat(1500.00),
		BankAmount:  decimal.NewFromFloat(2000.00),
		BankID:      &bankA.ID,
	})
	require.NoError(t, err)
	assert.True(t, sale3.TotalCashIn.Equal(decimal.NewFromFloat(1500.00)))
	assert.True(t, sale3.BankAmount.Equal(decimal.NewFromFloat(2000.00)))
	assert.True(t, sale3.CollectedAmount.Equal(decimal.NewFromFloat(3500.00)))
	assert.True(t, sale3.Balance.Equal(decimal.NewFromFloat(1500.00)))

	// Check customer balance increased by due amount (1500)
	updatedCust, err := svc.GetCustomerByID(ctx, tenantID, cust.ID)
	require.NoError(t, err)
	assert.True(t, updatedCust.CurrentBalance.Equal(decimal.NewFromFloat(1500.00)))

	// Verify Bank A balance updated again: 12000 + 2000 = 14000
	updatedBankA, err = svc.GetBankByID(ctx, tenantID, bankA.ID)
	require.NoError(t, err)
	assert.True(t, updatedBankA.CurrentBalance.Equal(decimal.NewFromFloat(14000.00)))

	// 4. Overpayment Validation (Collected > Total should be rejected)
	_, err = svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
		CustomerID:  cust.ID,
		TotalAmount: decimal.NewFromFloat(1000.00),
		TotalCashIn: decimal.NewFromFloat(600.00),
		BankAmount:  decimal.NewFromFloat(600.00),
		BankID:      &bankA.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collected amount cannot exceed sale total")

	// 5. Counter Sale with Bank Payment
	cs1, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
		Item:          "Rice 25kg",
		Price:         decimal.NewFromFloat(800.00),
		TotalAmount:   decimal.NewFromFloat(800.00),
		PaymentMethod: "bank",
		BankAmount:    decimal.NewFromFloat(800.00),
		BankID:        &bankA.ID,
	})
	require.NoError(t, err)
	assert.True(t, cs1.BankAmount.Equal(decimal.NewFromFloat(800.00)))
	assert.True(t, cs1.CollectedAmount.Equal(decimal.NewFromFloat(800.00)))

	// Bank A balance should now be: 14000 + 800 = 14800
	updatedBankA, err = svc.GetBankByID(ctx, tenantID, bankA.ID)
	require.NoError(t, err)
	assert.True(t, updatedBankA.CurrentBalance.Equal(decimal.NewFromFloat(14800.00)))

	// 6. Counter Sale Split Payment
	cs2, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
		Item:          "Sugar 50kg",
		Price:         decimal.NewFromFloat(1200.00),
		TotalAmount:   decimal.NewFromFloat(1200.00),
		PaymentMethod: "split",
		Cash:          decimal.NewFromFloat(500.00),
		BankAmount:    decimal.NewFromFloat(700.00),
		BankID:        &bankA.ID,
	})
	require.NoError(t, err)
	assert.True(t, cs2.Cash.Equal(decimal.NewFromFloat(500.00)))
	assert.True(t, cs2.BankAmount.Equal(decimal.NewFromFloat(700.00)))
	assert.True(t, cs2.CollectedAmount.Equal(decimal.NewFromFloat(1200.00)))

	// Bank A balance: 14800 + 700 = 15500
	updatedBankA, err = svc.GetBankByID(ctx, tenantID, bankA.ID)
	require.NoError(t, err)
	assert.True(t, updatedBankA.CurrentBalance.Equal(decimal.NewFromFloat(15500.00)))
}

func TestOperationsService_IndividualBankBalanceTracking(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	// Create Bank 1 (SBI) and Bank 2 (ICICI)
	sbi, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "SBI Current",
		AccountNumber:  "SBI001",
		OpeningBalance: decimal.NewFromFloat(25000.00),
	})
	require.NoError(t, err)

	icici, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "ICICI Current",
		AccountNumber:  "ICICI001",
		OpeningBalance: decimal.NewFromFloat(50000.00),
	})
	require.NoError(t, err)

	// Deposit to SBI via Line Sale
	cust1, _ := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName: "Retailer 1",
		Phone:        "9000000000",
	})
	_, err = svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
		CustomerID:  cust1.ID,
		TotalAmount: decimal.NewFromFloat(5000.00),
		BankAmount:  decimal.NewFromFloat(5000.00),
		BankID:      &sbi.ID,
	})
	require.NoError(t, err)

	// Deposit to ICICI via Counter Sale
	_, err = svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
		Item:          "Items",
		Price:         decimal.NewFromFloat(8000.00),
		TotalAmount:   decimal.NewFromFloat(8000.00),
		PaymentMethod: "bank",
		BankAmount:    decimal.NewFromFloat(8000.00),
		BankID:        &icici.ID,
	})
	require.NoError(t, err)

	// SBI balance should be 30000, ICICI balance should be 58000
	sbiUpdated, err := svc.GetBankByID(ctx, tenantID, sbi.ID)
	require.NoError(t, err)
	assert.True(t, sbiUpdated.CurrentBalance.Equal(decimal.NewFromFloat(30000.00)))

	iciciUpdated, err := svc.GetBankByID(ctx, tenantID, icici.ID)
	require.NoError(t, err)
	assert.True(t, iciciUpdated.CurrentBalance.Equal(decimal.NewFromFloat(58000.00)))

	// Check Financial Metrics contains both individual banks with separate balances
	metrics, err := svc.GetFinancialMetrics(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, metrics.BankBalances, 2)
	for _, b := range metrics.BankBalances {
		if b.ID == sbi.ID {
			assert.True(t, b.CurrentBalance.Equal(decimal.NewFromFloat(30000.00)))
		} else if b.ID == icici.ID {
			assert.True(t, b.CurrentBalance.Equal(decimal.NewFromFloat(58000.00)))
		}
	}
}

func TestOperationsService_EmployeeDailySalaryAndAttendance(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()
	today := todayString()

	// Create Employee with base salary 600/day
	emp, err := svc.CreateEmployee(ctx, tenantID, &dto.CreateEmployeeRequest{
		Name:   "Sunil Kumar",
		Phone:  "9876543200",
		Salary: decimal.NewFromFloat(600.00),
		OTRate: decimal.NewFromFloat(100.00),
	})
	require.NoError(t, err)

	// 1. Mark Absent: Salary balance should NOT increase
	attAbsent, err := svc.RecordAttendance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: emp.ID,
		Date:       today,
		Status:     "absent",
	})
	require.NoError(t, err)
	assert.Equal(t, "absent", attAbsent.Status)
	assert.True(t, attAbsent.DailySalary.IsZero())

	sal, err := svc.GetSalaryByEmployeeID(ctx, tenantID, emp.ID)
	require.NoError(t, err)
	assert.True(t, sal.Balance.IsZero())

	// 2. Mark Present: Salary balance should increase by configured daily salary (600)
	attPresent, err := svc.RecordAttendance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: emp.ID,
		Date:       today,
		Status:     "present",
	})
	require.NoError(t, err)
	assert.Equal(t, "present", attPresent.Status)
	assert.True(t, attPresent.DailySalary.Equal(decimal.NewFromFloat(600.00)))

	sal, err = svc.GetSalaryByEmployeeID(ctx, tenantID, emp.ID)
	require.NoError(t, err)
	assert.True(t, sal.Balance.Equal(decimal.NewFromFloat(600.00)))

	// Check Financial Metrics today's earned employee salary reflects 600
	metrics, err := svc.GetFinancialMetrics(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, metrics.TodayEmployeeSalary.Equal(decimal.NewFromFloat(600.00)))

	// 3. Idempotent Retry: Calling RecordAttendance again with "present" should NOT double-credit
	attRetry, err := svc.RecordAttendance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: emp.ID,
		Date:       today,
		Status:     "present",
	})
	require.NoError(t, err)
	assert.Equal(t, "present", attRetry.Status)

	sal, err = svc.GetSalaryByEmployeeID(ctx, tenantID, emp.ID)
	require.NoError(t, err)
	assert.True(t, sal.Balance.Equal(decimal.NewFromFloat(600.00)), "Retry must not duplicate salary credit")

	// 4. Change status back to "absent": Should reverse the daily salary
	attReversed, err := svc.RecordAttendance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: emp.ID,
		Date:       today,
		Status:     "absent",
	})
	require.NoError(t, err)
	assert.Equal(t, "absent", attReversed.Status)
	assert.True(t, attReversed.DailySalary.IsZero())

	sal, err = svc.GetSalaryByEmployeeID(ctx, tenantID, emp.ID)
	require.NoError(t, err)
	assert.True(t, sal.Balance.IsZero(), "Reversing status must reverse salary balance")
}

func TestOperationsService_CustomerBalanceAdjustmentAndTenantIsolation(t *testing.T) {
	ctx := context.Background()
	svcA, tenantIDA := setupTestOperationsService()
	svcB, tenantIDB := setupTestOperationsService()

	// Create Customer for Tenant A
	custA, err := svcA.CreateCustomer(ctx, tenantIDA, &dto.CreateCustomerRequest{
		CustomerName: "Kisan Traders",
		Phone:        "9811111111",
	})
	require.NoError(t, err)
	assert.True(t, custA.CurrentBalance.IsZero())

	// 1. Manually set new balance to 2500
	newBal := decimal.NewFromFloat(2500.00)
	custUpdated1, err := svcA.AdjustCustomerBalance(ctx, tenantIDA, custA.ID, &dto.AdjustCustomerBalanceRequest{
		NewBalance: &newBal,
		Reason:     "Initial ledger balance adjustment",
	})
	require.NoError(t, err)
	assert.True(t, custUpdated1.CurrentBalance.Equal(newBal))

	// 2. Adjust with delta (-500)
	delta := decimal.NewFromFloat(-500.00)
	custUpdated2, err := svcA.AdjustCustomerBalance(ctx, tenantIDA, custA.ID, &dto.AdjustCustomerBalanceRequest{
		AdjustmentAmount: &delta,
		Reason:           "Discount credit waiver",
	})
	require.NoError(t, err)
	assert.True(t, custUpdated2.CurrentBalance.Equal(decimal.NewFromFloat(2000.00)))

	// Verify adjustments list
	adjs, total, err := svcA.ListCustomerAdjustments(ctx, tenantIDA, custA.ID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, adjs, 2)
	assert.True(t, adjs[0].PreviousBalance.IsZero())
	assert.True(t, adjs[0].NewBalance.Equal(newBal))
	assert.True(t, adjs[0].AdjustmentAmount.Equal(newBal))
	assert.True(t, adjs[1].PreviousBalance.Equal(decimal.NewFromFloat(2500.00)))
	assert.True(t, adjs[1].NewBalance.Equal(decimal.NewFromFloat(2000.00)))
	assert.True(t, adjs[1].AdjustmentAmount.Equal(delta))

	// 3. Tenant Isolation Check: Tenant B cannot adjust or view Tenant A's customer
	_, err = svcB.AdjustCustomerBalance(ctx, tenantIDB, custA.ID, &dto.AdjustCustomerBalanceRequest{
		NewBalance: &newBal,
		Reason:     "Cross tenant attempt",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid customer for tenant")

	_, _, err = svcB.ListCustomerAdjustments(ctx, tenantIDB, custA.ID, 1, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "customer not found")
}

func TestOperationsService_TodayOverview_Empty(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	overview, err := svc.GetTodayOverview(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, overview)

	assert.Equal(t, 0, overview.Attendance.Present)
	assert.Equal(t, 0, overview.Attendance.Total)
	assert.False(t, overview.Attendance.HasRecords)
	assert.True(t, overview.LineSale.IsZero())
	assert.True(t, overview.CounterSale.IsZero())
	assert.True(t, overview.EmployeeAdvance.IsZero())
	assert.Equal(t, struct{}{}, overview.CurrentItem)
}

func TestOperationsService_TodayOverview_NormalAndMultiple(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()
	today := todayString()

	// 1. Create 3 employees
	emp1, err := svc.CreateEmployee(ctx, tenantID, &dto.CreateEmployeeRequest{
		Name:   "Worker Alpha",
		Phone:  "9876543210",
		Salary: decimal.NewFromFloat(15000),
		Status: "active",
	})
	require.NoError(t, err)

	emp2, err := svc.CreateEmployee(ctx, tenantID, &dto.CreateEmployeeRequest{
		Name:   "Worker Beta",
		Phone:  "9876543211",
		Salary: decimal.NewFromFloat(12000),
		Status: "active",
	})
	require.NoError(t, err)

	emp3, err := svc.CreateEmployee(ctx, tenantID, &dto.CreateEmployeeRequest{
		Name:   "Worker Gamma",
		Phone:  "9876543212",
		Salary: decimal.NewFromFloat(10000),
		Status: "active",
	})
	require.NoError(t, err)
	_ = emp3

	// 2. Mark attendance for emp1 (present) and emp2 (absent)
	_, err = svc.RecordAttendance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: emp1.ID,
		Date:       today,
		Status:     "present",
	})
	require.NoError(t, err)

	_, err = svc.RecordAttendance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAttendanceRequest{
		EmployeeID: emp2.ID,
		Date:       today,
		Status:     "absent",
	})
	require.NoError(t, err)

	// 3. Record advances for emp1 (1500) and emp2 (2000)
	_, err = svc.RecordAdvance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAdvanceRequest{
		EmployeeID: emp1.ID,
		Date:       today,
		Amount:     decimal.NewFromFloat(1500.00),
	})
	require.NoError(t, err)

	_, err = svc.RecordAdvance(ctx, tenantID, auth.RoleTenantAdmin, &dto.RecordAdvanceRequest{
		EmployeeID: emp2.ID,
		Date:       today,
		Amount:     decimal.NewFromFloat(2000.00),
	})
	require.NoError(t, err)

	// 4. Record a purchase item
	purch, err := svc.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreatePurchaseRequest{
		Item:        "Raw Milk 500L",
		Quantity:    500,
		TotalAmount: decimal.NewFromFloat(25000.00),
		TotalPaid:   decimal.NewFromFloat(25000.00),
	})
	require.NoError(t, err)
	require.NotNil(t, purch)

	// 5. Update stats mock with today's aggregates
	_ = svc.statsRepo.Upsert(ctx, &domain.TenantDailyStats{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Date:              today,
		LineSaleAmount:    decimal.NewFromFloat(12450.00),
		CounterSaleAmount: decimal.NewFromFloat(8750.00),
		AdvanceAmount:     decimal.NewFromFloat(3500.00),
		AttendancePresent: 1,
		AttendanceAbsent:  1,
	})

	overview, err := svc.GetTodayOverview(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, overview)

	// Verify Staff Attendance: 1 present / 3 total employees
	assert.Equal(t, 1, overview.Attendance.Present)
	assert.Equal(t, 3, overview.Attendance.Total)
	assert.Equal(t, 1, overview.Attendance.Absent)
	assert.True(t, overview.Attendance.HasRecords)

	// Verify Sales & Advances
	assert.True(t, overview.LineSale.Equal(decimal.NewFromFloat(12450.00)))
	assert.True(t, overview.CounterSale.Equal(decimal.NewFromFloat(8750.00)))
	assert.True(t, overview.EmployeeAdvance.Equal(decimal.NewFromFloat(3500.00)))

	// Verify Current Item
	ci, ok := overview.CurrentItem.(*domain.CurrentItemOverview)
	require.True(t, ok)
	assert.Equal(t, "Raw Milk 500L", ci.Name)
	assert.Equal(t, 500, ci.Quantity)
	assert.True(t, ci.TotalAmount.Equal(decimal.NewFromFloat(25000.00)))
	assert.True(t, ci.IsToday)
	assert.Equal(t, "purchase", ci.Source)

	// Also verify that GetFinancialMetrics contains the TodayOverview
	metrics, err := svc.GetFinancialMetrics(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, metrics)
	require.NotNil(t, metrics.TodayOverview)
	assert.Equal(t, 1, metrics.TodayOverview.Attendance.Present)
	assert.Equal(t, 3, metrics.TodayOverview.Attendance.Total)
	assert.True(t, metrics.TodayOverview.LineSale.Equal(decimal.NewFromFloat(12450.00)))
}

func TestOperationsService_TodayOverview_DateFiltering(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()
	today := todayString()
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// Store yesterday stats
	_ = svc.statsRepo.Upsert(ctx, &domain.TenantDailyStats{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Date:              yesterday,
		LineSaleAmount:    decimal.NewFromFloat(50000.00),
		CounterSaleAmount: decimal.NewFromFloat(30000.00),
		AdvanceAmount:     decimal.NewFromFloat(5000.00),
		AttendancePresent: 4,
		AttendanceAbsent:  0,
	})

	// For today, empty stats (no sales today)
	_ = svc.statsRepo.Upsert(ctx, &domain.TenantDailyStats{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Date:              today,
		LineSaleAmount:    decimal.Zero,
		CounterSaleAmount: decimal.Zero,
		AdvanceAmount:     decimal.Zero,
		AttendancePresent: 0,
		AttendanceAbsent:  0,
	})

	overview, err := svc.GetTodayOverview(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, overview)

	// Today's metrics must strictly be for today, not yesterday!
	assert.True(t, overview.LineSale.IsZero())
	assert.True(t, overview.CounterSale.IsZero())
	assert.True(t, overview.EmployeeAdvance.IsZero())
	assert.Equal(t, 0, overview.Attendance.Present)
	assert.False(t, overview.Attendance.HasRecords)
}

func TestOperationsService_TodayOverview_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	svcA, tenantIDA := setupTestOperationsService()
	svcB, tenantIDB := setupTestOperationsService()
	today := todayString()

	// Tenant A has 2 present, ₹20,000 line sale, ₹5,000 counter sale, ₹1,000 advance
	_ = svcA.statsRepo.Upsert(ctx, &domain.TenantDailyStats{
		ID:                uuid.New(),
		TenantID:          tenantIDA,
		Date:              today,
		LineSaleAmount:    decimal.NewFromFloat(20000.00),
		CounterSaleAmount: decimal.NewFromFloat(5000.00),
		AdvanceAmount:     decimal.NewFromFloat(1000.00),
		AttendancePresent: 2,
		AttendanceAbsent:  0,
	})

	overviewB, err := svcB.GetTodayOverview(ctx, tenantIDB)
	require.NoError(t, err)
	require.NotNil(t, overviewB)

	// Tenant B must have zero metrics, completely isolated from Tenant A
	assert.True(t, overviewB.LineSale.IsZero())
	assert.True(t, overviewB.CounterSale.IsZero())
	assert.True(t, overviewB.EmployeeAdvance.IsZero())
	assert.Equal(t, 0, overviewB.Attendance.Present)
}

func TestOperationsService_FinancialConsistencyOnMutationsAndDeletions(t *testing.T) {
	ctx := context.Background()
	svc, tenantID := setupTestOperationsService()

	t.Run("Purchase lifecycle updates TotalPayable and CashBalance atomically", func(t *testing.T) {
		// Initial balances: Cash = 50,000, TotalPayable = 0
		metricsBefore, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.True(t, metricsBefore.TotalPayable.IsZero())
		assert.Equal(t, 50000.0, metricsBefore.CashBalance.InexactFloat64())

		// 1. Create Purchase: Total 5000, Paid 2000 cash, Remaining = 3000
		purch, err := svc.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreatePurchaseRequest{
			Item:        "Wholesale Rice Bags",
			Quantity:    10,
			TotalAmount: decimal.NewFromFloat(5000),
			TotalPaid:   decimal.NewFromFloat(2000),
		})
		require.NoError(t, err)

		metricsAfterCreate, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 3000.0, metricsAfterCreate.TotalPayable.InexactFloat64())
		assert.Equal(t, 48000.0, metricsAfterCreate.CashBalance.InexactFloat64())

		// 2. Update Purchase: Total 6000, Paid 3000 cash, Remaining = 3000 (delta total = +1000, delta paid = +1000)
		newTotal := decimal.NewFromFloat(6000)
		newPaid := decimal.NewFromFloat(3000)
		_, err = svc.UpdatePurchase(ctx, tenantID, purch.ID, auth.RoleTenantAdmin, &dto.UpdatePurchaseRequest{
			TotalAmount: &newTotal,
			TotalPaid:   &newPaid,
		})
		require.NoError(t, err)

		metricsAfterUpdate, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 3000.0, metricsAfterUpdate.TotalPayable.InexactFloat64())
		assert.Equal(t, 47000.0, metricsAfterUpdate.CashBalance.InexactFloat64())

		// 3. Delete Purchase: reverses 3000 payable (becomes 0), and refunds 3000 cash (becomes 50,000)
		err = svc.DeletePurchase(ctx, tenantID, purch.ID, auth.RoleTenantAdmin)
		require.NoError(t, err)

		metricsAfterDelete, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 0.0, metricsAfterDelete.TotalPayable.InexactFloat64())
		assert.Equal(t, 50000.0, metricsAfterDelete.CashBalance.InexactFloat64())
	})

	t.Run("Line sale lifecycle updates customer balance and summary receivables atomically", func(t *testing.T) {
		// Create customer
		cust, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
			CustomerName: "Ramesh Customer",
		})
		require.NoError(t, err)
		assert.True(t, cust.CurrentBalance.IsZero())

		// Initial cash: 50,000, Receivables: 0
		metricsBefore, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.True(t, metricsBefore.TotalReceivable.IsZero())

		// 1. Create Line Sale: Total 10,000, Received 4000 cash, Balance/Receivable = 6000
		sale, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(10000),
			TotalCashIn: decimal.NewFromFloat(4000),
		})
		require.NoError(t, err)

		metricsAfterSale, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 6000.0, metricsAfterSale.TotalReceivable.InexactFloat64())
		assert.Equal(t, 54000.0, metricsAfterSale.CashBalance.InexactFloat64())

		// Check customer balance
		custUpdated, err := svc.GetCustomerByID(ctx, tenantID, cust.ID)
		require.NoError(t, err)
		assert.Equal(t, 6000.0, custUpdated.CurrentBalance.InexactFloat64())

		// 2. Update Line Sale: Total 12,000, Received 5000 cash, Balance = 7000 (delta remaining = +1000, delta cash = +1000)
		newTotal := decimal.NewFromFloat(12000)
		newCashIn := decimal.NewFromFloat(5000)
		_, err = svc.UpdateLineSale(ctx, tenantID, sale.ID, auth.RoleTenantAdmin, &dto.UpdateLineSaleRequest{
			TotalAmount: &newTotal,
			TotalCashIn: &newCashIn,
		})
		require.NoError(t, err)

		metricsAfterUpdate, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 7000.0, metricsAfterUpdate.TotalReceivable.InexactFloat64())
		assert.Equal(t, 55000.0, metricsAfterUpdate.CashBalance.InexactFloat64())

		// 3. Delete Line Sale: removes 7000 from receivable, reverts 5000 cash
		err = svc.DeleteLineSale(ctx, tenantID, sale.ID, auth.RoleTenantAdmin)
		require.NoError(t, err)

		metricsAfterDelete, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 0.0, metricsAfterDelete.TotalReceivable.InexactFloat64())
		assert.Equal(t, 50000.0, metricsAfterDelete.CashBalance.InexactFloat64())

		custAfterDelete, err := svc.GetCustomerByID(ctx, tenantID, cust.ID)
		require.NoError(t, err)
		assert.Equal(t, 0.0, custAfterDelete.CurrentBalance.InexactFloat64())
	})

	t.Run("Counter sale lifecycle updates cash/bank and daily stats", func(t *testing.T) {
		// 1. Create counter sale: 2000 total (1500 cash, 500 bank)
		bank, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
			BankName:      "HDFC Current",
			AccountNumber: "1234567890",
		})
		require.NoError(t, err)

		sale, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:          "Counter Items",
			Price:         decimal.NewFromFloat(2000),
			TotalAmount:   decimal.NewFromFloat(2000),
			PaymentMethod: "split",
			Cash:          decimal.NewFromFloat(1500),
			BankAmount:    decimal.NewFromFloat(500),
			BankID:        &bank.ID,
		})
		require.NoError(t, err)

		metrics, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 51500.0, metrics.CashBalance.InexactFloat64())
		assert.Equal(t, 100500.0, metrics.BankBalance.InexactFloat64())

		// 2. Delete counter sale: reverts balances back
		err = svc.DeleteCounterSale(ctx, tenantID, sale.ID, auth.RoleTenantAdmin)
		require.NoError(t, err)

		metricsAfterDelete, err := svc.GetFinancialMetrics(ctx, tenantID)
		require.NoError(t, err)
		assert.Equal(t, 50000.0, metricsAfterDelete.CashBalance.InexactFloat64())
		assert.Equal(t, 100000.0, metricsAfterDelete.BankBalance.InexactFloat64())
	})
}

// ----------------------------------------------------
// Customer Association & Live Balance Tests
// ----------------------------------------------------

func TestOperationsService_PurchaseCustomerAssociation(t *testing.T) {
	ctx := context.Background()
	svc, tenantID, custRepo, _, _, _, _, _ := setupTestOperationsServiceWithRepos()

	// 1. Create a customer for tenantID
	cust, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName:   "Alpha Materials Supplier",
		OpeningBalance: decimal.NewFromFloat(5000),
		Status:         "active",
	})
	require.NoError(t, err)

	// Another tenant and customer for isolation check
	otherTenantID := uuid.New()
	otherCust := &domain.TenantCustomer{
		ID:             uuid.New(),
		TenantID:       otherTenantID,
		CustomerName:   "Beta Other Tenant Supplier",
		CurrentBalance: decimal.Zero,
		Status:         "active",
	}
	_ = custRepo.Create(ctx, otherCust)

	// Inactive customer in same tenant
	inactiveCust, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName:   "Inactive Supplier",
		OpeningBalance: decimal.Zero,
		Status:         "inactive",
	})
	require.NoError(t, err)

	t.Run("Successfully associate customer with purchase", func(t *testing.T) {
		purch, err := svc.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreatePurchaseRequest{
			CustomerID:  &cust.ID,
			Item:        "Packaging Raw Materials",
			Quantity:    10,
			TotalAmount: decimal.NewFromFloat(15000),
			TotalPaid:   decimal.NewFromFloat(10000),
		})
		require.NoError(t, err)
		require.NotNil(t, purch.CustomerID)
		assert.Equal(t, cust.ID, *purch.CustomerID)
		assert.Equal(t, "Alpha Materials Supplier", purch.CustomerName)

		// Fetch purchase by ID
		fetched, err := svc.GetPurchaseByID(ctx, tenantID, purch.ID)
		require.NoError(t, err)
		require.NotNil(t, fetched.CustomerID)
		assert.Equal(t, cust.ID, *fetched.CustomerID)
		assert.Equal(t, "Alpha Materials Supplier", fetched.CustomerName)
	})

	t.Run("Purchase without customer succeeds (backward compatibility)", func(t *testing.T) {
		purch, err := svc.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreatePurchaseRequest{
			Item:        "Office Stationery",
			Quantity:    2,
			TotalAmount: decimal.NewFromFloat(500),
			TotalPaid:   decimal.NewFromFloat(500),
		})
		require.NoError(t, err)
		assert.Nil(t, purch.CustomerID)
		assert.Empty(t, purch.CustomerName)
	})

	t.Run("Customer from another tenant is rejected", func(t *testing.T) {
		_, err := svc.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreatePurchaseRequest{
			CustomerID:  &otherCust.ID,
			Item:        "Unauthorized Purchase",
			Quantity:    1,
			TotalAmount: decimal.NewFromFloat(1000),
			TotalPaid:   decimal.Zero,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid customer for tenant")
	})

	t.Run("Inactive customer is rejected", func(t *testing.T) {
		_, err := svc.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreatePurchaseRequest{
			CustomerID:  &inactiveCust.ID,
			Item:        "Inactive Purchase",
			Quantity:    1,
			TotalAmount: decimal.NewFromFloat(1000),
			TotalPaid:   decimal.Zero,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "customer is not active")
	})

	t.Run("Non-existent customer ID is rejected", func(t *testing.T) {
		randomID := uuid.New()
		_, err := svc.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreatePurchaseRequest{
			CustomerID:  &randomID,
			Item:        "Fake Customer Purchase",
			Quantity:    1,
			TotalAmount: decimal.NewFromFloat(1000),
			TotalPaid:   decimal.Zero,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid customer for tenant")
	})

	t.Run("Update purchase modifies customer association", func(t *testing.T) {
		purch, err := svc.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreatePurchaseRequest{
			Item:        "Test Item",
			Quantity:    1,
			TotalAmount: decimal.NewFromFloat(1000),
			TotalPaid:   decimal.Zero,
		})
		require.NoError(t, err)
		assert.Nil(t, purch.CustomerID)

		newQty := 2
		newTotal := decimal.NewFromFloat(2000)
		newPaid := decimal.NewFromFloat(500)
		updated, err := svc.UpdatePurchase(ctx, tenantID, purch.ID, auth.RoleTenantAdmin, &dto.UpdatePurchaseRequest{
			CustomerID:  &cust.ID,
			Item:        "Test Item Updated",
			Quantity:    &newQty,
			TotalAmount: &newTotal,
			TotalPaid:   &newPaid,
		})
		require.NoError(t, err)
		require.NotNil(t, updated.CustomerID)
		assert.Equal(t, cust.ID, *updated.CustomerID)
		assert.Equal(t, "Alpha Materials Supplier", updated.CustomerName)
	})
}

func TestOperationsService_CustomerBalance(t *testing.T) {
	ctx := context.Background()
	svc, tenantID, custRepo, _, _, _, _, _ := setupTestOperationsServiceWithRepos()

	// 1. Customer with positive balance (outstanding)
	cPositive, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName:   "Outstanding Customer",
		OpeningBalance: decimal.NewFromFloat(12500.50),
		Status:         "active",
	})
	require.NoError(t, err)

	// 2. Customer with zero balance
	cZero, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName:   "Zero Balance Customer",
		OpeningBalance: decimal.Zero,
		Status:         "active",
	})
	require.NoError(t, err)

	// 3. Customer with negative balance (credit/advance)
	cNegative, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName:   "Credit Advance Customer",
		OpeningBalance: decimal.NewFromFloat(-3400.00),
		Status:         "active",
	})
	require.NoError(t, err)

	// 4. Other tenant customer
	otherTenantID := uuid.New()
	cOther := &domain.TenantCustomer{
		ID:             uuid.New(),
		TenantID:       otherTenantID,
		CustomerName:   "Other Tenant Cust",
		CurrentBalance: decimal.NewFromFloat(5000),
		Status:         "active",
	}
	_ = custRepo.Create(ctx, cOther)

	t.Run("Positive outstanding balance returned correctly", func(t *testing.T) {
		bal, err := svc.GetCustomerBalance(ctx, tenantID, cPositive.ID)
		require.NoError(t, err)
		assert.Equal(t, cPositive.ID, bal.CustomerID)
		assert.Equal(t, "Outstanding Customer", bal.CustomerName)
		assert.True(t, bal.CurrentBalance.Equal(decimal.NewFromFloat(12500.50)))
		assert.Equal(t, "active", bal.Status)
	})

	t.Run("Zero balance returned correctly", func(t *testing.T) {
		bal, err := svc.GetCustomerBalance(ctx, tenantID, cZero.ID)
		require.NoError(t, err)
		assert.True(t, bal.CurrentBalance.IsZero())
	})

	t.Run("Negative credit/advance balance returned correctly", func(t *testing.T) {
		bal, err := svc.GetCustomerBalance(ctx, tenantID, cNegative.ID)
		require.NoError(t, err)
		assert.True(t, bal.CurrentBalance.Equal(decimal.NewFromFloat(-3400.00)))
		assert.True(t, bal.CurrentBalance.IsNegative())
	})

	t.Run("Customer from another tenant is rejected", func(t *testing.T) {
		_, err := svc.GetCustomerBalance(ctx, tenantID, cOther.ID)
		require.Error(t, err)
	})
}

// ----------------------------------------------------
// Multi-Bank & Split Payment Tests for Sales
// ----------------------------------------------------

func TestOperationsService_LineSale_MultiBankAndSplitPayments(t *testing.T) {
	ctx := context.Background()
	svc, tenantID, custRepo, bankRepo, _, _, _, _ := setupTestOperationsServiceWithRepos()

	// Seed customer
	cust, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName:   "Retail Wholesale Mart",
		OpeningBalance: decimal.Zero,
		Status:         "active",
	})
	require.NoError(t, err)

	// Seed multiple banks
	bankHDFC, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "HDFC Bank",
		AccountNumber:  "HDFC1001",
		OpeningBalance: decimal.NewFromFloat(50000),
	})
	require.NoError(t, err)

	bankSBI, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "SBI Bank",
		AccountNumber:  "SBI2002",
		OpeningBalance: decimal.NewFromFloat(30000),
	})
	require.NoError(t, err)

	bankICICI, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "ICICI Bank",
		AccountNumber:  "ICICI3003",
		OpeningBalance: decimal.NewFromFloat(20000),
	})
	require.NoError(t, err)

	// Inactive bank in same tenant
	bankInactive, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:      "Inactive Bank",
		AccountNumber: "INACT000",
		Status:        "inactive",
	})
	require.NoError(t, err)

	// Bank belonging to another tenant
	otherTenantID := uuid.New()
	bankOther := &domain.TenantBank{
		ID:             uuid.New(),
		TenantID:       otherTenantID,
		BankName:       "Other Tenant Bank",
		AccountNumber:  "OTHER999",
		CurrentBalance: decimal.NewFromFloat(10000),
		Status:         "active",
	}
	_ = bankRepo.Create(ctx, bankOther)

	t.Run("Cash-only payment", func(t *testing.T) {
		cashAmt := decimal.NewFromFloat(5000)
		sale, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(10000),
			CashAmount:  &cashAmt,
		})
		require.NoError(t, err)
		assert.Equal(t, 5000.0, sale.TotalCashIn.InexactFloat64())
		assert.Equal(t, 0.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 5000.0, sale.Balance.InexactFloat64())
	})

	t.Run("Bank-only single bank payment", func(t *testing.T) {
		sale, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(10000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(6000)},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 0.0, sale.TotalCashIn.InexactFloat64())
		assert.Equal(t, 6000.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 4000.0, sale.Balance.InexactFloat64())

		// Verify HDFC bank balance was credited
		hdfc, err := svc.GetBankByID(ctx, tenantID, bankHDFC.ID)
		require.NoError(t, err)
		assert.Equal(t, 56000.0, hdfc.CurrentBalance.InexactFloat64())
	})

	t.Run("Cash + single Bank payment", func(t *testing.T) {
		cashAmt := decimal.NewFromFloat(3000)
		sale, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(10000),
			CashAmount:  &cashAmt,
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankSBI.ID, Amount: decimal.NewFromFloat(4000)},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 3000.0, sale.TotalCashIn.InexactFloat64())
		assert.Equal(t, 4000.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 3000.0, sale.Balance.InexactFloat64())
	})

	t.Run("Cash + Multiple Banks payment (Cash + HDFC + SBI)", func(t *testing.T) {
		cashAmt := decimal.NewFromFloat(2000)
		sale, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(10000),
			CashAmount:  &cashAmt,
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(4000)},
				{BankID: bankSBI.ID, Amount: decimal.NewFromFloat(4000)},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 2000.0, sale.TotalCashIn.InexactFloat64())
		assert.Equal(t, 8000.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 0.0, sale.Balance.InexactFloat64()) // Full payment collected!

		// HDFC balance was 56000 + 4000 = 60000
		hdfc, _ := svc.GetBankByID(ctx, tenantID, bankHDFC.ID)
		assert.Equal(t, 60000.0, hdfc.CurrentBalance.InexactFloat64())

		// SBI balance was 30000 + 4000 (prev) + 4000 = 38000
		sbi, _ := svc.GetBankByID(ctx, tenantID, bankSBI.ID)
		assert.Equal(t, 38000.0, sbi.CurrentBalance.InexactFloat64())
	})

	t.Run("Multiple Banks only (HDFC + SBI + ICICI)", func(t *testing.T) {
		sale, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(10000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(5000)},
				{BankID: bankSBI.ID, Amount: decimal.NewFromFloat(3000)},
				{BankID: bankICICI.ID, Amount: decimal.NewFromFloat(2000)},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 0.0, sale.TotalCashIn.InexactFloat64())
		assert.Equal(t, 10000.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 0.0, sale.Balance.InexactFloat64())
	})

	t.Run("Duplicate bank in bank payments is rejected", func(t *testing.T) {
		_, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(10000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(3000)},
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(2000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate bank")
	})

	t.Run("Bank from another tenant is rejected", func(t *testing.T) {
		_, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(5000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankOther.ID, Amount: decimal.NewFromFloat(2000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid bank account")
	})

	t.Run("Inactive bank is rejected", func(t *testing.T) {
		_, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(5000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankInactive.ID, Amount: decimal.NewFromFloat(2000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bank account is inactive")
	})

	t.Run("Negative cash amount is rejected", func(t *testing.T) {
		negCash := decimal.NewFromFloat(-100)
		_, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(5000),
			CashAmount:  &negCash,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cash amount cannot be negative")
	})

	t.Run("Zero or negative bank amount is rejected", func(t *testing.T) {
		_, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(5000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(-500)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bank payment amount must be greater than zero")
	})

	t.Run("Payment exceeding total invoice amount is rejected", func(t *testing.T) {
		cashAmt := decimal.NewFromFloat(6000)
		_, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(10000),
			CashAmount:  &cashAmt,
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(5000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed sale total")
	})

	t.Run("Customer from another tenant is rejected", func(t *testing.T) {
		otherCust := &domain.TenantCustomer{
			ID:           uuid.New(),
			TenantID:     otherTenantID,
			CustomerName: "Beta Other Customer",
			Status:       "active",
		}
		_ = custRepo.Create(ctx, otherCust)

		_, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  otherCust.ID,
			TotalAmount: decimal.NewFromFloat(5000),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid customer for tenant")
	})
}

func TestOperationsService_CounterSale_MultiBankAndSplitPayments(t *testing.T) {
	ctx := context.Background()
	svc, tenantID, _, bankRepo, _, _, _, _ := setupTestOperationsServiceWithRepos()

	bankHDFC, _ := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "HDFC Bank",
		AccountNumber:  "HDFC101",
		OpeningBalance: decimal.NewFromFloat(10000),
	})
	bankSBI, _ := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "SBI Bank",
		AccountNumber:  "SBI202",
		OpeningBalance: decimal.NewFromFloat(10000),
	})

	// Bank belonging to another tenant
	otherTenantID := uuid.New()
	bankOther := &domain.TenantBank{
		ID:             uuid.New(),
		TenantID:       otherTenantID,
		BankName:       "Other Tenant Bank",
		AccountNumber:  "OTHER888",
		CurrentBalance: decimal.NewFromFloat(10000),
		Status:         "active",
	}
	_ = bankRepo.Create(ctx, bankOther)

	t.Run("Cash-only counter sale", func(t *testing.T) {
		cashAmt := decimal.NewFromFloat(1500)
		sale, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:        "Retail item",
			TotalAmount: decimal.NewFromFloat(1500),
			CashAmount:  &cashAmt,
		})
		require.NoError(t, err)
		assert.Equal(t, 1500.0, sale.Cash.InexactFloat64())
		assert.Equal(t, 0.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 0.0, sale.Account.InexactFloat64())
	})

	t.Run("Bank-only counter sale with single bank", func(t *testing.T) {
		sale, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:        "Wholesale Box",
			TotalAmount: decimal.NewFromFloat(2000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(2000)},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 0.0, sale.Cash.InexactFloat64())
		assert.Equal(t, 2000.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 0.0, sale.Account.InexactFloat64())
	})

	t.Run("Cash + Multiple Banks counter sale (Cash + HDFC + SBI)", func(t *testing.T) {
		cashAmt := decimal.NewFromFloat(500)
		sale, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:        "Mixed Sale",
			TotalAmount: decimal.NewFromFloat(3000),
			CashAmount:  &cashAmt,
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(1000)},
				{BankID: bankSBI.ID, Amount: decimal.NewFromFloat(1500)},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 500.0, sale.Cash.InexactFloat64())
		assert.Equal(t, 2500.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 0.0, sale.Account.InexactFloat64())
	})

	t.Run("Multiple Banks only counter sale (HDFC + SBI)", func(t *testing.T) {
		sale, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:        "Digital Payment Only",
			TotalAmount: decimal.NewFromFloat(2500),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(1500)},
				{BankID: bankSBI.ID, Amount: decimal.NewFromFloat(1000)},
			},
		})
		require.NoError(t, err)
		assert.Equal(t, 0.0, sale.Cash.InexactFloat64())
		assert.Equal(t, 2500.0, sale.BankAmount.InexactFloat64())
		assert.Equal(t, 0.0, sale.Account.InexactFloat64())
	})

	t.Run("Duplicate bank in counter sale is rejected", func(t *testing.T) {
		_, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:        "Item",
			TotalAmount: decimal.NewFromFloat(2000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(1000)},
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(1000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate bank")
	})

	t.Run("Bank from another tenant is rejected in counter sale", func(t *testing.T) {
		_, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:        "Item",
			TotalAmount: decimal.NewFromFloat(2000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankOther.ID, Amount: decimal.NewFromFloat(1000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid bank account")
	})

	t.Run("Payment exceeding total allowed in counter sale is rejected", func(t *testing.T) {
		cashAmt := decimal.NewFromFloat(1500)
		_, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:        "Item",
			TotalAmount: decimal.NewFromFloat(2000),
			CashAmount:  &cashAmt,
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bankHDFC.ID, Amount: decimal.NewFromFloat(1000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot exceed sale total")
	})
}

func TestOperationsService_TransactionRollbackScenarios(t *testing.T) {
	ctx := context.Background()
	svc, tenantID, _, bankRepo, _, _, _, _ := setupTestOperationsServiceWithRepos()

	cust, err := svc.CreateCustomer(ctx, tenantID, &dto.CreateCustomerRequest{
		CustomerName: "Rollback Customer",
		Status:       "active",
	})
	require.NoError(t, err)

	bank, err := svc.CreateBank(ctx, tenantID, &dto.CreateBankRequest{
		BankName:       "HDFC Test Bank",
		AccountNumber:  "HDFC9999",
		OpeningBalance: decimal.NewFromFloat(10000),
	})
	require.NoError(t, err)

	t.Run("Line sale with bank payment fails when bank transaction persistence fails", func(t *testing.T) {
		// Set failOnTx to trigger failure during transaction
		bankRepo.failOnTx = true
		defer func() { bankRepo.failOnTx = false }()

		_, err := svc.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateLineSaleRequest{
			CustomerID:  cust.ID,
			TotalAmount: decimal.NewFromFloat(5000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bank.ID, Amount: decimal.NewFromFloat(2000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "simulated bank transaction persistence failure")
	})

	t.Run("Counter sale with bank payment fails when bank transaction persistence fails", func(t *testing.T) {
		bankRepo.failOnTx = true
		defer func() { bankRepo.failOnTx = false }()

		_, err := svc.CreateCounterSale(ctx, tenantID, auth.RoleTenantAdmin, &dto.CreateCounterSaleRequest{
			Item:        "Counter Rollback Item",
			TotalAmount: decimal.NewFromFloat(5000),
			BankPayments: []dto.BankPaymentSplitRequest{
				{BankID: bank.ID, Amount: decimal.NewFromFloat(2000)},
			},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "simulated bank transaction persistence failure")
	})
}
