package forex

import (
	"time"

	"github.com/google/uuid"
)

// ExchangeRate stores currency conversion rates with spread and fees.
type ExchangeRate struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	FromCurrency  string     `gorm:"size:3;not null;uniqueIndex:idx_from_to" json:"from_currency"`
	ToCurrency    string     `gorm:"size:3;not null;uniqueIndex:idx_from_to" json:"to_currency"`
	Rate          float64    `gorm:"not null" json:"rate"`                   // mid-market rate
	SpreadPct     float64    `gorm:"default:0.5" json:"spread_pct"`          // spread percentage (e.g. 0.5%)
	FeePct        float64    `gorm:"default:1.0" json:"fee_pct"`             // conversion fee percentage
	FeeFixed      float64    `gorm:"default:0" json:"fee_fixed"`             // fixed fee per conversion
	EffectiveRate float64    `gorm:"not null" json:"effective_rate"`         // rate after spread (what user gets)
	Source        string     `gorm:"size:50;default:'manual'" json:"source"` // manual, ecb, openexchangerates
	ValidFrom     time.Time  `json:"valid_from"`
	ValidTo       *time.Time `json:"valid_to,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// ConversionRecord is an audit trail entry for every currency conversion.
type ConversionRecord struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	FromCurrency   string     `gorm:"size:3;not null" json:"from_currency"`
	ToCurrency     string     `gorm:"size:3;not null" json:"to_currency"`
	FromAmount     float64    `gorm:"not null" json:"from_amount"`
	ToAmount       float64    `gorm:"not null" json:"to_amount"`
	MidRate        float64    `gorm:"not null" json:"mid_rate"`
	EffectiveRate  float64    `gorm:"not null" json:"effective_rate"`
	SpreadAmount   float64    `gorm:"not null" json:"spread_amount"`
	FeeAmount      float64    `gorm:"not null" json:"fee_amount"`
	WalletTxnID    *uuid.UUID `gorm:"type:uuid;index" json:"wallet_txn_id,omitempty"`
	IdempotencyKey string     `gorm:"size:128;index" json:"idempotency_key,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// ConvertReq is the request body for a currency conversion.
type ConvertReq struct {
	FromCurrency   string  `json:"from_currency" binding:"required"`
	ToCurrency     string  `json:"to_currency" binding:"required"`
	Amount         float64 `json:"amount" binding:"required,gt=0"`
	IdempotencyKey string  `json:"idempotency_key"`
}
