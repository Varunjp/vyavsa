package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PlatformAdmin represents a system super-administrator
type PlatformAdmin struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone,omitempty"`
	Status       string    `json:"status"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PlatformPlan represents a SaaS subscription plan
type PlatformPlan struct {
	ID        uuid.UUID       `json:"id"`
	PlanName  string          `json:"plan_name"`
	Note      string          `json:"note,omitempty"`
	Price     decimal.Decimal `json:"price"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// PlatformSubscription represents an active subscription for a tenant
type PlatformSubscription struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	CurrentPlanID   uuid.UUID  `json:"current_plan_id"`
	CurrentPlanName string     `json:"current_plan_name"`
	Status          string     `json:"status"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// PlatformPlanTransaction tracks payment transactions for subscription plans
type PlatformPlanTransaction struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	TransactionID string          `json:"transaction_id"`
	PaymentMethod string          `json:"payment_method"`
	PlanID        uuid.UUID       `json:"plan_id"`
	PlanName      string          `json:"plan_name,omitempty"`
	Amount        decimal.Decimal `json:"amount"`
	Status        string          `json:"status"`
	FailureReason string          `json:"failure_reason,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// PlatformSubscriptionWithTenant includes tenant business details for platform administrative listings
type PlatformSubscriptionWithTenant struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	TenantName      string     `json:"tenant_name"`
	TenantEmail     string     `json:"tenant_email"`
	CurrentPlanID   uuid.UUID  `json:"current_plan_id"`
	CurrentPlanName string     `json:"current_plan_name"`
	Status          string     `json:"status"`
	StartDate       *time.Time `json:"start_date,omitempty"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// PlatformPlanTransactionWithTenant includes tenant business details for platform administrative transaction records
type PlatformPlanTransactionWithTenant struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenant_id"`
	TenantName    string          `json:"tenant_name"`
	TenantEmail   string          `json:"tenant_email"`
	TransactionID string          `json:"transaction_id"`
	PaymentMethod string          `json:"payment_method"`
	PlanID        uuid.UUID       `json:"plan_id"`
	PlanName      string          `json:"plan_name"`
	Amount        decimal.Decimal `json:"amount"`
	Status        string          `json:"status"`
	FailureReason string          `json:"failure_reason,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
