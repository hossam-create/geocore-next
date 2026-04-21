package fraud

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository provides persistence for fraud feedback and feature snapshots.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a fraud repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// SaveFeedback persists a fraud feedback record.
func (r *Repository) SaveFeedback(f *FraudFeedback) error {
	return r.db.Create(f).Error
}

// GetFeedbackByEvent retrieves feedback for a specific event.
func (r *Repository) GetFeedbackByEvent(eventID uuid.UUID) ([]FraudFeedback, error) {
	var feedback []FraudFeedback
	err := r.db.Where("event_id = ?", eventID).Find(&feedback).Error
	return feedback, err
}

// GetUserFraudRate calculates the fraud rate for a user based on feedback.
// Returns (total_decisions, fraud_count, fraud_rate).
func (r *Repository) GetUserFraudRate(userID uuid.UUID, since time.Duration) (total, fraudCount int64, rate float64) {
	cutoff := time.Now().Add(-since)
	r.db.Model(&FraudFeedback{}).
		Where("user_id = ? AND created_at > ?", userID, cutoff).
		Count(&total)

	r.db.Model(&FraudFeedback{}).
		Where("user_id = ? AND outcome = ? AND created_at > ?", userID, "FRAUD", cutoff).
		Count(&fraudCount)

	if total > 0 {
		rate = float64(fraudCount) / float64(total)
	}
	return
}

// GetDecisionAccuracy calculates overall fraud decision accuracy.
// Returns (total, correct_count, accuracy_rate).
func (r *Repository) GetDecisionAccuracy(since time.Duration) (total, correctCount int64, accuracy float64) {
	cutoff := time.Now().Add(-since)
	r.db.Model(&FraudFeedback{}).
		Where("created_at > ?", cutoff).
		Count(&total)

	// Correct = BLOCK+FRAUD or ALLOW+LEGIT
	r.db.Model(&FraudFeedback{}).
		Where("created_at > ? AND ((decision = 'BLOCK' AND outcome = 'FRAUD') OR (decision = 'ALLOW' AND outcome = 'LEGIT'))", cutoff).
		Count(&correctCount)

	if total > 0 {
		accuracy = float64(correctCount) / float64(total)
	}
	return
}

// GetRecentDecisions returns the last N decisions for a user (for learning).
func (r *Repository) GetRecentDecisions(userID uuid.UUID, limit int) ([]FraudFeedback, error) {
	var decisions []FraudFeedback
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&decisions).Error
	return decisions, err
}
