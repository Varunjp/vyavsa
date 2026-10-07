package tenant_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/auth"
	"github.com/Varunjp/vyavsa/internal/domain"
	tenantHandlerPkg "github.com/Varunjp/vyavsa/internal/handler/tenant"
	"github.com/Varunjp/vyavsa/internal/metrics"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockReportService struct {
	getFn func(ctx context.Context, tenantID uuid.UUID, dateStr string) (*domain.DailyReport, error)
	pdfFn func(ctx context.Context, tenantID uuid.UUID, dateStr string) ([]byte, string, error)
}

func (m *mockReportService) GetDailyReport(ctx context.Context, tenantID uuid.UUID, dateStr string) (*domain.DailyReport, error) {
	if m.getFn != nil {
		return m.getFn(ctx, tenantID, dateStr)
	}
	return &domain.DailyReport{
		Tenant: domain.TenantReportInfo{ID: tenantID, Name: "Test Tenant"},
		Date:   dateStr,
	}, nil
}

func (m *mockReportService) GeneratePDF(ctx context.Context, tenantID uuid.UUID, dateStr string) ([]byte, string, error) {
	if m.pdfFn != nil {
		return m.pdfFn(ctx, tenantID, dateStr)
	}
	return []byte("%PDF-1.4 test report"), "Test_Tenant_Daily_Report_2026-10-07.pdf", nil
}

func (m *mockReportService) SetLocation(loc *time.Location)     {}
func (m *mockReportService) SetMetrics(metric *metrics.Metrics) {}

func setupReportTestRouter(svc *mockReportService, tenantID *uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handler := tenantHandlerPkg.NewReportHandler(svc)

	// Middleware mimicking auth
	r.Use(func(c *gin.Context) {
		if tenantID != nil {
			tID := *tenantID
			uID := uuid.New()
			auth.SetClaims(c, &auth.CustomClaims{
				UserID:   uID,
				TenantID: &tID,
				Role:     auth.RoleTenantUser,
			})
		}
		c.Next()
	})

	r.GET("/api/v1/tenant/reports/daily", handler.GetDailyReport)
	r.GET("/api/v1/tenant/reports/daily/pdf", handler.DownloadDailyReportPDF)

	return r
}

func TestReportHandler_Unauthenticated(t *testing.T) {
	svc := &mockReportService{}
	router := setupReportTestRouter(svc, nil) // no tenant claims

	// 1. JSON endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily?date=2026-10-07", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 2. PDF endpoint
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily/pdf?date=2026-10-07", nil)
	router.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestReportHandler_GetDailyReport_TenantIsolation(t *testing.T) {
	authenticatedTenantID := uuid.New()
	var capturedTenantID uuid.UUID

	svc := &mockReportService{
		getFn: func(ctx context.Context, tID uuid.UUID, dateStr string) (*domain.DailyReport, error) {
			capturedTenantID = tID
			return &domain.DailyReport{
				Tenant: domain.TenantReportInfo{ID: tID, Name: "Isolated Tenant"},
				Date:   dateStr,
			}, nil
		},
	}

	router := setupReportTestRouter(svc, &authenticatedTenantID)

	w := httptest.NewRecorder()
	// Malicious client tries to send an arbitrary tenant_id in query params
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily?date=2026-10-07&tenant_id="+uuid.New().String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Server must have derived tenant exclusively from authenticated context
	assert.Equal(t, authenticatedTenantID, capturedTenantID)

	var resp struct {
		Success bool                `json:"success"`
		Data    *domain.DailyReport `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "Isolated Tenant", resp.Data.Tenant.Name)
}

func TestReportHandler_GetDailyReport_InvalidDate(t *testing.T) {
	tenantID := uuid.New()
	svc := &mockReportService{
		getFn: func(ctx context.Context, tID uuid.UUID, dateStr string) (*domain.DailyReport, error) {
			return nil, appErrors.NewBadRequest("invalid report date format: expected YYYY-MM-DD")
		},
	}

	router := setupReportTestRouter(svc, &tenantID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily?date=invalid-date", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReportHandler_DownloadDailyReportPDF_Success(t *testing.T) {
	tenantID := uuid.New()
	svc := &mockReportService{
		pdfFn: func(ctx context.Context, tID uuid.UUID, dateStr string) ([]byte, string, error) {
			return []byte("%PDF-1.4 dummy binary content"), "Acme_Stores_Daily_Report_2026-10-07.pdf", nil
		},
	}

	router := setupReportTestRouter(svc, &tenantID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily/pdf?date=2026-10-07", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	assert.Equal(t, `attachment; filename="Acme_Stores_Daily_Report_2026-10-07.pdf"`, w.Header().Get("Content-Disposition"))
	assert.Equal(t, "%PDF-1.4 dummy binary content", w.Body.String())
}

func TestReportHandler_DownloadDailyReportPDF_Failure(t *testing.T) {
	tenantID := uuid.New()
	svc := &mockReportService{
		pdfFn: func(ctx context.Context, tID uuid.UUID, dateStr string) ([]byte, string, error) {
			return nil, "", appErrors.NewInternal(errors.New("pdf generation engine error"))
		},
	}

	router := setupReportTestRouter(svc, &tenantID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/tenant/reports/daily/pdf?date=2026-10-07", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
