package exchange

// fees.go — Sprint 19/20 Exchange Fee Calculation (tier-aware).
//
// IMPORTANT: Fees are calculated and recorded here for transparency but are
// collected EXTERNALLY (off-ledger / via external payment provider).
// Platform NEVER deducts from a wallet balance.

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FeeConfig holds the fee rates. These can be loaded from DB/config in future.
type FeeConfig struct {
	MatchFeeRate    float64 // 0.01 = 1%
	PriorityFeeRate float64 // 0.005 = 0.5%
	ProtectionRate  float64 // 0.005 = 0.5%
}

var defaultFeeConfig = FeeConfig{
	MatchFeeRate:    0.015, // 1.5%
	PriorityFeeRate: 0.005, // 0.5%
	ProtectionRate:  0.005, // 0.5%
}

// ExchangeFeeBreakdown is the summary returned to the caller.
type ExchangeFeeBreakdown struct {
	MatchFee      float64 `json:"match_fee"`
	PriorityFee   float64 `json:"priority_fee"`
	ProtectionFee float64 `json:"protection_fee"`
	TotalFee      float64 `json:"total_fee"`
	Currency      string  `json:"currency"`
	Tier          string  `json:"tier"`
	Discount      float64 `json:"discount_pct"`
}

// CalculateExchangeFee computes the full fee breakdown at the FREE tier.
// hasPriority signals whether the user selected the optional priority listing.
// hasProtection signals whether the user opted into dispute-protection.
func CalculateExchangeFee(amount float64, currency string, hasPriority, hasProtection bool) ExchangeFeeBreakdown {
	return CalculateExchangeFeeForTier(amount, currency, hasPriority, hasProtection, TierFree)
}

// CalculateExchangeFeeForTier computes the fee breakdown applying the tier discount.
func CalculateExchangeFeeForTier(amount float64, currency string, hasPriority, hasProtection bool, tier string) ExchangeFeeBreakdown {
	cfg := defaultFeeConfig
	mult := TierFeeMultiplier(tier)
	matchFee := amount * cfg.MatchFeeRate * mult
	var priorityFee, protectionFee float64
	if hasPriority {
		priorityFee = amount * cfg.PriorityFeeRate * mult
	}
	if hasProtection {
		protectionFee = amount * cfg.ProtectionRate * mult
	}
	discountPct := (1 - mult) * 100
	return ExchangeFeeBreakdown{
		MatchFee:      round2(matchFee),
		PriorityFee:   round2(priorityFee),
		ProtectionFee: round2(protectionFee),
		TotalFee:      round2(matchFee + priorityFee + protectionFee),
		Currency:      currency,
		Tier:          tier,
		Discount:      round2(discountPct),
	}
}

// calculateFees builds []ExchangeFee rows for one participant in a match.
// Applies tier-based discount. Called from match_engine.go — does NOT write to DB.
func calculateFees(db *gorm.DB, reqA, reqB *ExchangeRequest, userID uuid.UUID) []ExchangeFee {
	amount := reqA.Amount
	currency := reqA.FromCurrency
	if userID == reqB.UserID {
		amount = reqB.Amount
		currency = reqB.FromCurrency
	}
	tier := GetUserTier(db, userID)
	bd := CalculateExchangeFeeForTier(amount, currency, false, false, tier)
	fees := []ExchangeFee{
		{
			ID:       uuid.New(),
			UserID:   userID,
			FeeType:  FeeTypeMatch,
			Amount:   bd.MatchFee,
			Currency: currency,
		},
	}
	return fees
}

// round2 rounds to 2 decimal places.
func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
