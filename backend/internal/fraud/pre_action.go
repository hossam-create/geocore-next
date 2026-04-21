package fraud

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PreActionRiskResult is the output of a pre-action risk check.
type PreActionRiskResult struct {
	Allowed       bool     `json:"allowed"`
	RiskScore     int      `json:"risk_score"`    // 0–100
	Action        string   `json:"action"`        // allow, manual_review, block
	Reason        string   `json:"reason"`        // human-readable reason
	Flags         []string `json:"flags"`         // detected risk flags
	TrustLevel    string   `json:"trust_level"`   // low, normal, high
	TrustScore    float64  `json:"trust_score"`   // 0–100
}

// PreActionRiskThresholds defines score boundaries for pre-action decisions.
var PreActionRiskThresholds = struct {
	AllowMax      int
	ReviewMin     int
	BlockMin      int
}{
	AllowMax:  29, // <30 → allow
	ReviewMin: 30, // 30–60 → manual review
	BlockMin:  61, // >60 → block
}

// CheckRiskBeforeAcceptOffer evaluates risk before accepting an offer.
func CheckRiskBeforeAcceptOffer(ctx context.Context, db *gorm.DB, buyerID, sellerID uuid.UUID, amount float64, ip string) PreActionRiskResult {
	result := PreActionRiskResult{}
	riskScore := 0

	// ── 1. Buyer trust score ────────────────────────────────────────────────
	buyerTrust := reputation.GetOverallScore(db, buyerID)
	result.TrustScore = buyerTrust
	result.TrustLevel = reputation.GetTrustLevel(buyerTrust)

	if buyerTrust < 30 {
		riskScore += 40
		result.Flags = append(result.Flags, "low_trust_buyer")
	} else if buyerTrust < 50 {
		riskScore += 15
		result.Flags = append(result.Flags, "below_average_trust")
	}

	// ── 2. Seller trust score ────────────────────────────────────────────────
	sellerTrust := reputation.GetOverallScore(db, sellerID)
	if sellerTrust < 30 {
		riskScore += 30
		result.Flags = append(result.Flags, "low_trust_seller")
	}

	// ── 3. Amount vs trust limits ────────────────────────────────────────────
	maxAmount := reputation.GetMaxTransactionAmount(db, buyerID)
	if maxAmount > 0 && amount > maxAmount {
		riskScore += 50
		result.Flags = append(result.Flags, "amount_exceeds_trust_limit")
	}

	// ── 4. Marketplace fraud rules ──────────────────────────────────────────
	marketResult := CheckMarketplaceRules(ctx, db, MarketplaceCheckInput{
		UserID:    buyerID,
		IP:        ip,
		EventType: "offer",
		Amount:    amount,
	})
	riskScore += marketResult.Score
	result.Flags = append(result.Flags, marketResult.Flags...)

	// ── 5. Collusion check ──────────────────────────────────────────────────
	if reputation.IsCollusionOrder(db, buyerID, sellerID) {
		riskScore += 30
		result.Flags = append(result.Flags, "collusion_suspected")
	}

	// Clamp
	if riskScore > 100 {
		riskScore = 100
	}
	result.RiskScore = riskScore

	// Determine action
	switch {
	case riskScore >= PreActionRiskThresholds.BlockMin:
		result.Action = "block"
		result.Allowed = false
		result.Reason = fmt.Sprintf("risk score %d exceeds block threshold (%d)", riskScore, PreActionRiskThresholds.BlockMin)
	case riskScore >= PreActionRiskThresholds.ReviewMin:
		result.Action = "manual_review"
		result.Allowed = false
		result.Reason = fmt.Sprintf("risk score %d requires manual review (%d–%d)", riskScore, PreActionRiskThresholds.ReviewMin, PreActionRiskThresholds.BlockMin-1)
	default:
		result.Action = "allow"
		result.Allowed = true
		result.Reason = "risk score within acceptable range"
	}

	slog.Info("fraud: pre-action risk check",
		"action", "accept_offer",
		"buyer_id", buyerID,
		"seller_id", sellerID,
		"amount", amount,
		"risk_score", riskScore,
		"decision", result.Action,
	)

	return result
}

