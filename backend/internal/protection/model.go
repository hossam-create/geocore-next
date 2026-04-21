package protection

import (
	"time"

	"github.com/google/uuid"
)

// ── Order Protection (superset of insurance) ────────────────────────────────────

type OrderProtection struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	OrderID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"order_id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	PriceCents      int64     `gorm:"not null;default:0" json:"price_cents"`
	HasCancellation bool      `gorm:"not null;default:false" json:"has_cancellation"`
	HasDelay        bool      `gorm:"not null;default:false" json:"has_delay"`
	HasFull         bool      `gorm:"not null;default:false" json:"has_full"`
	CoveragePercent float64   `gorm:"type:numeric(5,2);not null;default:100" json:"coverage_percent"`
	RiskFactor      float64   `gorm:"type:numeric(5,4);not null;default:0" json:"risk_factor"`
	UrgencyFactor   float64   `gorm:"type:numeric(5,4);not null;default:0" json:"urgency_factor"`
	IsUsed          bool      `gorm:"not null;default:false" json:"is_used"`
	FirstOrderFree  bool      `gorm:"not null;default:false" json:"first_order_free"`
	ABVariant       string    `gorm:"size:10;not null;default:'control'" json:"ab_variant"`
	CreatedAt       time.Time `json:"created_at"`
}

func (OrderProtection) TableName() string { return "order_protections" }

// ── Guarantee Claim ────────────────────────────────────────────────────────────

type ClaimType string

const (
	ClaimNoShow   ClaimType = "no_show"
	ClaimDelay    ClaimType = "delay"
	ClaimMismatch ClaimType = "mismatch"
)

type ClaimStatus string

const (
	ClaimPending      ClaimStatus = "pending"
	ClaimAutoApproved ClaimStatus = "auto_approved"
	ClaimApproved     ClaimStatus = "approved"
	ClaimRejected     ClaimStatus = "rejected"
)

type GuaranteeClaim struct {
	ID                uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	OrderID           uuid.UUID   `gorm:"type:uuid;not null;index" json:"order_id"`
	UserID            uuid.UUID   `gorm:"type:uuid;not null;index" json:"user_id"`
	TravelerID        uuid.UUID   `gorm:"type:uuid;not null" json:"traveler_id"`
	Type              ClaimType   `gorm:"size:20;not null" json:"type"`
	EvidenceJSON      string      `gorm:"type:jsonb;not null;default:'{}'" json:"evidence_json"`
	Status            ClaimStatus `gorm:"size:20;not null;default:'pending';index" json:"status"`
	RefundCents       int64       `gorm:"not null;default:0" json:"refund_cents"`
	CompensationCents int64       `gorm:"not null;default:0" json:"compensation_cents"`
	TravelerPenalty   bool        `gorm:"not null;default:false" json:"traveler_penalty"`
	AutoEvaluated     bool        `gorm:"not null;default:false" json:"auto_evaluated"`
	ReviewerID        *uuid.UUID  `gorm:"type:uuid" json:"reviewer_id,omitempty"`
	ResolvedAt        *time.Time  `json:"resolved_at,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
}

func (GuaranteeClaim) TableName() string { return "guarantee_claims" }

// ── A/B Variant Assignment ──────────────────────────────────────────────────────

type ABVariantAssignment struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_exp,2" json:"user_id"`
	Experiment string    `gorm:"size:50;not null;uniqueIndex:idx_user_exp,2" json:"experiment"`
	Variant    string    `gorm:"size:20;not null" json:"variant"`
	AssignedAt time.Time `gorm:"not null;default:NOW()" json:"assigned_at"`
}

func (ABVariantAssignment) TableName() string { return "ab_variant_assignments" }

// ── A/B Event ────────────────────────────────────────────────────────────────────

