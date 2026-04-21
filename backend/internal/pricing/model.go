package pricing

import (
	"time"

	"github.com/google/uuid"
)

// ── Pricing Context (all features for a pricing decision) ──────────────────────

type PricingContext struct {
	UserID              uuid.UUID `json:"user_id"`
	OrderID             uuid.UUID `json:"order_id"`
	OrderPriceCents     int64     `json:"order_price_cents"`
	Category            string    `json:"category"`
	DeliveryType        string    `json:"delivery_type"`

	// ── User Features ────────────────────────────────────────────────────────
	TrustScore          float64   `json:"trust_score"`           // 0-100
	CancellationRate    float64   `json:"cancellation_rate"`     // 0-1
	PastInsuranceUsage  int       `json:"past_insurance_usage"`  // count of past purchases
	AvgOrderValue       float64   `json:"avg_order_value"`       // in cents
	AbuseFlags          int       `json:"abuse_flags"`           // count of abuse signals
	AccountAgeDays      float64   `json:"account_age_days"`      // days since registration

	// ── Order Features ──────────────────────────────────────────────────────
	TravelerRating      float64   `json:"traveler_rating"`       // 0-5
	DeliveryRiskScore   float64   `json:"delivery_risk_score"`   // 0-1 (delay probability)
	RouteRisk           float64   `json:"route_risk"`            // 0-1

	// ── Context Features ────────────────────────────────────────────────────
	TimeOfDay           int       `json:"time_of_day"`           // 0-23 hour
	IsRushHour          bool      `json:"is_rush_hour"`
	LiveDemand          float64   `json:"live_demand"`           // 0-1 (traffic intensity)
	UrgencyScore        float64   `json:"urgency_score"`         // 0-1

	// ── Behavior Features ───────────────────────────────────────────────────
	InsuranceBuyRate    float64   `json:"insurance_buy_rate"`    // 0-1 (how often they buy)
	LastInsurancePrice  int64     `json:"last_insurance_price"`  // cents
	PriceSensitivity    float64   `json:"price_sensitivity"`     // 0-1 (1 = very sensitive)
}

// ── Price Result ────────────────────────────────────────────────────────────────

type PriceResult struct {
	PriceCents      int64             `json:"price_cents"`
	BasePriceCents  int64             `json:"base_price_cents"`
	PricePercent    float64           `json:"price_percent"`
	Adjustments     PriceAdjustments  `json:"adjustments"`
	BuyProbability  float64           `json:"buy_probability"`
	Confidence      float64           `json:"confidence"`
	Strategy        PricingStrategy   `json:"strategy"`
	AnchorPrice     int64             `json:"anchor_price,omitempty"` // "was X" for anchoring
	Features        FeatureVector     `json:"features,omitempty"`
}

type PriceAdjustments struct {
	RiskAdj      int64   `json:"risk_adj_cents"`
	BehaviorAdj  int64   `json:"behavior_adj_cents"`
	ContextAdj   int64   `json:"context_adj_cents"`
	AIAdj        int64   `json:"ai_adj_cents,omitempty"`
}

// ── Pricing Strategy ────────────────────────────────────────────────────────────

type PricingStrategy string

const (
	StrategyStatic PricingStrategy = "static"  // fixed 2%
	StrategyRules  PricingStrategy = "rules"   // rule-based dynamic
	StrategyAI     PricingStrategy = "ai"      // ML model prediction
)

// ── Feature Vector (for ML model input) ──────────────────────────────────────────

type FeatureVector struct {
	Features []float64 `json:"features"`
	Names    []string  `json:"names"`
}

