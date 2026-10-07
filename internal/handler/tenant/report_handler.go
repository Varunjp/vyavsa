package tenant

import (
	"fmt"
	"net/http"

	"github.com/Varunjp/vyavsa/internal/service"
	"github.com/Varunjp/vyavsa/pkg/response"
	"github.com/gin-gonic/gin"
)

// ReportHandler handles HTTP requests for tenant financial reports
type ReportHandler struct {
	reportService service.DailyReportService
}

// NewReportHandler creates a new ReportHandler instance
func NewReportHandler(reportService service.DailyReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// GetDailyReport handles GET /api/v1/tenant/reports/daily
func (h *ReportHandler) GetDailyReport(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	dateStr := c.Query("date")
	report, err := h.reportService.GetDailyReport(c.Request.Context(), tenantID, dateStr)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, report, "daily business report retrieved successfully")
}

// DownloadDailyReportPDF handles GET /api/v1/tenant/reports/daily/pdf
func (h *ReportHandler) DownloadDailyReportPDF(c *gin.Context) {
	tenantID, _, _, err := getTenantContext(c)
	if err != nil {
		response.Error(c, err)
		return
	}

	dateStr := c.Query("date")
	pdfBytes, filename, err := h.reportService.GeneratePDF(c.Request.Context(), tenantID, dateStr)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
