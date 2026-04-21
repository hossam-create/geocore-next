package reputation

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TrustIndicators represents the trust signals shown to users in the UI.
// This is the UX layer that makes trust visible.
type TrustIndicators struct {
	UserID       uuid.UUID `json:"user_id"`
	Score        float64   `json:"score"`          // 0–100
	Level        string    `json:"level"`          // low, normal, high
	LevelLabel   string    `json:"level_label"`    // 🟢 Verified, 🟡 Normal, 🔴 Low Trust
	IsVerified   bool      `json:"is_verified"`    // score > 60
	StarRating   float64   `json:"star_rating"`    // 1–5 stars
	TotalOrders  int       `json:"total_orders"`   // completed orders
	Badge        string    `json:"badge"`          // badge name: new, bronze, silver, gold, platinum
	EscrowBadge  string    `json:"escrow_badge"`   // 🔒 Escrow Protected (always shown)
}

// GetTrustIndicators returns the UX-friendly trust indicators for a user.
func GetTrustIndicators(db *gorm.DB, userID uuid.UUID) TrustIndicators {
	score := GetOverallScore(db, userID)
	level := GetTrustLevel(score)

	// Get best role rep for star rating
	var reps []UserReputation
	db.Where("user_id=?", userID).Find(&reps)
	bestRating := 3.0
	totalOrders := 0
	for _, r := range reps {
		if r.AvgRating > bestRating {
			bestRating = r.AvgRating
		}
		totalOrders += r.CompletedOrders
	}

	// Level label with emoji
	levelLabel := "🔴 Low Trust"
	if level == TrustNormal {
		levelLabel = "🟡 Normal"
	} else if level == TrustHigh {
		levelLabel = "🟢 Verified"
	}

	// Badge based on score and order count
	badge := "new"
	switch {
	case score >= 90 && totalOrders >= 50:
		badge = "platinum"
	case score >= 75 && totalOrders >= 20:
		badge = "gold"
	case score >= 60 && totalOrders >= 10:
		badge = "silver"
	case score >= 40 && totalOrders >= 5:
		badge = "bronze"
	}

	return TrustIndicators{
		UserID:      userID,
		Score:       score,
		Level:       level,
		LevelLabel:  levelLabel,
		IsVerified:  score > 60,
		StarRating:  bestRating,
		TotalOrders: totalOrders,
		Badge:       badge,
		EscrowBadge: "🔒 Escrow Protected",
	}
}

// ListingTrustInfo returns trust info for a listing's seller.
type ListingTrustInfo struct {
	SellerScore      float64 `json:"seller_score"`
	SellerLevel      string  `json:"seller_level"`
	SellerVerified   bool    `json:"seller_verified"`
	SellerBadge      string  `json:"seller_badge"`
	SellerStarRating float64 `json:"seller_star_rating"`
	EscrowProtected  bool    `json:"escrow_protected"` // always true for active listings
}

// GetListingTrustInfo returns trust indicators for a listing's seller.
func GetListingTrustInfo(db *gorm.DB, sellerID uuid.UUID) ListingTrustInfo {
	indicators := GetTrustIndicators(db, sellerID)
	return ListingTrustInfo{
		SellerScore:      indicators.Score,
		SellerLevel:      indicators.Level,
		SellerVerified:   indicators.IsVerified,
		SellerBadge:      indicators.Badge,
		SellerStarRating: indicators.StarRating,
		EscrowProtected:  true,
	}
}

// ActionTrustInfo returns what actions a user can perform.
type ActionTrustInfo struct {
	CanBuyNow      bool    `json:"can_buy_now"`
	CanMakeOffer   bool    `json:"can_make_offer"`
	CanBoost       bool    `json:"can_boost"`
	CanAutoClose   bool    `json:"can_auto_close"`
	CanWithdrawMax float64 `json:"can_withdraw_max"` // 0 = unlimited
	MaxTxAmount    float64 `json:"max_tx_amount"`
}

// GetActionTrustInfo returns which actions a user is allowed to perform.
func GetActionTrustInfo(db *gorm.DB, userID uuid.UUID) ActionTrustInfo {
	score := GetOverallScore(db, userID)

	return ActionTrustInfo{
		CanBuyNow:      score >= 30,
		CanMakeOffer:   score >= 30,
		CanBoost:       score >= 40,
		CanAutoClose:   score >= 50,
		CanWithdrawMax: GetMaxTransactionAmount(db, userID),
		MaxTxAmount:    GetMaxTransactionAmount(db, userID),
	}
}
