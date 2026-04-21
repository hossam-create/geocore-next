package cancellation

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// DefaultTokensPerMonth is the number of free cancellations per month.
	DefaultTokensPerMonth = 2
)

// tryUseToken attempts to use a free cancellation token for the user.
// Returns true if a token was successfully consumed, false otherwise.
func tryUseToken(db *gorm.DB, userID uuid.UUID) bool {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0)

	var token UserCancellationToken
	if err := db.Where("user_id = ? AND period_start = ?", userID, periodStart).
		First(&token).Error; err != nil {
		// No token record for this period — create one
		token = UserCancellationToken{
			UserID:          userID,
			RemainingTokens: DefaultTokensPerMonth,
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
		}
		if err := db.Create(&token).Error; err != nil {
			return false // couldn't create, deny token
		}
	}

	if token.RemainingTokens <= 0 {
		return false // no tokens left
	}

	// Consume one token
	token.RemainingTokens--
	if err := db.Save(&token).Error; err != nil {
		return false
	}
	return true
}

// GetRemainingTokens returns the number of free cancellation tokens for the current month.
func GetRemainingTokens(db *gorm.DB, userID uuid.UUID) int {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var token UserCancellationToken
	if err := db.Where("user_id = ? AND period_start = ?", userID, periodStart).
		First(&token).Error; err != nil {
		return DefaultTokensPerMonth
	}
	return token.RemainingTokens
}

// ResetMonthlyTokens resets all users' tokens for the new month.
// Should be called by a cron job on the 1st of each month.
func ResetMonthlyTokens(db *gorm.DB) error {
	now := time.Now()
	periodStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	periodEnd := periodStart.AddDate(0, 1, 0)

	// Get all users who had tokens last month
	var lastMonthTokens []UserCancellationToken
	lastMonth := periodStart.AddDate(0, -1, 0)
	db.Where("period_start = ?", lastMonth).Find(&lastMonthTokens)

	for _, t := range lastMonthTokens {
		newToken := UserCancellationToken{
			UserID:          t.UserID,
			RemainingTokens: DefaultTokensPerMonth,
			PeriodStart:     periodStart,
			PeriodEnd:       periodEnd,
		}
		db.Create(&newToken)
	}

	return nil
}
