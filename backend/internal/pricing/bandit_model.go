package pricing

import (
	"time"

	"github.com/google/uuid"
)

// ── Bandit Arm ──────────────────────────────────────────────────────────────────

// BanditArm represents a price point that the bandit can select.
type BanditArm struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Segment       string    `gorm:"size:50;not null;uniqueIndex:idx_bandit_arm_seg_price,1" json:"segment"` // e.g. "high_trust", "low_trust", "high_value", "default"
	PricePercent  float64   `gorm:"type:numeric(5,2);not null;uniqueIndex:idx_bandit_arm_seg_price,2" json:"price_percent"` // 1, 2, 3, 4
	Impressions   int64     `gorm:"not null;default:0" json:"impressions"`
	Conversions   int64     `gorm:"not null;default:0" json:"conversions"`
	TotalReward   float64   `gorm:"type:numeric(12,2);not null;default:0" json:"total_reward"`
	Alpha         float64   `gorm:"type:numeric(8,2);not null;default:1" json:"alpha"` // Beta distribution α (successes + 1)
	Beta          float64   `gorm:"type:numeric(8,2);not null;default:1" json:"beta"`  // Beta distribution β (failures + 1)
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (BanditArm) TableName() string { return "bandit_arms" }

// ── Bandit Event (impression + outcome) ──────────────────────────────────────────

type BanditEvent struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderID       uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	Segment       string    `gorm:"size:50;not null;index" json:"segment"`
	ArmID         uuid.UUID `gorm:"type:uuid;not null;index" json:"arm_id"`
	PricePercent  float64   `gorm:"type:numeric(5,2);not null" json:"price_percent"`
	PriceCents    int64     `gorm:"not null" json:"price_cents"`
	DidBuy        bool      `gorm:"not null;default:false" json:"did_buy"`
	Reward        float64   `gorm:"type:numeric(12,2);not null;default:0" json:"reward"`
	ClaimCost     float64   `gorm:"type:numeric(12,2);not null;default:0" json:"claim_cost"`
	Algorithm     string    `gorm:"size:20;not null;default:'thompson'" json:"algorithm"` // thompson | ucb | epsilon_greedy
	CreatedAt     time.Time `json:"created_at"`
}

func (BanditEvent) TableName() string { return "bandit_events" }

// ── Bandit Config ────────────────────────────────────────────────────────────────

type BanditConfig struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Algorithm            string    `gorm:"size:20;not null;default:'thompson'" json:"algorithm"` // thompson | ucb | epsilon_greedy
	Epsilon              float64   `gorm:"type:numeric(5,2);not null;default:0.2" json:"epsilon"` // ε for ε-greedy (0.2 = 20% explore)
	MinPricePercent      float64   `gorm:"type:numeric(5,2);not null;default:1" json:"min_price_percent"`
	MaxPricePercent      float64   `gorm:"type:numeric(5,2);not null;default:4" json:"max_price_percent"`
	ConversionDropThreshold float64 `gorm:"type:numeric(5,2);not null;default:0.10" json:"conversion_drop_threshold"` // 10% drop → kill switch
	SessionCooldownMinutes int    `gorm:"not null;default:5" json:"session_cooldown_minutes"`
	MinImpressionsBeforeExploit int `gorm:"not null;default:100" json:"min_impressions_before_exploit"` // need N impressions before trusting stats
	IsActive             bool      `gorm:"not null;default:true" json:"is_active"`
	KillSwitchActive     bool      `gorm:"not null;default:false" json:"kill_switch_active"`
	FallbackPricePercent float64   `gorm:"type:numeric(5,2);not null;default:2" json:"fallback_price_percent"` // used when kill switch on
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (BanditConfig) TableName() string { return "bandit_configs" }

// ── Bandit Segment ──────────────────────────────────────────────────────────────

// UserSegment classifies a user for contextual bandit pricing.
type UserSegment string

const (
	SegmentDefault    UserSegment = "default"
	SegmentHighTrust  UserSegment = "high_trust"  // trust_score > 70
	SegmentLowTrust   UserSegment = "low_trust"   // trust_score < 30
	SegmentHighValue  UserSegment = "high_value"  // order > 500 AED
	SegmentLowValue   UserSegment = "low_value"   // order < 100 AED
	SegmentHighRisk   UserSegment = "high_risk"   // risk_score > 50
	SegmentFrequent   UserSegment = "frequent"    // past_insurance_usage > 3
	SegmentResistant  UserSegment = "resistant"   // insurance_buy_rate < 0.2
)

// ClassifySegment determines which bandit segment a user belongs to.
func ClassifySegment(ctx *PricingContext) UserSegment {
	// Priority: high_risk > low_trust > high_trust > frequent > resistant > high_value > low_value > default
	if ctx.DeliveryRiskScore > 0.5 || ctx.CancellationRate > 0.5 {
		return SegmentHighRisk
	}
	if ctx.TrustScore < 30 {
		return SegmentLowTrust
	}
	if ctx.TrustScore > 70 {
		return SegmentHighTrust
	}
	if ctx.PastInsuranceUsage > 3 {
		return SegmentFrequent
	}
	if ctx.InsuranceBuyRate < 0.2 && ctx.PastInsuranceUsage > 1 {
		return SegmentResistant
	}
	if ctx.OrderPriceCents > 50000 { // > 500 AED
		return SegmentHighValue
	}
	if ctx.OrderPriceCents < 10000 { // < 100 AED
		return SegmentLowValue
	}
	return SegmentDefault
}

// ── Bandit Selection Result ─────────────────────────────────────────────────────

type BanditSelectionResult struct {
	ArmID          uuid.UUID `json:"arm_id"`
	Segment        string    `json:"segment"`
	PricePercent   float64   `json:"price_percent"`
	PriceCents     int64     `json:"price_cents"`
	Algorithm      string    `json:"algorithm"`
	SampleValue    float64   `json:"sample_value"`     // the sampled value from the arm
	Confidence     float64   `json:"confidence"`       // how confident we are
	IsExploration  bool      `json:"is_exploration"`    // true if this was an exploration pick
	AnchorPrice    int64     `json:"anchor_price"`     // "was X" for anchoring
	KillSwitchOn   bool      `json:"kill_switch_on"`   // true if kill switch activated
}

// ── Bandit Stats (admin view) ────────────────────────────────────────────────────

type BanditArmStats struct {
	Segment       string  `json:"segment"`
	PricePercent  float64 `json:"price_percent"`
	Impressions   int64   `json:"impressions"`
	Conversions   int64   `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	AvgReward     float64 `json:"avg_reward"`
	TotalReward   float64 `json:"total_reward"`
	Alpha         float64 `json:"alpha"`
	Beta          float64 `json:"beta"`
	SampleValue   float64 `json:"sample_value"` // latest Thompson sample
}

type BanditDashboard struct {
	Config          BanditConfig       `json:"config"`
	Arms            []BanditArmStats   `json:"arms"`
	TotalImpressions int64             `json:"total_impressions"`
	TotalConversions int64             `json:"total_conversions"`
	OverallAttachRate float64          `json:"overall_attach_rate"`
	OverallRevenue   float64           `json:"overall_revenue"`
	KillSwitchActive bool              `json:"kill_switch_active"`
	BestArm         *BanditArmStats    `json:"best_arm,omitempty"`
	ConversionTrend  string            `json:"conversion_trend"` // "stable", "dropping", "improving"
}
