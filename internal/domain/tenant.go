package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Tenant represents an isolated business organization
type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TenantUser represents a member/user within a tenant organisation
type TenantUser struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Name         string    `json:"name"`
	Role         string    `json:"role"` // admin or user
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TenantBank represents a bank account owned by the tenant
type TenantBank struct {
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

// TenantEmployee represents a staff member of the tenant
type TenantEmployee struct {
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

// TenantCustomer represents a buyer/client of the tenant
type TenantCustomer struct {
	ID             uuid.UUID       `json:"id"`
	TenantID       uuid.UUID       `json:"tenant_id"`
	CustomerName   string          `json:"customer_name"`
	Phone          string          `json:"phone,omitempty"`
	OpeningBalance decimal.Decimal `json:"opening_balance"`
	CurrentBalance decimal.Decimal `json:"current_balance"`
	Status         string          `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// TenantFinancialSummary holds authoritative cash and balance summaries
type TenantFinancialSummary struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	CashBalance     decimal.Decimal `json:"cash_balance"`
	BankBalance     decimal.Decimal `json:"bank_balance"`
	TotalReceivable decimal.Decimal `json:"total_receivable"`
	TotalPayable    decimal.Decimal `json:"total_payable"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// CachedTenantPlan represents cached subscription and plan validation status in Redis
type CachedTenantPlan struct {
	Active bool       `json:"active"`
	PlanID *uuid.UUID `json:"plan_id,omitempty"`
	Status string     `json:"status"`
}
