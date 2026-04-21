package admin

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/geocore-next/backend/internal/payments"
	"github.com/geocore-next/backend/internal/settlement"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/signintech/gopdf"
	"gorm.io/gorm"
)

// financeReportRow holds one line of the financial summary.
type financeReportRow struct {
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Count    int64   `json:"count"`
}

// buildFinanceReport queries the DB for a financial summary within the
// requested date range and returns structured rows.
func buildFinanceReport(db *gorm.DB, from, to time.Time) []financeReportRow {
	var rows []financeReportRow

	// Revenue (succeeded payments)
	var revenue struct {
		Sum float64
		Cnt int64
	}
	db.Model(&payments.Payment{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", "succeeded", from, to).
		Select("COALESCE(SUM(amount),0) as sum, COUNT(*) as cnt").
		Scan(&revenue)
	rows = append(rows, financeReportRow{Category: "Revenue", Amount: revenue.Sum, Count: revenue.Cnt})

	// Fees collected (from escrow fee column)
	var feesCollected struct {
		Sum string
		Cnt int64
	}
	db.Model(&wallet.Escrow{}).
		Where("created_at BETWEEN ? AND ?", from, to).
		Select("COALESCE(SUM(fee),0) as sum, COUNT(*) as cnt").
		Scan(&feesCollected)
	feeAmt, _ := decimal.NewFromString(feesCollected.Sum)
	rows = append(rows, financeReportRow{Category: "Fees Collected", Amount: feeAmt.InexactFloat64(), Count: feesCollected.Cnt})

	// Refunds
	var refunds struct {
		Sum float64
		Cnt int64
	}
	db.Model(&payments.Payment{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", "refunded", from, to).
		Select("COALESCE(SUM(amount),0) as sum, COUNT(*) as cnt").
		Scan(&refunds)
	rows = append(rows, financeReportRow{Category: "Refunds", Amount: refunds.Sum, Count: refunds.Cnt})

	// Escrow held (PENDING status = funds held in escrow)
	var escrow struct {
		Sum string
		Cnt int64
	}
	db.Model(&wallet.Escrow{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", "PENDING", from, to).
		Select("COALESCE(SUM(amount),0) as sum, COUNT(*) as cnt").
		Scan(&escrow)
	escrowAmt, _ := decimal.NewFromString(escrow.Sum)
	rows = append(rows, financeReportRow{Category: "Escrow Held", Amount: escrowAmt.InexactFloat64(), Count: escrow.Cnt})

	// Payouts
	var payouts struct {
		Sum float64
		Cnt int64
	}
	db.Model(&settlement.Payout{}).
		Where("status = ? AND created_at BETWEEN ? AND ?", "completed", from, to).
		Select("COALESCE(SUM(amount),0) as sum, COUNT(*) as cnt").
		Scan(&payouts)
	rows = append(rows, financeReportRow{Category: "Payouts", Amount: payouts.Sum, Count: payouts.Cnt})

	return rows
}

// parseDateRange extracts from/to from query params with sensible defaults.
func parseDateRange(c *gin.Context) (from, to time.Time) {
	now := time.Now()
	from = now.AddDate(0, -1, 0) // default: last 30 days
	to = now

	if v := c.Query("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = t
		}
	}
	if v := c.Query("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = t.Add(24 * time.Hour) // include the full "to" day
		}
	}
	return
}

// GetFinanceReport dispatches to CSV or PDF export based on ?format= query param.
func (h *Handler) GetFinanceReport(c *gin.Context) {
	format := c.DefaultQuery("format", "csv")
	switch format {
	case "pdf":
		h.GetFinanceReportPDF(c)
	default:
		h.GetFinanceReportCSV(c)
	}
}

// GetFinanceReportCSV exports the financial summary as a CSV file.
func (h *Handler) GetFinanceReportCSV(c *gin.Context) {
	from, to := parseDateRange(c)
	rows := buildFinanceReport(h.db, from, to)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"Category", "Amount", "Count"})
	for _, r := range rows {
		_ = w.Write([]string{r.Category, fmt.Sprintf("%.2f", r.Amount), fmt.Sprintf("%d", r.Count)})
	}
	w.Flush()

	filename := fmt.Sprintf("finance_report_%s_to_%s.csv", from.Format("2006-01-02"), to.Format("2006-01-02"))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv", buf.Bytes())
}

// GetFinanceReportPDF exports the financial summary as a PDF file.
func (h *Handler) GetFinanceReportPDF(c *gin.Context) {
	from, to := parseDateRange(c)
	rows := buildFinanceReport(h.db, from, to)

	pdf := gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: gopdf.Rect{W: 595, H: 842}}) // A4
	pdf.AddPage()

	// Title
	pdf.SetXY(40, 40)
	pdf.Cell(nil, "GeoCore Next — Finance Report")
	pdf.SetXY(40, 60)
	pdf.Cell(nil, fmt.Sprintf("Period: %s to %s", from.Format("2006-01-02"), to.Format("2006-01-02")))

	// Table header
	y := 100.0
	pdf.SetXY(40, y)
	pdf.Cell(nil, "Category")
	pdf.SetXY(250, y)
	pdf.Cell(nil, "Amount")
	pdf.SetXY(400, y)
	pdf.Cell(nil, "Count")
	y += 8
	pdf.Line(40, y, 500, y)
	y += 12

	// Table rows
	for _, r := range rows {
		pdf.SetXY(40, y)
		pdf.Cell(nil, r.Category)
		pdf.SetXY(250, y)
		pdf.Cell(nil, fmt.Sprintf("%.2f", r.Amount))
		pdf.SetXY(400, y)
		pdf.Cell(nil, fmt.Sprintf("%d", r.Count))
		y += 20
	}

	// Net line
	y += 8
	pdf.Line(40, y, 500, y)
	y += 12
	net := 0.0
	for _, r := range rows {
		switch r.Category {
		case "Revenue", "Fees Collected":
			net += r.Amount
		case "Refunds", "Payouts", "Escrow Held":
			net -= r.Amount
		}
	}
	pdf.SetXY(40, y)
	pdf.Cell(nil, "Net")
	pdf.SetXY(250, y)
	pdf.Cell(nil, fmt.Sprintf("%.2f", net))

	filename := fmt.Sprintf("finance_report_%s_to_%s.pdf", from.Format("2006-01-02"), to.Format("2006-01-02"))
	c.Header("Content-Disposition", "attachment; filename="+filename)

	var buf bytes.Buffer
	_, _ = pdf.WriteTo(&buf)
	c.Data(http.StatusOK, "application/pdf", buf.Bytes())
}
