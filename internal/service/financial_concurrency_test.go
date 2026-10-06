package service_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/logger"
	"github.com/Varunjp/vyavsa/internal/service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Thread-safe mock implementations for concurrent testing

type threadSafeTransactor struct{}

func (t *threadSafeTransactor) WithinTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

type failingTransactor struct {
	fail bool
}

func (t *failingTransactor) WithinTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	if t.fail {
		return errors.New("simulated transaction abort")
	}
	return fn(ctx)
}

type threadSafeSummaryRepo struct {
	mu      sync.RWMutex
	summary *domain.TenantFinancialSummary
}

func (r *threadSafeSummaryRepo) Create(ctx context.Context, s *domain.TenantFinancialSummary) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.summary = s
	return nil
}

func (r *threadSafeSummaryRepo) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.summary == nil {
		return nil, errors.New("not found")
	}
	cp := *r.summary
	return &cp, nil
}

func (r *threadSafeSummaryRepo) GetByTenantIDForUpdate(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	return r.GetByTenantID(ctx, tenantID)
}

func (r *threadSafeSummaryRepo) Update(ctx context.Context, s *domain.TenantFinancialSummary) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.summary = s
	return nil
}

func (r *threadSafeSummaryRepo) SyncFromSourceRecords(ctx context.Context, tenantID uuid.UUID) (*domain.TenantFinancialSummary, error) {
	return r.GetByTenantID(ctx, tenantID)
}

func (r *threadSafeSummaryRepo) AdjustBalances(ctx context.Context, tenantID uuid.UUID, cashDelta, bankDelta, receivableDelta, payableDelta decimal.Decimal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.summary == nil {
		r.summary = &domain.TenantFinancialSummary{
			TenantID: tenantID,
		}
	}
	r.summary.CashBalance = r.summary.CashBalance.Add(cashDelta)
	r.summary.BankBalance = r.summary.BankBalance.Add(bankDelta)
	r.summary.TotalReceivable = r.summary.TotalReceivable.Add(receivableDelta)
	r.summary.TotalPayable = r.summary.TotalPayable.Add(payableDelta)
	return nil
}

type threadSafeCustomerRepo struct {
	mu        sync.RWMutex
	customers map[uuid.UUID]*domain.TenantCustomer
}

func (r *threadSafeCustomerRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantCustomer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.customers[id]
	if !ok {
		return nil, errors.New("customer not found")
	}
	return c, nil
}

func (r *threadSafeCustomerRepo) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantCustomer, error) {
	return r.GetByID(ctx, tenantID, id)
}

func (r *threadSafeCustomerRepo) AdjustBalance(ctx context.Context, tenantID, id uuid.UUID, delta decimal.Decimal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.customers[id]
	if !ok {
		return errors.New("customer not found")
	}
	c.CurrentBalance = c.CurrentBalance.Add(delta)
	return nil
}

func (r *threadSafeCustomerRepo) SetBalance(ctx context.Context, tenantID, id uuid.UUID, balance decimal.Decimal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.customers[id]
	if !ok {
		return errors.New("customer not found")
	}
	c.CurrentBalance = balance
	return nil
}

func (r *threadSafeCustomerRepo) Create(ctx context.Context, cust *domain.TenantCustomer) error {
	return nil
}
func (r *threadSafeCustomerRepo) Update(ctx context.Context, cust *domain.TenantCustomer) error {
	return nil
}
func (r *threadSafeCustomerRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return nil
}
func (r *threadSafeCustomerRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantCustomer, int64, error) {
	return nil, 0, nil
}
func (r *threadSafeCustomerRepo) ListAdjustments(ctx context.Context, tenantID, customerID uuid.UUID, page, pageSize int) ([]domain.CustomerBalanceAdjustment, int64, error) {
	return nil, 0, nil
}
func (r *threadSafeCustomerRepo) RecordAdjustment(ctx context.Context, adj *domain.CustomerBalanceAdjustment) error {
	return nil
}

type threadSafeBankRepo struct {
	mu    sync.RWMutex
	banks map[uuid.UUID]*domain.TenantBank
}

func (r *threadSafeBankRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantBank, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.banks[id]
	if !ok {
		return nil, errors.New("bank not found")
	}
	return b, nil
}

func (r *threadSafeBankRepo) AdjustBalance(ctx context.Context, tenantID, id uuid.UUID, delta decimal.Decimal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.banks[id]
	if !ok {
		return errors.New("bank not found")
	}
	b.CurrentBalance = b.CurrentBalance.Add(delta)
	return nil
}

