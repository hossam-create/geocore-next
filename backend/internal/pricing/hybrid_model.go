package pricing

import (
	"time"

	"github.com/google/uuid"
)

// ── Hybrid Pricing Engine Models ─────────────────────────────────────────────────
//
// The hybrid engine orchestrates: Guardrails → RL → Bandit fallback → Clamp
// Source tracking: every decision records which engine made it and why.

// PricingSource identifies which engine produced the final price.
type PricingSource string

const (
	SourceRules     PricingSource = "rules"     // Hard guardrails override
	SourceRL        PricingSource = "rl"        // RL model (high confidence)
	SourceBandit    PricingSource = "bandit"    // Bandit fallback (low RL confidence)
	SourceBlend     PricingSource = "blend"     // Soft blend of RL + Bandit
	SourceEmergency PricingSource = "emergency" // Emergency/static fallback
	SourceSession   PricingSource = "session"   // Session stickiness
	SourceShadow    PricingSource = "shadow"    // Shadow mode (log only)
)

// HybridDecision is the final output of the hybrid pricing engine.
type HybridDecision struct {
	PriceCents     int64         `json:"price_cents"`
	PricePercent   float64       `json:"price_percent"`
	AnchorPrice    int64         `json:"anchor_price"`
	Source         PricingSource `json:"source"`          // which engine decided
	Confidence     float64       `json:"confidence"`      // overall confidence
	IsExploration  bool          `json:"is_exploration"`   // was this an exploration pick
	IsShadow       bool          `json:"is_shadow"`        // shadow mode (don't execute)

	// ── Sub-decision details (observability) ────────────────────────────────
	RLOutput       *RLSubDecision   `json:"rl_output,omitempty"`
	BanditOutput   *BanditSubDecision `json:"bandit_output,omitempty"`
	RulesOutput    *RulesSubDecision  `json:"rules_output,omitempty"`

	// ── UX ─────────────────────────────────────────────────────────────────
	UXVariant      string        `json:"ux_variant"`
	RecommendedLabel string      `json:"recommended_label"`
	SessionStep     int           `json:"session_step"`

	// ── Guardrails applied ──────────────────────────────────────────────────
	GuardrailsApplied []string   `json:"guardrails_applied"` // list of guardrails that fired
	Clamped          bool        `json:"clamped"`            // was price clamped
}

// RLSubDecision captures the RL engine's intermediate output.
type RLSubDecision struct {
	PriceCents    int64   `json:"price_cents"`
	PricePercent  float64 `json:"price_percent"`
	Confidence    float64 `json:"confidence"`
	QValue        float64 `json:"q_value"`
	StateKey      string  `json:"state_key"`
	ActionIndex   int     `json:"action_index"`
	UXVariant     string  `json:"ux_variant"`
}

// BanditSubDecision captures the bandit engine's intermediate output.
type BanditSubDecision struct {
	PriceCents    int64   `json:"price_cents"`
	PricePercent  float64 `json:"price_percent"`
	SampleValue   float64 `json:"sample_value"`
	Segment       string  `json:"segment"`
	ArmID         uuid.UUID `json:"arm_id"`
}

// RulesSubDecision captures the rule engine's intermediate output.
type RulesSubDecision struct {
	PriceCents    int64   `json:"price_cents"`
	PricePercent  float64 `json:"price_percent"`
	RiskAdj       int64   `json:"risk_adj_cents"`
	BehaviorAdj   int64   `json:"behavior_adj_cents"`
	ContextAdj    int64   `json:"context_adj_cents"`
	RuleName      string  `json:"rule_name"` // which rule fired
}

// ── Hybrid Config ────────────────────────────────────────────────────────────────

