package pricing

import (
	"fmt"
	"math"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Decision Engine ──────────────────────────────────────────────────────────────
//
// IF model confidence > threshold → use AI price
// ELSE → fallback to rule engine
//
// The decision also considers the user's A/B variant:
// - static → always use fixed 2%
// - rules  → always use rule engine
// - ai     → use AI when confident, rules otherwise

// CalculateDynamicPrice is the main entry point for dynamic insurance pricing.
func CalculateDynamicPrice(db *gorm.DB, ctx *PricingContext) (*PriceResult, error) {
	// 1. Load active model config
	cfg := loadModelConfig(db)

	// 2. Determine user's A/B variant
	variant := getPricingVariant(db, ctx.UserID)

	// 3. Route to the appropriate pricing strategy
	var result PriceResult

	switch variant {
	case "static":
		result = CalculateStaticPrice(ctx, cfg)
	case "rules":
		result = CalculateRuleBasedPrice(ctx, cfg)
	case "ai":
		result = calculateAIPrice(ctx, cfg)
	default:
		result = CalculateRuleBasedPrice(ctx, cfg)
	}

	// 4. Anti-abuse: adjust price for abusers
	result = applyAntiAbusePricing(ctx, &result)

	// 5. Clamp final price
	result.PricePercent = math.Max(cfg.MinPricePercent, math.Min(cfg.MaxPricePercent, result.PricePercent))
	result.PriceCents = int64(float64(ctx.OrderPriceCents) * result.PricePercent / 100.0)
	if result.PriceCents < 50 {
		result.PriceCents = 50
	}

	// 6. Set anchor price for UX anchoring
	if result.AnchorPrice == 0 {
		result.AnchorPrice = int64(float64(result.PriceCents) * 1.4)
	}

	// 7. Record pricing event
	go recordPricingEvent(db, ctx, &result, variant)

	return &result, nil
}

// calculateAIPrice uses the ML model with rule-based fallback.
func calculateAIPrice(ctx *PricingContext, cfg *PricingModelConfig) PriceResult {
	fv := ctx.BuildFeatureVector()

	// Try AI prediction
	buyProb, priceMultiplier, confidence, err := Predict(fv.Features)

	if err != nil || confidence < cfg.ConfidenceThreshold {
		// Fallback to rules
		ruleResult := CalculateRuleBasedPrice(ctx, cfg)
		ruleResult.Strategy = StrategyRules
		if err != nil {
			ruleResult.Confidence = 0.5 // low confidence due to model failure
		}
		return ruleResult
	}

	// AI prediction successful — compute optimal price
	basePct := cfg.BasePricePercent
	optimalPct := basePct + priceMultiplier*100.0 // convert multiplier to percentage points

	// Clamp
	optimalPct = math.Max(cfg.MinPricePercent, math.Min(cfg.MaxPricePercent, optimalPct))

	basePriceCents := int64(float64(ctx.OrderPriceCents) * basePct / 100.0)
	finalPriceCents := int64(float64(ctx.OrderPriceCents) * optimalPct / 100.0)
	if finalPriceCents < 50 {
		finalPriceCents = 50
	}

	// Compute adjustments relative to base
	aiAdjCents := finalPriceCents - basePriceCents

	// Also compute rule-based price for comparison
	ruleResult := CalculateRuleBasedPrice(ctx, cfg)

	return PriceResult{
		PriceCents:     finalPriceCents,
		BasePriceCents: basePriceCents,
		PricePercent:   optimalPct,
		Adjustments: PriceAdjustments{
			RiskAdj:     ruleResult.Adjustments.RiskAdj,
			BehaviorAdj: ruleResult.Adjustments.BehaviorAdj,
			ContextAdj:  ruleResult.Adjustments.ContextAdj,
			AIAdj:       aiAdjCents,
		},
		BuyProbability: buyProb,
		Confidence:     confidence,
		Strategy:       StrategyAI,
		AnchorPrice:    int64(float64(finalPriceCents) * 1.4),
		Features:       fv,
	}
}

// loadModelConfig fetches the active pricing model config from DB.
func loadModelConfig(db *gorm.DB) *PricingModelConfig {
	var cfg PricingModelConfig
	if err := db.Where("is_active = ?", true).Order("created_at DESC").First(&cfg).Error; err != nil {
		// Fallback to defaults
		return &PricingModelConfig{
			BasePricePercent:    DefaultBasePricePercent,
			MinPricePercent:     DefaultMinPricePercent,
			MaxPricePercent:     DefaultMaxPricePercent,
			StaticPricePercent:  DefaultStaticPricePercent,
			ConfidenceThreshold: 0.7,
		}
	}

	// If model JSON exists, try to load it
	if cfg.ModelJSON != "" {
		_ = LoadModelFromJSON([]byte(cfg.ModelJSON))
	}

	return &cfg
}

// getPricingVariant returns the user's A/B variant for pricing.
func getPricingVariant(db *gorm.DB, userID uuid.UUID) string {
	var assignment PricingABAssignment
	if err := db.Where("user_id = ? AND experiment = ?", userID, "pricing_strategy").
		First(&assignment).Error; err == nil {
		return assignment.Variant
	}

	// Default: rules (safe middle ground)
	return "rules"
}

// recordPricingEvent logs the pricing decision for training and tracking.
func recordPricingEvent(db *gorm.DB, ctx *PricingContext, result *PriceResult, variant string) {
	event := PricingEvent{
		UserID:         ctx.UserID,
		OrderID:        ctx.OrderID,
		Strategy:       result.Strategy,
		PriceCents:     result.PriceCents,
		BuyProbability: result.BuyProbability,
		Confidence:     result.Confidence,
		ABVariant:      variant,
	}
	db.Create(&event)
}

// ── Expected Revenue Calculation ──────────────────────────────────────────────────

// CalculateExpectedRevenue computes: revenue = price * P_buy - Expected_cost
func CalculateExpectedRevenue(priceCents int64, buyProb float64, expectedCostCents int64) float64 {
	return float64(priceCents)*buyProb - float64(expectedCostCents)
}

// FindOptimalPrice searches for the price that maximizes expected revenue.
func FindOptimalPrice(ctx *PricingContext, cfg *PricingModelConfig) int64 {
	bestPrice := int64(0)
	bestRevenue := 0.0

	// Search over price range in 0.25% increments
	for pct := cfg.MinPricePercent; pct <= cfg.MaxPricePercent; pct += 0.25 {
		priceCents := int64(float64(ctx.OrderPriceCents) * pct / 100.0)
		buyProb := estimateDemandAtPrice(ctx, pct)
		expectedCost := estimateExpectedCost(ctx)

		revenue := CalculateExpectedRevenue(priceCents, buyProb, expectedCost)
		if revenue > bestRevenue {
			bestRevenue = revenue
			bestPrice = priceCents
		}
	}

	return bestPrice
}

// estimateDemandAtPrice estimates buy probability at a given price point.
func estimateDemandAtPrice(ctx *PricingContext, pricePct float64) float64 {
	// Simple demand curve: higher price → lower probability
	// Based on price elasticity
	baseProb := 0.70 // at 1% price
	elasticity := ctx.PriceSensitivity * 0.3

	priceRatio := pricePct / 1.0 // relative to 1% base
	prob := baseProb - elasticity*(priceRatio-1.0)

	return math.Max(0.05, math.Min(0.95, prob))
}

// estimateExpectedCost estimates the expected claim payout.
func estimateExpectedCost(ctx *PricingContext) int64 {
	// Expected cost = P(claim) * avg_payout
	claimProb := ctx.CancellationRate*0.3 + ctx.DeliveryRiskScore*0.2
	avgPayout := int64(float64(ctx.OrderPriceCents) * 0.15) // 15% avg refund

	return int64(float64(avgPayout) * claimProb)
}

// estimateBuyProbability is a convenience wrapper for the rules engine.
// This is the same function from rules.go but accessible from decision.go.
func estimateBuyProbabilityWrapper(ctx *PricingContext, pricePct float64) float64 {
	return estimateBuyProbability(ctx, pricePct)
}

// Ensure the function is available (suppress unused warning)
var _ = estimateBuyProbabilityWrapper

// Ensure fmt is used
var _ = fmt.Sprintf
