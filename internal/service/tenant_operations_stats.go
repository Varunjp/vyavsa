package service

import (
	"context"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	appErrors "github.com/Varunjp/vyavsa/pkg/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func hasStatsActivity(s *domain.TenantDailyStats) bool {
	if s == nil {
		return false
	}
	return s.TotalSales.GreaterThan(decimal.Zero) ||
		s.PurchaseAmount.GreaterThan(decimal.Zero) ||
		s.ExpenseAmount.GreaterThan(decimal.Zero) ||
		s.AttendancePresent > 0 ||
		s.AttendanceAbsent > 0 ||
		s.AdvanceAmount.GreaterThan(decimal.Zero) ||
		s.WagesAmount.GreaterThan(decimal.Zero) ||
		s.AmountReceived.GreaterThan(decimal.Zero) ||
		s.AmountPaid.GreaterThan(decimal.Zero)
}

func calculateDaysOld(requestedDate, dataDate string) int {
	if requestedDate == dataDate || requestedDate == "" || dataDate == "" {
		return 0
	}
	reqTime, err1 := time.Parse("2006-01-02", requestedDate)
	dateTime, err2 := time.Parse("2006-01-02", dataDate)
	if err1 != nil || err2 != nil {
		return 0
	}
	days := int(reqTime.Sub(dateTime).Hours() / 24)
	if days < 1 {
		days = 1
	}
	return days
}

// ==========================================
// 11. Financial Summary & Daily Statistics
// ==========================================

func (s *TenantOperationsService) GetDailyStats(ctx context.Context, tenantID uuid.UUID, date string) (*domain.TenantDailyStats, error) {
	if s.statsRepo == nil {
		return &domain.TenantDailyStats{TenantID: tenantID}, nil
	}
	today := todayString()
	isExplicitHistorical := date != "" && date != today

	if isExplicitHistorical {
		stats, err := s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, date)
		if err != nil {
			return nil, err
		}
		if stats != nil {
			stats.RequestedDate = date
			stats.DataDate = date
			isCur := false
			stats.IsCurrent = &isCur
			stats.DaysOld = calculateDaysOld(today, date)
		}
		return stats, nil
	}

	// For current business day:
	todayStats, err := s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, today)
	if err != nil {
		return nil, err
	}

	if hasStatsActivity(todayStats) {
		todayStats.RequestedDate = today
		todayStats.DataDate = today
		isCur := true
		todayStats.IsCurrent = &isCur
		todayStats.DaysOld = 0
		return todayStats, nil
	}

	// If no activity today, look for latest available previous-day data
	prevStats, err := s.statsRepo.GetLatestAvailable(ctx, tenantID, today)
	if err == nil && prevStats != nil && prevStats.Date != "" {
		prevStats.RequestedDate = today
		prevStats.DataDate = prevStats.Date
		isCur := false
		prevStats.IsCurrent = &isCur
		prevStats.DaysOld = calculateDaysOld(today, prevStats.Date)
		return prevStats, nil
	}

	// Fallback to today's (zeroed) stats if no prior data exists
	todayStats.RequestedDate = today
	todayStats.DataDate = today
	isCur := true
	todayStats.IsCurrent = &isCur
	todayStats.DaysOld = 0
	return todayStats, nil
}

