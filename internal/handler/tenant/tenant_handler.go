package tenant

import (
	"math"
	"strconv"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/service"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// TenantHandler handles tenant HTTP operations
type TenantHandler struct {
	tenantService service.TenantService
}

// NewTenantHandler creates a new TenantHandler
func NewTenantHandler(tenantService service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// Register handles self-service registration for a new tenant
func (h *TenantHandler) Register(c *gin.Context) {
	var req dto.TenantRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	result, err := h.tenantService.RegisterTenant(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, result, "tenant registered successfully")
}

// GetProfile returns the business profile of the authenticated tenant
func (h *TenantHandler) GetProfile(c *gin.Context) {
	claims, err := auth.GetClaims(c)
	if err != nil || claims.TenantID == nil {
		response.Error(c, appErrors.NewUnauthorized("missing or invalid tenant claims"))
		return
	}

	profile, err := h.tenantService.GetTenantProfile(c.Request.Context(), *claims.TenantID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, profile, "tenant profile retrieved")
}

// UpdateSettings handles updating business organization details by tenant admin
func (h *TenantHandler) UpdateSettings(c *gin.Context) {
	claims, err := auth.GetClaims(c)
	if err != nil || claims.TenantID == nil {
		response.Error(c, appErrors.NewUnauthorized("missing or invalid tenant claims"))
		return
	}

	var req dto.UpdateTenantSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	updated, err := h.tenantService.UpdateTenantSettings(c.Request.Context(), *claims.TenantID, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, updated, "business details updated successfully")
}

// GetFinancialSummary returns authoritative real-time balances for the authenticated tenant
func (h *TenantHandler) GetFinancialSummary(c *gin.Context) {
	claims, err := auth.GetClaims(c)
	if err != nil || claims.TenantID == nil {
		response.Error(c, appErrors.NewUnauthorized("missing or invalid tenant claims"))
		return
	}

	summary, err := h.tenantService.GetTenantFinancialSummary(c.Request.Context(), *claims.TenantID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, summary, "tenant financial summary retrieved")
}

// GetSubscription returns subscription and plan details for the authenticated tenant
func (h *TenantHandler) GetSubscription(c *gin.Context) {
	claims, err := auth.GetClaims(c)
	if err != nil || claims.TenantID == nil {
		response.Error(c, appErrors.NewUnauthorized("missing or invalid tenant claims"))
		return
	}

	sub, err := h.tenantService.GetTenantSubscription(c.Request.Context(), *claims.TenantID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, sub, "tenant subscription retrieved")
}

// ListPlans returns all available active subscription plans for the tenant to browse or purchase
func (h *TenantHandler) ListPlans(c *gin.Context) {
	plans, err := h.tenantService.ListAvailablePlans(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, plans, "available subscription plans retrieved")
}

// PurchasePlan handles purchasing or upgrading a subscription plan
func (h *TenantHandler) PurchasePlan(c *gin.Context) {
	claims, err := auth.GetClaims(c)
	if err != nil || claims.TenantID == nil {
		response.Error(c, appErrors.NewUnauthorized("missing or invalid tenant claims"))
		return
	}

	var req dto.PurchasePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	sub, txn, err := h.tenantService.PurchasePlan(c.Request.Context(), *claims.TenantID, req)
	if err != nil {
		if txn != nil {
			c.JSON(400, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PAYMENT_FAILED",
					"message": err.Error(),
				},
				"data": gin.H{
					"transaction": txn,
				},
			})
			return
		}
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"subscription": sub,
		"transaction":  txn,
	}, "subscription updated successfully")
}

// ListTransactions returns the paginated payment/transaction history for subscription plans
func (h *TenantHandler) ListTransactions(c *gin.Context) {
	claims, err := auth.GetClaims(c)
	if err != nil || claims.TenantID == nil {
		response.Error(c, appErrors.NewUnauthorized("missing or invalid tenant claims"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	txns, total, err := h.tenantService.ListTenantTransactions(c.Request.Context(), *claims.TenantID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, txns, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "transaction history retrieved successfully")
}
