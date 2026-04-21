package crowdshipping

import (
	"fmt"
	"log/slog"
	"math"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TrustScore represents a composite trust score for a user.
// Used in matching, ranking, and deal approval.
type TrustScore struct {
	UserID          uuid.UUID `json:"user_id"`
	SellerScore     float64   `json:"seller_score"`     // 0–100
	TravelerScore   float64   `json:"traveler_score"`   // 0–100
	OverallScore    float64   `json:"overall_score"`    // 0–100
	Rating          float64   `json:"rating"`           // avg rating 1–5
	CompletionRate  float64   `json:"completion_rate"`  // 0–1
	DisputeRate     float64   `json:"dispute_rate"`     // 0–1
	DeliverySuccess float64   `json:"delivery_success"` // 0–1 (traveler)
	DeliverySpeed   float64   `json:"delivery_speed"`   // 0–1 (traveler)
	ReviewCount     int       `json:"review_count"`
	CompletedOrders int       `json:"completed_orders"`
	DisputedOrders  int       `json:"disputed_orders"`
}

// ComputeSellerScore calculates seller trust score.
// Components: rating(40%) + completion_rate(35%) + dispute_penalty(25%)
func ComputeSellerScore(rating float64, completionRate float64, disputeRate float64) float64 {
	ratingNorm := math.Max(0, math.Min(1, (rating-1)/4)) // 1→0, 5→1
	score := ratingNorm*40 + completionRate*35 + (1-math.Min(1, disputeRate*5))*25
	return math.Max(0, math.Min(100, score))
}

// ComputeTravelerScore calculates traveler trust score.
// Components: delivery_success(40%) + speed(25%) + rating(20%) + completion(15%)
func ComputeTravelerScore(deliverySuccess, speed, rating, completionRate float64) float64 {
	ratingNorm := math.Max(0, math.Min(1, (rating-1)/4))
	score := deliverySuccess*40 + speed*25 + ratingNorm*20 + completionRate*15
	return math.Max(0, math.Min(100, score))
}

// GetTrustScore computes a user's trust score from DB signals.
// Delegates to the reputation package for seller/buyer scores.
func GetTrustScore(db *gorm.DB, userID uuid.UUID) TrustScore {
	var ts TrustScore
	ts.UserID = userID

	// Gather order signals
	type OrderSignals struct {
		Total     int64
		Completed int64
		Disputed  int64
	}
	var sigs OrderSignals
	db.Table("orders").
		Where("buyer_id=? OR traveler_id=?", userID, userID).
		Count(&sigs.Total)
	db.Table("orders").
		Where("(buyer_id=? OR traveler_id=?) AND status=?", userID, userID, "completed").
		Count(&sigs.Completed)
	db.Table("orders").
		Where("(buyer_id=? OR traveler_id=?) AND status=?", userID, userID, "disputed").
		Count(&sigs.Disputed)

	ts.CompletedOrders = int(sigs.Completed)
	ts.DisputedOrders = int(sigs.Disputed)

	if sigs.Total > 0 {
		ts.CompletionRate = float64(sigs.Completed) / float64(sigs.Total)
		ts.DisputeRate = float64(sigs.Disputed) / float64(sigs.Total)
	} else {
		ts.CompletionRate = 1.0 // neutral default
		ts.DisputeRate = 0
	}

	// Use reputation package for seller score
	sellerRep := reputation.GetUserScore(db, userID, "seller")
	ts.SellerScore = sellerRep.Score
	ts.Rating = sellerRep.AvgRating

	// Traveler-specific signals
	ts.DeliverySuccess = 1.0 // neutral default
	ts.DeliverySpeed = 0.5   // neutral default
	if sigs.Completed > 0 {
		// Count on-time deliveries
		var onTime int64
		db.Table("orders").
			Where("traveler_id=? AND status=? AND delivered_at<=estimated_delivery", userID, "completed").
			Count(&onTime)
		if sigs.Completed > 0 {
			ts.DeliverySuccess = float64(onTime) / float64(sigs.Completed)
			ts.DeliverySpeed = ts.DeliverySuccess // simplified: speed = on-time rate
		}
	}

	ts.TravelerScore = ComputeTravelerScore(ts.DeliverySuccess, ts.DeliverySpeed, ts.Rating, ts.CompletionRate)

	// Overall: delegate to reputation package
	ts.OverallScore = reputation.GetOverallScore(db, userID)

	ts.ReviewCount = int(sigs.Completed)

	return ts
}

// getReputationScore reads the reputation score via the reputation package.
func getReputationScore(db *gorm.DB, userID uuid.UUID) (float64, error) {
	return reputation.GetOverallScore(db, userID), nil
}

// ApplyTrustScoreToMatching adjusts a match score based on trust.
// Higher trust = better match ranking.
func ApplyTrustScoreToMatching(baseScore float64, trust TrustScore) float64 {
	// Trust multiplier: 0.5x (low trust) to 1.2x (high trust)
	trustMultiplier := 0.5 + (trust.OverallScore/100)*0.7
	adjusted := baseScore * trustMultiplier
	return math.Max(0, math.Min(1, adjusted))
}

// IsTrustApprovedForDeal checks if a user's trust level is sufficient for auto-deal.
func IsTrustApprovedForDeal(trust TrustScore) bool {
	return trust.OverallScore >= 40 && trust.DisputeRate < 0.2
}

// Log trust score computation
func logTrustScore(ts TrustScore) {
	slog.Info("trust_engine: computed",
		"user_id", ts.UserID,
		"seller", fmt.Sprintf("%.1f", ts.SellerScore),
		"traveler", fmt.Sprintf("%.1f", ts.TravelerScore),
		"overall", fmt.Sprintf("%.1f", ts.OverallScore),
		"completion", fmt.Sprintf("%.2f", ts.CompletionRate),
		"disputes", fmt.Sprintf("%.2f", ts.DisputeRate),
	)
}