func (s *TenantOperationsService) GetFinancialMetrics(ctx context.Context, tenantID uuid.UUID) (*domain.FinancialMetrics, error) {
	if s.summaryRepo == nil {
		todayOverview, _ := s.GetTodayOverview(ctx, tenantID)
		return &domain.FinancialMetrics{
			TodayOverview: todayOverview,
		}, nil
	}
	var summary *domain.TenantFinancialSummary
	var err error
	if s.summaryRepo != nil {
		summary, err = s.summaryRepo.SyncFromSourceRecords(ctx, tenantID)
		if err != nil {
			summary, err = s.summaryRepo.GetByTenantID(ctx, tenantID)
		}
	}
	if err != nil || summary == nil {
		if err == nil || appErrors.IsNotFound(err) {
			summary = &domain.TenantFinancialSummary{
				TenantID:        tenantID,
				CashBalance:     decimal.Zero,
				BankBalance:     decimal.Zero,
				TotalReceivable: decimal.Zero,
				TotalPayable:    decimal.Zero,
			}
		} else {
			return nil, err
		}
	}

	// Calculate pending salaries directly from employee_salary table
	var totalPendingSalary decimal.Decimal
	if s.salaryRepo != nil {
		pendingSalaries, _, _ := s.salaryRepo.ListPending(ctx, tenantID, 1, 500)
		for _, ps := range pendingSalaries {
			if ps.Balance.GreaterThan(decimal.Zero) {
				totalPendingSalary = totalPendingSalary.Add(ps.Balance)
			}
		}
	}

	today := todayString()
	var todayStats *domain.TenantDailyStats
	if s.statsRepo != nil {
		todayStats, _ = s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, today)
	}

	requestedDate := today
	dataDate := today
	isCurrent := true
	daysOld := 0
	activeStats := todayStats

	if hasStatsActivity(todayStats) {
		if todayStats != nil {
			todayStats.RequestedDate = today
			todayStats.DataDate = today
			isCur := true
			todayStats.IsCurrent = &isCur
			todayStats.DaysOld = 0
		}
	} else if s.statsRepo != nil {
		prevStats, err := s.statsRepo.GetLatestAvailable(ctx, tenantID, today)
		if err == nil && prevStats != nil && prevStats.Date != "" {
			activeStats = prevStats
			dataDate = prevStats.Date
			isCurrent = false
			daysOld = calculateDaysOld(today, prevStats.Date)
			isCur := false
			prevStats.RequestedDate = today
			prevStats.DataDate = prevStats.Date
			prevStats.IsCurrent = &isCur
			prevStats.DaysOld = daysOld
		} else if todayStats != nil {
			todayStats.RequestedDate = today
			todayStats.DataDate = today
			isCur := true
			todayStats.IsCurrent = &isCur
			todayStats.DaysOld = 0
		}
	}

	// Net dues is total payable on purchases + pending employee salaries
	netDues := summary.TotalPayable.Add(totalPendingSalary)

	// Fetch individual bank balances for dashboard/reporting
	var activeBanks []domain.TenantBank
	if s.bankRepo != nil {
		activeBanks, _, _ = s.bankRepo.List(ctx, tenantID, 1, 100, "", "active")
	}

	// Calculate today's earned employee salary from attendance
	var todaySalaryEarned decimal.Decimal
	if s.attRepo != nil {
		todaySalaryEarned, _ = s.attRepo.GetTodaySalaryEarned(ctx, tenantID, today)
	}

	todayOverview, _ := s.GetTodayOverview(ctx, tenantID)
	if todayOverview != nil && todayOverview.OutstandingPayable.IsZero() && !summary.TotalPayable.IsZero() {
		todayOverview.OutstandingPayable = summary.TotalPayable
	}

	return &domain.FinancialMetrics{
		CashBalance:         summary.CashBalance,
		BankBalance:         summary.BankBalance,
		TotalReceivable:     summary.TotalReceivable,
		TotalPayable:        summary.TotalPayable,
		NetDues:             netDues,
		NetReceivables:      summary.TotalReceivable,
		PendingSalary:       totalPendingSalary,
		TodayEmployeeSalary: todaySalaryEarned,
		BankBalances:        activeBanks,
		TodayStats:          activeStats,
		TodayOverview:       todayOverview,
		RequestedDate:       requestedDate,
		DataDate:            dataDate,
		IsCurrent:           isCurrent,
		DaysOld:             daysOld,
	}, nil
}

