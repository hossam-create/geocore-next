package pricing

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ── Cross-System RL Coordinator Models ────────────────────────────────────────────
//
// Unified decision engine across Pricing + Ranking + Recommendations.
// Instead of 3 separate decisions, one coordinated bundle that maintains consistency.
//
// Objective:
//
//	max E[Σ γ^t (w1·GMV + w2·CTR - w3·ClaimCost - w4·Churn)]

// ── Cross State ────────────────────────────────────────────────────────────────────

// CrossState is the unified state across all subsystems.
type CrossState struct {
	// ── User ────────────────────────────────────────────────────────────────
	UserTrust      float64 `json:"user_trust"`
	UserSegment    string  `json:"user_segment"` // vip, regular, new, at_risk
	CancelRate     float64 `json:"cancel_rate"`
	BuyRate        float64 `json:"buy_rate"`
	AccountAgeDays float64 `json:"account_age_days"`

	// ── Session ──────────────────────────────────────────────────────────────
	SessionStep    int       `json:"session_step"`
	Device         string    `json:"device"` // mobile, desktop, tablet
	Geo            string    `json:"geo"`    // country code
	RefusalCount   int       `json:"refusal_count"`
	PreviousPrices []float64 `json:"previous_prices"`

	// ── Market ────────────────────────────────────────────────────────────────
	DemandScore float64 `json:"demand_score"` // 0-1
	SupplyScore float64 `json:"supply_score"` // 0-1
	IsLiveHot   bool    `json:"is_live_hot"`  // trending session active

	// ── Item ─────────────────────────────────────────────────────────────────
	ItemPriceCents int64   `json:"item_price_cents"`
	CategoryPath   string  `json:"category_path"` // e.g. "electronics/phones"
	DeliveryRisk   float64 `json:"delivery_risk"`

	// ── Derived ──────────────────────────────────────────────────────────────
	RiskScore    float64 `json:"risk_score"` // 1 - trust/100
	UrgencyScore float64 `json:"urgency_score"`
}

// CrossStateKey is a discretized key for Q-table lookup.
type CrossStateKey string

// Discretize converts CrossState to a discrete key.
func (s *CrossState) Discretize() CrossStateKey {
	trustBin := discretize(s.UserTrust, 0, 100, 5)
	segCode := segmentCode(s.UserSegment)
	riskBin := discretize(s.RiskScore, 0, 1, 4)
	demandBin := discretize(s.DemandScore, 0, 1, 3)
	refusalBin := s.RefusalCount
	if refusalBin > 3 {
		refusalBin = 3
	}

	return CrossStateKey(fmt.Sprintf("t%d_s%s_r%d_d%d_f%d",
		trustBin, segCode, riskBin, demandBin, refusalBin))
}

func segmentCode(seg string) string {
	codes := map[string]string{
		"vip":     "v",
		"regular": "r",
		"new":     "n",
		"at_risk": "a",
	}
	if c, ok := codes[seg]; ok {
		return c
	}
	return "r"
}

// ── Bundle Action ──────────────────────────────────────────────────────────────────

// BundleAction is the coordinated output: price + ranking + recommendations.
type BundleAction struct {
	// ── Pricing ──────────────────────────────────────────────────────────────
	PriceCents   int64   `json:"price_cents"`
	PricePercent float64 `json:"price_percent"`

	// ── Ranking ──────────────────────────────────────────────────────────────
	BoostScore int `json:"boost_score"` // 0-100, how much to boost this item

	// ── Recommendations ──────────────────────────────────────────────────────
	RecIDs      []string `json:"rec_ids"`      // top-K recommended item IDs
	RecStrategy string   `json:"rec_strategy"` // rl, cf, popular, similar

	// ── Meta ─────────────────────────────────────────────────────────────────
	Source        string  `json:"source"` // rl, fallback, rules, blend
	Confidence    float64 `json:"confidence"`
	IsExploration bool    `json:"is_exploration"`
	IsShadow      bool    `json:"is_shadow"`

	// ── Per-head sources (observability) ──────────────────────────────────────
	SourcePricing string `json:"source_pricing"` // rl | bandit | rules
	SourceRanking string `json:"source_ranking"` // rl | heuristic
	SourceRecs    string `json:"source_recs"`    // rl | cf | popular

	// ── UX ────────────────────────────────────────────────────────────────────
	UXVariant   string `json:"ux_variant"`
	AnchorPrice int64  `json:"anchor_price"`
}

