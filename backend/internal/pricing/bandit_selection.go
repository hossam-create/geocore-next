package pricing

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Bandit Selection Engine ──────────────────────────────────────────────────────
//
// Main entry point: SelectPrice(ctx) → BanditSelectionResult
//
// Flow:
//   1. Classify user segment (contextual bandit)
//   2. Load arms for that segment
//   3. Check kill switch → fallback if active
//   4. Check session stickiness → return cached if within cooldown
//   5. Select arm using configured algorithm (Thompson / UCB / ε-Greedy)
//   6. Apply safety clamps
//   7. Record impression
//   8. Return result

// SelectPrice selects the optimal insurance price using the bandit engine.
func SelectPrice(db *gorm.DB, ctx *PricingContext) (*BanditSelectionResult, error) {
	// 1. Load active config
	config := loadBanditConfig(db)

	// 2. Check kill switch
	if config.KillSwitchActive {
		return fallbackPrice(ctx, config), nil
	}

	// 3. Classify segment
	segment := ClassifySegment(ctx)

	// 4. Check session stickiness
	cached := checkSessionStickiness(db, ctx.UserID, string(segment), config)
	if cached != nil {
		return cached, nil
	}

	// 5. Load arms for this segment
	arms := loadArmsForSegment(db, string(segment))
	if len(arms) == 0 {
		// Initialize default arms if none exist
		arms = initializeSegmentArms(db, string(segment), config)
	}

	// 6. Select arm based on algorithm
	var selectedArm *BanditArm
	var sampleValue float64
	isExploration := false

	switch config.Algorithm {
	case "thompson":
		selectedArm, sampleValue = ThompsonSelect(arms)
		// Determine if this was exploration: if selected arm isn't the empirical best
		best := bestArmByAvgReward(arms)
		isExploration = best != nil && selectedArm != nil && selectedArm.ID != best.ID
	case "ucb":
		selectedArm, sampleValue = selectUCB(arms)
		isExploration = selectedArm != nil && selectedArm.Impressions < int64(config.MinImpressionsBeforeExploit)
	case "epsilon_greedy":
		selectedArm = EpsilonGreedySelect(arms, config.Epsilon)
		sampleValue = 0
		isExploration = randFloat() < config.Epsilon
	default:
		selectedArm, sampleValue = ThompsonSelect(arms)
	}

	if selectedArm == nil {
		return fallbackPrice(ctx, config), nil
	}

	// 7. Calculate price
	priceCents := int64(float64(ctx.OrderPriceCents) * selectedArm.PricePercent / 100.0)
	if priceCents < 50 {
		priceCents = 50
	}

	// 8. Safety clamp
	pricePercent := math.Max(config.MinPricePercent, math.Min(config.MaxPricePercent, selectedArm.PricePercent))

	// 9. Confidence: based on total impressions for this arm
	confidence := 0.0
	if selectedArm.Impressions > 0 {
		confidence = math.Min(float64(selectedArm.Impressions)/float64(config.MinImpressionsBeforeExploit), 1.0)
	}

	// 10. Record impression
	go recordBanditImpression(db, ctx, selectedArm, string(segment), config.Algorithm)

	// 11. Build result
	result := &BanditSelectionResult{
		ArmID:         selectedArm.ID,
		Segment:       string(segment),
		PricePercent:  pricePercent,
		PriceCents:    priceCents,
		Algorithm:     config.Algorithm,
		SampleValue:   sampleValue,
		Confidence:    confidence,
		IsExploration: isExploration,
		AnchorPrice:   int64(float64(priceCents) * 1.4),
		KillSwitchOn:  false,
	}

	return result, nil
}

// selectUCB selects an arm using Upper Confidence Bound.
func selectUCB(arms []*BanditArm) (*BanditArm, float64) {
	var totalImpressions int64
	for _, a := range arms {
		totalImpressions += a.Impressions
	}

	var best *BanditArm
	bestScore := math.Inf(-1)

	for _, arm := range arms {
		score := UCBScore(arm, totalImpressions)
		if score > bestScore {
			bestScore = score
			best = arm
		}
	}

	return best, bestScore
}

// fallbackPrice returns the static fallback when kill switch is active.
func fallbackPrice(ctx *PricingContext, config BanditConfig) *BanditSelectionResult {
	priceCents := int64(float64(ctx.OrderPriceCents) * config.FallbackPricePercent / 100.0)
	if priceCents < 50 {
		priceCents = 50
	}
	return &BanditSelectionResult{
		PricePercent: config.FallbackPricePercent,
		PriceCents:   priceCents,
		Algorithm:    "fallback",
		Confidence:   1.0,
		KillSwitchOn: true,
		AnchorPrice:  int64(float64(priceCents) * 1.4),
	}
}