// CheckRiskBeforeWithdraw evaluates risk before a wallet withdrawal.
func CheckRiskBeforeWithdraw(ctx context.Context, db *gorm.DB, userID uuid.UUID, amount float64, ip string) PreActionRiskResult {
	result := PreActionRiskResult{}
	riskScore := 0

	// ── 1. User trust score ──────────────────────────────────────────────────
	trust := reputation.GetOverallScore(db, userID)
	result.TrustScore = trust
	result.TrustLevel = reputation.GetTrustLevel(trust)

	if trust < 30 {
		riskScore += 50
		result.Flags = append(result.Flags, "low_trust_withdraw")
	} else if trust < 50 {
		riskScore += 20
		result.Flags = append(result.Flags, "below_average_trust_withdraw")
	}

	// ── 2. Amount vs trust limits ────────────────────────────────────────────
	maxAmount := reputation.GetMaxTransactionAmount(db, userID)
	if maxAmount > 0 && amount > maxAmount {
		riskScore += 60
		result.Flags = append(result.Flags, "withdraw_exceeds_trust_limit")
	}

	// ── 3. Marketplace fraud rules ──────────────────────────────────────────
	marketResult := CheckMarketplaceRules(ctx, db, MarketplaceCheckInput{
		UserID:    userID,
		IP:        ip,
		EventType: "withdraw",
		Amount:    amount,
	})
	riskScore += marketResult.Score
	result.Flags = append(result.Flags, marketResult.Flags...)

	// Clamp
	if riskScore > 100 {
		riskScore = 100
	}
	result.RiskScore = riskScore

	// Determine action
	switch {
	case riskScore >= PreActionRiskThresholds.BlockMin:
		result.Action = "block"
		result.Allowed = false
		result.Reason = fmt.Sprintf("withdrawal risk score %d exceeds block threshold", riskScore)
	case riskScore >= PreActionRiskThresholds.ReviewMin:
		result.Action = "manual_review"
		result.Allowed = false
		result.Reason = fmt.Sprintf("withdrawal risk score %d requires manual review", riskScore)
	default:
		result.Action = "allow"
		result.Allowed = true
		result.Reason = "withdrawal risk within acceptable range"
	}

	slog.Info("fraud: pre-action risk check",
		"action", "withdraw",
		"user_id", userID,
		"amount", amount,
		"risk_score", riskScore,
		"decision", result.Action,
	)

	return result
}

// CheckRiskBeforeBuyNow evaluates risk before a Buy Now action.
func CheckRiskBeforeBuyNow(ctx context.Context, db *gorm.DB, buyerID, sellerID uuid.UUID, amount float64, ip string) PreActionRiskResult {
	// Same logic as accept offer but with buy_now event type
	result := PreActionRiskResult{}
	riskScore := 0

	buyerTrust := reputation.GetOverallScore(db, buyerID)
	result.TrustScore = buyerTrust
	result.TrustLevel = reputation.GetTrustLevel(buyerTrust)

	if buyerTrust < 30 {
		riskScore += 40
		result.Flags = append(result.Flags, "low_trust_buyer")
	}

	sellerTrust := reputation.GetOverallScore(db, sellerID)
	if sellerTrust < 30 {
		riskScore += 30
		result.Flags = append(result.Flags, "low_trust_seller")
	}

	maxAmount := reputation.GetMaxTransactionAmount(db, buyerID)
	if maxAmount > 0 && amount > maxAmount {
		riskScore += 50
		result.Flags = append(result.Flags, "amount_exceeds_trust_limit")
	}

	marketResult := CheckMarketplaceRules(ctx, db, MarketplaceCheckInput{
		UserID:    buyerID,
		IP:        ip,
		EventType: "buy_now",
		Amount:    amount,
	})
	riskScore += marketResult.Score
	result.Flags = append(result.Flags, marketResult.Flags...)

	if riskScore > 100 {
		riskScore = 100
	}
	result.RiskScore = riskScore

	switch {
	case riskScore >= PreActionRiskThresholds.BlockMin:
		result.Action = "block"
		result.Allowed = false
		result.Reason = fmt.Sprintf("buy now risk score %d exceeds block threshold", riskScore)
	case riskScore >= PreActionRiskThresholds.ReviewMin:
		result.Action = "manual_review"
		result.Allowed = false
		result.Reason = fmt.Sprintf("buy now risk score %d requires manual review", riskScore)
	default:
		result.Action = "allow"
		result.Allowed = true
		result.Reason = "buy now risk within acceptable range"
	}

	return result
}
