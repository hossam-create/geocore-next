package reputation

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PenaltyReason defines why a penalty was applied.
type PenaltyReason string

const (
	PenaltyCancelAfterAccept PenaltyReason = "cancel_after_accept" // -10
	PenaltyDisputeLost       PenaltyReason = "dispute_lost"        // -20
	PenaltyFraudFlagged      PenaltyReason = "fraud_flagged"       // -40
	BonusSuccessfulDelivery  PenaltyReason = "successful_delivery" // +5
	BonusGoodReview          PenaltyReason = "good_review"         // +3
	BonusOnTimeDelivery      PenaltyReason = "on_time_delivery"    // +2
)

var penaltyValues = map[PenaltyReason]float64{
	PenaltyCancelAfterAccept: -10,
	PenaltyDisputeLost:       -20,
	PenaltyFraudFlagged:      -40,
	BonusSuccessfulDelivery:  5,
	BonusGoodReview:          3,
	BonusOnTimeDelivery:      2,
}

// PenaltyLog records all reputation changes for audit.
type PenaltyLog struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID     `gorm:"type:uuid;not null;index" json:"user_id"`
	Role      string        `gorm:"size:20;not null" json:"role"`
	Reason    PenaltyReason `gorm:"size:50;not null" json:"reason"`
	Delta     float64       `gorm:"type:numeric(6,2);not null" json:"delta"`
	NewScore  float64       `gorm:"type:numeric(5,2);not null" json:"new_score"`
	CreatedAt time.Time     `json:"created_at"`
}

func (PenaltyLog) TableName() string { return "penalty_logs" }

// ApplyPenalty applies a reputation penalty or bonus and logs it.
func ApplyPenalty(db *gorm.DB, notifSvc *notifications.Service, userID uuid.UUID, role string, reason PenaltyReason) error {
	delta, ok := penaltyValues[reason]
	if !ok {
		return fmt.Errorf("unknown penalty reason: %s", reason)
	}

	var newScore float64
	err := db.Transaction(func(tx *gorm.DB) error {
		var rep UserReputation
		if err := tx.Where("user_id=? AND role=?", userID, role).First(&rep).Error; err != nil {
			rep = UserReputation{UserID: userID, Role: role, Score: 50, AvgRating: 3.0}
		}

		rep.Score = clampScore(rep.Score + delta)
		rep.LastUpdated = time.Now()
		if err := tx.Save(&rep).Error; err != nil {
			return err
		}
		newScore = rep.Score

		log := PenaltyLog{
			UserID:   userID,
			Role:     role,
			Reason:   reason,
			Delta:    delta,
			NewScore: newScore,
		}
		return tx.Create(&log).Error
	})

	if err != nil {
		return err
	}

	// Notify user of reputation change
	if notifSvc != nil {
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: userID,
			Type:   "reputation_change",
			Title:  "Reputation Updated",
			Body:   fmt.Sprintf("Your reputation %s (%s). New score: %.0f", FormatScoreDelta(delta), string(reason), newScore),
			Data:   map[string]string{"delta": FormatScoreDelta(delta), "reason": string(reason), "new_score": fmt.Sprintf("%.0f", newScore)},
		})
	}

	slog.Info("reputation: penalty applied", "user_id", userID, "role", role, "reason", reason, "delta", delta, "new_score", newScore)
	return nil
}

// GetPenaltyHistory returns recent penalty logs for a user.
func GetPenaltyHistory(db *gorm.DB, userID uuid.UUID, limit int) []PenaltyLog {
	var logs []PenaltyLog
	db.Where("user_id=?", userID).Order("created_at DESC").Limit(limit).Find(&logs)
	return logs
}

func clampScore(s float64) float64 {
	if s < 0 {
		return 0
	}
	if s > 100 {
		return 100
	}
	return s
}
