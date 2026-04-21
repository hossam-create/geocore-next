package fraud

import (
	"time"

	"github.com/google/uuid"
)

// UserRiskSnapshot stores one pre-action risk evaluation. Immutable audit
// trail used to inspect why a user was throttled / blocked at any moment.
type UserRiskSnapshot struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index"  json:"user_id"`
	Score     int            `gorm:"not null;index"            json:"score"`
	Decision  string         `gorm:"size:20;not null"          json:"decision"`
	Factors   map[string]any `gorm:"type:jsonb;serializer:json;default:'{}'" json:"factors"`
	CreatedAt time.Time      `gorm:"index"                     json:"created_at"`
}

func (UserRiskSnapshot) TableName() string { return "user_risk_snapshots" }
