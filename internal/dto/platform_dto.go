package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PlatformDashboardMetricsResponse represents aggregate KPI metrics and revenue trends for platform admins
type PlatformDashboardMetricsResponse struct {
	ActiveTenants         int64                `json:"active_tenants"`
	TotalTenants          int64                `json:"total_tenants"`
	MonthlyReceivedIncome decimal.Decimal      `json:"monthly_received_income"`
	RecentRegistrations7d int64                `json:"recent_registrations_7d"`
	RevenueTrend          []MonthlyRevenueItem `json:"revenue_trend"`
}

// MonthlyRevenueItem represents monthly revenue aggregate for trend visualization
type MonthlyRevenueItem struct {
	Month   string          `json:"month"` // e.g. "Apr 2026"
	Revenue decimal.Decimal `json:"revenue"`
	Count   int64           `json:"count"`
}

// PlatformSubscriptionItemResponse represents a subscribed tenant entry in the platform admin view
type PlatformSubscriptionItemResponse struct {
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

// PlatformTransactionItemResponse represents a billing transaction in the platform admin view
type PlatformTransactionItemResponse struct {
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

// UpdatePlatformTenantRequest represents payload for a platform admin to update tenant details
type UpdatePlatformTenantRequest struct {
	Name   string `json:"name" binding:"required,min=2,max=255"`
	Email  string `json:"email" binding:"required,email"`
	Phone  string `json:"phone" binding:"omitempty,max=30"`
	Status string `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}
