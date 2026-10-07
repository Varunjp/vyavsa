package pdf

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPDFGenerator_EmptyReport(t *testing.T) {
	generator := NewPDFGenerator()

	report := &domain.DailyReport{
		Tenant: domain.TenantReportInfo{
			ID:    uuid.New(),
			Name:  "Test Store Pvt Ltd",
			Email: "test@example.com",
			Phone: "+91 9876543210",
		},
		Date:          "2026-10-07",
		FormattedDate: "07 October 2026",
		LineSales:     []domain.LineSaleReportItem{},
		CounterSales:  []domain.CounterSaleReportItem{},
		Purchases:     []domain.PurchaseReportItem{},
		Expenses:      []domain.ExpenseReportItem{},
		CashBalance: domain.CashBalanceReport{
			ClosingCash: decimal.NewFromInt(5000),
		},
		BankBalances: []domain.BankBalanceReportItem{},
		FinancialSummary: domain.ReportFinancialSummary{
			CashBalance:         decimal.NewFromInt(5000),
			TotalAvailableFunds: decimal.NewFromInt(5000),
		},
		GeneratedAt: time.Now(),
	}

	data, err := generator.GenerateDailyReportPDF(report)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Valid PDF header
	assert.True(t, bytes.HasPrefix(data, []byte("%PDF-")), "Generated file should have valid PDF signature")
}

func TestPDFGenerator_FullReportWithMultiPageData(t *testing.T) {
	generator := NewPDFGenerator()

	tenantID := uuid.New()
	report := &domain.DailyReport{
		Tenant: domain.TenantReportInfo{
			ID:    tenantID,
			Name:  "Super Wholesale Distributors",
			Email: "accounts@superwholesale.in",
			Phone: "+91 9811122233",
		},
		Date:          "2026-10-07",
		FormattedDate: "07 October 2026",
		LineSales:     make([]domain.LineSaleReportItem, 0),
		CounterSales:  make([]domain.CounterSaleReportItem, 0),
		Purchases:     make([]domain.PurchaseReportItem, 0),
		Expenses:      make([]domain.ExpenseReportItem, 0),
		BankBalances: []domain.BankBalanceReportItem{
			{
				BankID:         uuid.New(),
				BankName:       "State Bank of India",
				AccountNumber:  "****1234",
				CurrentBalance: decimal.NewFromInt(85000),
			},
			{
				BankID:         uuid.New(),
				BankName:       "HDFC Bank",
				AccountNumber:  "****5678",
				CurrentBalance: decimal.NewFromInt(42500),
			},
		},
		GeneratedAt: time.Now(),
	}

	// Add 30 line sales to verify multi-page pagination and table wrapping
	for i := 1; i <= 30; i++ {
		report.LineSales = append(report.LineSales, domain.LineSaleReportItem{
			ID:            uuid.New(),
			InvoiceNumber: fmt.Sprintf("LS-%04d", i),
			Time:          "10:15",
			CustomerName:  fmt.Sprintf("Customer %d Mart", i),
			Route:         "Route North",
			Salesman:      "Ramesh",
			TotalAmount:   decimal.NewFromInt(int64(i * 500)),
			CashIn:        decimal.NewFromInt(int64(i * 300)),
			BankAmount:    decimal.NewFromInt(int64(i * 200)),
			PaymentMethod: "Split (Cash+Bank)",
			CreatedAt:     time.Now(),
		})
	}

	// Add 15 counter sales
	for i := 1; i <= 15; i++ {
		report.CounterSales = append(report.CounterSales, domain.CounterSaleReportItem{
			ID:            uuid.New(),
			ReceiptNumber: fmt.Sprintf("CS-%04d", i),
			Time:          "11:30",
			Item:          fmt.Sprintf("Product %d", i),
			TotalAmount:   decimal.NewFromInt(int64(i * 150)),
			Cash:          decimal.NewFromInt(int64(i * 150)),
			PaymentMethod: "Cash",
			CreatedAt:     time.Now(),
		})
	}

	// Add purchases
	report.Purchases = append(report.Purchases, domain.PurchaseReportItem{
		ID:            uuid.New(),
		Reference:     "PUR-1001",
		Time:          "08:30",
		SupplierName:  "National Sugar Mills",
		Item:          "Sugar 50kg Bags",
		Quantity:      20,
		TotalAmount:   decimal.NewFromInt(45000),
		TotalPaid:     decimal.NewFromInt(45000),
		PaymentStatus: "PAID",
		CreatedAt:     time.Now(),
	})

	// Add expenses
	report.Expenses = append(report.Expenses, domain.ExpenseReportItem{
		ID:            uuid.New(),
		Reference:     "EXP-1001",
		Time:          "09:00",
		Category:      "Utilities",
		Description:   "Electricity Bill",
		Amount:        decimal.NewFromInt(3500),
		PaymentMethod: "Bank (HDFC)",
		CreatedAt:     time.Now(),
	})

	report.CashBalance = domain.CashBalanceReport{
		LineSalesCash:    decimal.NewFromInt(14000),
		CounterSalesCash: decimal.NewFromInt(18000),
		TotalCashInflow:  decimal.NewFromInt(32000),
		PurchaseCash:     decimal.NewFromInt(10000),
		ExpenseCash:      decimal.NewFromInt(2000),
		TotalCashOutflow: decimal.NewFromInt(12000),
		NetDailyCashFlow: decimal.NewFromInt(20000),
		ClosingCash:      decimal.NewFromInt(45000),
	}

	report.FinancialSummary = domain.ReportFinancialSummary{
		TotalLineSales:      decimal.NewFromInt(232500),
		TotalCounterSales:   decimal.NewFromInt(18000),
		TotalPurchases:      decimal.NewFromInt(45000),
		TotalExpenses:       decimal.NewFromInt(3500),
		CashBalance:         decimal.NewFromInt(45000),
		TotalBankBalance:    decimal.NewFromInt(127500),
		TotalAvailableFunds: decimal.NewFromInt(172500),
	}

	data, err := generator.GenerateDailyReportPDF(report)
	require.NoError(t, err)
	require.NotEmpty(t, data)
	assert.True(t, bytes.HasPrefix(data, []byte("%PDF-")))
}

func TestFormatCurrency(t *testing.T) {
	tests := []struct {
		input    decimal.Decimal
		expected string
	}{
		{decimal.NewFromInt(0), "Rs. 0.00"},
		{decimal.NewFromInt(500), "Rs. 500.00"},
		{decimal.NewFromInt(1250), "Rs. 1,250.00"},
		{decimal.NewFromInt(45000), "Rs. 45,000.00"},
		{decimal.NewFromInt(145750), "Rs. 1,45,750.00"},
		{decimal.NewFromFloat(1000000.50), "Rs. 10,00,000.50"},
		{decimal.NewFromInt(-1500), "-Rs. 1,500.00"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			res := formatCurrency(tt.input)
			assert.Equal(t, tt.expected, res)
		})
	}
}
