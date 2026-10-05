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
	"github.com/google/uuid"
)

// OperationsHandler handles HTTP requests for tenant operational modules
type OperationsHandler struct {
	svc *service.TenantOperationsService
}

func NewOperationsHandler(svc *service.TenantOperationsService) *OperationsHandler {
	return &OperationsHandler{svc: svc}
}

func getTenantContext(c *gin.Context) (uuid.UUID, uuid.UUID, string, error) {
	claims, err := auth.GetClaims(c)
	if err != nil || claims.TenantID == nil || *claims.TenantID == uuid.Nil {
		return uuid.Nil, uuid.Nil, "", appErrors.NewUnauthorized("unauthorized: missing or invalid tenant claims")
	}
	return *claims.TenantID, claims.UserID, claims.Role, nil
}

func parsePagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

func parseUUID(c *gin.Context, paramName string) (uuid.UUID, error) {
	raw := c.Param(paramName)
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, appErrors.NewValidation("invalid parameter format", map[string]string{paramName: "must be a valid UUID format"})
	}
	return id, nil
}

func validationErr(err error) *appErrors.AppError {
	return appErrors.ParseBindingError(err)
}

// ==========================================
// 1. Tenant Users Handlers (Admin only)
// ==========================================

func (h *OperationsHandler) CreateTenantUser(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreateTenantUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	user, err := h.svc.CreateTenantUser(c.Request.Context(), tenantID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToTenantUserItemResponse(user), "tenant user created successfully")
}

func (h *OperationsHandler) UpdateTenantUser(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateTenantUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	user, err := h.svc.UpdateTenantUser(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToTenantUserItemResponse(user), "tenant user updated successfully")
}

func (h *OperationsHandler) DeleteTenantUser(c *gin.Context) {
	tenantID, userID, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeleteTenantUser(c.Request.Context(), tenantID, userID, id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"deleted": true}, "tenant user deleted successfully")
}

func (h *OperationsHandler) GetTenantUserByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	user, err := h.svc.GetTenantUserByID(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToTenantUserItemResponse(user), "tenant user retrieved")
}

func (h *OperationsHandler) ListTenantUsers(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	search := c.Query("search")
	role := c.Query("role")
	status := c.Query("status")

	users, total, err := h.svc.ListTenantUsers(c.Request.Context(), tenantID, page, pageSize, search, role, status)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := make([]dto.TenantUserItemResponse, len(users))
	for i := range users {
		resp[i] = dto.ToTenantUserItemResponse(&users[i])
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, resp, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "tenant users listed")
}

// ==========================================
// 2. Employee Handlers (Admin only)
// ==========================================

func (h *OperationsHandler) CreateEmployee(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	emp, err := h.svc.CreateEmployee(c.Request.Context(), tenantID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToEmployeeResponse(emp), "employee created successfully")
}

func (h *OperationsHandler) UpdateEmployee(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	emp, err := h.svc.UpdateEmployee(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToEmployeeResponse(emp), "employee updated successfully")
}

func (h *OperationsHandler) DeleteEmployee(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeleteEmployee(c.Request.Context(), tenantID, id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"deleted": true}, "employee deleted successfully")
}

func (h *OperationsHandler) GetEmployeeByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	emp, err := h.svc.GetEmployeeByID(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToEmployeeResponse(emp), "employee retrieved")
}

func (h *OperationsHandler) ListEmployees(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	search := c.Query("search")
	status := c.Query("status")

	employees, total, err := h.svc.ListEmployees(c.Request.Context(), tenantID, page, pageSize, search, status)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := make([]dto.EmployeeResponse, len(employees))
	for i := range employees {
		resp[i] = dto.ToEmployeeResponse(&employees[i])
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, resp, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "employees listed")
}

// ==========================================
// 3. Customer Handlers (Admin only)
// ==========================================

func (h *OperationsHandler) CreateCustomer(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	cust, err := h.svc.CreateCustomer(c.Request.Context(), tenantID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToCustomerResponse(cust), "customer created successfully")
}

func (h *OperationsHandler) UpdateCustomer(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	cust, err := h.svc.UpdateCustomer(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToCustomerResponse(cust), "customer updated successfully")
}

func (h *OperationsHandler) DeleteCustomer(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeleteCustomer(c.Request.Context(), tenantID, id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"deleted": true}, "customer deleted successfully")
}

func (h *OperationsHandler) GetCustomerByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	detailResp, err := h.svc.GetCustomerDetails(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, detailResp, "customer retrieved")
}

