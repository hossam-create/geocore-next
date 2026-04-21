package waitlist

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Status represents the lifecycle state of a waitlist entry.
type Status string

const (
	StatusWaiting Status = "waiting"
	StatusInvited Status = "invited"
	StatusJoined  Status = "joined"
	StatusFlagged Status = "flagged"
)

// WaitlistUser is a single position in the pre-launch queue.
type WaitlistUser struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email            string    `gorm:"size:255;uniqueIndex;not null"                  json:"email"`
	Position         int       `gorm:"not null"                                       json:"position"`
	PreviousPosition int       `gorm:"default:0"                                      json:"-"` // for moved_today calc
	ReferralCode     string    `gorm:"size:16;uniqueIndex;not null"                   json:"referral_code"`
	ReferredBy       *string   `gorm:"size:16;index"                                  json:"referred_by,omitempty"`
	ReferralCount    int       `gorm:"default:0"                                      json:"referral_count"`
	PriorityScore    float64   `gorm:"default:0"                                      json:"priority_score"`
	Status           Status    `gorm:"size:16;default:'waiting';index"                json:"status"`
	IPAddress        string    `gorm:"size:64;index"                                  json:"-"`
	DeviceID         string    `gorm:"size:128;index"                                 json:"-"` // anti-gaming fingerprint
	CreatedAt        time.Time `json:"created_at"`
}

// OnboardingState tracks conversion progress after a user is invited.
type OnboardingState struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserEmail      string    `gorm:"size:255;uniqueIndex;not null"                  json:"user_email"`
	Step           int       `gorm:"not null;default:1"                             json:"step"`
	CompletedSteps int       `gorm:"not null;default:0"                             json:"completed_steps"`
	IsComplete     bool      `gorm:"default:false"                                  json:"is_complete"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (OnboardingState) TableName() string { return "waitlist_onboarding" }

// AutoMigrate creates / updates waitlist tables.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&WaitlistUser{}, &WaitlistConfig{}, &OnboardingState{})
}

// GenerateReferralCode returns a random 8-character uppercase hex code.
func GenerateReferralCode() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}
