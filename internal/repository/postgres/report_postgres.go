package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// DailyReportPostgres implements repository.DailyReportRepository
type DailyReportPostgres struct {
	pool *pgxpool.Pool
}

// NewDailyReportPostgres creates a new instance of DailyReportPostgres
func NewDailyReportPostgres(pool *pgxpool.Pool) *DailyReportPostgres {
	return &DailyReportPostgres{pool: pool}
}

// MaskAccountNumber masks an account number keeping only the trailing 4 characters for security
func MaskAccountNumber(acc string) string {
	clean := strings.TrimSpace(acc)
	if clean == "" {
		return "Active Account"
	}
	if len(clean) <= 4 {
		return "****" + clean
	}
	return "****" + clean[len(clean)-4:]
}

// GetDailyReportData queries all financial activity for the tenant between start and end
func (r *DailyReportPostgres) GetDailyReportData(
	ctx context.Context,
	tenantID uuid.UUID,
	start, end time.Time,
	loc *time.Location,
) (*domain.DailyReport, error) {
	if loc == nil {
		loc = time.UTC
	}
	exec := GetExecutor(ctx, r.pool)

	// 1. Fetch Tenant Metadata
	var tenantInfo domain.TenantReportInfo
	tenantQuery := `SELECT id, name, email, COALESCE(phone, '') FROM tenants WHERE id = $1`
	err := exec.QueryRow(ctx, tenantQuery, tenantID).Scan(
		&tenantInfo.ID,
		&tenantInfo.Name,
		&tenantInfo.Email,
		&tenantInfo.Phone,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, appErrors.NewNotFound("tenant not found")
		}
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to fetch tenant details: %w", err))
	}

	dateStr := start.In(loc).Format("2006-01-02")
	formattedDate := start.In(loc).Format("02 January 2006")

	report := &domain.DailyReport{
		Tenant:        tenantInfo,
		Date:          dateStr,
		FormattedDate: formattedDate,
		LineSales:     make([]domain.LineSaleReportItem, 0),
		CounterSales:  make([]domain.CounterSaleReportItem, 0),
		Purchases:     make([]domain.PurchaseReportItem, 0),
		Expenses:      make([]domain.ExpenseReportItem, 0),
		BankBalances:  make([]domain.BankBalanceReportItem, 0),
		GeneratedAt:   time.Now().In(loc),
	}

	// 2. Fetch Line Sales
	lineQuery := `
		SELECT id, customer_name, COALESCE(route, ''), COALESCE(salesman, ''), COALESCE(note, ''),
		       total_amount, total_cash_in, bank_amount, balance, created_at
		FROM line_sale
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at ASC
	`
	lineRows, err := exec.Query(ctx, lineQuery, tenantID, start, end)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to query line sales: %w", err))
	}
	defer lineRows.Close()

	var totalLineSales, totalLineCashIn decimal.Decimal
	lineSaleIDs := make([]uuid.UUID, 0)

	for lineRows.Next() {
		var item domain.LineSaleReportItem
		if err := lineRows.Scan(
			&item.ID,
			&item.CustomerName,
			&item.Route,
			&item.Salesman,
			&item.Note,
			&item.TotalAmount,
			&item.CashIn,
			&item.BankAmount,
			&item.Balance,
			&item.CreatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan line sale item: %w", err))
		}
		item.InvoiceNumber = fmt.Sprintf("LS-%s", strings.ToUpper(item.ID.String()[:8]))
		item.Time = item.CreatedAt.In(loc).Format("15:04")
		item.PaymentMethod = "Cash"
		if item.BankAmount.GreaterThan(decimal.Zero) {
			if item.CashIn.GreaterThan(decimal.Zero) {
				item.PaymentMethod = "Split (Cash+Bank)"
			} else {
				item.PaymentMethod = "Bank / UPI"
			}
		} else if item.TotalAmount.GreaterThan(decimal.Zero) && item.CashIn.IsZero() && item.Balance.GreaterThan(decimal.Zero) {
			item.PaymentMethod = "Credit / Due"
		}

		totalLineSales = totalLineSales.Add(item.TotalAmount)
		totalLineCashIn = totalLineCashIn.Add(item.CashIn)
		lineSaleIDs = append(lineSaleIDs, item.ID)
		report.LineSales = append(report.LineSales, item)
	}
	lineRows.Close()

	// 3. Fetch Counter Sales
	counterQuery := `
		SELECT id, item, price, total_amount, payment_method, cash, bank_amount, account, created_at
		FROM counter_sale
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at ASC
	`
	counterRows, err := exec.Query(ctx, counterQuery, tenantID, start, end)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to query counter sales: %w", err))
	}
	defer counterRows.Close()

	var totalCounterSales, totalCounterCashIn decimal.Decimal
	for counterRows.Next() {
		var item domain.CounterSaleReportItem
		var dummyPrice decimal.Decimal
		if err := counterRows.Scan(
			&item.ID,
			&item.Item,
			&dummyPrice,
			&item.TotalAmount,
			&item.PaymentMethod,
			&item.Cash,
			&item.BankAmount,
			&item.Account,
			&item.CreatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan counter sale item: %w", err))
		}
		item.ReceiptNumber = fmt.Sprintf("CS-%s", strings.ToUpper(item.ID.String()[:8]))
		item.Time = item.CreatedAt.In(loc).Format("15:04")
		if item.PaymentMethod == "" {
			if item.BankAmount.GreaterThan(decimal.Zero) && item.Cash.GreaterThan(decimal.Zero) {
				item.PaymentMethod = "Split"
			} else if item.BankAmount.GreaterThan(decimal.Zero) {
				item.PaymentMethod = "Bank / UPI"
			} else {
				item.PaymentMethod = "Cash"
			}
		}

		totalCounterSales = totalCounterSales.Add(item.TotalAmount)
		totalCounterCashIn = totalCounterCashIn.Add(item.Cash)
		report.CounterSales = append(report.CounterSales, item)
	}
	counterRows.Close()

	// 4. Fetch Purchases
	purchaseQuery := `
		SELECT id, COALESCE(customer_name, 'Direct Supplier'), item, quantity,
		       total_amount, total_paid, total_pending, payment_status, created_at
		FROM tenant_purchase
		WHERE tenant_id = $1 AND created_at >= $2 AND created_at < $3
		ORDER BY created_at ASC
	`
	purchRows, err := exec.Query(ctx, purchaseQuery, tenantID, start, end)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to query purchases: %w", err))
	}
	defer purchRows.Close()

	var totalPurchases decimal.Decimal
	for purchRows.Next() {
		var item domain.PurchaseReportItem
		if err := purchRows.Scan(
			&item.ID,
			&item.SupplierName,
			&item.Item,
			&item.Quantity,
			&item.TotalAmount,
			&item.TotalPaid,
			&item.TotalPending,
			&item.PaymentStatus,
			&item.CreatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan purchase item: %w", err))
		}
		item.Reference = fmt.Sprintf("PUR-%s", strings.ToUpper(item.ID.String()[:8]))
		item.Time = item.CreatedAt.In(loc).Format("15:04")
		totalPurchases = totalPurchases.Add(item.TotalAmount)
		report.Purchases = append(report.Purchases, item)
	}
	purchRows.Close()

	// 5. Fetch Expenses
	expenseQuery := `
		SELECT e.id, e.item, COALESCE(e.category, 'General'), COALESCE(emp.name, ''),
		       e.total_amount, e.created_at
		FROM tenant_expense e
		LEFT JOIN tenant_employees emp ON emp.id = e.employee_id
		WHERE e.tenant_id = $1 AND e.created_at >= $2 AND e.created_at < $3
		ORDER BY e.created_at ASC
	`
	expRows, err := exec.Query(ctx, expenseQuery, tenantID, start, end)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to query expenses: %w", err))
	}
	defer expRows.Close()

	var totalExpenses decimal.Decimal
	expenseIDs := make([]uuid.UUID, 0)
	for expRows.Next() {
		var item domain.ExpenseReportItem
		if err := expRows.Scan(
			&item.ID,
			&item.Description,
			&item.Category,
			&item.EmployeeName,
			&item.Amount,
			&item.CreatedAt,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan expense item: %w", err))
		}
		item.Reference = fmt.Sprintf("EXP-%s", strings.ToUpper(item.ID.String()[:8]))
		item.Time = item.CreatedAt.In(loc).Format("15:04")
		item.PaymentMethod = "Cash"
		totalExpenses = totalExpenses.Add(item.Amount)
		expenseIDs = append(expenseIDs, item.ID)
		report.Expenses = append(report.Expenses, item)
	}
	expRows.Close()

	// 5b. Enrich expense payment methods if recorded
	if len(expenseIDs) > 0 {
		expPmQuery := `
			SELECT expense_id, payment_method, bank_name
			FROM tenant_expense_payment
			WHERE tenant_id = $1 AND expense_id = ANY($2)
		`
		if pmRows, err := exec.Query(ctx, expPmQuery, tenantID, expenseIDs); err == nil {
			pmMap := make(map[uuid.UUID]string)
			for pmRows.Next() {
				var eid uuid.UUID
				var method, bName string
				if scanErr := pmRows.Scan(&eid, &method, &bName); scanErr == nil {
					if method == "bank" && bName != "" {
						pmMap[eid] = "Bank (" + bName + ")"
					} else {
						pmMap[eid] = strings.ToUpper(method[:1]) + strings.ToLower(method[1:])
					}
				}
			}
			pmRows.Close()
			for i := range report.Expenses {
				if pm, ok := pmMap[report.Expenses[i].ID]; ok {
					report.Expenses[i].PaymentMethod = pm
				}
			}
		}
	}

	// 6. Cash Purchases Paid Today
	var purchaseCashPaid decimal.Decimal
	purchPmQuery := `
		SELECT COALESCE(SUM(amount), 0)
		FROM tenant_purchase_payment
		WHERE tenant_id = $1 AND payment_method = 'cash' AND created_at >= $2 AND created_at < $3
	`
	_ = exec.QueryRow(ctx, purchPmQuery, tenantID, start, end).Scan(&purchaseCashPaid)

	// 7. Cash Expenses Paid Today
	var expenseCashPaid decimal.Decimal
	expPmQuery := `
		SELECT COALESCE(SUM(amount), 0)
		FROM tenant_expense_payment
		WHERE tenant_id = $1 AND payment_method = 'cash' AND created_at >= $2 AND created_at < $3
	`
	_ = exec.QueryRow(ctx, expPmQuery, tenantID, start, end).Scan(&expenseCashPaid)
	if expenseCashPaid.IsZero() && totalExpenses.GreaterThan(decimal.Zero) {
		// Fallback: if expense payment table wasn't separately populated, default cash expenses to total expenses
		expenseCashPaid = totalExpenses
	}

	// 8. Other Cash Outflows (Advances & Salary Payments)
	var advancesCashPaid, salaryCashPaid decimal.Decimal
	advQuery := `
		SELECT COALESCE(SUM(advance), 0)
		FROM attendance
		WHERE tenant_id = $1 AND date = $2::date
	`
	_ = exec.QueryRow(ctx, advQuery, tenantID, dateStr).Scan(&advancesCashPaid)

	salPmQuery := `
		SELECT COALESCE(SUM(amount), 0)
		FROM employee_salary_payment
		WHERE tenant_id = $1 AND payment_method = 'cash' AND created_at >= $2 AND created_at < $3
	`
	_ = exec.QueryRow(ctx, salPmQuery, tenantID, start, end).Scan(&salaryCashPaid)
	otherCashOutflows := advancesCashPaid.Add(salaryCashPaid)

	// 9. Fetch Bank Balances (Each bank separately)
	bankQuery := `
		SELECT id, bank_name, COALESCE(account_number, ''), current_balance
		FROM tenant_bank
		WHERE tenant_id = $1 AND status = 'active'
		ORDER BY bank_name ASC
	`
	bankRows, err := exec.Query(ctx, bankQuery, tenantID)
	if err != nil {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to query tenant banks: %w", err))
	}
	defer bankRows.Close()

	var totalBankBalance decimal.Decimal
	for bankRows.Next() {
		var b domain.BankBalanceReportItem
		if err := bankRows.Scan(
			&b.BankID,
			&b.BankName,
			&b.AccountNumber,
			&b.CurrentBalance,
		); err != nil {
			return nil, appErrors.NewDatabase(fmt.Errorf("failed to scan bank balance item: %w", err))
		}
		b.AccountNumber = MaskAccountNumber(b.AccountNumber)
		totalBankBalance = totalBankBalance.Add(b.CurrentBalance)
		report.BankBalances = append(report.BankBalances, b)
	}
	bankRows.Close()

	// 10. Fetch Authoritative Closing Cash Balance from Tenant Financial Summary
	var closingCash decimal.Decimal
	summaryQuery := `
		SELECT cash_balance
		FROM tenant_financial_summary
		WHERE tenant_id = $1
	`
	err = exec.QueryRow(ctx, summaryQuery, tenantID).Scan(&closingCash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, appErrors.NewDatabase(fmt.Errorf("failed to fetch financial summary cash balance: %w", err))
	}

	// 11. Calculate Daily Cash Movement Summary
	totalCashInflow := totalLineCashIn.Add(totalCounterCashIn)
	totalCashOutflow := purchaseCashPaid.Add(expenseCashPaid).Add(otherCashOutflows)
	netDailyCashFlow := totalCashInflow.Sub(totalCashOutflow)

	report.CashBalance = domain.CashBalanceReport{
		LineSalesCash:    totalLineCashIn,
		CounterSalesCash: totalCounterCashIn,
		OtherCashInflows: decimal.Zero,
		TotalCashInflow:  totalCashInflow,
		PurchaseCash:     purchaseCashPaid,
		ExpenseCash:      expenseCashPaid,
		OtherCashOutflow: otherCashOutflows,
		TotalCashOutflow: totalCashOutflow,
		NetDailyCashFlow: netDailyCashFlow,
		ClosingCash:      closingCash,
	}

	// 12. Financial Summary
	report.FinancialSummary = domain.ReportFinancialSummary{
		TotalLineSales:      totalLineSales,
		TotalCounterSales:   totalCounterSales,
		TotalPurchases:      totalPurchases,
		TotalExpenses:       totalExpenses,
		CashBalance:         closingCash,
		TotalBankBalance:    totalBankBalance,
		TotalAvailableFunds: closingCash.Add(totalBankBalance),
	}

	return report, nil
}