func (h *OperationsHandler) GetCustomerStatement(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	stmt, err := h.svc.GetCustomerStatement(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, stmt, "customer statement retrieved")
}

func (h *OperationsHandler) RecordSupplierPayment(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.RecordSupplierPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.NewBadRequest("invalid payment payload: "+err.Error()))
		return
	}

	if id, err := parseUUID(c, "id"); err == nil && id != uuid.Nil {
		req.CustomerID = &id
	}

	resp, err := h.svc.RecordSupplierPayment(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp, "supplier payment recorded successfully")
}

func (h *OperationsHandler) GetCustomerBalance(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	balanceResp, err := h.svc.GetCustomerBalance(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, balanceResp, "customer balance retrieved")
}

func (h *OperationsHandler) ListCustomers(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	search := c.Query("search")
	status := c.Query("status")

	customers, total, err := h.svc.ListCustomers(c.Request.Context(), tenantID, page, pageSize, search, status)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, customers, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "customers listed")
}

func (h *OperationsHandler) AdjustCustomerBalance(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.AdjustCustomerBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	cust, err := h.svc.AdjustCustomerBalance(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToCustomerResponse(cust), "customer balance adjusted successfully")
}

func (h *OperationsHandler) ListCustomerAdjustments(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	adjustments, total, err := h.svc.ListCustomerAdjustments(c.Request.Context(), tenantID, id, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := make([]dto.CustomerBalanceAdjustmentResponse, len(adjustments))
	for i, a := range adjustments {
		resp[i] = dto.CustomerBalanceAdjustmentResponse{
			ID:               a.ID,
			TenantID:         a.TenantID,
			CustomerID:       a.CustomerID,
			CustomerName:     a.CustomerName,
			PreviousBalance:  a.PreviousBalance,
			NewBalance:       a.NewBalance,
			AdjustmentAmount: a.AdjustmentAmount,
			Reason:           a.Reason,
			CreatedAt:        a.CreatedAt,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, resp, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "customer adjustments listed")
}

// ==========================================
// 4. Bank Handlers (Admin only)
// ==========================================

func (h *OperationsHandler) CreateBank(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreateBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	bank, err := h.svc.CreateBank(c.Request.Context(), tenantID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, dto.ToBankResponse(bank), "bank account created successfully")
}

func (h *OperationsHandler) UpdateBank(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateBankRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	bank, err := h.svc.UpdateBank(c.Request.Context(), tenantID, id, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToBankResponse(bank), "bank account updated successfully")
}

func (h *OperationsHandler) DeleteBank(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeleteBank(c.Request.Context(), tenantID, id); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"deleted": true}, "bank account deleted successfully")
}

func (h *OperationsHandler) GetBankByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	bank, err := h.svc.GetBankByID(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, dto.ToBankResponse(bank), "bank account retrieved")
}

func (h *OperationsHandler) ListBanks(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	search := c.Query("search")
	status := c.Query("status")

	banks, total, err := h.svc.ListBanks(c.Request.Context(), tenantID, page, pageSize, search, status)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := make([]dto.BankResponse, len(banks))
	for i := range banks {
		resp[i] = dto.ToBankResponse(&banks[i])
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, resp, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "bank accounts listed")
}

func (h *OperationsHandler) ListBankTransactions(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	txs, total, err := h.svc.ListBankTransactions(c.Request.Context(), tenantID, id, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	resp := make([]dto.BankTransactionResponse, len(txs))
	for i, t := range txs {
		resp[i] = dto.BankTransactionResponse{
			ID:              t.ID,
			TenantID:        t.TenantID,
			BankID:          t.BankID,
			Amount:          t.Amount,
			TransactionType: t.TransactionType,
			Reason:          t.Reason,
			SaleType:        t.SaleType,
			SaleID:          t.SaleID,
			CreatedAt:       t.CreatedAt,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, resp, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "bank transactions listed")
}

// ==========================================
// 5. Line Sale Handlers
// ==========================================

func (h *OperationsHandler) CreateLineSale(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreateLineSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	sale, err := h.svc.CreateLineSale(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, sale, "line sale recorded successfully")
}

func (h *OperationsHandler) UpdateLineSale(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateLineSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	sale, err := h.svc.UpdateLineSale(c.Request.Context(), tenantID, id, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, sale, "line sale updated successfully")
}

func (h *OperationsHandler) DeleteLineSale(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeleteLineSale(c.Request.Context(), tenantID, id, role); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"deleted": true}, "line sale deleted successfully")
}