// loadBanditConfig loads the active bandit configuration.
func loadBanditConfig(db *gorm.DB) BanditConfig {
	var config BanditConfig
	if err := db.Where("is_active = ?", true).Order("created_at DESC").First(&config).Error; err != nil {
		return BanditConfig{
			Algorithm:                   "thompson",
			Epsilon:                     0.2,
			MinPricePercent:             1,
			MaxPricePercent:             4,
			ConversionDropThreshold:     0.10,
			SessionCooldownMinutes:      5,
			MinImpressionsBeforeExploit: 100,
			FallbackPricePercent:        2,
		}
	}
	return config
}

// loadArmsForSegment loads all arms for a given segment.
func loadArmsForSegment(db *gorm.DB, segment string) []*BanditArm {
	var arms []BanditArm
	db.Where("segment = ?", segment).Find(&arms)
	result := make([]*BanditArm, len(arms))
	for i := range arms {
		result[i] = &arms[i]
	}
	return result
}

// initializeSegmentArms creates default price arms for a new segment.
func initializeSegmentArms(db *gorm.DB, segment string, config BanditConfig) []*BanditArm {
	percents := []float64{1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0}
	arms := make([]*BanditArm, len(percents))

	for i, pct := range percents {
		arm := BanditArm{
			Segment:      segment,
			PricePercent: pct,
			Impressions:  0,
			Conversions:  0,
			TotalReward:  0,
			Alpha:        1, // Beta(1,1) = Uniform — no prior bias
			Beta:         1,
		}
		db.Create(&arm)
		arms[i] = &arm
	}

	return arms
}

// checkSessionStickiness returns a cached price if within cooldown period.
func checkSessionStickiness(db *gorm.DB, userID uuid.UUID, segment string, config BanditConfig) *BanditSelectionResult {
	cooldown := time.Duration(config.SessionCooldownMinutes) * time.Minute

	var event BanditEvent
	if err := db.Where("user_id = ? AND segment = ? AND created_at > ?",
		userID, segment, time.Now().Add(-cooldown)).
		Order("created_at DESC").First(&event).Error; err != nil {
		return nil
	}

	return &BanditSelectionResult{
		ArmID:        event.ArmID,
		Segment:      event.Segment,
		PricePercent: event.PricePercent,
		PriceCents:   event.PriceCents,
		Algorithm:    "session_sticky",
		Confidence:   1.0,
	}
}

// recordBanditImpression records a bandit impression event.
func recordBanditImpression(db *gorm.DB, ctx *PricingContext, arm *BanditArm, segment, algorithm string) {
	priceCents := int64(float64(ctx.OrderPriceCents) * arm.PricePercent / 100.0)

	event := BanditEvent{
		UserID:       ctx.UserID,
		OrderID:      ctx.OrderID,
		Segment:      segment,
		ArmID:        arm.ID,
		PricePercent: arm.PricePercent,
		PriceCents:   priceCents,
		Algorithm:    algorithm,
	}
	db.Create(&event)

	// Increment arm impressions
	db.Model(&BanditArm{}).Where("id = ?", arm.ID).
		Updates(map[string]interface{}{
			"impressions": gorm.Expr("impressions + 1"),
			"beta":        gorm.Expr("beta + 1"), // Beta(α, β+1) — one more failure observed
		})
}

// RecordBanditOutcome records the outcome (buy/no-buy) and updates arm stats.
func RecordBanditOutcome(db *gorm.DB, userID, orderID uuid.UUID, didBuy bool, claimCostCents float64) error {
	// Find the most recent bandit event for this user+order
	var event BanditEvent
	if err := db.Where("user_id = ? AND order_id = ?", userID, orderID).
		Order("created_at DESC").First(&event).Error; err != nil {
		return fmt.Errorf("no bandit event found for this order")
	}

	// Calculate reward
	reward := CalculateBanditReward(event.PriceCents, didBuy, claimCostCents)

	// Update the event
	db.Model(&BanditEvent{}).Where("id = ?", event.ID).Updates(map[string]interface{}{
		"did_buy":    didBuy,
		"reward":     reward,
		"claim_cost": claimCostCents,
	})

	// Update arm stats
	if didBuy {
		db.Model(&BanditArm{}).Where("id = ?", event.ArmID).
			Updates(map[string]interface{}{
				"conversions":  gorm.Expr("conversions + 1"),
				"total_reward": gorm.Expr("total_reward + ?", reward),
				"alpha":        gorm.Expr("alpha + 1"), // Beta(α+1, β) — one more success
			})
	} else {
		db.Model(&BanditArm{}).Where("id = ?", event.ArmID).
			Updates(map[string]interface{}{
				"total_reward": gorm.Expr("total_reward + ?", reward),
			})
		// beta already incremented during impression
	}

	return nil
}

// randFloat returns a random float64 in [0,1).
func randFloat() float64 {
	return rand.Float64()
}

// Ensure fmt is used
var _ = fmt.Sprintf
