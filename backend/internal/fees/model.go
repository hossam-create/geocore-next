package fees

import (
	"time"

	"github.com/google/uuid"
)

// FeeType categorises each fee in the engine.
type FeeType string

const (
	FeeTypeTransaction  FeeType = "transaction"   // % of payment amount
	FeeTypeEscrow       FeeType = "escrow"         // % of escrowed amount
	FeeTypeForexSpread  FeeType = "forex_spread"   // extra spread on top of midmarket
	FeeTypeWithdrawal   FeeType = "withdrawal"     // % + fixed on withdrawals
	FeeTypeReferral     FeeType = "referral"        // bonus credited to referrer wallet
)

// FeeConfig stores a configurable fee rule.
// Rules are matched by type, and optionally scoped to a country or amount range.
// The most specific matching rule wins; fallback to the wildcard (Country="*").
type FeeConfig struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	FeeType    FeeType   `gorm:"size:50;not null;index" json:"fee_type"`
	Country    string    `gorm:"size:3;not null;default:'*'" json:"country"` // ISO-3 or "*" for default
	MinAmount  float64   `gorm:"default:0" json:"min_amount"`
	MaxAmount  float64   `gorm:"default:0" json:"max_amount"` // 0 = no upper limit
	FeePct     float64   `gorm:"default:0" json:"fee_pct"`   // e.g. 2.5 for 2.5%
	FeeFixed   float64   `gorm:"default:0" json:"fee_fixed"` // absolute fixed fee
	MinFee     float64   `gorm:"default:0" json:"min_fee"`   // floor
	MaxFee     float64   `gorm:"default:0" json:"max_fee"`   // cap (0 = no cap)
	IsActive   bool      `gorm:"default:true" json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// FeeResult is the computed fee breakdown for a transaction.
type FeeResult struct {
	GrossAmount float64 `json:"gross_amount"`
	FeeAmount   float64 `json:"fee_amount"`
	NetAmount   float64 `json:"net_amount"`
	FeePct      float64 `json:"fee_pct_applied"`
	FeeFixed    float64 `json:"fee_fixed_applied"`
	Rule        string  `json:"rule"` // which config matched
}
