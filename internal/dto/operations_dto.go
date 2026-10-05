package dto

import (
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ==========================================
// 1. Tenant User DTOs
// ==========================================

type CreateTenantUserRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=255"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Role     string `json:"role" binding:"required,oneof=admin user"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}

type UpdateTenantUserRequest struct {
	Name     string `json:"name" binding:"omitempty,min=2,max=255"`
	Role     string `json:"role" binding:"omitempty,oneof=admin user"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive suspended"`
	Password string `json:"password" binding:"omitempty,min=6,max=72"`
}

type TenantUserItemResponse struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Email     string    `json:"email"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ToTenantUserItemResponse(u *domain.TenantUser) TenantUserItemResponse {
	return TenantUserItemResponse{
		ID:        u.ID,
		TenantID:  u.TenantID,
		Name:      u.Name,
		Role:      u.Role,
		Email:     u.Email,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

// ==========================================
// 2. Employee DTOs
// ==========================================

type CreateEmployeeRequest struct {
	Name   string          `json:"name" binding:"required,min=2,max=255"`
	Phone  string          `json:"phone" binding:"omitempty,max=30"`
	Salary decimal.Decimal `json:"salary" binding:"required"`
	OTRate decimal.Decimal `json:"ot_rate" binding:"omitempty"`
	Status string          `json:"status" binding:"omitempty,oneof=active inactive terminated"`
}

type UpdateEmployeeRequest struct {
	Name   string           `json:"name" binding:"omitempty,min=2,max=255"`
	Phone  string           `json:"phone" binding:"omitempty,max=30"`
	Salary *decimal.Decimal `json:"salary" binding:"omitempty"`
	OTRate *decimal.Decimal `json:"ot_rate" binding:"omitempty"`
	Status string           `json:"status" binding:"omitempty,oneof=active inactive terminated"`
}

type EmployeeResponse struct {
	ID        uuid.UUID       `json:"id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Name      string          `json:"name"`
	Phone     string          `json:"phone,omitempty"`
	Salary    decimal.Decimal `json:"salary"`
	OTRate    decimal.Decimal `json:"ot_rate"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func ToEmployeeResponse(e *domain.TenantEmployee) EmployeeResponse {
	return EmployeeResponse{
		ID:        e.ID,
		TenantID:  e.TenantID,
		Name:      e.Name,
		Phone:     e.Phone,
		Salary:    e.Salary,
		OTRate:    e.OTRate,
		Status:    e.Status,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

// ==========================================
// 3. Customer DTOs
// ==========================================

type CreateCustomerRequest struct {
	CustomerName   string          `json:"customer_name" binding:"required,min=2,max=255"`
	Phone          string          `json:"phone" binding:"omitempty,max=30"`
	OpeningBalance decimal.Decimal `json:"opening_balance" binding:"omitempty"`
	Status         string          `json:"status" binding:"omitempty,oneof=active inactive"`
}

type UpdateCustomerRequest struct {
	CustomerName string `json:"customer_name" binding:"omitempty,min=2,max=255"`
	Phone        string `json:"phone" binding:"omitempty,max=30"`
	Status       string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type CustomerResponse struct {
	ID                 uuid.UUID       `json:"id"`
	TenantID           uuid.UUID       `json:"tenant_id"`
	CustomerName       string          `json:"customer_name"`
	Phone              string          `json:"phone,omitempty"`
	OpeningBalance     decimal.Decimal `json:"opening_balance"`
	CurrentBalance     decimal.Decimal `json:"current_balance"`
	TotalPurchases     decimal.Decimal `json:"total_purchases"`
	TotalPaid          decimal.Decimal `json:"total_paid"`
	OutstandingPayable decimal.Decimal `json:"outstanding_payable"`
	Status             string          `json:"status"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

func ToCustomerResponse(c *domain.TenantCustomer) CustomerResponse {
	return CustomerResponse{
		ID:             c.ID,
		TenantID:       c.TenantID,
		CustomerName:   c.CustomerName,
		Phone:          c.Phone,
		OpeningBalance: c.OpeningBalance,
		CurrentBalance: c.CurrentBalance,
		Status:         c.Status,
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

func ToCustomerDetailResponse(c *domain.TenantCustomer, summary *domain.CustomerPayableSummary) CustomerResponse {
	resp := ToCustomerResponse(c)
	if summary != nil {
		resp.TotalPurchases = summary.TotalPurchases
		resp.TotalPaid = summary.TotalPaid
		resp.OutstandingPayable = summary.OutstandingPayable
	}
	return resp
}

// ==========================================
// 4. Bank DTOs
// ==========================================

type CreateBankRequest struct {
	BankName       string          `json:"bank_name" binding:"required,min=2,max=255"`
	AccountNumber  string          `json:"account_number" binding:"omitempty,max=100"`
	IFSC           string          `json:"ifsc" binding:"omitempty,max=50"`
	OpeningBalance decimal.Decimal `json:"opening_balance" binding:"omitempty"`
	Status         string          `json:"status" binding:"omitempty,oneof=active inactive"`
}

type UpdateBankRequest struct {
	BankName      string `json:"bank_name" binding:"omitempty,min=2,max=255"`
	AccountNumber string `json:"account_number" binding:"omitempty,max=100"`
	IFSC          string `json:"ifsc" binding:"omitempty,max=50"`
	Status        string `json:"status" binding:"omitempty,oneof=active inactive"`
}

type BankResponse struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	BankName       string          `json:"bank_name"`
	AccountNumber  string          `json:"account_number,omitempty"`
	IFSC           string          `json:"ifsc,omitempty"`
	CurrentBalance decimal.Decimal `json:"current_balance"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func ToBankResponse(b *domain.TenantBank) BankResponse {
	return BankResponse{
		ID:             b.ID,
		TenantID:       b.TenantID,
		BankName:       b.BankName,
		AccountNumber:  b.AccountNumber,
		IFSC:           b.IFSC,
		CurrentBalance: b.CurrentBalance,
		Status:         b.Status,
		CreatedAt:      b.CreatedAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

// ==========================================
// 5. Line Sale DTOs
// ==========================================

type BankPaymentSplitRequest struct {
	BankID   uuid.UUID       `json:"bank_id" binding:"required"`
	BankName string          `json:"bank_name,omitempty"`
	Amount   decimal.Decimal `json:"amount" binding:"required"`
	Note     string          `json:"note,omitempty"`
}

type CustomerBalanceResponse struct {
	CustomerID         uuid.UUID       `json:"customer_id"`
	CustomerName       string          `json:"customer_name"`
	CurrentBalance     decimal.Decimal `json:"current_balance"`
	TotalPurchases     decimal.Decimal `json:"total_purchases"`
	TotalPaid          decimal.Decimal `json:"total_paid"`
	OutstandingPayable decimal.Decimal `json:"outstanding_payable"`
	Status             string          `json:"status"`
}

type LineSalePaymentRequest struct {
	PaymentMethod string          `json:"payment_method" binding:"required,oneof=cash bank cheque upi other"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	Note          string          `json:"note,omitempty"`
}

type CreateLineSaleRequest struct {
	CustomerID   uuid.UUID                 `json:"customer_id" binding:"required"`
	Route        string                    `json:"route" binding:"omitempty,max=255"`
	Salesman     string                    `json:"salesman" binding:"omitempty,max=255"`
	Note         string                    `json:"note" binding:"omitempty"`
	TotalAmount  decimal.Decimal           `json:"total_amount" binding:"required"`
	TotalCashIn  decimal.Decimal           `json:"total_cash_in" binding:"omitempty"`
	CashAmount   *decimal.Decimal          `json:"cash_amount,omitempty"`
	BankAmount   decimal.Decimal           `json:"bank_amount" binding:"omitempty"`
	BankID       *uuid.UUID                `json:"bank_id,omitempty"`
	Payments     []LineSalePaymentRequest  `json:"payments,omitempty"`
	BankPayments []BankPaymentSplitRequest `json:"bank_payments,omitempty"`
}

type UpdateLineSaleRequest struct {
	Route       string           `json:"route" binding:"omitempty,max=255"`
	Salesman    string           `json:"salesman" binding:"omitempty,max=255"`
	Note        string           `json:"note" binding:"omitempty"`
	TotalAmount *decimal.Decimal `json:"total_amount" binding:"omitempty"`
	TotalCashIn *decimal.Decimal `json:"total_cash_in" binding:"omitempty"`
}

// ==========================================
// 6. Counter Sale DTOs
// ==========================================

type CounterSalePaymentRequest struct {
	PaymentMethod string          `json:"payment_method" binding:"required"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	Note          string          `json:"note,omitempty"`
}

type CreateCounterSaleRequest struct {
	Item          string                      `json:"item" binding:"required,min=1,max=255"`
	Price         decimal.Decimal             `json:"price" binding:"required"`
	TotalAmount   decimal.Decimal             `json:"total_amount" binding:"required"`
	PaymentMethod string                      `json:"payment_method" binding:"omitempty"`
	Cash          decimal.Decimal             `json:"cash" binding:"omitempty"`
	CashAmount    *decimal.Decimal            `json:"cash_amount,omitempty"`
	BankAmount    decimal.Decimal             `json:"bank_amount" binding:"omitempty"`
	BankID        *uuid.UUID                  `json:"bank_id,omitempty"`
	Account       decimal.Decimal             `json:"account" binding:"omitempty"`
	Payments      []CounterSalePaymentRequest `json:"payments,omitempty"`
	BankPayments  []BankPaymentSplitRequest   `json:"bank_payments,omitempty"`
}

type UpdateCounterSaleRequest struct {
	Item          string           `json:"item" binding:"omitempty"`
	Price         *decimal.Decimal `json:"price" binding:"omitempty"`
	TotalAmount   *decimal.Decimal `json:"total_amount" binding:"omitempty"`
	PaymentMethod string           `json:"payment_method" binding:"omitempty"`
	Cash          *decimal.Decimal `json:"cash" binding:"omitempty"`
	BankAmount    *decimal.Decimal `json:"bank_amount" binding:"omitempty"`
	Account       *decimal.Decimal `json:"account" binding:"omitempty"`
}

// ==========================================
// 7. Purchase DTOs
// ==========================================

type PurchasePaymentRequest struct {
	PaymentMethod string          `json:"payment_method" binding:"required"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	Note          string          `json:"note,omitempty"`
}

type CreatePurchaseRequest struct {
	CustomerID   *uuid.UUID                `json:"customer_id,omitempty"`
	Item         string                    `json:"item" binding:"required,min=1,max=255"`
	Quantity     int                       `json:"quantity" binding:"required,min=1"`
	TotalAmount  decimal.Decimal           `json:"total_amount" binding:"required"`
	TotalPaid    decimal.Decimal           `json:"total_paid" binding:"omitempty"`
	CashAmount   *decimal.Decimal          `json:"cash_amount,omitempty"`
	Payments     []PurchasePaymentRequest  `json:"payments,omitempty"`
	BankPayments []BankPaymentSplitRequest `json:"bank_payments,omitempty"`
}

type UpdatePurchaseRequest struct {
	CustomerID  *uuid.UUID       `json:"customer_id,omitempty"`
	Item        string           `json:"item" binding:"omitempty"`
	Quantity    *int             `json:"quantity" binding:"omitempty,min=1"`
	TotalAmount *decimal.Decimal `json:"total_amount" binding:"omitempty"`
	TotalPaid   *decimal.Decimal `json:"total_paid" binding:"omitempty"`
}

// ==========================================
// 8. Expense DTOs
// ==========================================

type ExpensePaymentRequest struct {
	PaymentMethod string          `json:"payment_method" binding:"required"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	Note          string          `json:"note,omitempty"`
}

type CreateExpenseRequest struct {
	Item          string                  `json:"item" binding:"required,min=1,max=255"`
	TotalAmount   decimal.Decimal         `json:"total_amount" binding:"required"`
	PaymentMethod string                  `json:"payment_method" binding:"omitempty"`
	BankID        *uuid.UUID              `json:"bank_id,omitempty"`
	BankName      string                  `json:"bank_name,omitempty"`
	Payments      []ExpensePaymentRequest `json:"payments,omitempty"`
}

type UpdateExpenseRequest struct {
	Item        string           `json:"item" binding:"omitempty"`
	TotalAmount *decimal.Decimal `json:"total_amount" binding:"omitempty"`
}

// ==========================================
// 9. Attendance & OT & Advance DTOs
// ==========================================

type RecordAttendanceRequest struct {
	EmployeeID uuid.UUID       `json:"employee_id" binding:"required"`
	Date       string          `json:"date" binding:"omitempty"` // YYYY-MM-DD
	Status     string          `json:"status" binding:"required,oneof=present absent half_day leave"`
	OT         decimal.Decimal `json:"ot" binding:"omitempty"`
	Advance    decimal.Decimal `json:"advance" binding:"omitempty"`
}

type UpdateAttendanceRequest struct {
	Status  string           `json:"status" binding:"omitempty,oneof=present absent half_day leave"`
	OT      *decimal.Decimal `json:"ot" binding:"omitempty"`
	Advance *decimal.Decimal `json:"advance" binding:"omitempty"`
}

type RecordOvertimeRequest struct {
	EmployeeID uuid.UUID       `json:"employee_id" binding:"required"`
	Date       string          `json:"date" binding:"omitempty"`
	OT         decimal.Decimal `json:"ot" binding:"required"`
}

type RecordAdvanceRequest struct {
	EmployeeID    uuid.UUID       `json:"employee_id" binding:"required"`
	Date          string          `json:"date" binding:"omitempty"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	PaymentMethod string          `json:"payment_method" binding:"omitempty"` // cash or bank
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Note          string          `json:"note,omitempty"`
}

// ==========================================
// 10. Salary DTOs
// ==========================================

type PaySalaryRequest struct {
	PaymentMethod string          `json:"payment_method" binding:"required"`
	BankID        *uuid.UUID      `json:"bank_id,omitempty"`
	BankName      string          `json:"bank_name,omitempty"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
	Note          string          `json:"note,omitempty"`
}

type UpdateSalaryBalanceRequest struct {
	Balance decimal.Decimal `json:"balance" binding:"required"`
}

// ==========================================
// 11. Customer Balance Adjustment DTOs
// ==========================================

type AdjustCustomerBalanceRequest struct {
	NewBalance       *decimal.Decimal `json:"new_balance,omitempty"`
	AdjustmentAmount *decimal.Decimal `json:"adjustment_amount,omitempty"`
	Reason           string           `json:"reason" binding:"required,min=2,max=500"`
}

type CustomerBalanceAdjustmentResponse struct {
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

// ==========================================
// 12. Bank Transaction DTOs
// ==========================================

type BankTransactionResponse struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	BankID          uuid.UUID       `json:"bank_id"`
	Amount          decimal.Decimal `json:"amount"`
	TransactionType string          `json:"transaction_type"`
	Reason          string          `json:"reason"`
	SaleType        string          `json:"sale_type,omitempty"`
	SaleID          *uuid.UUID      `json:"sale_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ==========================================
// 13. Purchase & Supplier Payment DTOs
// ==========================================

type PurchaseResponse struct {
	ID                uuid.UUID                      `json:"id"`
	TenantID          uuid.UUID                      `json:"tenant_id"`
	CustomerID        *uuid.UUID                     `json:"customer_id,omitempty"`
	CustomerName      string                         `json:"customer_name,omitempty"`
	Item              string                         `json:"item"`
	Quantity          int                            `json:"quantity"`
	TotalAmount       decimal.Decimal                `json:"total_amount"`
	TotalPaid         decimal.Decimal                `json:"total_paid"`
	TotalPending      decimal.Decimal                `json:"total_pending"`
	OutstandingAmount decimal.Decimal                `json:"outstanding_amount"`
	PaymentStatus     string                         `json:"payment_status"`
	CreatedAt         time.Time                      `json:"created_at"`
	UpdatedAt         time.Time                      `json:"updated_at"`
	Payments          []domain.TenantPurchasePayment `json:"payments,omitempty"`
}

func ToPurchaseResponse(p *domain.TenantPurchase) PurchaseResponse {
	return PurchaseResponse{
		ID:                p.ID,
		TenantID:          p.TenantID,
		CustomerID:        p.CustomerID,
		CustomerName:      p.CustomerName,
		Item:              p.Item,
		Quantity:          p.Quantity,
		TotalAmount:       p.TotalAmount,
		TotalPaid:         p.TotalPaid,
		TotalPending:      p.TotalPending,
		OutstandingAmount: p.TotalPending,
		PaymentStatus:     p.PaymentStatus,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
		Payments:          p.Payments,
	}
}

type RecordSupplierPaymentRequest struct {
	CustomerID    *uuid.UUID                `json:"customer_id,omitempty"`
	PurchaseID    *uuid.UUID                `json:"purchase_id,omitempty"`
	Amount        decimal.Decimal           `json:"amount" binding:"required"`
	PaymentMethod string                    `json:"payment_method" binding:"required,oneof=cash bank"`
	BankID        *uuid.UUID                `json:"bank_id,omitempty"`
	BankPayments  []BankPaymentSplitRequest `json:"bank_payments,omitempty"`
	Note          string                    `json:"note,omitempty"`
}

type SupplierPaymentAllocation struct {
	PurchaseID      uuid.UUID       `json:"purchase_id"`
	Item            string          `json:"item"`
	PreviousPending decimal.Decimal `json:"previous_pending"`
	AmountSettled   decimal.Decimal `json:"amount_settled"`
	NewPending      decimal.Decimal `json:"new_pending"`
	PaymentStatus   string          `json:"payment_status"`
}

type SupplierPaymentResponse struct {
	PaymentID           uuid.UUID                   `json:"payment_id"`
	CustomerID          uuid.UUID                   `json:"customer_id"`
	CustomerName        string                      `json:"customer_name"`
	Amount              decimal.Decimal             `json:"amount"`
	PaymentMethod       string                      `json:"payment_method"`
	PreviousOutstanding decimal.Decimal             `json:"previous_outstanding"`
	NewOutstanding      decimal.Decimal             `json:"new_outstanding"`
	Allocations         []SupplierPaymentAllocation `json:"allocations"`
	CreatedAt           time.Time                   `json:"created_at"`
}

type CustomerStatementEntry struct {
	ID             uuid.UUID       `json:"id"`
	Date           time.Time       `json:"date"`
	FormattedDate  string          `json:"formatted_date"`
	Description    string          `json:"description"`
	EntryType      string          `json:"entry_type"` // "purchase", "settlement_payment"
	PurchaseAmount decimal.Decimal `json:"purchase_amount"`
	PaidAmount     decimal.Decimal `json:"paid_amount"`
	Balance        decimal.Decimal `json:"balance"` // Running cumulative balance
	PaymentMethod  string          `json:"payment_method,omitempty"`
	BankName       string          `json:"bank_name,omitempty"`
	ReferenceID    uuid.UUID       `json:"reference_id"`
}

type CustomerStatementResponse struct {
	CustomerID         uuid.UUID                `json:"customer_id"`
	CustomerName       string                   `json:"customer_name"`
	Phone              string                   `json:"phone,omitempty"`
	OpeningBalance     decimal.Decimal          `json:"opening_balance"`
	CurrentBalance     decimal.Decimal          `json:"current_balance"`
	TotalPurchases     decimal.Decimal          `json:"total_purchases"`
	TotalPaid          decimal.Decimal          `json:"total_paid"`
	OutstandingPayable decimal.Decimal          `json:"outstanding_payable"`
	Entries            []CustomerStatementEntry `json:"entries"`
}
