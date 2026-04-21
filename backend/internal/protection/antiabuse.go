package protection

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	// MaxClaimsPerMonth is the maximum number of claims a user can file per month.
	MaxClaimsPerMonth = 3

	// MaxApprovalRateBeforeFlag is the approval rate above which a user is flagged.
	MaxApprovalRateBeforeFlag = 0.80 // 80% approval rate is suspicious

	// TravelerDelayPenaltyThreshold is the number of delay claims before penalizing a traveler.
	TravelerDelayPenaltyThreshold = 3

	// AbuseScoreBlock is the abuse score above which protection is disabled.
	AbuseScoreBlock = 50.0
)

// ── Buyer Anti-Abuse ────────────────────────────────────────────────────────────

// CanFileClaim checks if a user is allowed to file a claim.
func CanFileClaim(db *gorm.DB, userID uuid.UUID) (bool, string) {
	// 1. Check monthly claim count
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	var claimCount int64
	db.Model(&GuaranteeClaim{}).
		Where("user_id = ? AND created_at >= ?", userID, monthStart).
		Count(&claimCount)

	if claimCount >= MaxClaimsPerMonth {
		return false, "monthly_claim_limit_reached"
	}

	// 2. Check abuse score
	abuseScore := calculateAbuseScore(db, userID)
	if abuseScore >= AbuseScoreBlock {
		return false, "abuse_detected"
	}

	// 3. Check risk score
	var profile struct {
		RiskScore float64
	}
	db.Table("user_risk_profiles").
		Select("risk_score").
		Where("user_id = ?", userID).
		Scan(&profile)

	if profile.RiskScore >= 80 {
		return false, "high_risk_user"
	}

	return true, ""
}

// calculateAbuseScore computes a 0-100 abuse score for a user.
func calculateAbuseScore(db *gorm.DB, userID uuid.UUID) float64 {
	var totalClaims int64
	db.Model(&GuaranteeClaim{}).Where("user_id = ?", userID).Count(&totalClaims)

	if totalClaims == 0 {
		return 0
	}

	var approvedClaims int64
	db.Model(&GuaranteeClaim{}).
		Where("user_id = ? AND status IN ?", userID, []string{"auto_approved", "approved"}).
		Count(&approvedClaims)

	approvalRate := float64(approvedClaims) / float64(totalClaims)

	// High approval rate + many claims = suspicious
	score := 0.0

	// Factor 1: Too many claims
	if totalClaims > 5 {
		score += 20
	}
	if totalClaims > 10 {
		score += 20
	}

	// Factor 2: Very high approval rate (suspicious if combined with many claims)
	if approvalRate > MaxApprovalRateBeforeFlag && totalClaims > 3 {
		score += 30
	}

	// Factor 3: Recent claims (last 30 days)
	var recentClaims int64
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	db.Model(&GuaranteeClaim{}).
		Where("user_id = ? AND created_at >= ?", userID, thirtyDaysAgo).
		Count(&recentClaims)
	if recentClaims > 2 {
		score += 20
	}

	// Factor 4: Cancellation rate
	var cancelStats struct {
		CancelRate float64
	}
	db.Table("user_cancellation_stats").
		Select("cancel_rate").
		Where("user_id = ?", userID).
		Scan(&cancelStats)
	if cancelStats.CancelRate > 0.5 {
		score += 10
	}

	return score
}

// ── Traveler Anti-Abuse ──────────────────────────────────────────────────────────

// CheckTravelerDelayHistory checks if a traveler has too many delay claims.
func CheckTravelerDelayHistory(db *gorm.DB, travelerID uuid.UUID) (int, bool) {
	var delayClaimCount int64
	db.Model(&GuaranteeClaim{}).
		Where("traveler_id = ? AND type = ? AND traveler_penalty = ?",
			travelerID, ClaimDelay, true).
		Count(&delayClaimCount)

	count := int(delayClaimCount)
	shouldPenalize := count >= TravelerDelayPenaltyThreshold
	return count, shouldPenalize
}

// ApplyTravelerDemotion reduces a traveler's trust and matching priority.
func ApplyTravelerDemotion(db *gorm.DB, travelerID uuid.UUID) error {
	// 1. Reduce geoscore
	db.Table("geo_scores").
		Where("user_id = ?", travelerID).
		Updates(map[string]interface{}{
			"score":      gorm.Expr("GREATEST(score - 25, 0)"),
			"updated_at": time.Now(),
		})

	// 2. Flag in risk profile
	db.Table("user_risk_profiles").
		Where("user_id = ?", travelerID).
		Updates(map[string]interface{}{
			"risk_score": gorm.Expr("LEAST(risk_score + 15, 100)"),
			"updated_at": time.Now(),
		})

	return nil
}

// ── Protection Purchase Guard ────────────────────────────────────────────────────

// CanPurchaseProtection checks if a user is allowed to buy protection.
func CanPurchaseProtection(db *gorm.DB, userID uuid.UUID) (bool, string) {
	// 1. Check abuse score
	abuseScore := calculateAbuseScore(db, userID)
	if abuseScore >= AbuseScoreBlock {
		return false, "abuse_detected"
	}

	// 2. Check risk score — very high risk users can't buy protection
	var profile struct {
		RiskScore float64
	}
	db.Table("user_risk_profiles").
		Select("risk_score").
		Where("user_id = ?", userID).
		Scan(&profile)

	if profile.RiskScore >= 80 {
		return false, "high_risk_user"
	}

	return true, ""
}

// GetTopRiskyUsers returns users with the highest abuse scores.
func GetTopRiskyUsers(db *gorm.DB, limit int) []RiskyUser {
	var users []struct {
		UserID      uuid.UUID
		ClaimsCount int
	}

	db.Table("guarantee_claims").
		Select("user_id, COUNT(*) as claims_count").
		Where("status IN ?", []string{"auto_approved", "approved"}).
		Group("user_id").
		Order("claims_count DESC").
		Limit(limit).
		Find(&users)

	result := make([]RiskyUser, 0, len(users))
	for _, u := range users {
		score := calculateAbuseScore(db, u.UserID)
		if score > 0 {
			result = append(result, RiskyUser{
				UserID:      u.UserID,
				ClaimsCount: u.ClaimsCount,
				AbuseScore:  score,
			})
		}
	}

	return result
}
