package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/Varunjp/vyavsa/internal/metrics"
	"github.com/Varunjp/vyavsa/internal/pdf"
	"github.com/Varunjp/vyavsa/internal/repository"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
)

// DailyReportService provides business operations for retrieving and exporting daily reports
type DailyReportService interface {
	GetDailyReport(ctx context.Context, tenantID uuid.UUID, dateStr string) (*domain.DailyReport, error)
	GeneratePDF(ctx context.Context, tenantID uuid.UUID, dateStr string) ([]byte, string, error)
	SetLocation(loc *time.Location)
	SetMetrics(m *metrics.Metrics)
}

type dailyReportService struct {
	reportRepo   repository.DailyReportRepository
	pdfGenerator pdf.DailyReportPDFGenerator
	location     *time.Location
	metrics      *metrics.Metrics
	logger       *slog.Logger
}

// NewDailyReportService creates a new instance of DailyReportService
func NewDailyReportService(
	reportRepo repository.DailyReportRepository,
	pdfGenerator pdf.DailyReportPDFGenerator,
	logger *slog.Logger,
) DailyReportService {
	if logger == nil {
		logger = slog.Default()
	}
	if pdfGenerator == nil {
		pdfGenerator = pdf.NewPDFGenerator()
	}
	return &dailyReportService{
		reportRepo:   reportRepo,
		pdfGenerator: pdfGenerator,
		location:     time.UTC,
		logger:       logger,
	}
}

func (s *dailyReportService) SetLocation(loc *time.Location) {
	if loc != nil {
		s.location = loc
	}
}

func (s *dailyReportService) SetMetrics(m *metrics.Metrics) {
	s.metrics = m
}

// parseReportDate parses and normalizes the date string in the configured business timezone
func (s *dailyReportService) parseReportDate(dateStr string) (time.Time, time.Time, string, error) {
	loc := s.location
	if loc == nil {
		loc = time.UTC
	}

	trimmed := strings.TrimSpace(dateStr)
	if trimmed == "" {
		now := time.Now().In(loc)
		trimmed = now.Format("2006-01-02")
	}

	t, err := time.ParseInLocation("2006-01-02", trimmed, loc)
	if err != nil {
		return time.Time{}, time.Time{}, "", appErrors.NewBadRequest("invalid report date format: expected YYYY-MM-DD")
	}

	start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)

	return start, end, trimmed, nil
}

// SanitizeFilename creates a safe filename component by stripping non-alphanumeric characters
var nonAlphaNumRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func SanitizeFilename(name string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return "Business"
	}
	clean = nonAlphaNumRegex.ReplaceAllString(clean, "_")
	clean = strings.Trim(clean, "_")
	if clean == "" {
		return "Business"
	}
	if len(clean) > 50 {
		clean = clean[:50]
	}
	return clean
}

// GetDailyReport compiles the daily financial figures and transaction breakdown
func (s *dailyReportService) GetDailyReport(ctx context.Context, tenantID uuid.UUID, dateStr string) (*domain.DailyReport, error) {
	startTimer := time.Now()
	s.logger.InfoContext(ctx, "daily report generation started",
		slog.String("tenant_id", tenantID.String()),
		slog.String("date", dateStr),
		slog.String("format", "json"),
	)

	start, end, normalizedDate, err := s.parseReportDate(dateStr)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordDailyReportFailed("json")
		}
		s.logger.WarnContext(ctx, "daily report date validation failed",
			slog.String("tenant_id", tenantID.String()),
			slog.String("input_date", dateStr),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	report, err := s.reportRepo.GetDailyReportData(ctx, tenantID, start, end, s.location)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordDailyReportFailed("json")
		}
		s.logger.ErrorContext(ctx, "daily report generation failed",
			slog.String("tenant_id", tenantID.String()),
			slog.String("date", normalizedDate),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	txnCount := len(report.LineSales) + len(report.CounterSales) + len(report.Purchases) + len(report.Expenses)
	duration := time.Since(startTimer)

	if s.metrics != nil {
		s.metrics.RecordDailyReportGenerated("json", duration, txnCount)
	}

	s.logger.InfoContext(ctx, "daily report generation completed",
		slog.String("tenant_id", tenantID.String()),
		slog.String("date", normalizedDate),
		slog.Duration("duration", duration),
		slog.Int("transaction_count", txnCount),
	)

	return report, nil
}

// GeneratePDF produces the downloadable PDF binary along with a sanitized filename
func (s *dailyReportService) GeneratePDF(ctx context.Context, tenantID uuid.UUID, dateStr string) ([]byte, string, error) {
	startTimer := time.Now()
	s.logger.InfoContext(ctx, "daily report generation started",
		slog.String("tenant_id", tenantID.String()),
		slog.String("date", dateStr),
		slog.String("format", "pdf"),
	)

	start, end, normalizedDate, err := s.parseReportDate(dateStr)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordDailyReportFailed("pdf")
		}
		return nil, "", err
	}

	report, err := s.reportRepo.GetDailyReportData(ctx, tenantID, start, end, s.location)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordDailyReportFailed("pdf")
		}
		s.logger.ErrorContext(ctx, "daily report data fetch failed for pdf",
			slog.String("tenant_id", tenantID.String()),
			slog.String("date", normalizedDate),
			slog.String("error", err.Error()),
		)
		return nil, "", err
	}

	pdfBytes, err := s.pdfGenerator.GenerateDailyReportPDF(report)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordDailyReportFailed("pdf")
		}
		s.logger.ErrorContext(ctx, "daily report PDF generation failed",
			slog.String("tenant_id", tenantID.String()),
			slog.String("date", normalizedDate),
			slog.String("error", err.Error()),
		)
		return nil, "", appErrors.NewInternal(fmt.Errorf("failed to generate PDF document: %w", err))
	}

	safeName := SanitizeFilename(report.Tenant.Name)
	filename := fmt.Sprintf("%s_Daily_Report_%s.pdf", safeName, normalizedDate)

	txnCount := len(report.LineSales) + len(report.CounterSales) + len(report.Purchases) + len(report.Expenses)
	duration := time.Since(startTimer)

	if s.metrics != nil {
		s.metrics.RecordDailyReportGenerated("pdf", duration, txnCount)
	}

	s.logger.InfoContext(ctx, "daily report generation completed",
		slog.String("tenant_id", tenantID.String()),
		slog.String("date", normalizedDate),
		slog.String("filename", filename),
		slog.Duration("duration", duration),
		slog.Int("transaction_count", txnCount),
	)

	return pdfBytes, filename, nil
}
