package dto

import (
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OnboardTenantRequest represents the payload for platform admins to onboard a new tenant
type OnboardTenantRequest struct {
	// Tenant Organization Details
	Name  string `json:"name" binding:"required,min=2,max=255"`
	Email string `json:"email" binding:"required,email"`
	Phone string `json:"phone" binding:"omitempty,max=30"`

	// Initial Tenant Administrator User Details
	AdminName     string `json:"admin_name" binding:"omitempty,min=2,max=255"` // Defaults to Name if empty
	AdminEmail    string `json:"admin_email" binding:"omitempty,email"`       // Defaults to Email if empty
	AdminPassword string `json:"admin_password" binding:"required,min=6,max=72"`

	// Subscription Plan Selection
	PlanID uuid.UUID `json:"plan_id" binding:"required"`
}

// TenantRegisterRequest represents self-service public registration for a new tenant
type TenantRegisterRequest struct {
	Name      string     `json:"name" binding:"required,min=2,max=255"`       // Organization / Business Name
	AdminName string     `json:"admin_name" binding:"omitempty,min=2,max=255"` // Optional: Defaults to Name if omitted
	Email     string     `json:"email" binding:"required,email"`              // Used for both Tenant Organization and Admin User
	Password  string     `json:"password" binding:"required,min=6,max=72"`     // Admin User Password
	Phone     string     `json:"phone" binding:"omitempty,max=30"`
	PlanID    *uuid.UUID `json:"plan_id,omitempty"`                           // Optional: Defaults to free plan if omitted
}

// UpdateTenantStatusRequest represents tenant status modification payload
type UpdateTenantStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive suspended"`
}

// ChangeSubscriptionRequest represents changing subscription plan
type ChangeSubscriptionRequest struct {
	PlanID uuid.UUID `json:"plan_id" binding:"required"`
}

// TenantResponse represents public summary of a tenant
type TenantResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToTenantResponse converts domain.Tenant to TenantResponse
func ToTenantResponse(t *domain.Tenant) TenantResponse {
	return TenantResponse{
		ID:        t.ID,
		Name:      t.Name,
		Email:     t.Email,
		Phone:     t.Phone,
		Status:    t.Status,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// ToTenantListResponse converts a slice of domain.Tenant to TenantResponse
func ToTenantListResponse(tenants []domain.Tenant) []TenantResponse {
	resp := make([]TenantResponse, len(tenants))
	for i := range tenants {
		resp[i] = ToTenantResponse(&tenants[i])
	}
	return resp
}

// TenantSubscriptionResponse represents subscription details for a tenant
type TenantSubscriptionResponse struct {
	ID              uuid.UUID  `json:"id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	CurrentPlanID   uuid.UUID  `json:"current_plan_id"`
	CurrentPlanName string     `json:"current_plan_name"`
	Status          string     `json:"status"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ToSubscriptionResponse converts domain.PlatformSubscription to TenantSubscriptionResponse
func ToSubscriptionResponse(s *domain.PlatformSubscription) TenantSubscriptionResponse {
	return TenantSubscriptionResponse{
		ID:              s.ID,
		TenantID:        s.TenantID,
		CurrentPlanID:   s.CurrentPlanID,
		CurrentPlanName: s.CurrentPlanName,
		Status:          s.Status,
		EndDate:         s.EndDate,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

// TenantFinancialSummaryResponse represents financial summary balances
type TenantFinancialSummaryResponse struct {
	CashBalance     decimal.Decimal `json:"cash_balance"`
	BankBalance     decimal.Decimal `json:"bank_balance"`
	TotalReceivable decimal.Decimal `json:"total_receivable"`
	TotalPayable    decimal.Decimal `json:"total_payable"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// ToFinancialSummaryResponse converts domain.TenantFinancialSummary to TenantFinancialSummaryResponse
func ToFinancialSummaryResponse(s *domain.TenantFinancialSummary) TenantFinancialSummaryResponse {
	return TenantFinancialSummaryResponse{
		CashBalance:     s.CashBalance,
		BankBalance:     s.BankBalance,
		TotalReceivable: s.TotalReceivable,
		TotalPayable:    s.TotalPayable,
		UpdatedAt:       s.UpdatedAt,
	}
}

// TenantAdminUserResponse represents the primary admin user info
type TenantAdminUserResponse struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Email  string    `json:"email"`
	Role   string    `json:"role"`
	Status string    `json:"status"`
}

// ToAdminUserResponse converts domain.TenantUser to TenantAdminUserResponse
func ToAdminUserResponse(u *domain.TenantUser) TenantAdminUserResponse {
	return TenantAdminUserResponse{
		ID:     u.ID,
		Name:   u.Name,
		Email:  u.Email,
		Role:   u.Role,
		Status: u.Status,
	}
}

// TenantDetailResponse provides full detailed tenant view
type TenantDetailResponse struct {
	Tenant           TenantResponse                  `json:"tenant"`
	AdminUser        *TenantAdminUserResponse        `json:"admin_user,omitempty"`
	Subscription     *TenantSubscriptionResponse     `json:"subscription,omitempty"`
	FinancialSummary *TenantFinancialSummaryResponse `json:"financial_summary,omitempty"`
}

// OnboardTenantResponse represents response returned after successful onboarding
type OnboardTenantResponse struct {
	Tenant           TenantResponse                 `json:"tenant"`
	AdminUser        TenantAdminUserResponse        `json:"admin_user"`
	Subscription     TenantSubscriptionResponse     `json:"subscription"`
	FinancialSummary TenantFinancialSummaryResponse `json:"financial_summary"`
}

// TenantRegisterResponse represents the result of successful tenant self-registration
type TenantRegisterResponse struct {
	Tokens           *TokenResponse                 `json:"tokens,omitempty"`
	Tenant           TenantResponse                 `json:"tenant"`
	AdminUser        TenantAdminUserResponse        `json:"admin_user"`
	Subscription     TenantSubscriptionResponse     `json:"subscription"`
	FinancialSummary TenantFinancialSummaryResponse `json:"financial_summary"`
}
