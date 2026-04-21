package billing

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Invoice is a billing record for one tenant over one calendar month.
type Invoice struct {
	ID          string            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID    string            `gorm:"type:uuid;not null;index"                       json:"tenant_id"`
	PeriodStart time.Time         `gorm:"not null"                                       json:"period_start"`
	PeriodEnd   time.Time         `gorm:"not null"                                       json:"period_end"`
	AmountCents int64             `gorm:"not null;default:0"                             json:"amount_cents"`
	AmountUSD   string            `gorm:"-"                                              json:"amount_usd"`
	Items       string            `gorm:"type:jsonb"                                     json:"-"`
	Breakdown   *InvoiceBreakdown `gorm:"-"                                              json:"breakdown,omitempty"`
	Status      string            `gorm:"not null;default:'draft'"                       json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
}

// CurrentInvoice fetches the existing draft invoice for this month,
// or generates a fresh one if none exists.
// It always recalculates the amount with the latest usage figures.
func CurrentInvoice(db *gorm.DB, tenantID string, plan Plan) (*Invoice, error) {
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	usage := Summarize(db, tenantID, start, now)
	breakdown := Compute(usage, plan)
	items, _ := json.Marshal(breakdown.Lines)

	var inv Invoice
	result := db.Where("tenant_id = ? AND period_start = ? AND status = 'draft'", tenantID, start).First(&inv)

	if result.Error == nil {
		// Update existing draft with current usage
		db.Model(&inv).Updates(map[string]interface{}{
			"amount_cents": breakdown.TotalCents,
			"items":        string(items),
			"period_end":   now,
		})
		inv.AmountCents = breakdown.TotalCents
		inv.AmountUSD = breakdown.TotalUSD
		inv.Breakdown = &breakdown
		return &inv, nil
	}

	inv = Invoice{
		TenantID:    tenantID,
		PeriodStart: start,
		PeriodEnd:   now,
		AmountCents: breakdown.TotalCents,
		AmountUSD:   breakdown.TotalUSD,
		Items:       string(items),
		Breakdown:   &breakdown,
		Status:      "draft",
	}
	if err := db.Create(&inv).Error; err != nil {
		return nil, fmt.Errorf("create invoice: %w", err)
	}
	return &inv, nil
}