// GetTodayOverview aggregates the current day's key business and staff metrics
func (s *TenantOperationsService) GetTodayOverview(ctx context.Context, tenantID uuid.UUID) (*domain.TodayOverview, error) {
	today := todayString()

	var todayStats *domain.TenantDailyStats
	if s.statsRepo != nil {
		todayStats, _ = s.statsRepo.ComputeAndSyncDailyStats(ctx, tenantID, today)
	}
	if todayStats == nil {
		todayStats = &domain.TenantDailyStats{
			TenantID:          tenantID,
			Date:              today,
			LineSaleAmount:    decimal.Zero,
			CounterSaleAmount: decimal.Zero,
			TotalSales:        decimal.Zero,
			PurchaseAmount:    decimal.Zero,
			ExpenseAmount:     decimal.Zero,
			AdvanceAmount:     decimal.Zero,
		}
	}

	// 1. Staff Attendance & Total Employee Count
	totalStaff := 0
	if s.empRepo != nil {
		_, activeCount, err := s.empRepo.List(ctx, tenantID, 1, 1, "", "active")
		if err == nil && activeCount > 0 {
			totalStaff = int(activeCount)
		} else {
			_, allCount, err := s.empRepo.List(ctx, tenantID, 1, 1, "", "")
			if err == nil {
				totalStaff = int(allCount)
			}
		}
	}

	recordedTotal := todayStats.AttendancePresent + todayStats.AttendanceAbsent
	if recordedTotal > totalStaff {
		totalStaff = recordedTotal
	}
	hasRecords := recordedTotal > 0

	attOverview := domain.TodayAttendanceOverview{
		Present:    todayStats.AttendancePresent,
		Total:      totalStaff,
		Absent:     todayStats.AttendanceAbsent,
		HasRecords: hasRecords,
	}

	// 2. Line Sale (today's transactions only)
	lineSale := todayStats.LineSaleAmount

	// 3. Counter Sale (today's transactions only)
	counterSale := todayStats.CounterSaleAmount

	// 4. Employee Total Advance (today's records only)
	empAdvance := todayStats.AdvanceAmount

	// 5. Current Item (from existing inventory/purchase data or fallback to counter sales)
	var currentItem *domain.CurrentItemOverview
	if s.purchRepo != nil {
		// First check if an inventory purchase occurred today
		todayPurchases, _, err := s.purchRepo.List(ctx, tenantID, 1, 1, today, nil, "")
		if err == nil && len(todayPurchases) > 0 {
			p := todayPurchases[0]
			currentItem = &domain.CurrentItemOverview{
				Name:         p.Item,
				Quantity:     p.Quantity,
				TotalAmount:  p.TotalAmount,
				TotalPaid:    p.TotalPaid,
				TotalPending: p.TotalPending,
				Date:         p.CreatedAt.Format("2006-01-02"),
				IsToday:      true,
				Source:       "purchase",
			}
		} else {
			// Check latest available inventory procurement
			allPurchases, _, err := s.purchRepo.List(ctx, tenantID, 1, 1, "", nil, "")
			if err == nil && len(allPurchases) > 0 {
				p := allPurchases[0]
				isToday := p.CreatedAt.Format("2006-01-02") == today
				currentItem = &domain.CurrentItemOverview{
					Name:         p.Item,
					Quantity:     p.Quantity,
					TotalAmount:  p.TotalAmount,
					TotalPaid:    p.TotalPaid,
					TotalPending: p.TotalPending,
					Date:         p.CreatedAt.Format("2006-01-02"),
					IsToday:      isToday,
					Source:       "purchase",
				}
			}
		}
	}

	// Fallback to counter sale item if no purchase item exists
	if currentItem == nil && s.countSaleRepo != nil {
		todaySales, _, err := s.countSaleRepo.List(ctx, tenantID, 1, 1, today, "")
		if err == nil && len(todaySales) > 0 {
			cs := todaySales[0]
			currentItem = &domain.CurrentItemOverview{
				Name:         cs.Item,
				Quantity:     1,
				TotalAmount:  cs.TotalAmount,
				TotalPaid:    cs.CollectedAmount,
				TotalPending: cs.Account,
				Date:         cs.CreatedAt.Format("2006-01-02"),
				IsToday:      true,
				Source:       "counter_sale",
			}
		} else {
			allSales, _, err := s.countSaleRepo.List(ctx, tenantID, 1, 1, "", "")
			if err == nil && len(allSales) > 0 {
				cs := allSales[0]
				isToday := cs.CreatedAt.Format("2006-01-02") == today
				currentItem = &domain.CurrentItemOverview{
					Name:         cs.Item,
					Quantity:     1,
					TotalAmount:  cs.TotalAmount,
					TotalPaid:    cs.CollectedAmount,
					TotalPending: cs.Account,
					Date:         cs.CreatedAt.Format("2006-01-02"),
					IsToday:      isToday,
					Source:       "counter_sale",
				}
			}
		}
	}

	var currentItemVal any = struct{}{}
	if currentItem != nil {
		currentItemVal = currentItem
	}

	var outstandingPayable decimal.Decimal
	if s.summaryRepo != nil {
		finSummary, err := s.summaryRepo.GetByTenantID(ctx, tenantID)
		if err != nil && appErrors.IsNotFound(err) {
			finSummary, err = s.summaryRepo.SyncFromSourceRecords(ctx, tenantID)
		}
		if err == nil && finSummary != nil {
			outstandingPayable = finSummary.TotalPayable
		}
	}

	return &domain.TodayOverview{
		Attendance:         attOverview,
		LineSale:           lineSale,
		CounterSale:        counterSale,
		EmployeeAdvance:    empAdvance,
		CurrentItem:        currentItemVal,
		OutstandingPayable: outstandingPayable,
		Date:               today,
	}, nil
}
