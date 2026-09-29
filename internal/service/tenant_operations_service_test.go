package service

import (
	"context"
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
	custs map[uuid.UUID]*domain.TenantCustomer
}

func newMockOpCustomerRepo() *mockOpCustomerRepo {
	return &mockOpCustomerRepo{custs: make(map[uuid.UUID]*domain.TenantCustomer)}
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
	banks map[uuid.UUID]*domain.TenantBank
}

func newMockOpBankRepo() *mockOpBankRepo {
	return &mockOpBankRepo{banks: make(map[uuid.UUID]*domain.TenantBank)}
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

func (m *mockOpDailyStatsRepo) ComputeAndSyncDailyStats(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	s := &domain.TenantDailyStats{
		ID:       uuid.New(),
		TenantID: tenantID,
		Date:     date,
	}
	m.stats[tenantID.String()+":"+date] = s
	return s, nil
}

func setupTestOperationsService() (*TenantOperationsService, uuid.UUID) {
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

	// Salary balance should reflect: 0 (initial) - 2000 (advance) + 600 (OT) = -1400
	salary, err := svc.GetSalaryByEmployeeID(ctx, tenantID, emp.ID)
	require.NoError(t, err)
	assert.True(t, salary.Balance.Equal(decimal.NewFromFloat(-1400.00)))

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
