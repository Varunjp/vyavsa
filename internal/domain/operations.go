package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LineSale represents field/route sales to a customer
type LineSale struct {
	ID           uuid.UUID         `json:"id"`
	TenantID     uuid.UUID         `json:"tenant_id"`
	CustomerID   uuid.UUID         `json:"customer_id"`
	CustomerName string            `json:"customer_name"`
	Route        string            `json:"route,omitempty"`
	Salesman     string            `json:"salesman,omitempty"`
	Note         string            `json:"note,omitempty"`
	TotalAmount  decimal.Decimal   `json:"total_amount"`
	TotalCashIn  decimal.Decimal   `json:"total_cash_in"`
	Balance      decimal.Decimal   `json:"balance"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Payments     []LineSalePayment `json:"payments,omitempty"`
}

// LineSalePayment represents a payment towards a line sale
type LineSalePayment struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	LineSaleID    uuid.UUID       `json:"line_sale_id"`
	PaymentMethod string          `json:"payment_method"` // cash, bank, cheque, upi, other
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	Note          string          `json:"note,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// CounterSale represents point-of-sale store transactions
type CounterSale struct {
	ID            uuid.UUID            `json:"id"`
	TenantID      uuid.UUID            `json:"tenant_id"`
	Item          string               `json:"item"`
	Price         decimal.Decimal      `json:"price"`
	TotalAmount   decimal.Decimal      `json:"total_amount"`
	PaymentMethod string               `json:"payment_method"`
	Cash          decimal.Decimal      `json:"cash"`
	Account       decimal.Decimal      `json:"account"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	Payments      []CounterSalePayment `json:"payments,omitempty"`
}

// CounterSalePayment represents a payment split on a counter sale
type CounterSalePayment struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	CounterSaleID uuid.UUID       `json:"counter_sale_id"`
	PaymentMethod string          `json:"payment_method"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	Note          string          `json:"note,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// TenantPurchase represents inventory/stock purchases
type TenantPurchase struct {
	ID           uuid.UUID               `json:"id"`
	TenantID     uuid.UUID               `json:"tenant_id"`
	Item         string                  `json:"item"`
	Quantity     int                     `json:"quantity"`
	TotalAmount  decimal.Decimal         `json:"total_amount"`
	TotalPaid    decimal.Decimal         `json:"total_paid"`
	TotalPending decimal.Decimal         `json:"total_pending"`
	CreatedAt    time.Time               `json:"created_at"`
	UpdatedAt    time.Time               `json:"updated_at"`
	Payments     []TenantPurchasePayment `json:"payments,omitempty"`
}

// TenantPurchasePayment represents payment towards a purchase
type TenantPurchasePayment struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	PurchaseID    uuid.UUID       `json:"purchase_id"`
	PaymentMethod string          `json:"payment_method"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	Note          string          `json:"note,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// TenantExpense represents operating and miscellaneous expenses
type TenantExpense struct {
	ID          uuid.UUID              `json:"id"`
	TenantID    uuid.UUID              `json:"tenant_id"`
	Item        string                 `json:"item"`
	TotalAmount decimal.Decimal        `json:"total_amount"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Payments    []TenantExpensePayment `json:"payments,omitempty"`
}

// TenantExpensePayment represents payment towards an expense
type TenantExpensePayment struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	ExpenseID     uuid.UUID       `json:"expense_id"`
	PaymentMethod string          `json:"payment_method"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	Note          string          `json:"note,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// Attendance represents employee daily attendance, overtime, and advance
type Attendance struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	EmployeeID   uuid.UUID       `json:"employee_id"`
	EmployeeName string          `json:"employee_name,omitempty"`
	Date         string          `json:"date"`    // YYYY-MM-DD
	Status       string          `json:"status"`  // present, absent, half_day, leave
	OT           decimal.Decimal `json:"ot"`      // Overtime hours/units
	Advance      decimal.Decimal `json:"advance"` // Cash advance paid
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// EmployeeSalary represents the salary payable ledger for an employee
type EmployeeSalary struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	EmployeeID   uuid.UUID       `json:"employee_id"`
	EmployeeName string          `json:"employee_name,omitempty"`
	SalaryRate   decimal.Decimal `json:"salary_rate,omitempty"`
	OTRate       decimal.Decimal `json:"ot_rate,omitempty"`
	Balance      decimal.Decimal `json:"balance"` // Pending salary amount
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// EmployeeSalaryPayment represents salary disbursement
type EmployeeSalaryPayment struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	EmployeeID    uuid.UUID       `json:"employee_id"`
	EmployeeName  string          `json:"employee_name,omitempty"`
	PaymentMethod string          `json:"payment_method"` // cash, bank, cheque, upi
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	Note          string          `json:"note,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// TenantDailyStats represents business performance aggregates for a day
type TenantDailyStats struct {
	ID                uuid.UUID       `json:"id"`
	TenantID          uuid.UUID       `json:"tenant_id"`
	Date              string          `json:"date"` // YYYY-MM-DD
	LineSaleAmount    decimal.Decimal `json:"line_sale_amount"`
	CounterSaleAmount decimal.Decimal `json:"counter_sale_amount"`
	TotalSales        decimal.Decimal `json:"total_sales"`
	PurchaseAmount    decimal.Decimal `json:"purchase_amount"`
	ExpenseAmount     decimal.Decimal `json:"expense_amount"`
	WagesAmount       decimal.Decimal `json:"wages_amount"`
	AdvanceAmount     decimal.Decimal `json:"advance_amount"`
	AttendancePresent int             `json:"attendance_present"`
	AttendanceAbsent  int             `json:"attendance_absent"`
	AmountReceived    decimal.Decimal `json:"amount_received"`
	AmountPaid        decimal.Decimal `json:"amount_paid"`
	CreditSale        decimal.Decimal `json:"credit_sale"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// FinancialMetrics aggregates all live cash, bank, dues, receivables, and pending salaries
type FinancialMetrics struct {
	CashBalance     decimal.Decimal   `json:"cash_balance"`
	BankBalance     decimal.Decimal   `json:"bank_balance"`
	TotalReceivable decimal.Decimal   `json:"total_receivable"`
	TotalPayable    decimal.Decimal   `json:"total_payable"`
	NetDues         decimal.Decimal   `json:"net_dues"`
	NetReceivables  decimal.Decimal   `json:"net_receivables"`
	PendingSalary   decimal.Decimal   `json:"pending_salary"`
	TodayStats      *TenantDailyStats `json:"today_stats,omitempty"`
}