func (r *threadSafeBankRepo) CreateTransaction(ctx context.Context, tx *domain.BankTransaction) error {
	return nil
}
func (r *threadSafeBankRepo) Create(ctx context.Context, bank *domain.TenantBank) error { return nil }
func (r *threadSafeBankRepo) Update(ctx context.Context, bank *domain.TenantBank) error { return nil }
func (r *threadSafeBankRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error  { return nil }
func (r *threadSafeBankRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, search, status string) ([]domain.TenantBank, int64, error) {
	return nil, 0, nil
}
func (r *threadSafeBankRepo) ListTransactions(ctx context.Context, tenantID, bankID uuid.UUID, page, pageSize int) ([]domain.BankTransaction, int64, error) {
	return nil, 0, nil
}

type threadSafeLineSaleRepo struct {
	mu    sync.RWMutex
	sales map[uuid.UUID]*domain.LineSale
}

func (r *threadSafeLineSaleRepo) Create(ctx context.Context, sale *domain.LineSale, payments []domain.LineSalePayment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	sale.ID = uuid.New()
	r.sales[sale.ID] = sale
	return nil
}
func (r *threadSafeLineSaleRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error) {
	return nil, nil
}
func (r *threadSafeLineSaleRepo) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.LineSale, error) {
	return nil, nil
}
func (r *threadSafeLineSaleRepo) Update(ctx context.Context, sale *domain.LineSale) error { return nil }
func (r *threadSafeLineSaleRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return nil
}
func (r *threadSafeLineSaleRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.LineSale, int64, error) {
	return nil, 0, nil
}
func (r *threadSafeLineSaleRepo) CreatePayment(ctx context.Context, payment *domain.LineSalePayment) error {
	return nil
}
func (r *threadSafeLineSaleRepo) ListPaymentsByLineSaleID(ctx context.Context, tenantID, lineSaleID uuid.UUID) ([]domain.LineSalePayment, error) {
	return nil, nil
}

type threadSafePurchaseRepo struct {
	mu        sync.RWMutex
	purchases map[uuid.UUID]*domain.TenantPurchase
}

func (r *threadSafePurchaseRepo) Create(ctx context.Context, purch *domain.TenantPurchase, payments []domain.TenantPurchasePayment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	purch.ID = uuid.New()
	r.purchases[purch.ID] = purch
	return nil
}
func (r *threadSafePurchaseRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error) {
	return nil, nil
}
func (r *threadSafePurchaseRepo) GetByIDForUpdate(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantPurchase, error) {
	return nil, nil
}
func (r *threadSafePurchaseRepo) GetCustomerPurchasesForUpdate(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.TenantPurchase, error) {
	return nil, nil
}
func (r *threadSafePurchaseRepo) Update(ctx context.Context, purch *domain.TenantPurchase) error {
	return nil
}
func (r *threadSafePurchaseRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return nil
}
func (r *threadSafePurchaseRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date string, customerID *uuid.UUID, search string) ([]domain.TenantPurchase, int64, error) {
	return nil, 0, nil
}
func (r *threadSafePurchaseRepo) CreatePayment(ctx context.Context, payment *domain.TenantPurchasePayment) error {
	return nil
}
func (r *threadSafePurchaseRepo) ListPaymentsByPurchaseID(ctx context.Context, tenantID, purchaseID uuid.UUID) ([]domain.TenantPurchasePayment, error) {
	return nil, nil
}
func (r *threadSafePurchaseRepo) ListPaymentsByCustomerID(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.TenantPurchasePayment, error) {
	return nil, nil
}
func (r *threadSafePurchaseRepo) GetCustomerPayableSummary(ctx context.Context, tenantID, customerID uuid.UUID) (decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	return decimal.Zero, decimal.Zero, decimal.Zero, nil
}
func (r *threadSafePurchaseRepo) GetCustomerPayableSummariesBatch(ctx context.Context, tenantID uuid.UUID, customerIDs []uuid.UUID) (map[uuid.UUID]domain.CustomerPayableSummary, error) {
	return nil, nil
}

type threadSafeExpenseRepo struct {
	mu       sync.RWMutex
	expenses map[uuid.UUID]*domain.TenantExpense
}

func (r *threadSafeExpenseRepo) Create(ctx context.Context, exp *domain.TenantExpense, payments []domain.TenantExpensePayment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	exp.ID = uuid.New()
	r.expenses[exp.ID] = exp
	return nil
}
func (r *threadSafeExpenseRepo) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.TenantExpense, error) {
	return nil, nil
}
func (r *threadSafeExpenseRepo) Update(ctx context.Context, exp *domain.TenantExpense) error {
	return nil
}
func (r *threadSafeExpenseRepo) Delete(ctx context.Context, tenantID, id uuid.UUID) error { return nil }
func (r *threadSafeExpenseRepo) List(ctx context.Context, tenantID uuid.UUID, page, pageSize int, date, search string) ([]domain.TenantExpense, int64, error) {
	return nil, 0, nil
}

