package geoscore

import (
	"time"

	"github.com/google/uuid"
)

// GeoScore is the persisted trust score for a user.
// Table: geo_scores
type GeoScore struct {
	UserID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	Score         float64   `gorm:"type:decimal(5,2);not null;default:0" json:"score"` // 0–100
	SuccessRate   float64   `gorm:"type:decimal(5,4);default:0" json:"success_rate"`
	DisputeRate   float64   `gorm:"type:decimal(5,4);default:0" json:"dispute_rate"`
	KYCScore      float64   `gorm:"type:decimal(5,4);default:0" json:"kyc_score"`
	DeliveryScore float64   `gorm:"type:decimal(5,4);default:0" json:"delivery_score"`
	FraudScore    float64   `gorm:"type:decimal(5,4);default:0" json:"fraud_score"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (GeoScore) TableName() string { return "geo_scores" }

// BehaviorEvent captures lightweight interaction telemetry for future ML.
// Table: behavior_events
type BehaviorEvent struct {
	ID        uuid.UUID              `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID              `gorm:"type:uuid;not null;index" json:"user_id"`
	EventType string                 `gorm:"size:50;not null;index" json:"event_type"`
	Metadata  map[string]interface{} `gorm:"type:jsonb;serializer:json" json:"metadata,omitempty"`
	CreatedAt time.Time              `gorm:"index" json:"created_at"`
}

func (BehaviorEvent) TableName() string { return "behavior_events" }

// TrackReq is the payload for the TrackEvent call.
type TrackReq struct {
	UserID    string                 `json:"user_id"`
	EventType string                 `json:"event_type"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
