package platform

import (
	"math"
	"strconv"

	"github.com/Varunjp/vyavsa/internal/dto"
	"github.com/Varunjp/vyavsa/internal/service"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantHandler handles platform-level HTTP operations for tenants
type TenantHandler struct {
	tenantService service.TenantService
}

// NewTenantHandler creates a new platform TenantHandler
func NewTenantHandler(tenantService service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// Onboard handles onboarding a new tenant organisation with atomic setup
func (h *TenantHandler) Onboard(c *gin.Context) {
	var req dto.OnboardTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	result, err := h.tenantService.OnboardTenant(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, result, "tenant onboarded successfully")
}

// GetByID handles fetching complete tenant details
func (h *TenantHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, appErrors.NewBadRequest("invalid tenant ID format"))
		return
	}

	tenant, err := h.tenantService.GetTenantByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, tenant, "tenant details retrieved")
}

// List handles listing tenants with pagination, status filtering, and search
func (h *TenantHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	search := c.Query("search")

	tenants, total, err := h.tenantService.ListTenants(c.Request.Context(), page, pageSize, status, search)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	pagination := response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}

	response.Paginated(c, tenants, pagination, "tenants retrieved")
}

// UpdateStatus handles updating a tenant's lifecycle status
func (h *TenantHandler) UpdateStatus(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, appErrors.NewBadRequest("invalid tenant ID format"))
		return
	}

	var req dto.UpdateTenantStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	if err := h.tenantService.UpdateTenantStatus(c.Request.Context(), id, req.Status); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"id": id, "status": req.Status}, "tenant status updated successfully")
}

// ChangeSubscription handles updating or renewing a tenant's subscription plan
func (h *TenantHandler) ChangeSubscription(c *gin.Context) {
	idParam := c.Param("id")
	tenantID, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, appErrors.NewBadRequest("invalid tenant ID format"))
		return
	}

	var req dto.ChangeSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.ParseBindingError(err))
		return
	}

	sub, err := h.tenantService.ChangeSubscription(c.Request.Context(), tenantID, req.PlanID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, sub, "tenant subscription updated successfully")
}
