package fraud

import (
	"time"

	"github.com/google/uuid"
)

// FraudFeedback records the actual outcome of a fraud decision.
// This closes the feedback loop so the system can self-improve.
type FraudFeedback struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	EventID   uuid.UUID `gorm:"type:uuid;index;not null" json:"event_id"`
	UserID    uuid.UUID `gorm:"type:uuid;index;not null" json:"user_id"`
	Decision  string    `gorm:"size:20;not null" json:"decision"`  // ALLOW / BLOCK / CHALLENGE
	Outcome   string    `gorm:"size:20;not null" json:"outcome"`  // LEGIT / FRAUD
	Notes     string    `gorm:"type:text" json:"notes,omitempty"`
	ReviewedBy uuid.UUID `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (FraudFeedback) TableName() string { return "fraud_feedback" }

// NewFeedback creates a feedback record for a fraud decision.
func NewFeedback(eventID, userID uuid.UUID, decision, outcome string) *FraudFeedback {
	return &FraudFeedback{
		ID:        uuid.New(),
		EventID:   eventID,
		UserID:    userID,
		Decision:  decision,
		Outcome:   outcome,
		CreatedAt: time.Now(),
	}
}

// IsCorrect returns true if the decision matched the actual outcome.
func (f *FraudFeedback) IsCorrect() bool {
	switch {
	case f.Decision == string(DecisionBlock) && f.Outcome == "FRAUD":
		return true
	case f.Decision == string(DecisionAllow) && f.Outcome == "LEGIT":
		return true
	case f.Decision == string(DecisionChallenge) && f.Outcome == "FRAUD":
		return true // challenge caught it
	default:
		return false
	}
}
