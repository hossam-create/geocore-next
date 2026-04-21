package cancellation

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// CancelRateThreshold is the rate above which abuse multiplier kicks in.
	CancelRateThreshold = 0.30 // 30%
	// AbuseMultiplierPenalty is applied when cancel_rate exceeds threshold.
	AbuseMultiplierPenalty = 1.5
	// HighRiskMultiplier for extreme abuse (cancel_rate > 50%).
	HighRiskMultiplier = 2.0
	// HighRiskThreshold is the rate above which the multiplier doubles.
	HighRiskThreshold = 0.50
)

// getOrCreateStats fetches or creates user cancellation stats.
func getOrCreateStats(db *gorm.DB, userID uuid.UUID) *UserCancellationStats {
	var stats UserCancellationStats
	if err := db.Where("user_id = ?", userID).First(&stats).Error; err != nil {
		// Create new stats record
		stats = UserCancellationStats{
			UserID:             userID,
			TotalOrders:        0,
			TotalCancellations: 0,
			CancelRate:         0,
			AbuseMultiplier:    1.0,
		}
		db.Create(&stats)
	}
	return &stats
}

// updateStatsAfterCancel increments cancellation count and recalculates rate.
func updateStatsAfterCancel(tx *gorm.DB, userID uuid.UUID) {
	var stats UserCancellationStats
	if err := tx.Where("user_id = ?", userID).First(&stats).Error; err != nil {
		// Stats don't exist yet — create them
		now := time.Now()
		stats = UserCancellationStats{
			UserID:             userID,
			TotalOrders:        1,
			TotalCancellations: 1,
			CancelRate:         1.0,
			AbuseMultiplier:    1.0,
			LastCancelAt:       &now,
		}
		tx.Create(&stats)
		return
	}

	now := time.Now()
	stats.TotalCancellations++
	stats.CancelRate = float64(stats.TotalCancellations) / float64(max(stats.TotalOrders, 1))
	stats.LastCancelAt = &now

	// Recalculate abuse multiplier
	switch {
	case stats.CancelRate >= HighRiskThreshold:
		stats.AbuseMultiplier = HighRiskMultiplier
	case stats.CancelRate >= CancelRateThreshold:
		stats.AbuseMultiplier = AbuseMultiplierPenalty
	default:
		stats.AbuseMultiplier = 1.0
	}

	tx.Save(&stats)
}

// IncrementOrderCount should be called when a user creates an order
// to keep the stats denominator accurate.
func IncrementOrderCount(db *gorm.DB, userID uuid.UUID) {
	var stats UserCancellationStats
	if err := db.Where("user_id = ?", userID).First(&stats).Error; err != nil {
		stats = UserCancellationStats{
			UserID:          userID,
			TotalOrders:     1,
			AbuseMultiplier: 1.0,
		}
		db.Create(&stats)
		return
	}
	stats.TotalOrders++
	stats.CancelRate = float64(stats.TotalCancellations) / float64(stats.TotalOrders)
	db.Save(&stats)
}

// GetUserAbuseMultiplier returns the current abuse multiplier for a user.
func GetUserAbuseMultiplier(db *gorm.DB, userID uuid.UUID) float64 {
	stats := getOrCreateStats(db, userID)
	return stats.AbuseMultiplier
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