type mockStatsEnqueuer struct {
	mu       sync.Mutex
	enqueued int
}

func (m *mockStatsEnqueuer) Enqueue(ctx context.Context, tenantID uuid.UUID, date string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enqueued++
	return nil
}

func TestFinancialOperations_Concurrency(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()
	bankID := uuid.New()

	initialCash := decimal.NewFromFloat(100000.00)
	initialBank := decimal.NewFromFloat(200000.00)

	summaryRepo := &threadSafeSummaryRepo{
		summary: &domain.TenantFinancialSummary{
			TenantID:        tenantID,
			CashBalance:     initialCash,
			BankBalance:     initialBank,
			TotalReceivable: decimal.Zero,
			TotalPayable:    decimal.Zero,
		},
	}

	custRepo := &threadSafeCustomerRepo{
		customers: map[uuid.UUID]*domain.TenantCustomer{
			customerID: {ID: customerID, TenantID: tenantID, CustomerName: "Customer 1", CurrentBalance: decimal.Zero},
		},
	}

	bankRepo := &threadSafeBankRepo{
		banks: map[uuid.UUID]*domain.TenantBank{
			bankID: {ID: bankID, TenantID: tenantID, BankName: "HDFC", CurrentBalance: initialBank},
		},
	}

	lineSaleRepo := &threadSafeLineSaleRepo{sales: make(map[uuid.UUID]*domain.LineSale)}
	purchRepo := &threadSafePurchaseRepo{purchases: make(map[uuid.UUID]*domain.TenantPurchase)}
	expRepo := &threadSafeExpenseRepo{expenses: make(map[uuid.UUID]*domain.TenantExpense)}
	statsEnqueuer := &mockStatsEnqueuer{}

	transactor := &threadSafeTransactor{}
	hasher := auth.NewBcryptHasher()
	log := logger.Default().Logger

	opsService := service.NewTenantOperationsService(
		nil, nil, custRepo, bankRepo,
		lineSaleRepo, nil, purchRepo, expRepo,
		nil, nil, nil, nil, nil,
		summaryRepo, transactor, hasher, log,
	)
	opsService.SetDailyStatsWorker(statsEnqueuer)

	// Concurrency scenario:
	// - 50 concurrent line sales:
	//     Total: 1000, Cash: 600, Credit/Receivable: 400
	// - 20 concurrent purchases:
	//     Total: 2000, Bank: 1200, Credit/Payable: 800
	// - 10 concurrent expenses:
	//     Total: 500, Cash: 500
	const numSales = 50
	const numPurchases = 20
	const numExpenses = 10

	var wg sync.WaitGroup
	wg.Add(numSales + numPurchases + numExpenses)

	errChan := make(chan error, numSales+numPurchases+numExpenses)

	ctx := context.Background()

	// Launch 50 concurrent sales
	for i := 0; i < numSales; i++ {
		go func() {
			defer wg.Done()
			req := &dto.CreateLineSaleRequest{
				CustomerID:  customerID,
				TotalAmount: decimal.NewFromFloat(1000.00),
				TotalCashIn: decimal.NewFromFloat(600.00),
			}
			_, err := opsService.CreateLineSale(ctx, tenantID, auth.RoleTenantAdmin, req)
			if err != nil {
				errChan <- err
			}
		}()
	}

	// Launch 20 concurrent purchases
	for i := 0; i < numPurchases; i++ {
		go func() {
			defer wg.Done()
			req := &dto.CreatePurchaseRequest{
				Item:        "Bulk Raw Materials",
				Quantity:    10,
				TotalAmount: decimal.NewFromFloat(2000.00),
				TotalPaid:   decimal.NewFromFloat(1200.00),
				Payments: []dto.PurchasePaymentRequest{
					{
						PaymentMethod: "bank",
						BankID:        &bankID,
						Amount:        decimal.NewFromFloat(1200.00),
					},
				},
			}
			_, err := opsService.CreatePurchase(ctx, tenantID, auth.RoleTenantAdmin, req)
			if err != nil {
				errChan <- err
			}
		}()
	}

	// Launch 10 concurrent expenses
	for i := 0; i < numExpenses; i++ {
		go func() {
			defer wg.Done()
			req := &dto.CreateExpenseRequest{
				Item:          "Electricity Bill",
				Category:      "Utilities",
				TotalAmount:   decimal.NewFromFloat(500.00),
				PaymentMethod: "cash",
			}
			_, err := opsService.CreateExpense(ctx, tenantID, auth.RoleTenantAdmin, req)
			if err != nil {
				errChan <- err
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		require.NoError(t, err, "concurrent operation failed")
	}

	// Verify all operations were recorded
	require.Equal(t, numSales, len(lineSaleRepo.sales), "expected all sales to be recorded")
	require.Equal(t, numPurchases, len(purchRepo.purchases), "expected all purchases to be recorded")
	require.Equal(t, numExpenses, len(expRepo.expenses), "expected all expenses to be recorded")

	// Calculate expected final financial balances:
	// Cash: 100,000 + (50 * 600) - (10 * 500) = 100,000 + 30,000 - 5,000 = 125,000
	expectedCash := initialCash.Add(decimal.NewFromFloat(float64(numSales * 600))).Sub(decimal.NewFromFloat(float64(numExpenses * 500)))

	// Bank: 200,000 - (20 * 1200) = 200,000 - 24,000 = 176,000
	expectedBank := initialBank.Sub(decimal.NewFromFloat(float64(numPurchases * 1200)))

	// Receivable: 50 * 400 = 20,000
	expectedReceivable := decimal.NewFromFloat(float64(numSales * 400))

	// Payable: 20 * 800 = 16,000
	expectedPayable := decimal.NewFromFloat(float64(numPurchases * 800))

	summary, err := summaryRepo.GetByTenantID(ctx, tenantID)
	require.NoError(t, err)

	require.True(t, expectedCash.Equal(summary.CashBalance), "Cash balance mismatch: expected %s, got %s", expectedCash, summary.CashBalance)
	require.True(t, expectedBank.Equal(summary.BankBalance), "Bank balance mismatch: expected %s, got %s", expectedBank, summary.BankBalance)
	require.True(t, expectedReceivable.Equal(summary.TotalReceivable), "Receivable mismatch: expected %s, got %s", expectedReceivable, summary.TotalReceivable)
	require.True(t, expectedPayable.Equal(summary.TotalPayable), "Payable mismatch: expected %s, got %s", expectedPayable, summary.TotalPayable)

	// Verify customer balance: 50 * 400 = 20,000
	cust, err := custRepo.GetByID(ctx, tenantID, customerID)
	require.NoError(t, err)
	require.True(t, expectedReceivable.Equal(cust.CurrentBalance), "Customer balance mismatch: expected %s, got %s", expectedReceivable, cust.CurrentBalance)

	// Verify bank balance: 200,000 - 24,000 = 176,000
	bank, err := bankRepo.GetByID(ctx, tenantID, bankID)
	require.NoError(t, err)
	require.True(t, expectedBank.Equal(bank.CurrentBalance), "Bank entity balance mismatch: expected %s, got %s", expectedBank, bank.CurrentBalance)

	// Verify stats enqueue events count
	statsEnqueuer.mu.Lock()
	defer statsEnqueuer.mu.Unlock()
	require.Equal(t, numSales+numPurchases+numExpenses, statsEnqueuer.enqueued, "expected stats enqueue for all transactions")
}

func TestFinancialOperations_TransactionRollbackConsistency(t *testing.T) {
	tenantID := uuid.New()
	customerID := uuid.New()

	initialCash := decimal.NewFromFloat(50000.00)
	summaryRepo := &threadSafeSummaryRepo{
		summary: &domain.TenantFinancialSummary{
			TenantID:        tenantID,
			CashBalance:     initialCash,
			BankBalance:     decimal.Zero,
			TotalReceivable: decimal.Zero,
			TotalPayable:    decimal.Zero,
		},
	}

	custRepo := &threadSafeCustomerRepo{
		customers: map[uuid.UUID]*domain.TenantCustomer{
			customerID: {ID: customerID, TenantID: tenantID, CustomerName: "Customer Test", CurrentBalance: decimal.Zero},
		},
	}

	lineSaleRepo := &threadSafeLineSaleRepo{sales: make(map[uuid.UUID]*domain.LineSale)}
	failingTx := &failingTransactor{fail: true}
	log := logger.Default().Logger

	opsService := service.NewTenantOperationsService(
		nil, nil, custRepo, nil,
		lineSaleRepo, nil, nil, nil,
		nil, nil, nil, nil, nil,
		summaryRepo, failingTx, auth.NewBcryptHasher(), log,
	)

	req := &dto.CreateLineSaleRequest{
		CustomerID:  customerID,
		TotalAmount: decimal.NewFromFloat(1000.00),
		TotalCashIn: decimal.NewFromFloat(1000.00),
	}

	_, err := opsService.CreateLineSale(context.Background(), tenantID, auth.RoleTenantAdmin, req)
	require.Error(t, err, "expected operation to fail due to transactor abort")

	// Verify no sale was recorded
	require.Equal(t, 0, len(lineSaleRepo.sales), "no sale record should exist after rollback")

	// Verify summary was NOT modified
	summary, err := summaryRepo.GetByTenantID(context.Background(), tenantID)
	require.NoError(t, err)
	require.True(t, initialCash.Equal(summary.CashBalance), "financial summary cash must remain untouched on rollback")
}
