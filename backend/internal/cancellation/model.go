package cancellation

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Cancellation Policy ────────────────────────────────────────────────────────

type CancellationPolicy struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CorridorKey   string    `gorm:"size:100;not null;uniqueIndex" json:"corridor_key"`
	GraceSeconds  int       `gorm:"not null;default:600" json:"grace_seconds"`
	Tier1Seconds  int       `gorm:"not null;default:3600" json:"tier1_seconds"`
	Tier2Seconds  int       `gorm:"not null;default:86400" json:"tier2_seconds"`
	FeeGracePct   float64   `gorm:"type:numeric(5,2);not null;default:0" json:"fee_grace_pct"`
	FeeTier1Pct   float64   `gorm:"type:numeric(5,2);not null;default:5" json:"fee_tier1_pct"`
	FeeTier2Pct   float64   `gorm:"type:numeric(5,2);not null;default:10" json:"fee_tier2_pct"`
	FeeMaxPct     float64   `gorm:"type:numeric(5,2);not null;default:15" json:"fee_max_pct"`
	TravelerSplit float64   `gorm:"type:numeric(5,2);not null;default:70" json:"traveler_split"`
	IsActive      bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (CancellationPolicy) TableName() string { return "cancellation_policies" }

// ── User Cancellation Stats (anti-abuse) ────────────────────────────────────────

type UserCancellationStats struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID             uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	TotalOrders        int        `gorm:"not null;default:0" json:"total_orders"`
	TotalCancellations int        `gorm:"not null;default:0" json:"total_cancellations"`
	CancelRate         float64    `gorm:"type:numeric(5,4);not null;default:0" json:"cancel_rate"`
	AbuseMultiplier    float64    `gorm:"type:numeric(5,2);not null;default:1.0" json:"abuse_multiplier"`
	LastCancelAt       *time.Time `json:"last_cancel_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func (UserCancellationStats) TableName() string { return "user_cancellation_stats" }

// ── Free Cancellation Tokens ────────────────────────────────────────────────────

type UserCancellationToken struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index:idx_user_period,2" json:"user_id"`
	RemainingTokens int       `gorm:"not null;default:2" json:"remaining_tokens"`
	PeriodStart     time.Time `gorm:"type:date;not null;index:idx_user_period,2" json:"period_start"`
	PeriodEnd       time.Time `gorm:"type:date;not null" json:"period_end"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (UserCancellationToken) TableName() string { return "user_cancellation_tokens" }

// ── Cancellation Ledger (audit trail) ────────────────────────────────────────────

type CancellationLedger struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	OrderID              uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	UserID               uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	FeeCents             int64     `gorm:"not null;default:0" json:"fee_cents"`
	TravelerCompensation int64     `gorm:"not null;default:0" json:"traveler_compensation"`
	PlatformFee          int64     `gorm:"not null;default:0" json:"platform_fee"`
	FeePercent           float64   `gorm:"type:numeric(5,2);not null;default:0" json:"fee_percent"`
	AbuseMultiplier      float64   `gorm:"type:numeric(5,2);not null;default:1.0" json:"abuse_multiplier"`
	TokenUsed            bool      `gorm:"not null;default:false" json:"token_used"`
	SecondsSinceAccept   int       `gorm:"not null;default:0" json:"seconds_since_accept"`
	Tier                 string    `gorm:"size:20;not null;default:'grace'" json:"tier"`
	Reason               string    `json:"reason,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

func (CancellationLedger) TableName() string { return "cancellation_ledger" }

// ── Fee Tier Constants ──────────────────────────────────────────────────────────

const (
	TierFree  = "free"  // Before acceptance
	TierGrace = "grace" // Within grace period
	Tier1     = "tier1" // Within tier1
	Tier2     = "tier2" // Within tier2
	TierMax   = "max"   // After tier2
)

// ── Calculation Result ──────────────────────────────────────────────────────────

type CancellationFeeResult struct {
	FeeCents             int64     `json:"fee_cents"`
	TravelerCompensation int64     `json:"traveler_compensation"`
	PlatformFee          int64     `json:"platform_fee"`
	FeePercent           float64   `json:"fee_percent"`
	Tier                 string    `json:"tier"`
	AbuseMultiplier      float64   `json:"abuse_multiplier"`
	TokenUsed            bool      `json:"token_used"`
	SecondsSinceAccept   int       `json:"seconds_since_accept"`
	InsuranceApplied     bool      `json:"insurance_applied"`
	OriginalFeeCents     int64     `json:"original_fee_cents,omitempty"`
	InsuranceID          uuid.UUID `json:"insurance_id,omitempty"`
}

// ── Admin Stats ──────────────────────────────────────────────────────────────────

type AdminCancellationStats struct {
	TotalCancellations  int64   `json:"total_cancellations"`
	TotalFeesCollected  int64   `json:"total_fees_collected_cents"`
	TravelerCompensated int64   `json:"traveler_compensated_cents"`
	PlatformFees        int64   `json:"platform_fees_cents"`
	AvgFeePercent       float64 `json:"avg_fee_percent"`
	TokensUsed          int64   `json:"tokens_used"`
	HighRiskUsers       int64   `json:"high_risk_users"` // cancel_rate > 30%
}

// ── Seed ──────────────────────────────────────────────────────────────────────────

func SeedDefaults(db *gorm.DB) {
	db.Where("corridor_key = ?", "global").FirstOrCreate(&CancellationPolicy{
		CorridorKey:   "global",
		GraceSeconds:  600,
		Tier1Seconds:  3600,
		Tier2Seconds:  86400,
		FeeGracePct:   0,
		FeeTier1Pct:   5,
		FeeTier2Pct:   10,
		FeeMaxPct:     15,
		TravelerSplit: 70,
		IsActive:      true,
	})
}
