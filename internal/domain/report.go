package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TenantReportInfo contains business identifying metadata for the report header
type TenantReportInfo struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Phone string    `json:"phone,omitempty"`
}

// LineSaleReportItem represents an individual line sale transaction in the daily report
type LineSaleReportItem struct {
	ID            uuid.UUID       `json:"id"`
	InvoiceNumber string          `json:"invoice_number"`
	Time          string          `json:"time"` // HH:MM
	CustomerName  string          `json:"customer_name"`
	Route         string          `json:"route,omitempty"`
	Salesman      string          `json:"salesman,omitempty"`
	Note          string          `json:"note,omitempty"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	CashIn        decimal.Decimal `json:"cash_in"`
	BankAmount    decimal.Decimal `json:"bank_amount"`
	Balance       decimal.Decimal `json:"balance"`
	PaymentMethod string          `json:"payment_method"`
	CreatedAt     time.Time       `json:"created_at"`
}

// CounterSaleReportItem represents an individual counter sale transaction in the daily report
type CounterSaleReportItem struct {
	ID            uuid.UUID       `json:"id"`
	ReceiptNumber string          `json:"receipt_number"`
	Time          string          `json:"time"` // HH:MM
	Item          string          `json:"item"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	Cash          decimal.Decimal `json:"cash"`
	BankAmount    decimal.Decimal `json:"bank_amount"`
	Account       decimal.Decimal `json:"account"` // Credit / Due balance
	PaymentMethod string          `json:"payment_method"`
	CreatedAt     time.Time       `json:"created_at"`
}

// PurchaseReportItem represents an individual inventory/stock purchase transaction in the daily report
type PurchaseReportItem struct {
	ID            uuid.UUID       `json:"id"`
	Reference     string          `json:"reference"`
	Time          string          `json:"time"` // HH:MM
	SupplierName  string          `json:"supplier_name"`
	Item          string          `json:"item"`
	Quantity      int             `json:"quantity"`
	TotalAmount   decimal.Decimal `json:"total_amount"`
	TotalPaid     decimal.Decimal `json:"total_paid"`
	TotalPending  decimal.Decimal `json:"total_pending"`
	PaymentStatus string          `json:"payment_status"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ExpenseReportItem represents an individual operating expense in the daily report
type ExpenseReportItem struct {
	ID            uuid.UUID       `json:"id"`
	Reference     string          `json:"reference"`
	Time          string          `json:"time"` // HH:MM
	Category      string          `json:"category"`
	Description   string          `json:"description"`
	EmployeeName  string          `json:"employee_name,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"`
	CreatedAt     time.Time       `json:"created_at"`
}

// BankBalanceReportItem represents an individual tenant bank account balance
type BankBalanceReportItem struct {
	BankID         uuid.UUID       `json:"bank_id"`
	BankName       string          `json:"bank_name"`
	AccountNumber  string          `json:"account_number"` // Masked: ****1234
	CurrentBalance decimal.Decimal `json:"current_balance"`
}

// CashBalanceReport provides day-level cash flow calculations and closing cash position
type CashBalanceReport struct {
	LineSalesCash    decimal.Decimal `json:"line_sales_cash"`
	CounterSalesCash decimal.Decimal `json:"counter_sales_cash"`
	OtherCashInflows decimal.Decimal `json:"other_cash_inflows"`
	TotalCashInflow  decimal.Decimal `json:"total_cash_inflow"`
	PurchaseCash     decimal.Decimal `json:"purchase_cash"`
	ExpenseCash      decimal.Decimal `json:"expense_cash"`
	OtherCashOutflow decimal.Decimal `json:"other_cash_outflow"`
	TotalCashOutflow decimal.Decimal `json:"total_cash_outflow"`
	NetDailyCashFlow decimal.Decimal `json:"net_daily_cash_flow"`
	ClosingCash      decimal.Decimal `json:"closing_cash"`
}

// ReportFinancialSummary aggregates the overall financial indicators for the report date
type ReportFinancialSummary struct {
	TotalLineSales      decimal.Decimal `json:"total_line_sales"`
	TotalCounterSales   decimal.Decimal `json:"total_counter_sales"`
	TotalPurchases      decimal.Decimal `json:"total_purchases"`
	TotalExpenses       decimal.Decimal `json:"total_expenses"`
	CashBalance         decimal.Decimal `json:"cash_balance"`
	TotalBankBalance    decimal.Decimal `json:"total_bank_balance"`
	TotalAvailableFunds decimal.Decimal `json:"total_available_funds"`
}

// DailyReport is the comprehensive domain aggregate for a single tenant business day
type DailyReport struct {
	Tenant           TenantReportInfo        `json:"tenant"`
	Date             string                  `json:"date"`           // YYYY-MM-DD
	FormattedDate    string                  `json:"formatted_date"` // e.g. "07 October 2026"
	LineSales        []LineSaleReportItem    `json:"line_sales"`
	CounterSales     []CounterSaleReportItem `json:"counter_sales"`
	Purchases        []PurchaseReportItem    `json:"purchases"`
	Expenses         []ExpenseReportItem     `json:"expenses"`
	CashBalance      CashBalanceReport       `json:"cash_balance"`
	BankBalances     []BankBalanceReportItem `json:"bank_balances"`
	FinancialSummary ReportFinancialSummary  `json:"financial_summary"`
	GeneratedAt      time.Time               `json:"generated_at"`
}
