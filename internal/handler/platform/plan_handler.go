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

// PlanHandler handles HTTP requests for subscription plans
type PlanHandler struct {
	planService service.PlatformPlanService
}

// NewPlanHandler creates a new PlanHandler
func NewPlanHandler(planService service.PlatformPlanService) *PlanHandler {
	return &PlanHandler{
		planService: planService,
	}
}

// Create handles creating a new subscription plan
func (h *PlanHandler) Create(c *gin.Context) {
	var req dto.CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.NewValidation("invalid request payload", map[string]string{
			"error": err.Error(),
		}))
		return
	}

	plan, err := h.planService.CreatePlan(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, plan, "subscription plan created successfully")
}

// GetByID handles retrieving a plan by UUID
func (h *PlanHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, appErrors.NewBadRequest("invalid plan ID format"))
		return
	}

	plan, err := h.planService.GetPlanByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, plan, "subscription plan retrieved")
}

// List handles listing subscription plans with optional status filter and pagination
func (h *PlanHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	if status == "" {
		status = "active"
	} else if status == "all" {
		status = ""
	}

	plans, total, err := h.planService.ListPlans(c.Request.Context(), page, pageSize, status)
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

	response.Paginated(c, plans, pagination, "subscription plans retrieved")
}

// Update handles modifying an existing subscription plan
func (h *PlanHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, appErrors.NewBadRequest("invalid plan ID format"))
		return
	}

	var req dto.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.NewValidation("invalid request payload", map[string]string{
			"error": err.Error(),
		}))
		return
	}

	plan, err := h.planService.UpdatePlan(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, plan, "subscription plan updated successfully")
}

// Archive handles soft-deleting/archiving a subscription plan
func (h *PlanHandler) Archive(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, appErrors.NewBadRequest("invalid plan ID format"))
		return
	}

	if err := h.planService.ArchivePlan(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"id": id, "status": "archived"}, "subscription plan archived successfully")
}