func (h *OperationsHandler) GetLineSaleByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	sale, err := h.svc.GetLineSaleByID(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, sale, "line sale retrieved")
}

func (h *OperationsHandler) ListLineSales(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	date := c.Query("date")
	search := c.Query("search")
	var customerID *uuid.UUID
	if cidStr := c.Query("customer_id"); cidStr != "" {
		if parsed, err := uuid.Parse(cidStr); err == nil {
			customerID = &parsed
		}
	}

	sales, total, err := h.svc.ListLineSales(c.Request.Context(), tenantID, role, page, pageSize, date, customerID, search)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, sales, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "line sales listed")
}

// ==========================================
// 6. Counter Sale Handlers
// ==========================================

func (h *OperationsHandler) CreateCounterSale(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreateCounterSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	sale, err := h.svc.CreateCounterSale(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, sale, "counter sale recorded successfully")
}

func (h *OperationsHandler) UpdateCounterSale(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateCounterSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	sale, err := h.svc.UpdateCounterSale(c.Request.Context(), tenantID, id, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, sale, "counter sale updated successfully")
}

func (h *OperationsHandler) DeleteCounterSale(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeleteCounterSale(c.Request.Context(), tenantID, id, role); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"deleted": true}, "counter sale deleted successfully")
}

func (h *OperationsHandler) GetCounterSaleByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	sale, err := h.svc.GetCounterSaleByID(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, sale, "counter sale retrieved")
}

func (h *OperationsHandler) ListCounterSales(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	date := c.Query("date")
	search := c.Query("search")

	sales, total, err := h.svc.ListCounterSales(c.Request.Context(), tenantID, role, page, pageSize, date, search)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, sales, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "counter sales listed")
}

// ==========================================
// 7. Purchase Handlers
// ==========================================

func (h *OperationsHandler) CreatePurchase(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreatePurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	purch, err := h.svc.CreatePurchase(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, purch, "purchase recorded successfully")
}

func (h *OperationsHandler) UpdatePurchase(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdatePurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	purch, err := h.svc.UpdatePurchase(c.Request.Context(), tenantID, id, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, purch, "purchase updated successfully")
}

func (h *OperationsHandler) DeletePurchase(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeletePurchase(c.Request.Context(), tenantID, id, role); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"deleted": true}, "purchase deleted successfully")
}

func (h *OperationsHandler) GetPurchaseByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	purch, err := h.svc.GetPurchaseByID(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, purch, "purchase retrieved")
}

func (h *OperationsHandler) ListPurchases(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	date := c.Query("date")
	search := c.Query("search")
	var customerID *uuid.UUID
	if cid := c.Query("customer_id"); cid != "" {
		if id, err := uuid.Parse(cid); err == nil && id != uuid.Nil {
			customerID = &id
		}
	}

	purchases, total, err := h.svc.ListPurchases(c.Request.Context(), tenantID, role, page, pageSize, date, customerID, search)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, purchases, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "purchases listed")
}

func (h *OperationsHandler) RecordPurchaseSettlementPayment(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.RecordSupplierPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, appErrors.NewBadRequest("invalid payment payload: "+err.Error()))
		return
	}

	if id, err := parseUUID(c, "id"); err == nil && id != uuid.Nil {
		req.PurchaseID = &id
	}

	resp, err := h.svc.RecordSupplierPayment(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, resp, "purchase settlement payment recorded successfully")
}

// ==========================================
// 8. Expense Handlers
// ==========================================

func (h *OperationsHandler) CreateExpense(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	exp, err := h.svc.CreateExpense(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, exp, "expense recorded successfully")
}

func (h *OperationsHandler) UpdateExpense(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	exp, err := h.svc.UpdateExpense(c.Request.Context(), tenantID, id, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, exp, "expense updated successfully")
}

func (h *OperationsHandler) DeleteExpense(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	if err := h.svc.DeleteExpense(c.Request.Context(), tenantID, id, role); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"deleted": true}, "expense deleted successfully")
}

func (h *OperationsHandler) GetExpenseByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	exp, err := h.svc.GetExpenseByID(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, exp, "expense retrieved")
}

func (h *OperationsHandler) ListExpenses(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	date := c.Query("date")
	search := c.Query("search")

	expenses, total, err := h.svc.ListExpenses(c.Request.Context(), tenantID, role, page, pageSize, date, search)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, expenses, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "expenses listed")
}