// ── Cross Config ──────────────────────────────────────────────────────────────────

type CrossConfig struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`

	// ── Reward weights ──────────────────────────────────────────────────────
	WeightGMV       float64 `gorm:"type:numeric(5,2);not null;default:0.4" json:"weight_gmv"`        // α
	WeightCTR       float64 `gorm:"type:numeric(5,2);not null;default:0.2" json:"weight_ctr"`        // β
	WeightClaimCost float64 `gorm:"type:numeric(5,2);not null;default:0.2" json:"weight_claim_cost"` // γ
	WeightChurn     float64 `gorm:"type:numeric(5,2);not null;default:0.2" json:"weight_churn"`      // δ

	// ── RL params ────────────────────────────────────────────────────────────
	LearningRate        float64 `gorm:"type:numeric(5,4);not null;default:0.1" json:"learning_rate"`
	DiscountFactor      float64 `gorm:"type:numeric(5,4);not null;default:0.95" json:"discount_factor"`
	Epsilon             float64 `gorm:"type:numeric(5,4);not null;default:0.1" json:"epsilon"`
	ConfidenceThreshold float64 `gorm:"type:numeric(5,2);not null;default:0.6" json:"confidence_threshold"`

	// ── Price bounds ────────────────────────────────────────────────────────
	MinPricePercent float64 `gorm:"type:numeric(5,2);not null;default:1" json:"min_price_percent"`
	MaxPricePercent float64 `gorm:"type:numeric(5,2);not null;default:4" json:"max_price_percent"`

	// ── Consistency rules ────────────────────────────────────────────────────
	MaxBoostWithHighPrice int     `gorm:"not null;default:30" json:"max_boost_with_high_price"` // cap boost when price > 3%
	HighPriceThreshold    float64 `gorm:"type:numeric(5,2);not null;default:3.0" json:"high_price_threshold"`

	// ── Safety ──────────────────────────────────────────────────────────────
	EmergencyModeActive     bool    `gorm:"not null;default:false" json:"emergency_mode_active"`
	ConversionDropThreshold float64 `gorm:"type:numeric(5,2);not null;default:0.08" json:"conversion_drop_threshold"`
	SessionCooldownMinutes  int     `gorm:"not null;default:5" json:"session_cooldown_minutes"`
	MaxSessionSteps         int     `gorm:"not null;default:3" json:"max_session_steps"`
	AnomalyDetectionEnabled bool    `gorm:"not null;default:true" json:"anomaly_detection_enabled"`

	// ── Rollout ──────────────────────────────────────────────────────────────
	RolloutPercent int `gorm:"not null;default:5" json:"rollout_percent"`

	// ── Fallback config ──────────────────────────────────────────────────────
	FallbackPricePercent float64 `gorm:"type:numeric(5,2);not null;default:2" json:"fallback_price_percent"`
	FallbackBoostScore   int     `gorm:"not null;default:50" json:"fallback_boost_score"`
	FallbackRecStrategy  string  `gorm:"size:20;not null;default:'popular'" json:"fallback_rec_strategy"`

	IsActive   bool      `gorm:"not null;default:true" json:"is_active"`
	QTableJSON string    `gorm:"type:text" json:"q_table_json,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (CrossConfig) TableName() string { return "cross_configs" }

// ── Cross Transition ──────────────────────────────────────────────────────────────

