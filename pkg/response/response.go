package response

import (
	"net/http"

	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/gin-gonic/gin"
)

// Response represents the standard API response structure
type Response struct {
	Success    bool        `json:"success"`
	Data       any         `json:"data,omitempty"`
	Message    string      `json:"message,omitempty"`
	Error      *ErrorInfo  `json:"error,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// ErrorInfo represents error details in an error response
type ErrorInfo struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

// Success sends a standard 200 OK success response
func Success(c *gin.Context, data any, message ...string) {
	msg := "operation successful"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Message: msg,
	})
}

// Created sends a standard 201 Created success response
func Created(c *gin.Context, data any, message ...string) {
	msg := "resource created successfully"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
		Message: msg,
	})
}

// Paginated sends a standard 200 OK paginated success response
func Paginated(c *gin.Context, data any, pagination Pagination, message ...string) {
	msg := "operation successful"
	if len(message) > 0 && message[0] != "" {
		msg = message[0]
	}

	c.JSON(http.StatusOK, Response{
		Success:    true,
		Data:       data,
		Pagination: &pagination,
		Message:    msg,
	})
}

// Error maps an error to standard JSON response using centralized AppError
func Error(c *gin.Context, err error) {
	appErr := appErrors.FromError(err)

	details := appErr.Details
	fields := appErr.Fields
	if details == nil && fields != nil {
		details = fields
	}
	if fields == nil && details != nil {
		fields = details
	}

	c.JSON(appErr.HTTPStatus, Response{
		Success: false,
		Message: appErr.Message,
		Error: &ErrorInfo{
			Code:    appErr.Code,
			Message: appErr.Message,
			Details: details,
			Fields:  fields,
		},
	})
}

// CustomError sends an explicit error with status code, code, and message
func CustomError(c *gin.Context, status int, code, message string, details ...map[string]string) {
	var d map[string]string
	if len(details) > 0 {
		d = details[0]
	}

	c.JSON(status, Response{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: d,
			Fields:  d,
		},
	})
}
