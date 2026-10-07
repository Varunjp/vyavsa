package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDailyReportRepo struct {
	reportFn func(ctx context.Context, tenantID uuid.UUID, start, end time.Time, loc *time.Location) (*domain.DailyReport, error)
}

func (m *mockDailyReportRepo) GetDailyReportData(ctx context.Context, tenantID uuid.UUID, start, end time.Time, loc *time.Location) (*domain.DailyReport, error) {
	if m.reportFn != nil {
		return m.reportFn(ctx, tenantID, start, end, loc)
	}
	return &domain.DailyReport{
		Tenant: domain.TenantReportInfo{ID: tenantID, Name: "Acme Corp"},
		Date:   start.Format("2006-01-02"),
	}, nil
}

type mockPDFGen struct {
	genFn func(report *domain.DailyReport) ([]byte, error)
}

func (m *mockPDFGen) GenerateDailyReportPDF(report *domain.DailyReport) ([]byte, error) {
	if m.genFn != nil {
		return m.genFn(report)
	}
	return []byte("%PDF-1.4 mock content"), nil
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC Stores", "ABC_Stores"},
		{"Vijay & Sons, Pvt. Ltd.!", "Vijay_Sons_Pvt_Ltd"},
		{"  Trimmed Name  ", "Trimmed_Name"},
		{"", "Business"},
		{"!@#$%^&*()", "Business"},
		{"A Very Long Name That Exceeds Fifty Characters In Length Should Be Cut Off Cleanly", "A_Very_Long_Name_That_Exceeds_Fifty_Characters_In_"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res := SanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, res)
		})
	}
}

func TestDailyReportService_GetDailyReport_InvalidDate(t *testing.T) {
	svc := NewDailyReportService(&mockDailyReportRepo{}, &mockPDFGen{}, nil)

	_, err := svc.GetDailyReport(context.Background(), uuid.New(), "invalid-date-format")
	require.Error(t, err)

	var appErr *appErrors.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, appErrors.CodeBadRequest, appErr.Code)
}

func TestDailyReportService_GetDailyReport_Success(t *testing.T) {
	tenantID := uuid.New()
	var capturedStart, capturedEnd time.Time

	repo := &mockDailyReportRepo{
		reportFn: func(ctx context.Context, tID uuid.UUID, start, end time.Time, loc *time.Location) (*domain.DailyReport, error) {
			capturedStart = start
			capturedEnd = end
			return &domain.DailyReport{
				Tenant:        domain.TenantReportInfo{ID: tID, Name: "Test Tenant"},
				Date:          "2026-10-07",
				FormattedDate: "07 October 2026",
			}, nil
		},
	}

	svc := NewDailyReportService(repo, &mockPDFGen{}, nil)
	svc.SetLocation(time.UTC)

	report, err := svc.GetDailyReport(context.Background(), tenantID, "2026-10-07")
	require.NoError(t, err)
	assert.Equal(t, "2026-10-07", report.Date)
	assert.Equal(t, "Test Tenant", report.Tenant.Name)

	assert.Equal(t, 2026, capturedStart.Year())
	assert.Equal(t, time.Month(10), capturedStart.Month())
	assert.Equal(t, 7, capturedStart.Day())
	assert.Equal(t, 0, capturedStart.Hour())

	assert.Equal(t, 2026, capturedEnd.Year())
	assert.Equal(t, time.Month(10), capturedEnd.Month())
	assert.Equal(t, 8, capturedEnd.Day())
	assert.Equal(t, 0, capturedEnd.Hour())
}

func TestDailyReportService_GeneratePDF_Success(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockDailyReportRepo{
		reportFn: func(ctx context.Context, tID uuid.UUID, start, end time.Time, loc *time.Location) (*domain.DailyReport, error) {
			return &domain.DailyReport{
				Tenant:        domain.TenantReportInfo{ID: tID, Name: "Apex Retail Mart"},
				Date:          "2026-10-07",
				FormattedDate: "07 October 2026",
			}, nil
		},
	}

	pdfGen := &mockPDFGen{
		genFn: func(report *domain.DailyReport) ([]byte, error) {
			return []byte("%PDF-1.4 dummy report"), nil
		},
	}

	svc := NewDailyReportService(repo, pdfGen, nil)
	pdfBytes, filename, err := svc.GeneratePDF(context.Background(), tenantID, "2026-10-07")
	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF-1.4 dummy report"), pdfBytes)
	assert.Equal(t, "Apex_Retail_Mart_Daily_Report_2026-10-07.pdf", filename)
}

func TestDailyReportService_GeneratePDF_ErrorPropagation(t *testing.T) {
	tenantID := uuid.New()
	repo := &mockDailyReportRepo{
		reportFn: func(ctx context.Context, tID uuid.UUID, start, end time.Time, loc *time.Location) (*domain.DailyReport, error) {
			return nil, appErrors.NewNotFound("tenant not found")
		},
	}

	svc := NewDailyReportService(repo, &mockPDFGen{}, nil)
	_, _, err := svc.GeneratePDF(context.Background(), tenantID, "2026-10-07")
	require.Error(t, err)
	assert.True(t, appErrors.IsNotFound(err))
}
