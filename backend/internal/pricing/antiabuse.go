package pricing

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Anti-Manipulation + Cooldown + Anchoring ──────────────────────────────────────

const (
	// CooldownDuration is how long a price stays fixed for the same user session.
	CooldownDuration = 5 * time.Minute

	// MaxAbusePriceMultiplier is the maximum price increase for abusers.
	MaxAbusePriceMultiplier = 2.0

	// AnchorMultiplier is the "was" price multiplier for anchoring effect.
	AnchorMultiplier = 1.4 // show 40% higher as "original" price
)

// applyAntiAbusePricing adjusts the price for users with abuse patterns.
func applyAntiAbusePricing(ctx *PricingContext, result *PriceResult) PriceResult {
	adjusted := *result

	// 1. Frequent canceller → increase price
	if ctx.CancellationRate > 0.5 {
		adjusted.PricePercent *= 1.3 // +30%
		adjusted.Adjustments.RiskAdj += int64(float64(result.PriceCents) * 0.3)
	} else if ctx.CancellationRate > 0.3 {
		adjusted.PricePercent *= 1.15 // +15%
		adjusted.Adjustments.RiskAdj += int64(float64(result.PriceCents) * 0.15)
	}

	// 2. Abuse flags → price increase
	if ctx.AbuseFlags > 3 {
		adjusted.PricePercent *= MaxAbusePriceMultiplier
	} else if ctx.AbuseFlags > 1 {
		adjusted.PricePercent *= 1.2
	}

	// 3. User gaming: if they always buy insurance and always claim
	if ctx.InsuranceBuyRate > 0.9 && ctx.CancellationRate > 0.4 {
		adjusted.PricePercent *= 1.5 // heavy gamer penalty
	}

	return adjusted
}

// PriceCooldown ensures the same user gets a stable price within a session.
type PriceCooldown struct {
	UserID      uuid.UUID
	PriceCents  int64
	PricePct    float64
	ExpiresAt   time.Time
}

// CheckPriceCooldown returns cached price if within cooldown period.
func CheckPriceCooldown(db *gorm.DB, userID uuid.UUID) *PriceCooldown {
	// In production, this would use Redis with TTL
	// For now, check recent pricing events
	var recent PricingEvent
	if err := db.Where("user_id = ? AND created_at > ?", userID, time.Now().Add(-CooldownDuration)).
		Order("created_at DESC").First(&recent).Error; err != nil {
		return nil
	}

	return &PriceCooldown{
		UserID:     userID,
		PriceCents: recent.PriceCents,
		PricePct:   float64(recent.PriceCents), // approximate
		ExpiresAt:  time.Now().Add(CooldownDuration),
	}
}

// GenerateAnchorPrice creates the "was X, now Y" anchoring effect.
func GenerateAnchorPrice(actualPriceCents int64) int64 {
	return int64(float64(actualPriceCents) * AnchorMultiplier)
}

// DetectPriceGaming identifies users who might be gaming the pricing system.
func DetectPriceGaming(db *gorm.DB, userID uuid.UUID) (isGaming bool, score float64) {
	// Signals of gaming:
	// 1. User checks price multiple times without buying
	// 2. User buys only when price drops
	// 3. User has high claim rate after buying insurance

	var events []PricingEvent
	db.Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&events)

	if len(events) < 5 {
		return false, 0
	}

	// Signal 1: many price checks without buying
	notBought := 0
	for _, e := range events {
		if !e.DidBuy {
			notBought++
		}
	}
	windowShoppingRate := float64(notBought) / float64(len(events))

	// Signal 2: buy only at lower prices
	var boughtPrices []int64
	var notBoughtPrices []int64
	for _, e := range events {
		if e.DidBuy {
			boughtPrices = append(boughtPrices, e.PriceCents)
		} else {
			notBoughtPrices = append(notBoughtPrices, e.PriceCents)
		}
	}

	priceSensitiveGaming := false
	if len(boughtPrices) > 0 && len(notBoughtPrices) > 0 {
		avgBought := avg(boughtPrices)
		avgNotBought := avg(notBoughtPrices)
		if avgNotBought > 0 && avgBought < avgNotBought*0.7 {
			priceSensitiveGaming = true
		}
	}

	// Signal 3: high claim rate
	claimCount := 0
	for _, e := range events {
		if e.ClaimFiled {
			claimCount++
		}
	}
	claimRate := float64(claimCount) / float64(len(events))

	// Aggregate gaming score
	score = 0
	if windowShoppingRate > 0.7 {
		score += 30
	}
	if priceSensitiveGaming {
		score += 40
	}
	if claimRate > 0.5 {
		score += 30
	}

	return score > 50, score
}

func avg(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum int64
	for _, v := range values {
		sum += v
	}
	return float64(sum) / float64(len(values))
}
