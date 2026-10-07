package pdf

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Varunjp/vyavsa/internal/domain"
	"github.com/go-pdf/fpdf"
	"github.com/shopspring/decimal"
)

// DailyReportPDFGenerator defines the interface for rendering daily business report PDFs
type DailyReportPDFGenerator interface {
	GenerateDailyReportPDF(report *domain.DailyReport) ([]byte, error)
}

// PDFGenerator is the concrete implementation of DailyReportPDFGenerator using fpdf
type PDFGenerator struct{}

// NewPDFGenerator creates a new instance of PDFGenerator
func NewPDFGenerator() *PDFGenerator {
	return &PDFGenerator{}
}

// formatCurrency formats a decimal amount with Indian numbering format (e.g. "Rs. 1,45,750.00")
func formatCurrency(d decimal.Decimal) string {
	isNeg := d.IsNegative()
	abs := d.Abs()
	parts := strings.Split(abs.StringFixed(2), ".")
	intPart := parts[0]
	decPart := parts[1]

	var formattedInt string
	n := len(intPart)
	if n <= 3 {
		formattedInt = intPart
	} else {
		last3 := intPart[n-3:]
		remaining := intPart[:n-3]
		var chunks []string
		for len(remaining) > 2 {
			chunks = append([]string{remaining[len(remaining)-2:]}, chunks...)
			remaining = remaining[:len(remaining)-2]
		}
		if len(remaining) > 0 {
			chunks = append([]string{remaining}, chunks...)
		}
		formattedInt = strings.Join(chunks, ",") + "," + last3
	}

	res := "Rs. " + formattedInt + "." + decPart
	if isNeg {
		return "-" + res
	}
	return res
}

