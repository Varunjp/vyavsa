package tenant

import (
	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/service"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// TenantHandler handles tenant-scoped HTTP operations
type TenantHandler struct {
	tenantService service.TenantService
}

// NewTenantHandler creates a new tenant-scoped TenantHandler
func NewTenantHandler(tenantService service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
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