// BuildFeatureVector extracts the ordered feature vector from PricingContext.
func (ctx *PricingContext) BuildFeatureVector() FeatureVector {
	return FeatureVector{
		Features: []float64{
			float64(ctx.OrderPriceCents) / 100.0, // order_value
			ctx.TrustScore / 100.0,               // normalized trust
			ctx.CancellationRate,                  // cancel_rate
			float64(ctx.PastInsuranceUsage),       // insurance_history
			float64(ctx.AbuseFlags),               // abuse_count
			ctx.TravelerRating / 5.0,              // normalized rating
			ctx.DeliveryRiskScore,                 // delay_prob
			ctx.RouteRisk,                         // route_risk
			float64(ctx.TimeOfDay) / 23.0,        // time_normalized
			ctx.UrgencyScore,                      // urgency
			ctx.InsuranceBuyRate,                  // buy_rate
			ctx.PriceSensitivity,                  // sensitivity
			float64(ctx.AccountAgeDays) / 365.0,   // account_age_norm
			ctx.LiveDemand,                        // demand
		},
		Names: []string{
			"order_value", "trust_score", "cancel_rate", "insurance_history",
			"abuse_count", "traveler_rating", "delay_prob", "route_risk",
			"time_of_day", "urgency", "buy_rate", "price_sensitivity",
			"account_age", "live_demand",
		},
	}
}

// ── Pricing Model Config ────────────────────────────────────────────────────────

type PricingModelConfig struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Version           string    `gorm:"size:20;not null" json:"version"`
	Strategy          string    `gorm:"size:20;not null;default:'rules'" json:"strategy"`
	MinPricePercent   float64   `gorm:"type:numeric(5,2);not null;default:1" json:"min_price_percent"`
	MaxPricePercent   float64   `gorm:"type:numeric(5,2);not null;default:4" json:"max_price_percent"`
	BasePricePercent  float64   `gorm:"type:numeric(5,2);not null;default:1.5" json:"base_price_percent"`
	StaticPricePercent float64  `gorm:"type:numeric(5,2);not null;default:2" json:"static_price_percent"`
	ConfidenceThreshold float64 `gorm:"type:numeric(5,2);not null;default:0.7" json:"confidence_threshold"`
	ModelJSON         string    `gorm:"type:text" json:"model_json,omitempty"`
	IsActive          bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (PricingModelConfig) TableName() string { return "pricing_model_configs" }

// ── Pricing Event (for training + tracking) ──────────────────────────────────────

type PricingEvent struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderID         uuid.UUID       `gorm:"type:uuid;not null" json:"order_id"`
	Strategy        PricingStrategy `gorm:"size:20;not null" json:"strategy"`
	PriceCents      int64           `gorm:"not null" json:"price_cents"`
	BuyProbability  float64         `gorm:"type:numeric(5,4)" json:"buy_probability"`
	Confidence      float64         `gorm:"type:numeric(5,4)" json:"confidence"`
	DidBuy          bool            `gorm:"not null;default:false" json:"did_buy"`
	DidCancel       bool            `gorm:"not null;default:false" json:"did_cancel"`
	ClaimFiled      bool            `gorm:"not null;default:false" json:"claim_filed"`
	ABVariant       string          `gorm:"size:20;not null;default:'control'" json:"ab_variant"`
	CreatedAt       time.Time       `json:"created_at"`
}

func (PricingEvent) TableName() string { return "pricing_events" }

// ── Pricing AB Assignment ────────────────────────────────────────────────────────

type PricingABAssignment struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_pricing_ab_user,2" json:"user_id"`
	Experiment string    `gorm:"size:50;not null;uniqueIndex:idx_pricing_ab_user,2" json:"experiment"`
	Variant    string    `gorm:"size:20;not null" json:"variant"` // static | rules | ai
	AssignedAt time.Time `gorm:"not null;default:NOW()" json:"assigned_at"`
}

func (PricingABAssignment) TableName() string { return "pricing_ab_assignments" }

// ── Admin Pricing Metrics ────────────────────────────────────────────────────────

type AdminPricingMetrics struct {
	TotalPriced        int64              `json:"total_priced"`
	TotalRevenueCents  int64              `json:"total_revenue_cents"`
	AvgPriceCents      int64              `json:"avg_price_cents"`
	AttachRate         float64            `json:"attach_rate"`
	AttachRateByStrategy map[string]float64 `json:"attach_rate_by_strategy"`
	RevenueByStrategy  map[string]int64   `json:"revenue_by_strategy"`
	AvgConfidence      float64            `json:"avg_confidence"`
	ModelUsage         map[string]int64   `json:"model_usage"` // strategy -> count
}
