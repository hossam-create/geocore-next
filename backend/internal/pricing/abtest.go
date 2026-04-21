package pricing

import (
	"hash/fnv"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Pricing A/B Test Engine ──────────────────────────────────────────────────────

const (
	PricingExperiment = "pricing_strategy"

	VariantStatic = "static" // fixed 2%
	VariantRules  = "rules"  // rule-based dynamic
	VariantAI     = "ai"     // ML model prediction
)

// AssignPricingVariant assigns a user to a pricing A/B variant.
// Uses deterministic hash for consistent assignment.
func AssignPricingVariant(db *gorm.DB, userID uuid.UUID) string {
	// Check existing assignment
	var assignment PricingABAssignment
	if err := db.Where("user_id = ? AND experiment = ?", userID, PricingExperiment).
		First(&assignment).Error; err == nil {
		return assignment.Variant
	}

	// Deterministic assignment: 33% each
	h := fnv.New32a()
	h.Write([]byte(userID.String() + PricingExperiment))
	hash := h.Sum32()

	variant := VariantRules // default
	switch hash % 3 {
	case 0:
		variant = VariantStatic
	case 1:
		variant = VariantRules
	case 2:
		variant = VariantAI
	}

	// Persist
	assignment = PricingABAssignment{
		UserID:     userID,
		Experiment: PricingExperiment,
		Variant:    variant,
	}
	db.Create(&assignment)

	return variant
}

// GetPricingABResults computes metrics for each pricing variant.
func GetPricingABResults(db *gorm.DB) (*AdminPricingMetrics, error) {
	metrics := &AdminPricingMetrics{
		AttachRateByStrategy: make(map[string]float64),
		RevenueByStrategy:    make(map[string]int64),
		ModelUsage:           make(map[string]int64),
	}

	// Total priced events
	db.Model(&PricingEvent{}).Count(&metrics.TotalPriced)

	// Revenue
	var revenue struct{ Total int64 }
	db.Model(&PricingEvent{}).
		Select("COALESCE(SUM(price_cents), 0) as total").
		Where("did_buy = ?", true).
		Scan(&revenue)
	metrics.TotalRevenueCents = revenue.Total

	// Average price
	var avgPrice struct{ Avg int64 }
	db.Model(&PricingEvent{}).
		Select("COALESCE(AVG(price_cents), 0) as avg").
		Scan(&avgPrice)
	metrics.AvgPriceCents = avgPrice.Avg

	// Attach rate: % of priced events where user bought
	var boughtCount int64
	db.Model(&PricingEvent{}).Where("did_buy = ?", true).Count(&boughtCount)
	if metrics.TotalPriced > 0 {
		metrics.AttachRate = float64(boughtCount) / float64(metrics.TotalPriced)
	}

	// Average confidence
	var avgConf struct{ Avg float64 }
	db.Model(&PricingEvent{}).
		Select("COALESCE(AVG(confidence), 0) as avg").
		Scan(&avgConf)
	metrics.AvgConfidence = avgConf.Avg

	// Per-strategy metrics
	strategies := []string{VariantStatic, VariantRules, VariantAI}
	for _, s := range strategies {
		var count int64
		db.Model(&PricingEvent{}).Where("strategy = ?", s).Count(&count)
		metrics.ModelUsage[s] = count

		var stratBought int64
		db.Model(&PricingEvent{}).Where("strategy = ? AND did_buy = ?", s, true).Count(&stratBought)
		if count > 0 {
			metrics.AttachRateByStrategy[s] = float64(stratBought) / float64(count)
		}

		var stratRevenue struct{ Total int64 }
		db.Model(&PricingEvent{}).
			Select("COALESCE(SUM(price_cents), 0) as total").
			Where("strategy = ? AND did_buy = ?", s, true).
			Scan(&stratRevenue)
		metrics.RevenueByStrategy[s] = stratRevenue.Total
	}

	return metrics, nil
}

// UpdatePricingOutcome records whether the user actually bought after seeing a price.
func UpdatePricingOutcome(db *gorm.DB, userID, orderID uuid.UUID, didBuy, didCancel, claimFiled bool) {
	db.Model(&PricingEvent{}).
		Where("user_id = ? AND order_id = ?", userID, orderID).
		Updates(map[string]interface{}{
			"did_buy":     didBuy,
			"did_cancel":  didCancel,
			"claim_filed": claimFiled,
		})
}