type ABEvent struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Experiment string     `gorm:"size:50;not null" json:"experiment"`
	Variant    string     `gorm:"size:20;not null" json:"variant"`
	EventType  string     `gorm:"size:50;not null" json:"event_type"`
	OrderID    *uuid.UUID `gorm:"type:uuid" json:"order_id,omitempty"`
	Metadata   string     `gorm:"type:jsonb;not null;default:'{}'" json:"metadata"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (ABEvent) TableName() string { return "ab_events" }

// ── Daily Metrics ────────────────────────────────────────────────────────────────

type ProtectionDailyMetrics struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Date               time.Time `gorm:"type:date;not null;uniqueIndex" json:"date"`
	TotalOrders        int64     `gorm:"not null;default:0" json:"total_orders"`
	ProtectionAttached int64     `gorm:"not null;default:0" json:"protection_attached"`
	AttachRate         float64   `gorm:"type:numeric(5,4);not null;default:0" json:"attach_rate"`
	RevenueCents       int64     `gorm:"not null;default:0" json:"revenue_cents"`
	ClaimsFiled        int64     `gorm:"not null;default:0" json:"claims_filed"`
	ClaimsApproved     int64     `gorm:"not null;default:0" json:"claims_approved"`
	PayoutsCents       int64     `gorm:"not null;default:0" json:"payouts_cents"`
	NetRevenueCents    int64     `gorm:"not null;default:0" json:"net_revenue_cents"`
	AvgRiskFactor      float64   `gorm:"type:numeric(5,4);not null;default:0" json:"avg_risk_factor"`
	CreatedAt          time.Time `json:"created_at"`
}

func (ProtectionDailyMetrics) TableName() string { return "protection_daily_metrics" }

// ── Protection Bundle Types ──────────────────────────────────────────────────────

type ProtectionBundle struct {
	Type            string  `json:"type"`
	Label           string  `json:"label"`
	Description     string  `json:"description"`
	PricePercent    float64 `json:"price_percent"`
	HasCancellation bool    `json:"has_cancellation"`
	HasDelay        bool    `json:"has_delay"`
	HasFull         bool    `json:"has_full"`
	CoveragePercent float64 `json:"coverage_percent"`
}

var ProtectionBundles = []ProtectionBundle{
	{
		Type:            "cancellation",
		Label:           "Cancellation Protection",
		Description:     "Cancel anytime without fees",
		PricePercent:    1.5,
		HasCancellation: true,
		CoveragePercent: 100,
	},
	{
		Type:            "delay",
		Label:           "Late Delivery Protection",
		Description:     "Get compensated if delivery is late",
		PricePercent:    1.0,
		HasDelay:        true,
		CoveragePercent: 100,
	},
	{
		Type:            "full",
		Label:           "Full Protection — best value 🔥",
		Description:     "Cancel anytime + delay coverage + priority support",
		PricePercent:    3.0,
		HasCancellation: true,
		HasDelay:        true,
		HasFull:         true,
		CoveragePercent: 100,
	},
}

// ── Claim Evaluation Result ────────────────────────────────────────────────────

type ClaimEvaluation struct {
	Decision          ClaimStatus `json:"decision"`
	RefundPercent     float64     `json:"refund_percent"`
	CompensationCents int64       `json:"compensation_cents"`
	TravelerPenalty   bool        `json:"traveler_penalty"`
	Reason            string      `json:"reason"`
	AutoEvaluated     bool        `json:"auto_evaluated"`
}

// ── Admin Metrics Summary ────────────────────────────────────────────────────────

type AdminProtectionMetrics struct {
	TotalProtected    int64       `json:"total_protected"`
	TotalRevenueCents int64       `json:"total_revenue_cents"`
	TotalPayoutsCents int64       `json:"total_payouts_cents"`
	NetRevenueCents   int64       `json:"net_revenue_cents"`
	AttachRate        float64     `json:"attach_rate"`
	ClaimsRate        float64     `json:"claims_rate"`
	ApprovalRate      float64     `json:"approval_rate"`
	AbuseRate         float64     `json:"abuse_rate"`
	TopRiskyUsers     []RiskyUser `json:"top_risky_users,omitempty"`
}

type RiskyUser struct {
	UserID      uuid.UUID `json:"user_id"`
	ClaimsCount int       `json:"claims_count"`
	AbuseScore  float64   `json:"abuse_score"`
}

// ── A/B Test Results ────────────────────────────────────────────────────────────

type ABTestResults struct {
	Experiment string             `json:"experiment"`
	Variants   []ABVariantMetrics `json:"variants"`
	Winner     string             `json:"winner,omitempty"`
}

type ABVariantMetrics struct {
	Variant            string  `json:"variant"`
	Users              int64   `json:"users"`
	AttachRate         float64 `json:"attach_rate"`
	ConversionRate     float64 `json:"conversion_rate"`
	AvgRevenuePerOrder float64 `json:"avg_revenue_per_order"`
	CancellationRate   float64 `json:"cancellation_rate"`
}
