package cancellation

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// MaxInsuranceCancellationsPerMonth is the limit on how many times
	// insurance can be used for free cancellation per month.
	MaxInsuranceCancellationsPerMonth = 3

	// RiskScoreInsuranceBlock is the risk score above which insurance
	// purchase is disabled.
	RiskScoreInsuranceBlock = 70.0

	// RiskScoreReducedCoverage is the risk score above which coverage
	// is reduced (max_fee_covered_pct capped at 50%).
	RiskScoreReducedCoverage = 50.0
)

// canBuyInsurance checks if a user is allowed to purchase insurance.
// Blocked if: high risk score, or insurance abuse detected.
func canBuyInsurance(db *gorm.DB, userID uuid.UUID) bool {
	// 1. Check fraud risk score
	var profile struct {
		RiskScore float64
	}
	db.Table("user_risk_profiles").
		Select("risk_score").
		Where("user_id = ?", userID).
		Scan(&profile)

	if profile.RiskScore >= RiskScoreInsuranceBlock {
		return false
	}

	// 2. Check cancellation abuse rate
	stats := getOrCreateStats(db, userID)
	if stats.AbuseMultiplier >= 2.0 {
		return false // extreme abuser — no insurance
	}

	// 3. Check monthly insurance usage
	usage := getOrCreateUsage(db, userID)
	if usage.CancellationsUsed >= MaxInsuranceCancellationsPerMonth {
		return false // already used insurance too many times this month
	}

	return true
}

// getOrCreateUsage fetches or creates the monthly insurance usage record.
func getOrCreateUsage(db *gorm.DB, userID uuid.UUID) *UserInsuranceUsage {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var usage UserInsuranceUsage
	if err := db.Where("user_id = ? AND month = ?", userID, monthStart).First(&usage).Error; err != nil {
		usage = UserInsuranceUsage{
			UserID: userID,
			Month:  monthStart,
		}
		db.Create(&usage)
	}
	return &usage
}

// incrementInsuranceUsage records that insurance was used for a cancellation.
func incrementInsuranceUsage(db *gorm.DB, userID uuid.UUID) {
	usage := getOrCreateUsage(db, userID)
	usage.CancellationsUsed++
	db.Save(usage)
}

// getInsuranceCoverageForUser returns the effective coverage percentage
// based on the user's risk profile. High-risk users get reduced coverage.
func getInsuranceCoverageForUser(db *gorm.DB, userID uuid.UUID, defaultCoverage float64) float64 {
	var profile struct {
		RiskScore float64
	}
	db.Table("user_risk_profiles").
		Select("risk_score").
		Where("user_id = ?", userID).
		Scan(&profile)

	if profile.RiskScore >= RiskScoreReducedCoverage {
		// Reduce coverage to 50% max
		if defaultCoverage > 50 {
			return 50
		}
	}
	return defaultCoverage
}