// CrossTransition records (s, a_bundle, r, s') for offline training.
type CrossTransition struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	SessionID string    `gorm:"size:50;not null;index" json:"session_id"`

	// State
	StateKey  string `gorm:"size:120;not null;index" json:"state_key"`
	StateJSON string `gorm:"type:text" json:"state_json"`

	// Action bundle
	PriceCents   int64   `gorm:"not null" json:"price_cents"`
	PricePercent float64 `gorm:"type:numeric(5,2);not null" json:"price_percent"`
	BoostScore   int     `gorm:"not null" json:"boost_score"`
	RecIDsJSON   string  `gorm:"type:text" json:"rec_ids_json"` // JSON array
	RecStrategy  string  `gorm:"size:20;not null" json:"rec_strategy"`

	// Reward components
	RewardGMV       float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_gmv"`
	RewardCTR       float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_ctr"`
	RewardClaimCost float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_claim_cost"`
	RewardChurn     float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_churn"`
	RewardTotal     float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_total"`

	// Outcome
	DidBuy   bool `gorm:"not null;default:false" json:"did_buy"`
	DidClick bool `gorm:"not null;default:false" json:"did_click"`
	DidClaim bool `gorm:"not null;default:false" json:"did_claim"`
	DidChurn bool `gorm:"not null;default:false" json:"did_churn"`

	// Next state
	NextStateKey string `gorm:"size:120;not null;default:''" json:"next_state_key"`
	IsTerminal   bool   `gorm:"not null;default:false" json:"is_terminal"`
	EpisodeID    string `gorm:"size:50;index" json:"episode_id"`

	CreatedAt time.Time `json:"created_at"`
}

func (CrossTransition) TableName() string { return "cross_transitions" }

// ── Cross Event (observability) ──────────────────────────────────────────────────

type CrossEvent struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderID        uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	SourcePricing  string    `gorm:"size:20;not null;default:'rules'" json:"source_pricing"`
	SourceRanking  string    `gorm:"size:20;not null;default:'heuristic'" json:"source_ranking"`
	SourceRecs     string    `gorm:"size:20;not null;default:'popular'" json:"source_recs"`
	PriceCents     int64     `gorm:"not null" json:"price_cents"`
	PricePercent   float64   `gorm:"type:numeric(5,2);not null" json:"price_percent"`
	BoostScore     int       `gorm:"not null" json:"boost_score"`
	RecIDsJSON     string    `gorm:"type:text" json:"rec_ids_json"`
	Confidence     float64   `gorm:"type:numeric(5,4);not null" json:"confidence"`
	IsShadow       bool      `gorm:"not null;default:false" json:"is_shadow"`
	GuardrailsJSON string    `gorm:"type:text" json:"guardrails_json"`
	DidBuy         bool      `gorm:"not null;default:false" json:"did_buy"`
	DidClick       bool      `gorm:"not null;default:false" json:"did_click"`
	Reward         float64   `gorm:"type:numeric(12,2);not null;default:0" json:"reward"`
	CreatedAt      time.Time `json:"created_at"`
}

func (CrossEvent) TableName() string { return "cross_events" }

// ── Cross Feedback ────────────────────────────────────────────────────────────────

type CrossFeedback struct {
	OrderID        string  `json:"order_id" binding:"required"`
	DidBuy         bool    `json:"did_buy"`
	DidClick       bool    `json:"did_click"`
	DidClaim       bool    `json:"did_claim"`
	DidChurn       bool    `json:"did_churn"`
	ClaimCostCents float64 `json:"claim_cost_cents"`
	ClickedRecID   string  `json:"clicked_rec_id"` // which rec was clicked
}

// ── Cross Dashboard ────────────────────────────────────────────────────────────────

type CrossDashboard struct {
	Config                CrossConfig      `json:"config"`
	TotalDecisions        int64            `json:"total_decisions"`
	DecisionsBySource     map[string]int64 `json:"decisions_by_source"`
	AvgConfidence         float64          `json:"avg_confidence"`
	AttachRate            float64          `json:"attach_rate"`
	ClickRate             float64          `json:"click_rate"`
	AvgReward             float64          `json:"avg_reward"`
	EmergencyModeActive   bool             `json:"emergency_mode_active"`
	RolloutPercent        int              `json:"rollout_percent"`
	ConsistencyViolations int64            `json:"consistency_violations"`
	TopGuardrails         []GuardrailStats `json:"top_guardrails"`
}
