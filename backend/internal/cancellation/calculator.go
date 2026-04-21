package cancellation

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CalculateCancellationFee computes the dynamic cancellation fee for an order.
//
// Logic:
//   - Order not yet accepted by traveler → 0 (free)
//   - Within grace period (default 10 min) → 0–2%
//   - Within tier1 (default 1h) → 5%
//   - Within tier2 (default 24h) → 10%
//   - After tier2 → 15%
//
// Anti-abuse: if user cancel_rate > threshold, multiply fee.
// Free tokens: if user has remaining tokens, waive the fee entirely.
func CalculateCancellationFee(db *gorm.DB, orderID, userID uuid.UUID) (*CancellationFeeResult, error) {
	// 1. Load order
	var ord struct {
		ID           uuid.UUID
		BuyerID      uuid.UUID
		SellerID     uuid.UUID
		Status       string
		Total        float64
		Currency     string
		DeliveryType string
		ConfirmedAt  *time.Time
		CreatedAt    time.Time
	}
	if err := db.Table("orders").
		Select("id, buyer_id, seller_id, status, total, currency, delivery_type, confirmed_at, created_at").
		Where("id = ?", orderID).First(&ord).Error; err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	// Only the buyer can cancel with fee calculation
	if ord.BuyerID != userID {
		return nil, fmt.Errorf("only the buyer can initiate cancellation")
	}

	// If order is already cancelled/completed, no fee
	if ord.Status == "cancelled" || ord.Status == "completed" || ord.Status == "delivered" {
		return &CancellationFeeResult{Tier: TierFree}, nil
	}

	// 2. If order not yet accepted → free cancellation
	if ord.ConfirmedAt == nil {
		return &CancellationFeeResult{
			FeeCents: 0,
			Tier:     TierFree,
		}, nil
	}

	// 3. Calculate seconds since acceptance
	secondsSinceAccept := int(time.Since(*ord.ConfirmedAt).Seconds())

	// 4. Load cancellation policy
	policy, err := loadPolicy(db)
	if err != nil {
		return nil, fmt.Errorf("policy load failed: %w", err)
	}

	// 5. Determine tier and base fee percent
	var tier string
	var feePct float64
	switch {
	case secondsSinceAccept <= policy.GraceSeconds:
		tier = TierGrace
		feePct = policy.FeeGracePct
	case secondsSinceAccept <= policy.Tier1Seconds:
		tier = Tier1
		feePct = policy.FeeTier1Pct
	case secondsSinceAccept <= policy.Tier2Seconds:
		tier = Tier2
		feePct = policy.FeeTier2Pct
	default:
		tier = TierMax
		feePct = policy.FeeMaxPct
	}

	// 6. Anti-abuse: check user cancellation stats
	stats := getOrCreateStats(db, userID)
	abuseMultiplier := stats.AbuseMultiplier

	// Apply multiplier to fee percent
	effectivePct := feePct * abuseMultiplier
	if effectivePct > policy.FeeMaxPct*2 { // hard cap at 2x max
		effectivePct = policy.FeeMaxPct * 2
	}

	// 7. Check free cancellation tokens
	tokenUsed := false
	if effectivePct > 0 {
		tokenUsed = tryUseToken(db, userID)
		if tokenUsed {
			effectivePct = 0
		}
	}

	// 8. Check cancellation insurance
	insuranceApplied := false
	originalFeeCents := int64(0)
	var insuranceID *uuid.UUID
	if effectivePct > 0 && IsInsuranceEnabled() {
		var insurance OrderInsurance
		if err := db.Where("order_id = ? AND is_active = ? AND is_used = ?", orderID, true, false).
			First(&insurance).Error; err == nil {
			// Insurance found — apply coverage
			coveragePct := getInsuranceCoverageForUser(db, userID, insurance.MaxFeeCoveredPct)

			totalCents := int64(ord.Total * 100)
			originalFeeCents = int64(float64(totalCents) * effectivePct / 100.0)

			// Coverage reduces the fee
			feeCoveredCents := int64(float64(originalFeeCents) * coveragePct / 100.0)
			finalFeeCents := originalFeeCents - feeCoveredCents

			effectivePct = effectivePct * (100 - coveragePct) / 100.0
			_ = finalFeeCents // will be recalculated below
			insuranceApplied = true
			insuranceID = &insurance.ID

			// Mark insurance as used
			db.Model(&insurance).Updates(map[string]interface{}{
				"is_used": true,
			})
			incrementInsuranceUsage(db, userID)
		}
	}

	// 9. Calculate fee in cents
	totalCents := int64(ord.Total * 100)
	feeCents := int64(float64(totalCents) * effectivePct / 100.0)

	// 10. Distribute: traveler_compensation + platform_fee
	travelerComp := int64(float64(feeCents) * policy.TravelerSplit / 100.0)
	platformFee := feeCents - travelerComp

	result := &CancellationFeeResult{
		FeeCents:             feeCents,
		TravelerCompensation: travelerComp,
		PlatformFee:          platformFee,
		FeePercent:           effectivePct,
		Tier:                 tier,
		AbuseMultiplier:      abuseMultiplier,
		TokenUsed:            tokenUsed,
		SecondsSinceAccept:   secondsSinceAccept,
		InsuranceApplied:     insuranceApplied,
		OriginalFeeCents:     originalFeeCents,
	}
	if insuranceID != nil {
		result.InsuranceID = *insuranceID
	}
	return result, nil
}

// loadPolicy fetches the active global cancellation policy.
func loadPolicy(db *gorm.DB) (*CancellationPolicy, error) {
	var policy CancellationPolicy
	if err := db.Where("is_active = ? AND corridor_key = ?", true, "global").
		First(&policy).Error; err != nil {
		// Fallback to hardcoded defaults
		return &CancellationPolicy{
			GraceSeconds:  600,
			Tier1Seconds:  3600,
			Tier2Seconds:  86400,
			FeeGracePct:   0,
			FeeTier1Pct:   5,
			FeeTier2Pct:   10,
			FeeMaxPct:     15,
			TravelerSplit: 70,
		}, nil
	}
	return &policy, nil
}
