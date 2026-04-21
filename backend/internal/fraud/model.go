package fraud

import (
	"time"

	"github.com/google/uuid"
)

// ── Enums ───────────────────────────────────────────────────────────────────

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type AlertStatus string

const (
	AlertPending      AlertStatus = "pending"
	AlertInvestigating AlertStatus = "investigating"
	AlertConfirmed    AlertStatus = "confirmed"
	AlertFalsePositive AlertStatus = "false_positive"
	AlertResolved     AlertStatus = "resolved"
)

type TargetType string

const (
	TargetUser        TargetType = "user"
	TargetOrder       TargetType = "order"
	TargetTransaction TargetType = "transaction"
	TargetListing     TargetType = "listing"
	TargetReview      TargetType = "review"
)

// ── Models ──────────────────────────────────────────────────────────────────

type FraudAlert struct {
	ID         uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TargetType TargetType  `gorm:"size:50;not null"  json:"target_type"`
	TargetID   uuid.UUID   `gorm:"type:uuid;not null" json:"target_id"`
	AlertType  string      `gorm:"size:100;not null" json:"alert_type"`
	Severity   Severity    `gorm:"size:20;not null;default:'medium'" json:"severity"`
	RiskScore  float64     `gorm:"type:numeric(5,2);default:0" json:"risk_score"`
	DetectedBy string      `gorm:"size:100;default:'rule_engine'" json:"detected_by"`
	Confidence float64     `gorm:"type:numeric(4,3);default:0" json:"confidence"`
	Indicators string      `gorm:"type:jsonb;default:'[]'" json:"indicators"`
	RawData    *string     `gorm:"type:jsonb" json:"raw_data,omitempty"`
	Status     AlertStatus `gorm:"size:50;not null;default:'pending'" json:"status"`
	ReviewedBy *uuid.UUID  `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time  `json:"reviewed_at,omitempty"`
	Resolution string      `gorm:"type:text" json:"resolution,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

func (FraudAlert) TableName() string { return "fraud_alerts" }

type FraudRule struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name        string    `gorm:"size:200;uniqueIndex" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	RuleType    string    `gorm:"size:50;not null" json:"rule_type"`
	Conditions  string    `gorm:"type:jsonb;default:'{}'" json:"conditions"`
	Severity    Severity  `gorm:"size:20;not null;default:'medium'" json:"severity"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (FraudRule) TableName() string { return "fraud_rules" }

type UserRiskProfile struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	RiskScore     float64   `gorm:"type:numeric(5,2);default:0" json:"risk_score"`
	RiskLevel     string    `gorm:"size:20;default:'low'" json:"risk_level"`
	TotalOrders   int       `gorm:"default:0" json:"total_orders"`
	TotalSpent    float64   `gorm:"type:numeric(14,2);default:0" json:"total_spent"`
	AvgOrderValue float64   `gorm:"type:numeric(12,2);default:0" json:"avg_order_value"`
	Flags         string    `gorm:"type:jsonb;default:'[]'" json:"flags"`
	LastAssessed  time.Time `json:"last_assessed"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (UserRiskProfile) TableName() string { return "user_risk_profiles" }