type HybridConfig struct {
	ID                        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	RLConfidenceThreshold     float64   `gorm:"type:numeric(5,2);not null;default:0.6" json:"rl_confidence_threshold"` // below → bandit fallback
	BlendWeightRL             float64   `gorm:"type:numeric(5,2);not null;default:0.7" json:"blend_weight_rl"` // 0.7 = 70% RL, 30% bandit
	EnableSoftBlend           bool      `gorm:"not null;default:false" json:"enable_soft_blend"` // soft blend vs hard switch
	MinPricePercent           float64   `gorm:"type:numeric(5,2);not null;default:1" json:"min_price_percent"`
	MaxPricePercent           float64   `gorm:"type:numeric(5,2);not null;default:4" json:"max_price_percent"`
	EmergencyModeActive       bool      `gorm:"not null;default:false" json:"emergency_mode_active"`
	EmergencyPricePercent     float64   `gorm:"type:numeric(5,2);not null;default:2" json:"emergency_price_percent"`
	ConversionDropThreshold   float64   `gorm:"type:numeric(5,2);not null;default:0.08" json:"conversion_drop_threshold"`
	SessionCooldownMinutes    int       `gorm:"not null;default:5" json:"session_cooldown_minutes"`
	MaxSessionSteps           int       `gorm:"not null;default:3" json:"max_session_steps"`
	AnomalyDetectionEnabled   bool      `gorm:"not null;default:true" json:"anomaly_detection_enabled"`
	RolloutPercent            int       `gorm:"not null;default:5" json:"rollout_percent"` // what % of traffic uses hybrid
	IsActive                  bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt                 time.Time `json:"created_at"`
	UpdatedAt                 time.Time `json:"updated_at"`
}

func (HybridConfig) TableName() string { return "hybrid_configs" }

// ── Hybrid Event (observability) ────────────────────────────────────────────────

type HybridEvent struct {
	ID              uuid.UUID    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID    `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderID         uuid.UUID    `gorm:"type:uuid;not null" json:"order_id"`
	Source          string       `gorm:"size:20;not null;index" json:"source"` // rules/rl/bandit/blend/emergency
	PriceCents      int64        `gorm:"not null" json:"price_cents"`
	PricePercent    float64      `gorm:"type:numeric(5,2);not null" json:"price_percent"`
	Confidence      float64      `gorm:"type:numeric(5,4);not null" json:"confidence"`
	IsExploration   bool         `gorm:"not null;default:false" json:"is_exploration"`
	IsShadow        bool         `gorm:"not null;default:false" json:"is_shadow"`
	UXVariant       string       `gorm:"size:30;not null;default:'standard'" json:"ux_variant"`
	GuardrailsJSON  string       `gorm:"type:text" json:"guardrails_json"` // JSON array of applied guardrails
	DidBuy          bool         `gorm:"not null;default:false" json:"did_buy"`
	Reward          float64      `gorm:"type:numeric(12,2);not null;default:0" json:"reward"`
	CreatedAt       time.Time    `json:"created_at"`
}

func (HybridEvent) TableName() string { return "hybrid_events" }

// ── Hybrid Feedback ──────────────────────────────────────────────────────────────

type HybridFeedback struct {
	OrderID        string  `json:"order_id" binding:"required"`
	DidBuy         bool    `json:"did_buy"`
	DidClaim       bool    `json:"did_claim"`
	DidChurn       bool    `json:"did_churn"`
	ClaimCostCents float64 `json:"claim_cost_cents"`
}

// ── Hybrid Dashboard ──────────────────────────────────────────────────────────────

type HybridDashboard struct {
	Config              HybridConfig       `json:"config"`
	TotalDecisions      int64              `json:"total_decisions"`
	DecisionsBySource   map[string]int64   `json:"decisions_by_source"`
	AvgConfidence       float64            `json:"avg_confidence"`
	AttachRate          float64            `json:"attach_rate"`
	TotalRevenue        float64            `json:"total_revenue"`
	EmergencyModeActive bool               `json:"emergency_mode_active"`
	RolloutPercent      int                `json:"rollout_percent"`
	ConversionTrend     string             `json:"conversion_trend"`
	TopGuardrails       []GuardrailStats   `json:"top_guardrails"`
}

type GuardrailStats struct {
	Name    string `json:"name"`
	Count   int64  `json:"count"`
	Percent float64 `json:"percent"` // % of decisions where this guardrail fired
}
