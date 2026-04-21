package referral

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReferralStatus represents the state of a referral
type ReferralStatus string

const (
	StatusPending   ReferralStatus = "pending"
	StatusCompleted ReferralStatus = "completed"
	StatusExpired   ReferralStatus = "expired"
)

// DefaultRewardPoints is the number of loyalty points awarded to the referrer
const DefaultRewardPoints = 100

// Referral tracks a single referral relationship between two users
type Referral struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ReferrerID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"referrer_id"`
	RefereeID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"referee_id"`
	Code         string         `gorm:"size:16;not null;index" json:"code"`
	Status       ReferralStatus `gorm:"size:20;not null;default:'pending'" json:"status"`
	RewardPoints int            `gorm:"not null;default:100" json:"reward_points"`
	RewardPaidAt *time.Time     `json:"reward_paid_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Referral) TableName() string { return "referrals" }

func (r *Referral) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// GenerateCode produces a short unique code from a user UUID
func GenerateCode(userID uuid.UUID) string {
	raw := strings.ReplaceAll(userID.String(), "-", "")
	return strings.ToUpper(raw[:8])
}

// ReferralStats is a summary returned from the stats endpoint
type ReferralStats struct {
	Code          string `json:"code"`
	ShareURL      string `json:"share_url"`
	TotalReferrals int   `json:"total_referrals"`
	Pending       int    `json:"pending"`
	Completed     int    `json:"completed"`
	TotalEarned   int    `json:"total_earned_points"`
}