// ==========================================
// 9. Attendance & Overtime & Advances Handlers
// ==========================================

func (h *OperationsHandler) RecordAttendance(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.RecordAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	att, err := h.svc.RecordAttendance(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, att, "attendance recorded successfully")
}

func (h *OperationsHandler) RecordOvertime(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.RecordOvertimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	att, err := h.svc.RecordOvertime(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, att, "employee overtime recorded successfully")
}

func (h *OperationsHandler) RecordAdvance(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.RecordAdvanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	att, err := h.svc.RecordAdvance(c.Request.Context(), tenantID, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, att, "employee advance recorded successfully")
}

func (h *OperationsHandler) UpdateAttendance(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateAttendanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	att, err := h.svc.UpdateAttendance(c.Request.Context(), tenantID, id, role, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, att, "attendance record updated successfully")
}

func (h *OperationsHandler) GetAttendanceByID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	id, err := parseUUID(c, "id")
	if err != nil {
		response.Error(c, err)
		return
	}

	att, err := h.svc.GetAttendanceByID(c.Request.Context(), tenantID, id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, att, "attendance retrieved")
}

func (h *OperationsHandler) ListAttendance(c *gin.Context) {
	tenantID, _, role, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	date := c.Query("date")
	var empID *uuid.UUID
	if empStr := c.Query("employee_id"); empStr != "" {
		if parsed, err := uuid.Parse(empStr); err == nil {
			empID = &parsed
		}
	}

	list, total, err := h.svc.ListAttendance(c.Request.Context(), tenantID, role, page, pageSize, date, empID)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, list, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "attendance listed")
}

// ==========================================
// 10. Salary Handlers (Admin only)
// ==========================================

func (h *OperationsHandler) ListSalaries(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	search := c.Query("search")

	salaries, total, err := h.svc.ListSalaries(c.Request.Context(), tenantID, page, pageSize, search)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, salaries, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "salaries listed")
}

func (h *OperationsHandler) ListPendingSalaries(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	salaries, total, err := h.svc.ListPendingSalaries(c.Request.Context(), tenantID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, salaries, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "pending salaries listed")
}

func (h *OperationsHandler) GetSalaryByEmployeeID(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	empID, err := parseUUID(c, "employee_id")
	if err != nil {
		response.Error(c, err)
		return
	}

	salary, err := h.svc.GetSalaryByEmployeeID(c.Request.Context(), tenantID, empID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, salary, "salary balance retrieved")
}

func (h *OperationsHandler) UpdateSalaryBalance(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	empID, err := parseUUID(c, "employee_id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.UpdateSalaryBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	salary, err := h.svc.UpdateSalaryBalance(c.Request.Context(), tenantID, empID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, salary, "salary balance updated successfully")
}

func (h *OperationsHandler) PaySalary(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	empID, err := parseUUID(c, "employee_id")
	if err != nil {
		response.Error(c, err)
		return
	}

	var req dto.PaySalaryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, validationErr(err))
		return
	}

	payment, err := h.svc.PaySalary(c.Request.Context(), tenantID, empID, &req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, payment, "salary payment processed successfully")
}

func (h *OperationsHandler) ListSalaryPayments(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	page, pageSize := parsePagination(c)
	var empID *uuid.UUID
	if raw := c.Param("employee_id"); raw != "" {
		if parsed, err := uuid.Parse(raw); err == nil {
			empID = &parsed
		}
	}

	payments, total, err := h.svc.ListSalaryPayments(c.Request.Context(), tenantID, empID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))
	response.Paginated(c, payments, response.Pagination{
		Page:       page,
		PageSize:   pageSize,
		TotalItems: total,
		TotalPages: totalPages,
	}, "salary payments listed")
}

// ==========================================
// 11. Statistics & Financial Dashboard Handlers
// ==========================================

func (h *OperationsHandler) GetDailyStats(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	date := c.Query("date")
	stats, err := h.svc.GetDailyStats(c.Request.Context(), tenantID, date)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, stats, "daily business statistics retrieved")
}

func (h *OperationsHandler) GetFinancialMetrics(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	metrics, err := h.svc.GetFinancialMetrics(c.Request.Context(), tenantID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, metrics, "financial summary and dashboard metrics retrieved")
}

func (h *OperationsHandler) GetTodayOverview(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	overview, err := h.svc.GetTodayOverview(c.Request.Context(), tenantID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, overview, "today's overview metrics retrieved")
}