func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// GenerateDailyReportPDF renders the complete, multi-page, production-ready PDF
func (g *PDFGenerator) GenerateDailyReportPDF(report *domain.DailyReport) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 15, 14)
	pdf.SetAutoPageBreak(true, 22)
	pdf.AliasNbPages("{nb}")

	// Subtle background watermark on every page
	pdf.SetHeaderFunc(func() {
		curX, curY := pdf.GetX(), pdf.GetY()
		curR, curG, curB := 15, 23, 42 // store default text color

		// Watermark: Faded light Vyavsa branding across the page
		pdf.SetTextColor(241, 245, 249) // very light slate
		pdf.SetFont("Arial", "B", 60)
		pdf.SetXY(14, 130)
		pdf.CellFormat(182, 30, "VYAVSA", "", 0, "C", false, 0, "")

		// Restore position and color
		pdf.SetTextColor(curR, curG, curB)
		pdf.SetXY(curX, curY)
	})

	// Professional footer on every page
	pdf.SetFooterFunc(func() {
		pdf.SetY(-16)
		pdf.SetDrawColor(226, 232, 240)
		pdf.SetLineWidth(0.3)
		pdf.Line(14, pdf.GetY(), 196, pdf.GetY())

		pdf.SetY(-13)
		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(100, 116, 139)

		leftText := fmt.Sprintf("Vyavsa  |  Daily Business Report  |  %s", report.FormattedDate)
		pdf.CellFormat(120, 6, leftText, "", 0, "L", false, 0, "")

		rightText := fmt.Sprintf("Page %d of {nb}", pdf.PageNo())
		pdf.CellFormat(62, 6, rightText, "", 0, "R", false, 0, "")
	})

	pdf.AddPage()

	// -------------------------------------------------------------
	// 1. REPORT HEADER
	// -------------------------------------------------------------
	businessName := report.Tenant.Name
	if businessName == "" {
		businessName = "Business Organization"
	}

	// Business Name
	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(30, 41, 59) // slate-800
	pdf.CellFormat(182, 8, businessName, "", 1, "L", false, 0, "")

	// Subtitle & Date
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(79, 70, 229) // Indigo primary
	pdf.CellFormat(120, 6, "DAILY FINANCIAL & OPERATIONS REPORT", "", 0, "L", false, 0, "")

	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(62, 6, fmt.Sprintf("Date: %s", report.FormattedDate), "", 1, "R", false, 0, "")

	// Metadata line
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(100, 116, 139)
	contactInfo := ""
	if report.Tenant.Email != "" {
		contactInfo = report.Tenant.Email
	}
	if report.Tenant.Phone != "" {
		if contactInfo != "" {
			contactInfo += "  |  "
		}
		contactInfo += report.Tenant.Phone
	}
	genInfo := fmt.Sprintf("Generated: %s", report.GeneratedAt.Format("02 Jan 2006, 15:04 MST"))
	pdf.CellFormat(120, 4, contactInfo, "", 0, "L", false, 0, "")
	pdf.CellFormat(62, 4, genInfo, "", 1, "R", false, 0, "")

	pdf.Ln(3)
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetLineWidth(0.4)
	pdf.Line(14, pdf.GetY(), 196, pdf.GetY())
	pdf.Ln(4)

	// -------------------------------------------------------------
	// 2. FINANCIAL SUMMARY OVERVIEW
	// -------------------------------------------------------------
	drawSectionHeading(pdf, "FINANCIAL SUMMARY")

	// Summary Cards Box (2 rows of 3 columns, clean grid)
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetLineWidth(0.2)

	colW := 60.66
	boxH := 15.0

	// Row 1: Line Sales | Counter Sales | Total Purchases
	yStart := pdf.GetY()
	drawMetricBox(pdf, 14, yStart, colW, boxH, "Line Sales", formatCurrency(report.FinancialSummary.TotalLineSales), 79, 70, 229)
	drawMetricBox(pdf, 14+colW, yStart, colW, boxH, "Counter Sales", formatCurrency(report.FinancialSummary.TotalCounterSales), 79, 70, 229)
	drawMetricBox(pdf, 14+colW*2, yStart, colW, boxH, "Total Purchases", formatCurrency(report.FinancialSummary.TotalPurchases), 225, 29, 72)

	// Row 2: Total Expenses | Closing Cash | Total Bank Balance
	yRow2 := yStart + boxH + 2
	drawMetricBox(pdf, 14, yRow2, colW, boxH, "Total Expenses", formatCurrency(report.FinancialSummary.TotalExpenses), 234, 88, 12)
	drawMetricBox(pdf, 14+colW, yRow2, colW, boxH, "Closing Cash Balance", formatCurrency(report.FinancialSummary.CashBalance), 16, 185, 129)
	drawMetricBox(pdf, 14+colW*2, yRow2, colW, boxH, "Total Bank Balance", formatCurrency(report.FinancialSummary.TotalBankBalance), 14, 165, 233)

	// Row 3: Full Width Available Funds Banner
	yRow3 := yRow2 + boxH + 2
	pdf.SetXY(14, yRow3)
	pdf.SetFillColor(248, 250, 252)
	pdf.Rect(14, yRow3, 182, 10, "DF")
	pdf.SetFont("Arial", "B", 9)
	pdf.SetTextColor(71, 85, 105)
	pdf.SetXY(18, yRow3+2.5)
	pdf.CellFormat(90, 5, "TOTAL AVAILABLE FUNDS (Cash + Bank):", "", 0, "L", false, 0, "")
	pdf.SetFont("Arial", "B", 11)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(86, 5, formatCurrency(report.FinancialSummary.TotalAvailableFunds), "", 1, "R", false, 0, "")

	pdf.SetY(yRow3 + 14)

	// -------------------------------------------------------------
	// 3. LINE SALES SECTION
	// -------------------------------------------------------------
	ensurePageSpace(pdf, 35)
	drawSectionHeading(pdf, fmt.Sprintf("1. LINE SALES (%d transactions)", len(report.LineSales)))

	if len(report.LineSales) == 0 {
		drawEmptyMessage(pdf, "No line sales recorded for this date.")
	} else {
		headers := []string{"#", "Time", "Invoice", "Customer", "Route / Notes", "Amount", "Cash In", "Bank"}
		widths := []float64{8, 14, 24, 40, 36, 22, 18, 20}
		aligns := []string{"C", "C", "L", "L", "L", "R", "R", "R"}

		drawLineSalesHeader(pdf, headers, widths, aligns)

		for i, sale := range report.LineSales {
			ensureTableRowSpace(pdf, 7, func() {
				drawLineSalesHeader(pdf, headers, widths, aligns)
			})

			fill := (i % 2) == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetDrawColor(226, 232, 240)

			notes := sale.Route
			if sale.Salesman != "" {
				if notes != "" {
					notes += " / "
				}
				notes += sale.Salesman
			}
			if sale.Note != "" {
				if notes != "" {
					notes += " - "
				}
				notes += sale.Note
			}

			pdf.CellFormat(widths[0], 6.5, fmt.Sprintf("%d", i+1), "B", 0, aligns[0], fill, 0, "")
			pdf.CellFormat(widths[1], 6.5, sale.Time, "B", 0, aligns[1], fill, 0, "")
			pdf.CellFormat(widths[2], 6.5, sale.InvoiceNumber, "B", 0, aligns[2], fill, 0, "")
			pdf.CellFormat(widths[3], 6.5, truncateString(sale.CustomerName, 22), "B", 0, aligns[3], fill, 0, "")
			pdf.CellFormat(widths[4], 6.5, truncateString(notes, 20), "B", 0, aligns[4], fill, 0, "")
			pdf.CellFormat(widths[5], 6.5, formatCurrency(sale.TotalAmount), "B", 0, aligns[5], fill, 0, "")
			pdf.CellFormat(widths[6], 6.5, formatCurrency(sale.CashIn), "B", 0, aligns[6], fill, 0, "")
			pdf.CellFormat(widths[7], 6.5, formatCurrency(sale.BankAmount), "B", 1, aligns[7], fill, 0, "")
		}

		// Line Sales Subtotal Row
		drawSubtotalRow(pdf, "Total Line Sales:", formatCurrency(report.FinancialSummary.TotalLineSales))
	}
	pdf.Ln(4)

	// -------------------------------------------------------------
	// 4. COUNTER SALES SECTION
	// -------------------------------------------------------------
	ensurePageSpace(pdf, 35)
	drawSectionHeading(pdf, fmt.Sprintf("2. COUNTER SALES (%d transactions)", len(report.CounterSales)))

	if len(report.CounterSales) == 0 {
		drawEmptyMessage(pdf, "No counter sales recorded for this date.")
	} else {
		headers := []string{"#", "Time", "Receipt", "Item", "Amount", "Cash", "Bank", "Due"}
		widths := []float64{8, 14, 24, 44, 24, 22, 22, 24}
		aligns := []string{"C", "C", "L", "L", "R", "R", "R", "R"}

		drawGenericHeader(pdf, headers, widths, aligns)

		for i, sale := range report.CounterSales {
			ensureTableRowSpace(pdf, 7, func() {
				drawGenericHeader(pdf, headers, widths, aligns)
			})

			fill := (i % 2) == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetDrawColor(226, 232, 240)

			pdf.CellFormat(widths[0], 6.5, fmt.Sprintf("%d", i+1), "B", 0, aligns[0], fill, 0, "")
			pdf.CellFormat(widths[1], 6.5, sale.Time, "B", 0, aligns[1], fill, 0, "")
			pdf.CellFormat(widths[2], 6.5, sale.ReceiptNumber, "B", 0, aligns[2], fill, 0, "")
			pdf.CellFormat(widths[3], 6.5, truncateString(sale.Item, 25), "B", 0, aligns[3], fill, 0, "")
			pdf.CellFormat(widths[4], 6.5, formatCurrency(sale.TotalAmount), "B", 0, aligns[4], fill, 0, "")
			pdf.CellFormat(widths[5], 6.5, formatCurrency(sale.Cash), "B", 0, aligns[5], fill, 0, "")
			pdf.CellFormat(widths[6], 6.5, formatCurrency(sale.BankAmount), "B", 0, aligns[6], fill, 0, "")
			pdf.CellFormat(widths[7], 6.5, formatCurrency(sale.Account), "B", 1, aligns[7], fill, 0, "")
		}

		drawSubtotalRow(pdf, "Total Counter Sales:", formatCurrency(report.FinancialSummary.TotalCounterSales))
	}
	pdf.Ln(4)

	// -------------------------------------------------------------
	// 5. PURCHASES SECTION
	// -------------------------------------------------------------
	ensurePageSpace(pdf, 35)
	drawSectionHeading(pdf, fmt.Sprintf("3. PURCHASES (%d transactions)", len(report.Purchases)))

	if len(report.Purchases) == 0 {
		drawEmptyMessage(pdf, "No purchases recorded for this date.")
	} else {
		headers := []string{"#", "Time", "Reference", "Supplier", "Item", "Qty", "Total Amount", "Paid"}
		widths := []float64{8, 14, 24, 40, 34, 12, 24, 26}
		aligns := []string{"C", "C", "L", "L", "L", "C", "R", "R"}

		drawGenericHeader(pdf, headers, widths, aligns)

		for i, purch := range report.Purchases {
			ensureTableRowSpace(pdf, 7, func() {
				drawGenericHeader(pdf, headers, widths, aligns)
			})

			fill := (i % 2) == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetDrawColor(226, 232, 240)

			pdf.CellFormat(widths[0], 6.5, fmt.Sprintf("%d", i+1), "B", 0, aligns[0], fill, 0, "")
			pdf.CellFormat(widths[1], 6.5, purch.Time, "B", 0, aligns[1], fill, 0, "")
			pdf.CellFormat(widths[2], 6.5, purch.Reference, "B", 0, aligns[2], fill, 0, "")
			pdf.CellFormat(widths[3], 6.5, truncateString(purch.SupplierName, 22), "B", 0, aligns[3], fill, 0, "")
			pdf.CellFormat(widths[4], 6.5, truncateString(purch.Item, 20), "B", 0, aligns[4], fill, 0, "")
			pdf.CellFormat(widths[5], 6.5, fmt.Sprintf("%d", purch.Quantity), "B", 0, aligns[5], fill, 0, "")
			pdf.CellFormat(widths[6], 6.5, formatCurrency(purch.TotalAmount), "B", 0, aligns[6], fill, 0, "")
			pdf.CellFormat(widths[7], 6.5, formatCurrency(purch.TotalPaid), "B", 1, aligns[7], fill, 0, "")
		}

		drawSubtotalRow(pdf, "Total Purchases:", formatCurrency(report.FinancialSummary.TotalPurchases))
	}
	pdf.Ln(4)

	// -------------------------------------------------------------
	// 6. EXPENSES SECTION
	// -------------------------------------------------------------
	ensurePageSpace(pdf, 35)
	drawSectionHeading(pdf, fmt.Sprintf("4. EXPENSES (%d transactions)", len(report.Expenses)))

	if len(report.Expenses) == 0 {
		drawEmptyMessage(pdf, "No expenses recorded for this date.")
	} else {
		headers := []string{"#", "Time", "Reference", "Category", "Description", "Staff / Notes", "Amount"}
		widths := []float64{8, 14, 24, 28, 44, 34, 30}
		aligns := []string{"C", "C", "L", "L", "L", "L", "R"}

		drawGenericHeader(pdf, headers, widths, aligns)

		for i, exp := range report.Expenses {
			ensureTableRowSpace(pdf, 7, func() {
				drawGenericHeader(pdf, headers, widths, aligns)
			})

			fill := (i % 2) == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetDrawColor(226, 232, 240)

			staffOrNote := exp.EmployeeName
			if staffOrNote == "" {
				staffOrNote = exp.PaymentMethod
			} else if exp.PaymentMethod != "" {
				staffOrNote += " (" + exp.PaymentMethod + ")"
			}

			pdf.CellFormat(widths[0], 6.5, fmt.Sprintf("%d", i+1), "B", 0, aligns[0], fill, 0, "")
			pdf.CellFormat(widths[1], 6.5, exp.Time, "B", 0, aligns[1], fill, 0, "")
			pdf.CellFormat(widths[2], 6.5, exp.Reference, "B", 0, aligns[2], fill, 0, "")
			pdf.CellFormat(widths[3], 6.5, truncateString(exp.Category, 16), "B", 0, aligns[3], fill, 0, "")
			pdf.CellFormat(widths[4], 6.5, truncateString(exp.Description, 26), "B", 0, aligns[4], fill, 0, "")
			pdf.CellFormat(widths[5], 6.5, truncateString(staffOrNote, 20), "B", 0, aligns[5], fill, 0, "")
			pdf.CellFormat(widths[6], 6.5, formatCurrency(exp.Amount), "B", 1, aligns[6], fill, 0, "")
		}

		drawSubtotalRow(pdf, "Total Expenses:", formatCurrency(report.FinancialSummary.TotalExpenses))
	}
	pdf.Ln(4)

	// -------------------------------------------------------------
	// 7. CASH BALANCE SECTION
	// -------------------------------------------------------------
	ensurePageSpace(pdf, 40)
	drawSectionHeading(pdf, "5. CASH BALANCE BREAKDOWN")

	cashBoxY := pdf.GetY()
	pdf.SetDrawColor(226, 232, 240)
	pdf.SetFillColor(250, 250, 252)
	pdf.Rect(14, cashBoxY, 182, 32, "DF")

	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(71, 85, 105)

	// Column 1: Inflows
	pdf.SetXY(18, cashBoxY+3)
	pdf.CellFormat(55, 5, "CASH INFLOWS", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetX(18)
	pdf.CellFormat(35, 4.5, "Line Sales Cash:", "", 0, "L", false, 0, "")
	pdf.CellFormat(20, 4.5, formatCurrency(report.CashBalance.LineSalesCash), "", 1, "R", false, 0, "")
	pdf.SetX(18)
	pdf.CellFormat(35, 4.5, "Counter Sales Cash:", "", 0, "L", false, 0, "")
	pdf.CellFormat(20, 4.5, formatCurrency(report.CashBalance.CounterSalesCash), "", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "B", 8)
	pdf.SetX(18)
	pdf.CellFormat(35, 5, "Total Inflows:", "T", 0, "L", false, 0, "")
	pdf.CellFormat(20, 5, formatCurrency(report.CashBalance.TotalCashInflow), "T", 1, "R", false, 0, "")

	// Column 2: Outflows
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.SetXY(80, cashBoxY+3)
	pdf.CellFormat(55, 5, "CASH OUTFLOWS", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetX(80)
	pdf.CellFormat(35, 4.5, "Cash Purchases:", "", 0, "L", false, 0, "")
	pdf.CellFormat(20, 4.5, formatCurrency(report.CashBalance.PurchaseCash), "", 1, "R", false, 0, "")
	pdf.SetX(80)
	pdf.CellFormat(35, 4.5, "Cash Expenses:", "", 0, "L", false, 0, "")
	pdf.CellFormat(20, 4.5, formatCurrency(report.CashBalance.ExpenseCash), "", 1, "R", false, 0, "")
	pdf.SetX(80)
	pdf.CellFormat(35, 4.5, "Advances & Wages:", "", 0, "L", false, 0, "")
	pdf.CellFormat(20, 4.5, formatCurrency(report.CashBalance.OtherCashOutflow), "", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "B", 8)
	pdf.SetX(80)
	pdf.CellFormat(35, 5, "Total Outflows:", "T", 0, "L", false, 0, "")
	pdf.CellFormat(20, 5, formatCurrency(report.CashBalance.TotalCashOutflow), "T", 1, "R", false, 0, "")

	// Column 3: Net Cash & Closing Position
	pdf.SetXY(142, cashBoxY+3)
	pdf.SetFont("Arial", "B", 8)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(50, 5, "NET CASH POSITION", "", 1, "L", false, 0, "")
	pdf.SetFont("Arial", "", 8)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetX(142)
	pdf.CellFormat(26, 5, "Net Day Flow:", "", 0, "L", false, 0, "")
	pdf.CellFormat(24, 5, formatCurrency(report.CashBalance.NetDailyCashFlow), "", 1, "R", false, 0, "")

	// Closing Cash Highlight Banner
	pdf.SetXY(142, cashBoxY+15)
	pdf.SetFillColor(236, 253, 245) // emerald-50
	pdf.SetDrawColor(167, 243, 208) // emerald-200
	pdf.Rect(142, cashBoxY+15, 50, 14, "DF")
	pdf.SetFont("Arial", "B", 7)
	pdf.SetTextColor(6, 95, 70) // emerald-800
	pdf.SetXY(144, cashBoxY+16.5)
	pdf.CellFormat(46, 3.5, "CLOSING CASH BALANCE", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(5, 150, 105) // emerald-600
	pdf.SetX(144)
	pdf.CellFormat(46, 6, formatCurrency(report.CashBalance.ClosingCash), "", 1, "C", false, 0, "")

	pdf.SetY(cashBoxY + 36)

	// -------------------------------------------------------------
	// 8. BANK BALANCES SECTION
	// -------------------------------------------------------------
	ensurePageSpace(pdf, 30)
	drawSectionHeading(pdf, fmt.Sprintf("6. BANK BALANCES (%d accounts)", len(report.BankBalances)))

	if len(report.BankBalances) == 0 {
		drawEmptyMessage(pdf, "No bank accounts found.")
	} else {
		bankHeaders := []string{"Bank Name", "Account Number", "Current Balance"}
		bankWidths := []float64{70, 52, 60}
		bankAligns := []string{"L", "C", "R"}

		drawGenericHeader(pdf, bankHeaders, bankWidths, bankAligns)

		for i, bank := range report.BankBalances {
			ensureTableRowSpace(pdf, 7, func() {
				drawGenericHeader(pdf, bankHeaders, bankWidths, bankAligns)
			})

			fill := (i % 2) == 1
			if fill {
				pdf.SetFillColor(248, 250, 252)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}

			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(15, 23, 42)
			pdf.SetDrawColor(226, 232, 240)

			pdf.CellFormat(bankWidths[0], 6.5, truncateString(bank.BankName, 35), "B", 0, bankAligns[0], fill, 0, "")
			pdf.CellFormat(bankWidths[1], 6.5, bank.AccountNumber, "B", 0, bankAligns[1], fill, 0, "")
			pdf.CellFormat(bankWidths[2], 6.5, formatCurrency(bank.CurrentBalance), "B", 1, bankAligns[2], fill, 0, "")
		}

		drawSubtotalRow(pdf, "Total Bank Balance:", formatCurrency(report.FinancialSummary.TotalBankBalance))
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to encode PDF buffer: %w", err)
	}

	return buf.Bytes(), nil
}

// Helper functions for rendering PDF components

func drawSectionHeading(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(30, 41, 59)
	pdf.SetFillColor(241, 245, 249)
	pdf.CellFormat(182, 6.5, "  "+title, "B", 1, "L", true, 0, "")
	pdf.Ln(1.5)
}

func drawEmptyMessage(pdf *fpdf.Fpdf, message string) {
	pdf.SetFont("Arial", "I", 8.5)
	pdf.SetTextColor(148, 163, 184)
	pdf.CellFormat(182, 7, "     "+message, "", 1, "L", false, 0, "")
}

func drawMetricBox(pdf *fpdf.Fpdf, x, y, w, h float64, label, value string, r, g, b int) {
	pdf.SetXY(x, y)
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(x, y, w, h, "DF")

	// Left colored accent indicator bar
	pdf.SetFillColor(r, g, b)
	pdf.Rect(x, y, 2.5, h, "F")

	// Label
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetTextColor(100, 116, 139)
	pdf.SetXY(x+4.5, y+2)
	pdf.CellFormat(w-6, 4, strings.ToUpper(label), "", 1, "L", false, 0, "")

	// Value
	pdf.SetFont("Arial", "B", 10)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetXY(x+4.5, y+6.5)
	pdf.CellFormat(w-6, 6, value, "", 1, "L", false, 0, "")
}

func drawLineSalesHeader(pdf *fpdf.Fpdf, headers []string, widths []float64, aligns []string) {
	drawGenericHeader(pdf, headers, widths, aligns)
}

func drawGenericHeader(pdf *fpdf.Fpdf, headers []string, widths []float64, aligns []string) {
	pdf.SetFont("Arial", "B", 7.5)
	pdf.SetFillColor(241, 245, 249)
	pdf.SetTextColor(51, 65, 85)
	pdf.SetDrawColor(203, 213, 225)
	pdf.SetLineWidth(0.3)

	for i := range headers {
		pdf.CellFormat(widths[i], 6, headers[i], "BT", 0, aligns[i], true, 0, "")
	}
	pdf.Ln(-1)
}

func drawSubtotalRow(pdf *fpdf.Fpdf, label, total string) {
	pdf.SetFont("Arial", "B", 8.5)
	pdf.SetFillColor(248, 250, 252)
	pdf.SetTextColor(15, 23, 42)
	pdf.SetDrawColor(203, 213, 225)
	pdf.CellFormat(120, 6.5, label, "T", 0, "R", true, 0, "")
	pdf.CellFormat(62, 6.5, total, "T", 1, "R", true, 0, "")
}

func ensurePageSpace(pdf *fpdf.Fpdf, requiredHeight float64) {
	if pdf.GetY()+requiredHeight > 275 {
		pdf.AddPage()
	}
}

func ensureTableRowSpace(pdf *fpdf.Fpdf, rowHeight float64, onNewPage func()) {
	if pdf.GetY()+rowHeight > 275 {
		pdf.AddPage()
		if onNewPage != nil {
			onNewPage()
		}
	}
}
