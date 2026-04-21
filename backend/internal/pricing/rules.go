package pricing

import (
	"math"
)

// ── Rule-Based Pricing Engine ────────────────────────────────────────────────────

const (
	DefaultBasePricePercent   = 1.5
	DefaultMinPricePercent    = 1.0
	DefaultMaxPricePercent    = 4.0
	DefaultStaticPricePercent = 2.0
)

// CalculateRuleBasedPrice computes insurance price using deterministic rules.
//
// Step 1 — Base price: 1.5% of order value
// Step 2 — Risk adjustment: +1% high risk, -0.5% trusted
// Step 3 — Behavior adjustment: +0.5% if always buys, -0.5% if refuses
// Step 4 — Context adjustment: +0.5% rush/urgent, -0.25% low demand
// Step 5 — Clamp: final price between 1% and 4%
func CalculateRuleBasedPrice(ctx *PricingContext, cfg *PricingModelConfig) PriceResult {
	basePct := cfg.BasePricePercent
	minPct := cfg.MinPricePercent
	maxPct := cfg.MaxPricePercent

	// Step 1 — Base price
	basePriceCents := int64(float64(ctx.OrderPriceCents) * basePct / 100.0)

	// Step 2 — Risk adjustment
	riskAdjCents, riskAdjPct := riskAdjustment(ctx)

	// Step 3 — Behavior adjustment
	behaviorAdjCents, behaviorAdjPct := behaviorAdjustment(ctx)

	// Step 4 — Context adjustment
	contextAdjCents, contextAdjPct := contextAdjustment(ctx)

	// Step 5 — Calculate total and clamp
	totalPct := basePct + riskAdjPct + behaviorAdjPct + contextAdjPct
	totalPct = math.Max(minPct, math.Min(maxPct, totalPct))

	finalPriceCents := int64(float64(ctx.OrderPriceCents) * totalPct / 100.0)
	if finalPriceCents < 50 {
		finalPriceCents = 50
	}

	// Estimate buy probability from price sensitivity
	buyProb := estimateBuyProbability(ctx, totalPct)

	// Anchor price: show a higher "was" price for anchoring effect
	anchorCents := int64(float64(finalPriceCents) * 1.4) // 40% higher

	return PriceResult{
		PriceCents:     finalPriceCents,
		BasePriceCents: basePriceCents,
		PricePercent:   totalPct,
		Adjustments: PriceAdjustments{
			RiskAdj:     riskAdjCents,
			BehaviorAdj: behaviorAdjCents,
			ContextAdj:  contextAdjCents,
		},
		BuyProbability: buyProb,
		Confidence:     0.85, // rules have high confidence by design
		Strategy:       StrategyRules,
		AnchorPrice:    anchorCents,
	}
}

// riskAdjustment adjusts price based on user and order risk.
func riskAdjustment(ctx *PricingContext) (int64, float64) {
	adjPct := 0.0

	// User risk: high cancellation rate → higher price
	if ctx.CancellationRate > 0.5 {
		adjPct += 1.0 // +1% for frequent cancellers
	} else if ctx.CancellationRate > 0.3 {
		adjPct += 0.5 // +0.5% for moderate cancellers
	}

	// Trust score: low trust → higher price
	if ctx.TrustScore < 30 {
		adjPct += 0.75
	} else if ctx.TrustScore > 70 {
		adjPct -= 0.5 // -0.5% for trusted users
	}

	// Abuse flags
	if ctx.AbuseFlags > 2 {
		adjPct += 1.0
	} else if ctx.AbuseFlags > 0 {
		adjPct += 0.25
	}

	// Delivery risk
	if ctx.DeliveryRiskScore > 0.7 {
		adjPct += 0.5
	} else if ctx.DeliveryRiskScore < 0.3 {
		adjPct -= 0.25
	}

	// Route risk
	if ctx.RouteRisk > 0.6 {
		adjPct += 0.25
	}

	adjCents := int64(float64(ctx.OrderPriceCents) * adjPct / 100.0)
	return adjCents, adjPct
}

// behaviorAdjustment adjusts price based on user's past insurance behavior.
func behaviorAdjustment(ctx *PricingContext) (int64, float64) {
	adjPct := 0.0

	// User always buys insurance → can charge slightly more (less elastic)
	if ctx.InsuranceBuyRate > 0.8 {
		adjPct += 0.5 // +0.5% (they'll buy anyway)
	} else if ctx.InsuranceBuyRate < 0.2 && ctx.PastInsuranceUsage > 2 {
		// User rarely buys → lower price to entice
		adjPct -= 0.5 // -0.5%
	}

	// Price sensitivity: high sensitivity → lower price
	if ctx.PriceSensitivity > 0.7 {
		adjPct -= 0.5
	} else if ctx.PriceSensitivity < 0.3 {
		adjPct += 0.25
	}

	// Account age: new users → slightly lower (first impression)
	if ctx.AccountAgeDays < 30 {
		adjPct -= 0.25
	}

	adjCents := int64(float64(ctx.OrderPriceCents) * adjPct / 100.0)
	return adjCents, adjPct
}

// contextAdjustment adjusts price based on time, demand, urgency.
func contextAdjustment(ctx *PricingContext) (int64, float64) {
	adjPct := 0.0

	// Rush hour premium
	if ctx.IsRushHour {
		adjPct += 0.25
	}

	// High demand → slight premium
	if ctx.LiveDemand > 0.7 {
		adjPct += 0.25
	} else if ctx.LiveDemand < 0.3 {
		adjPct -= 0.25
	}

	// Urgency: high urgency → user values protection more
	if ctx.UrgencyScore > 0.7 {
		adjPct += 0.5
	} else if ctx.UrgencyScore > 0.4 {
		adjPct += 0.25
	}

	adjCents := int64(float64(ctx.OrderPriceCents) * adjPct / 100.0)
	return adjCents, adjPct
}

// estimateBuyProbability estimates P(buy) based on price and user features.
func estimateBuyProbability(ctx *PricingContext, pricePct float64) float64 {
	// Base probability: ~60% at 2% price
	prob := 0.60

	// Price elasticity: higher price → lower probability
	if pricePct > 3.0 {
		prob -= 0.15
	} else if pricePct > 2.5 {
		prob -= 0.05
	} else if pricePct < 1.5 {
		prob += 0.10
	}

	// User buy rate: past behavior predicts future
	if ctx.InsuranceBuyRate > 0.7 {
		prob += 0.15
	} else if ctx.InsuranceBuyRate < 0.3 {
		prob -= 0.10
	}

	// Price sensitivity
	if ctx.PriceSensitivity > 0.7 {
		prob -= 0.10
	}

	// Urgency: urgent users more likely to buy
	if ctx.UrgencyScore > 0.6 {
		prob += 0.10
	}

	return math.Max(0.05, math.Min(0.95, prob))
}

// CalculateStaticPrice returns a fixed percentage price (control group).
func CalculateStaticPrice(ctx *PricingContext, cfg *PricingModelConfig) PriceResult {
	pct := cfg.StaticPricePercent
	priceCents := int64(float64(ctx.OrderPriceCents) * pct / 100.0)
	if priceCents < 50 {
		priceCents = 50
	}

	return PriceResult{
		PriceCents:     priceCents,
		BasePriceCents: priceCents,
		PricePercent:   pct,
		Adjustments:    PriceAdjustments{},
		BuyProbability: 0.60,
		Confidence:     1.0,
		Strategy:       StrategyStatic,
	}
}
