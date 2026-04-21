package wallet

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetWithdrawLimit returns the daily withdrawal limit based on trust level.
// Low: $100/day, Normal: $1000/day, High: unlimited (0)
func GetWithdrawLimit(db *gorm.DB, userID uuid.UUID) float64 {
	return reputation.GetMaxTransactionAmount(db, userID)
}

// CheckWithdrawLimit verifies a withdrawal doesn't exceed the user's daily limit.
func CheckWithdrawLimit(db *gorm.DB, userID uuid.UUID, amount float64) error {
	limit := GetWithdrawLimit(db, userID)
	if limit <= 0 {
		return nil // unlimited
	}

	// Check total withdrawn today
	var withdrawnToday float64
	db.Table("wallet_transactions").
		Where("user_id=? AND type='withdrawal' AND created_at>?", userID, time.Now().Truncate(24*time.Hour)).
		Select("COALESCE(SUM(amount),0)").Scan(&withdrawnToday)

	if withdrawnToday+amount > limit {
		remaining := limit - withdrawnToday
		if remaining < 0 {
			remaining = 0
		}
		return fmt.Errorf("daily withdraw limit $%.0f reached (withdrawn $%.2f today, remaining $%.2f)", limit, withdrawnToday, remaining)
	}
	return nil
}

// GetEscrowReleaseDelay returns how long to delay escrow release based on trust.
// Low trust: 24h delay, High trust: instant (0)
func GetEscrowReleaseDelay(db *gorm.DB, userID uuid.UUID) time.Duration {
	score := reputation.GetOverallScore(db, userID)
	level := reputation.GetTrustLevel(score)

	switch level {
	case reputation.TrustLow:
		return 24 * time.Hour
	case reputation.TrustNormal:
		return 2 * time.Hour
	default: // TrustHigh
		return 0 // instant
	}
}

// IsEscrowReleaseReady checks if the escrow hold period has passed.
func IsEscrowReleaseReady(db *gorm.DB, escrowID uuid.UUID) bool {
	var escrow Escrow
	if err := db.Where("id=?", escrowID).First(&escrow).Error; err != nil {
		return false
	}

	delay := GetEscrowReleaseDelay(db, escrow.SellerID)
	if delay == 0 {
		return true // instant for high trust
	}

	return time.Since(escrow.CreatedAt) >= delay
}

// LogRiskDecision logs a risk-related wallet decision for audit.
func LogRiskDecision(db *gorm.DB, userID uuid.UUID, action string, allowed bool, reason string) {
	slog.Info("wallet: risk decision",
		"user_id", userID,
		"action", action,
		"allowed", allowed,
		"reason", reason,
	)
}
