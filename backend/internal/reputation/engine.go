package reputation

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TrustLevel constants
const (
	TrustLow    = "low"    // score < 40
	TrustNormal = "normal" // 40–70
	TrustHigh   = "high"   // > 70
)

// UserReputation stores per-role reputation data.
type UserReputation struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_role" json:"user_id"`
	Role            string    `gorm:"size:20;not null;uniqueIndex:idx_user_role" json:"role"` // seller, traveler, buyer
	Score           float64   `gorm:"type:numeric(5,2);not null;default:50" json:"score"`
	TotalOrders     int       `gorm:"default:0" json:"total_orders"`
	CompletedOrders int       `gorm:"default:0" json:"completed_orders"`
	CancelledOrders int       `gorm:"default:0" json:"cancelled_orders"`
	DisputeCount    int       `gorm:"default:0" json:"dispute_count"`
	AvgRating       float64   `gorm:"type:numeric(3,2);default:3.0" json:"avg_rating"`
	LastUpdated     time.Time `json:"last_updated"`
}

func (UserReputation) TableName() string { return "user_reputations" }

// ComputeScore calculates reputation score.
// Formula: completion_rate*40 + avg_rating*30 + (1-dispute_rate)*20 + activity*10
// Weighted by log(totalOrders+1) to prevent manipulation via fake orders.
func ComputeScore(totalOrders, completedOrders, cancelledOrders, disputeCount int, avgRating float64) float64 {
	completionRate := 1.0
	if totalOrders > 0 {
		completionRate = float64(completedOrders) / float64(totalOrders)
	}

	disputeRate := 0.0
	if totalOrders > 0 {
		disputeRate = float64(disputeCount) / float64(totalOrders)
	}

	// Activity score: more completed orders = higher, capped at 1.0
	activity := math.Min(1.0, float64(completedOrders)/20.0)

	// Normalize rating from 1-5 scale to 0-1
	ratingNorm := math.Max(0, math.Min(1, (avgRating-1)/4))

	baseScore := completionRate*40 + ratingNorm*30 + (1-math.Min(1, disputeRate*5))*20 + activity*10

	// Weight by log(totalOrders+1) to prevent manipulation via fake orders.
	// New user with 0 orders: weight=0 → score capped at 50 (neutral).
	// 5 orders: weight≈0.7, 20 orders: weight≈1.0, 100 orders: weight≈1.4
	weight := math.Log(float64(totalOrders)+1) / math.Log(21) // normalized so 20 orders = 1.0
	if weight < 0 {
		weight = 0
	}

	// Weighted: low order count pulls score toward neutral (50)
	score := 50 + (baseScore-50)*math.Min(weight, 1.5)
	return math.Max(0, math.Min(100, score))
}

// GetTrustLevel returns the trust level label for a score.
func GetTrustLevel(score float64) string {
	if score < 40 {
		return TrustLow
	}
	if score > 70 {
		return TrustHigh
	}
	return TrustNormal
}

// GetUserScore returns the reputation score for a user in a given role.
func GetUserScore(db *gorm.DB, userID uuid.UUID, role string) UserReputation {
	var rep UserReputation
	if err := db.Where("user_id=? AND role=?", userID, role).First(&rep).Error; err != nil {
		// Return default
		return UserReputation{
			UserID:    userID,
			Role:      role,
			Score:     50,
			AvgRating: 3.0,
		}
	}
	return rep
}

// GetOverallScore returns the best score across all roles for a user.
func GetOverallScore(db *gorm.DB, userID uuid.UUID) float64 {
	var reps []UserReputation
	db.Where("user_id=?", userID).Find(&reps)
	if len(reps) == 0 {
		return 50 // neutral default
	}
	best := 0.0
	for _, r := range reps {
		if r.Score > best {
			best = r.Score
		}
	}
	return best
}

// UpdateAfterOrder updates reputation after an order completes or cancels.
// Only DELIVERED orders count toward reputation (verified transactions only).
func UpdateAfterOrder(db *gorm.DB, userID uuid.UUID, role string, completed bool) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var rep UserReputation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id=? AND role=?", userID, role).First(&rep).Error; err != nil {
			rep = UserReputation{
				UserID:    userID,
				Role:      role,
				AvgRating: 3.0,
			}
		}

		rep.TotalOrders++
		if completed {
			rep.CompletedOrders++
		} else {
			rep.CancelledOrders++
		}
		rep.Score = ComputeScore(rep.TotalOrders, rep.CompletedOrders, rep.CancelledOrders, rep.DisputeCount, rep.AvgRating)
		rep.LastUpdated = time.Now()

		return tx.Save(&rep).Error
	})
}

