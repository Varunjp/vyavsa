package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LineSale represents field/route sales to a customer
type LineSale struct {
	ID              uuid.UUID         `json:"id"`
	TenantID        uuid.UUID         `json:"tenant_id"`
	CustomerID      uuid.UUID         `json:"customer_id"`
	CustomerName    string            `json:"customer_name"`
	Route           string            `json:"route,omitempty"`
	Salesman        string            `json:"salesman,omitempty"`
	Note            string            `json:"note,omitempty"`
	TotalAmount     decimal.Decimal   `json:"total_amount"`
	TotalCashIn     decimal.Decimal   `json:"total_cash_in"`
	BankAmount      decimal.Decimal   `json:"bank_amount"`
	CollectedAmount decimal.Decimal   `json:"collected_amount"`
	Balance         decimal.Decimal   `json:"balance"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Payments        []LineSalePayment `json:"payments,omitempty"`
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
	PaymentDate   string          `json:"payment_date,omitempty"`
	ReferenceID   string          `json:"reference_id,omitempty"`
	Status        string          `json:"status,omitempty"`
	Note          string          `json:"note,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// CounterSale represents point-of-sale store transactions
type CounterSale struct {
	ID              uuid.UUID            `json:"id"`
	TenantID        uuid.UUID            `json:"tenant_id"`
	Item            string               `json:"item"`
	Price           decimal.Decimal      `json:"price"`
	TotalAmount     decimal.Decimal      `json:"total_amount"`
	PaymentMethod   string               `json:"payment_method"`
	Cash            decimal.Decimal      `json:"cash"`
	BankAmount      decimal.Decimal      `json:"bank_amount"`
	BankID          *uuid.UUID           `json:"bank_id,omitempty"`
	CollectedAmount decimal.Decimal      `json:"collected_amount"`
	Account         decimal.Decimal      `json:"account"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	Payments        []CounterSalePayment `json:"payments,omitempty"`
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
	PaymentDate   string          `json:"payment_date,omitempty"`
	ReferenceID   string          `json:"reference_id,omitempty"`
	Status        string          `json:"status,omitempty"`
	Note          string          `json:"note,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// Purchase payment status constants
const (
	PurchasePaymentStatusPaid          = "PAID"
	PurchasePaymentStatusPartiallyPaid = "PARTIALLY_PAID"
	PurchasePaymentStatusUnpaid        = "UNPAID"
)

// ComputePurchasePaymentStatus derives the payment status from total amount and paid amount
func ComputePurchasePaymentStatus(totalAmount, totalPaid decimal.Decimal) string {
	if totalAmount.IsZero() || totalPaid.GreaterThanOrEqual(totalAmount) {
		return PurchasePaymentStatusPaid
	}
	if totalPaid.GreaterThan(decimal.Zero) {
		return PurchasePaymentStatusPartiallyPaid
	}
	return PurchasePaymentStatusUnpaid
}

// CustomerPayableSummary represents aggregated purchase and payment figures for a customer/supplier
type CustomerPayableSummary struct {
	TotalPurchases     decimal.Decimal `json:"total_purchases"`
	TotalPaid          decimal.Decimal `json:"total_paid"`
	OutstandingPayable decimal.Decimal `json:"outstanding_payable"`
}

// TenantPurchase represents inventory/stock purchases
type TenantPurchase struct {
	ID                uuid.UUID               `json:"id"`
	TenantID          uuid.UUID               `json:"tenant_id"`
	CustomerID        *uuid.UUID              `json:"customer_id,omitempty"`
	CustomerName      string                  `json:"customer_name,omitempty"`
	Item              string                  `json:"item"`
	Quantity          int                     `json:"quantity"`
	TotalAmount       decimal.Decimal         `json:"total_amount"`
	TotalPaid         decimal.Decimal         `json:"total_paid"`
	TotalPending      decimal.Decimal         `json:"total_pending"`
	OutstandingAmount decimal.Decimal         `json:"outstanding_amount"`
	PaymentStatus     string                  `json:"payment_status"`
	CreatedAt         time.Time               `json:"created_at"`
	UpdatedAt         time.Time               `json:"updated_at"`
	Payments          []TenantPurchasePayment `json:"payments,omitempty"`
}

// TenantPurchasePayment represents payment towards a purchase
type TenantPurchasePayment struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	PurchaseID    uuid.UUID       `json:"purchase_id"`
	CustomerID    *uuid.UUID      `json:"customer_id,omitempty"`
	PaymentMethod string          `json:"payment_method"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentDate   string          `json:"payment_date,omitempty"`
	ReferenceID   string          `json:"reference_id,omitempty"`
	Status        string          `json:"status,omitempty"`
	Note          string          `json:"note,omitempty"`
	IsSettlement  bool            `json:"is_settlement"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// ExpensePaymentBreakdown represents the detailed payment breakdown for an expense
type ExpensePaymentBreakdown struct {
	CashAmount decimal.Decimal        `json:"cash_amount"`
	BankAmount decimal.Decimal        `json:"bank_amount"`
	Banks      []ExpenseBankBreakdown `json:"banks"`
}

// ExpenseBankBreakdown represents individual bank payment breakdown in an expense
type ExpenseBankBreakdown struct {
	BankAccountID uuid.UUID       `json:"bank_account_id"`
	BankName      string          `json:"bank_name"`
	Amount        decimal.Decimal `json:"amount"`
}

// TenantExpense represents operating and miscellaneous expenses
type TenantExpense struct {
	ID               uuid.UUID                `json:"id"`
	TenantID         uuid.UUID                `json:"tenant_id"`
	Item             string                   `json:"item"`
	Category         string                   `json:"category,omitempty"`
	EmployeeID       *uuid.UUID               `json:"employee_id,omitempty"`
	EmployeeName     string                   `json:"employee_name,omitempty"`
	TotalAmount      decimal.Decimal          `json:"total_amount"`
	Amount           decimal.Decimal          `json:"amount"` // alias for TotalAmount
	PaymentMethod    string                   `json:"payment_method"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
	Payments         []TenantExpensePayment   `json:"payments,omitempty"`
	PaymentBreakdown *ExpensePaymentBreakdown `json:"payment_breakdown,omitempty"`
}

// BuildPaymentBreakdown populates Amount, PaymentBreakdown, and ensures consistent PaymentMethod
func (e *TenantExpense) BuildPaymentBreakdown() {
	if e.Amount.IsZero() && !e.TotalAmount.IsZero() {
		e.Amount = e.TotalAmount
	} else if e.TotalAmount.IsZero() && !e.Amount.IsZero() {
		e.TotalAmount = e.Amount
	}

	var cashAmount, bankAmount decimal.Decimal
	banks := make([]ExpenseBankBreakdown, 0)

	for _, p := range e.Payments {
		if p.PaymentMethod == "cash" {
			cashAmount = cashAmount.Add(p.Amount)
		} else if p.PaymentMethod == "bank" {
			bankAmount = bankAmount.Add(p.Amount)
			var bID uuid.UUID
			if p.BankID != nil {
				bID = *p.BankID
			}
			banks = append(banks, ExpenseBankBreakdown{
				BankAccountID: bID,
				BankName:      p.BankName,
				Amount:        p.Amount,
			})
		}
	}

	if e.PaymentMethod == "" {
		if cashAmount.GreaterThan(decimal.Zero) && bankAmount.GreaterThan(decimal.Zero) {
			e.PaymentMethod = "cash_bank"
		} else if bankAmount.GreaterThan(decimal.Zero) {
			e.PaymentMethod = "bank"
		} else {
			e.PaymentMethod = "cash"
		}
	}

	e.PaymentBreakdown = &ExpensePaymentBreakdown{
		CashAmount: cashAmount,
		BankAmount: bankAmount,
		Banks:      banks,
	}
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
	PaymentDate   string          `json:"payment_date,omitempty"`
	ReferenceID   string          `json:"reference_id,omitempty"`
	Status        string          `json:"status,omitempty"`
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
	Date         string          `json:"date"`   // YYYY-MM-DD
	Status       string          `json:"status"` // present, absent, half_day, leave
	DailySalary  decimal.Decimal `json:"daily_salary"`
	OT           decimal.Decimal `json:"ot"`        // Historical overtime hours/units
	OTAmount     decimal.Decimal `json:"ot_amount"` // Overtime direct amount
	Advance      decimal.Decimal `json:"advance"`   // Cash advance paid
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// EmployeeAdvance represents an individual cash/bank advance transaction to an employee
type EmployeeAdvance struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	EmployeeID    uuid.UUID       `json:"employee_id"`
	EmployeeName  string          `json:"employee_name,omitempty"`
	ExpenseID     *uuid.UUID      `json:"expense_id,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	PaymentMethod string          `json:"payment_method"` // cash, bank
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	ReferenceID   string          `json:"reference_id,omitempty"`
	AdvanceDate   string          `json:"advance_date"` // YYYY-MM-DD
	Notes         string          `json:"notes,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// EmployeeOvertime represents an individual overtime amount logged for an employee
type EmployeeOvertime struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	EmployeeID   uuid.UUID       `json:"employee_id"`
	EmployeeName string          `json:"employee_name,omitempty"`
	Amount       decimal.Decimal `json:"amount"`
	OvertimeDate string          `json:"overtime_date"` // YYYY-MM-DD
	ReferenceID  string          `json:"reference_id,omitempty"`
	Notes        string          `json:"notes,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// EmployeeSalary represents the salary payable ledger for an employee
type EmployeeSalary struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	EmployeeID    uuid.UUID       `json:"employee_id"`
	EmployeeName  string          `json:"employee_name,omitempty"`
	SalaryRate    decimal.Decimal `json:"salary_rate,omitempty"` // Base Salary
	OTRate        decimal.Decimal `json:"ot_rate,omitempty"`
	TotalOvertime decimal.Decimal `json:"total_overtime"`
	GrossSalary   decimal.Decimal `json:"gross_salary"`
	TotalAdvances decimal.Decimal `json:"total_advances"`
	TotalPaid     decimal.Decimal `json:"total_paid"`
	Balance       decimal.Decimal `json:"balance"` // Pending net payable salary amount
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// EmployeeSalaryStatement represents an employee's comprehensive financial ledger statement
type EmployeeSalaryStatement struct {
	EmployeeID    uuid.UUID               `json:"employee_id"`
	EmployeeName  string                  `json:"employee_name"`
	BaseSalary    decimal.Decimal         `json:"base_salary"`
	TotalOvertime decimal.Decimal         `json:"total_overtime"`
	GrossSalary   decimal.Decimal         `json:"gross_salary"`
	TotalAdvances decimal.Decimal         `json:"total_advances"`
	TotalPaid     decimal.Decimal         `json:"total_paid"`
	NetPayable    decimal.Decimal         `json:"net_payable"`
	Overtimes     []EmployeeOvertime      `json:"overtimes"`
	Advances      []EmployeeAdvance       `json:"advances"`
	Payments      []EmployeeSalaryPayment `json:"payments"`
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
	PaymentDate   string          `json:"payment_date,omitempty"` // YYYY-MM-DD
	ReferenceID   string          `json:"reference_id,omitempty"`
	Status        string          `json:"status,omitempty"`
	Note          string          `json:"note,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// TenantDailyStats represents business performance aggregates for a day
type TenantDailyStats struct {
	ID                  uuid.UUID       `json:"id"`
	TenantID            uuid.UUID       `json:"tenant_id"`
	Date                string          `json:"date"` // YYYY-MM-DD
	LineSaleAmount      decimal.Decimal `json:"line_sale_amount"`
	CounterSaleAmount   decimal.Decimal `json:"counter_sale_amount"`
	TotalSales          decimal.Decimal `json:"total_sales"`
	PurchaseAmount      decimal.Decimal `json:"purchase_amount"`
	ExpenseAmount       decimal.Decimal `json:"expense_amount"`
	WagesAmount         decimal.Decimal `json:"wages_amount"`
	AdvanceAmount       decimal.Decimal `json:"advance_amount"`
	AttendancePresent   int             `json:"attendance_present"`
	AttendanceAbsent    int             `json:"attendance_absent"`
	TodayEmployeeSalary decimal.Decimal `json:"today_employee_salary"`
	AmountReceived      decimal.Decimal `json:"amount_received"`
	AmountPaid          decimal.Decimal `json:"amount_paid"`
	CreditSale          decimal.Decimal `json:"credit_sale"`
	RequestedDate       string          `json:"requested_date,omitempty"`
	DataDate            string          `json:"data_date,omitempty"`
	IsCurrent           *bool           `json:"is_current,omitempty"`
	DaysOld             int             `json:"days_old,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// TodayAttendanceOverview represents today's staff attendance details
type TodayAttendanceOverview struct {
	Present    int  `json:"present"`
	Total      int  `json:"total"`
	Absent     int  `json:"absent"`
	HasRecords bool `json:"has_records"`
}

// CurrentItemOverview represents inventory/stock intake item information
type CurrentItemOverview struct {
	Name         string          `json:"name"`
	Quantity     int             `json:"quantity,omitempty"`
	TotalAmount  decimal.Decimal `json:"total_amount"`
	TotalPaid    decimal.Decimal `json:"total_paid,omitempty"`
	TotalPending decimal.Decimal `json:"total_pending,omitempty"`
	Date         string          `json:"date,omitempty"`
	IsToday      bool            `json:"is_today"`
	Source       string          `json:"source,omitempty"` // purchase, counter_sale
}

// TodayOverview aggregates the key daily metrics for the tenant dashboard
type TodayOverview struct {
	Attendance         TodayAttendanceOverview `json:"attendance"`
	LineSale           decimal.Decimal         `json:"line_sale"`
	CounterSale        decimal.Decimal         `json:"counter_sale"`
	EmployeeAdvance    decimal.Decimal         `json:"employee_advance"`
	CurrentItem        any                     `json:"current_item"`
	OutstandingPayable decimal.Decimal         `json:"outstanding_payable"`
	Date               string                  `json:"date"`
}

// FinancialMetrics aggregates all live cash, bank, dues, receivables, and pending salaries
type FinancialMetrics struct {
	CashBalance         decimal.Decimal   `json:"cash_balance"`
	BankBalance         decimal.Decimal   `json:"bank_balance"`
	TotalReceivable     decimal.Decimal   `json:"total_receivable"`
	TotalPayable        decimal.Decimal   `json:"total_payable"`
	NetDues             decimal.Decimal   `json:"net_dues"`
	NetReceivables      decimal.Decimal   `json:"net_receivables"`
	PendingSalary       decimal.Decimal   `json:"pending_salary"`
	TodayEmployeeSalary decimal.Decimal   `json:"today_employee_salary"`
	BankBalances        []TenantBank      `json:"bank_balances,omitempty"`
	TodayStats          *TenantDailyStats `json:"today_stats,omitempty"`
	TodayOverview       *TodayOverview    `json:"today_overview,omitempty"`
	RequestedDate       string            `json:"requested_date,omitempty"`
	DataDate            string            `json:"data_date,omitempty"`
	IsCurrent           bool              `json:"is_current"`
	DaysOld             int               `json:"days_old"`
}

// BankTransaction represents individual ledger transactions on tenant bank accounts
type BankTransaction struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	BankID          uuid.UUID       `json:"bank_id"`
	Amount          decimal.Decimal `json:"amount"`
	TransactionType string          `json:"transaction_type"` // credit, debit
	Reason          string          `json:"reason"`
	SaleType        string          `json:"sale_type,omitempty"` // line_sale, counter_sale
	SaleID          *uuid.UUID      `json:"sale_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// CustomerBalanceAdjustment tracks manual adjustments to customer balances
type CustomerBalanceAdjustment struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	CustomerID       uuid.UUID       `json:"customer_id"`
	CustomerName     string          `json:"customer_name,omitempty"`
	PreviousBalance  decimal.Decimal `json:"previous_balance"`
	NewBalance       decimal.Decimal `json:"new_balance"`
	AdjustmentAmount decimal.Decimal `json:"adjustment_amount"`
	Reason           string          `json:"reason"`
	CreatedAt        time.Time       `json:"created_at"`
}
