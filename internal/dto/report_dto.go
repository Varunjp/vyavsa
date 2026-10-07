package dto

import "github.com/Varunjp/vyavsa/internal/domain"

// DailyReportQueryRequest holds query parameters for retrieving daily reports
type DailyReportQueryRequest struct {
	Date string `form:"date" binding:"required"`
}

// DailyReportResponse represents the JSON response envelope containing a daily business report
type DailyReportResponse struct {
	Report *domain.DailyReport `json:"report"`
}