// UpdateAfterDispute updates reputation after a dispute is filed.
func UpdateAfterDispute(db *gorm.DB, userID uuid.UUID, role string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var rep UserReputation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id=? AND role=?", userID, role).First(&rep).Error; err != nil {
			rep = UserReputation{
				UserID:    userID,
				Role:      role,
				AvgRating: 3.0,
			}
		}

		rep.DisputeCount++
		rep.Score = ComputeScore(rep.TotalOrders, rep.CompletedOrders, rep.CancelledOrders, rep.DisputeCount, rep.AvgRating)
		rep.LastUpdated = time.Now()

		return tx.Save(&rep).Error
	})
}

// UpdateRating updates the average rating for a user.
func UpdateRating(db *gorm.DB, userID uuid.UUID, role string, newRating float64) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var rep UserReputation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id=? AND role=?", userID, role).First(&rep).Error; err != nil {
			rep = UserReputation{
				UserID:    userID,
				Role:      role,
				AvgRating: 3.0,
			}
		}

		// Exponential moving average
		alpha := 0.3
		rep.AvgRating = rep.AvgRating*(1-alpha) + newRating*alpha
		rep.Score = ComputeScore(rep.TotalOrders, rep.CompletedOrders, rep.CancelledOrders, rep.DisputeCount, rep.AvgRating)
		rep.LastUpdated = time.Now()

		return tx.Save(&rep).Error
	})
}

// ApplyScoreDelta directly adjusts a reputation score by a delta (for penalties/bonuses).
func ApplyScoreDelta(db *gorm.DB, userID uuid.UUID, role string, delta float64, reason string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var rep UserReputation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id=? AND role=?", userID, role).First(&rep).Error; err != nil {
			rep = UserReputation{
				UserID:    userID,
				Role:      role,
				AvgRating: 3.0,
			}
		}

		rep.Score = math.Max(0, math.Min(100, rep.Score+delta))
		rep.LastUpdated = time.Now()

		slog.Info("reputation: score delta applied", "user_id", userID, "role", role, "delta", delta, "reason", reason, "new_score", rep.Score)

		return tx.Save(&rep).Error
	})
}

// CheckTrustGate verifies if a user's trust level allows a specific action.
func CheckTrustGate(db *gorm.DB, userID uuid.UUID, action string) error {
	score := GetOverallScore(db, userID)
	level := GetTrustLevel(score)

	gates := map[string]float64{
		"auto_close":     50,
		"boost":          40,
		"withdraw_high":  70,
		"create_listing": 30,
		"make_offer":     30,
		"buy_now":        30,
	}

	required, ok := gates[action]
	if !ok {
		return nil // no gate = allowed
	}

	if score < required {
		return fmt.Errorf("trust gate: %s requires score %.0f, user has %.0f (%s trust)", action, required, score, level)
	}

	return nil
}

// GetMaxTransactionAmount returns the max transaction amount based on trust level.
func GetMaxTransactionAmount(db *gorm.DB, userID uuid.UUID) float64 {
	score := GetOverallScore(db, userID)
	level := GetTrustLevel(score)

	switch level {
	case TrustLow:
		return 200
	case TrustNormal:
		return 1000
	default: // TrustHigh
		return 0 // unlimited
	}
}

// IsCollusionOrder checks if the same buyer-seller pair has transacted >3 times.
// Used to ignore collusion orders in reputation scoring.
func IsCollusionOrder(db *gorm.DB, buyerID, sellerID uuid.UUID) bool {
	var cnt int64
	db.Table("orders").
		Where("buyer_id=? AND seller_id=?", buyerID, sellerID).
		Count(&cnt)
	return cnt > 3
}

// UpdateAfterVerifiedOrder updates reputation only for verified (delivered) orders.
// Skips orders flagged as collusion.
func UpdateAfterVerifiedOrder(db *gorm.DB, buyerID, sellerID, userID uuid.UUID, role string) error {
	// Anti-collusion: ignore if same buyer-seller > 3 times
	if IsCollusionOrder(db, buyerID, sellerID) {
		slog.Warn("reputation: collusion detected, skipping order", "buyer_id", buyerID, "seller_id", sellerID, "user_id", userID)
		return nil
	}
	return UpdateAfterOrder(db, userID, role, true)
}

// FormatScoreDelta creates a human-readable delta string for notifications.
func FormatScoreDelta(delta float64) string {
	if delta > 0 {
		return fmt.Sprintf("+%.0f", delta)
	}
	return fmt.Sprintf("%.0f", delta)
}
