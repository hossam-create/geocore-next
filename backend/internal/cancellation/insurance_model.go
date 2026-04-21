package cancellation

import (
	"time"

	"github.com/google/uuid"
)

// ── Coverage Type ──────────────────────────────────────────────────────────────

type CoverageType string

const (
	CoverageBasic    CoverageType = "basic"    // Free cancel within grace only
	CoveragePlus     CoverageType = "plus"     // Free cancel anytime before execution
	CoveragePremium  CoverageType = "premium"  // Free cancel + priority support + faster refund
)

// ── Order Insurance ────────────────────────────────────────────────────────────

type OrderInsurance struct {
	ID                 uuid.UUID    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	OrderID            uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex" json:"order_id"`
	UserID             uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	PriceCents         int64        `gorm:"not null;default:0" json:"price_cents"`
	CoverageType       CoverageType `gorm:"size:20;not null;default:'basic'" json:"coverage_type"`
	MaxFeeCoveredPct   float64      `gorm:"type:numeric(5,2);not null;default:100" json:"max_fee_covered_pct"`
	IsActive           bool         `gorm:"not null;default:true" json:"is_active"`
	IsUsed             bool         `gorm:"not null;default:false" json:"is_used"`
	FirstOrderFree     bool         `gorm:"not null;default:false" json:"first_order_free"`
	CreatedAt          time.Time    `json:"created_at"`
}

func (OrderInsurance) TableName() string { return "order_insurances" }

// ── User Insurance Usage (anti-abuse tracking) ──────────────────────────────────

type UserInsuranceUsage struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID             uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_month,2" json:"user_id"`
	Month              time.Time `gorm:"type:date;not null;uniqueIndex:idx_user_month,2" json:"month"`
	CancellationsUsed  int       `gorm:"not null;default:0" json:"cancellations_used"`
	InsurancePurchased int       `gorm:"not null;default:0" json:"insurance_purchased"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (UserInsuranceUsage) TableName() string { return "user_insurance_usage" }

// ── Insurance Pricing Tiers ────────────────────────────────────────────────────

type InsurancePriceTier struct {
	CoverageType     CoverageType `json:"coverage_type"`
	PricePercent     float64      `json:"price_percent"`      // % of order total
	MaxFeeCoveredPct float64      `json:"max_fee_covered_pct"` // % of cancellation fee covered
	Label           string       `json:"label"`
	Description     string       `json:"description"`
}

var InsuranceTiers = []InsurancePriceTier{
	{
		CoverageType:     CoverageBasic,
		PricePercent:     1.0,  // 1% of order
		MaxFeeCoveredPct: 100,  // full waive within grace
		Label:           "Basic",
		Description:     "Free cancellation within grace period",
	},
	{
		CoverageType:     CoveragePlus,
		PricePercent:     2.0,  // 2% of order
		MaxFeeCoveredPct: 100,  // full waive anytime before execution
		Label:           "Plus",
		Description:     "Cancel anytime before execution — no fees",
	},
	{
		CoverageType:     CoveragePremium,
		PricePercent:     3.0,  // 3% of order
		MaxFeeCoveredPct: 100,  // full waive + priority support
		Label:           "Premium",
		Description:     "Cancel anytime + priority support + faster refund",
	},
}

// ── Insurance Purchase Result ──────────────────────────────────────────────────

type InsurancePurchaseResult struct {
	InsuranceID   uuid.UUID    `json:"insurance_id"`
	OrderID       uuid.UUID    `json:"order_id"`
	PriceCents    int64        `json:"price_cents"`
	CoverageType  CoverageType `json:"coverage_type"`
	FirstOrderFree bool        `json:"first_order_free"`
}

// ── Insurance Cancellation Result ──────────────────────────────────────────────

type InsuranceCancellationResult struct {
	OriginalFeeCents   int64 `json:"original_fee_cents"`
	InsuranceApplied   bool  `json:"insurance_applied"`
	FeeCoveredCents    int64 `json:"fee_covered_cents"`
	FinalFeeCents      int64 `json:"final_fee_cents"`
}

// ── Admin Insurance Stats ────────────────────────────────────────────────────────

type AdminInsuranceStats struct {
	TotalPurchased    int64   `json:"total_purchased"`
	TotalRevenueCents int64   `json:"total_revenue_cents"`
	TotalUsed         int64   `json:"total_used"`           // how many times insurance was invoked
	UsageRate         float64 `json:"usage_rate"`           // used / purchased
	FirstOrderFreeCount int64 `json:"first_order_free_count"`
	AvgPriceCents     int64   `json:"avg_price_cents"`
	ByCoverageType    map[string]int64 `json:"by_coverage_type"`
}
