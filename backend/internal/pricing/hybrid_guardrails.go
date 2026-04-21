package pricing

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Hybrid Guardrails ────────────────────────────────────────────────────────────
//
// These are HARD rules that override any AI decision.
// They run BEFORE the RL/Bandit engines and can short-circuit the entire pipeline.
//
// Guardrails:
// 1. Price boundaries: 1% ≤ price ≤ 4%
// 2. Risk override: risk > 0.9 → force max price
// 3. Trust override: trust < 20 → force min price
// 4. Claim protection: claim_rate > 0.3 → force higher price
// 5. User fairness: same user, same session → same price
// 6. Emergency mode: global switch → static pricing
// 7. Anomaly detection: unusual patterns → fallback
// 8. Cold start: new user → bandit (not RL)
// 9. Model lag: if RL response too slow → bandit

// GuardrailResult is the output of the guardrail check.
type GuardrailResult struct {
	Override     bool    `json:"override"`      // true = skip AI, use this price
	PriceCents   int64   `json:"price_cents"`
	PricePercent float64 `json:"price_percent"`
	RuleName     string  `json:"rule_name"`     // which guardrail fired
	Reason       string  `json:"reason"`        // human-readable reason
}

// ApplyHardRules checks all guardrails and returns an override if any fires.
func ApplyHardRules(db *gorm.DB, ctx *PricingContext) *GuardrailResult {
	config := loadHybridConfig(db)

	// ── 1. Emergency mode ──────────────────────────────────────────────────
	if config.EmergencyModeActive {
		return &GuardrailResult{
			Override:     true,
			PricePercent: config.EmergencyPricePercent,
			PriceCents:   int64(float64(ctx.OrderPriceCents) * config.EmergencyPricePercent / 100.0),
			RuleName:     "emergency_mode",
			Reason:       "Emergency mode active — using static pricing",
		}
	}

	// ── 2. Risk override ────────────────────────────────────────────────────
	riskScore := 1.0 - ctx.TrustScore/100.0
	if riskScore > 0.9 {
		pct := config.MaxPricePercent
		return &GuardrailResult{
			Override:     true,
			PricePercent: pct,
			PriceCents:   int64(float64(ctx.OrderPriceCents) * pct / 100.0),
			RuleName:     "high_risk_override",
			Reason:       "User risk score > 0.9 — forcing maximum price",
		}
	}

	// ── 3. Trust override ────────────────────────────────────────────────────
	if ctx.TrustScore < 20 {
		pct := config.MinPricePercent
		return &GuardrailResult{
			Override:     true,
			PricePercent: pct,
			PriceCents:   int64(float64(ctx.OrderPriceCents) * pct / 100.0),
			RuleName:     "low_trust_override",
			Reason:       "User trust < 20 — forcing minimum price for fairness",
		}
	}

	// ── 4. Claim protection ──────────────────────────────────────────────────
	if ctx.CancellationRate > 0.3 {
		// Force higher price to cover expected claims
		pct := math.Min(config.MaxPricePercent, 3.0) // at least 3%
		return &GuardrailResult{
			Override:     true,
			PricePercent: pct,
			PriceCents:   int64(float64(ctx.OrderPriceCents) * pct / 100.0),
			RuleName:     "high_claim_rate",
			Reason:       "Cancellation rate > 30% — forcing higher price for claim protection",
		}
	}

	// ── 5. User fairness (session stickiness) ────────────────────────────────
	cooldown := time.Duration(config.SessionCooldownMinutes) * time.Minute
	var event HybridEvent
	if err := db.Where("user_id = ? AND created_at > ?", ctx.UserID, time.Now().Add(-cooldown)).
		Order("created_at DESC").First(&event).Error; err == nil {
		return &GuardrailResult{
			Override:     true,
			PricePercent: event.PricePercent,
			PriceCents:   event.PriceCents,
			RuleName:     "session_stickiness",
			Reason:       "Same user within session cooldown — keeping previous price",
		}
	}

	// ── 6. Cold start check ──────────────────────────────────────────────────
	// New users should NOT go through RL — use bandit instead
	// This is handled by the engine, not as an override

	// ── 7. Anomaly detection ──────────────────────────────────────────────────
	if config.AnomalyDetectionEnabled {
		if isAnomalous(db, ctx) {
			pct := config.EmergencyPricePercent // safe fallback
			return &GuardrailResult{
				Override:     true,
				PricePercent: pct,
				PriceCents:   int64(float64(ctx.OrderPriceCents) * pct / 100.0),
				RuleName:     "anomaly_detected",
				Reason:       "Anomalous pricing pattern detected — using safe price",
			}
		}
	}

	// No guardrail fired → let AI decide
	return nil
}

// isAnomalous checks for unusual pricing patterns.
func isAnomalous(db *gorm.DB, ctx *PricingContext) bool {
	// Signal 1: User has seen very different prices recently
	var recentEvents []HybridEvent
	db.Where("user_id = ? AND created_at > ?", ctx.UserID, time.Now().Add(-1*time.Hour)).
		Order("created_at DESC").Limit(5).Find(&recentEvents)

	if len(recentEvents) >= 3 {
		// Check price variance
		var sum, sumSq float64
		for _, e := range recentEvents {
			pct := e.PricePercent
			sum += pct
			sumSq += pct * pct
		}
		n := float64(len(recentEvents))
		mean := sum / n
		variance := sumSq/n - mean*mean
		stdDev := math.Sqrt(variance)

		// If standard deviation > 1%, that's anomalous
		if stdDev > 1.0 {
			return true
		}
	}

	// Signal 2: Order value is extreme outlier
	if ctx.OrderPriceCents > 500000 || ctx.OrderPriceCents < 100 {
		return true
	}

	return false
}

// ── Hybrid Config Loader ──────────────────────────────────────────────────────────

func loadHybridConfig(db *gorm.DB) HybridConfig {
	var config HybridConfig
	if err := db.Where("is_active = ?", true).Order("created_at DESC").First(&config).Error; err != nil {
		return HybridConfig{
			RLConfidenceThreshold:   0.6,
			BlendWeightRL:           0.7,
			EnableSoftBlend:         false,
			MinPricePercent:         1,
			MaxPricePercent:         4,
			EmergencyModeActive:     false,
			EmergencyPricePercent:   2,
			ConversionDropThreshold: 0.08,
			SessionCooldownMinutes:  5,
			MaxSessionSteps:         3,
			AnomalyDetectionEnabled: true,
			RolloutPercent:          5,
		}
	}
	return config
}

// ── Hybrid Event Recording ────────────────────────────────────────────────────────

func recordHybridEvent(db *gorm.DB, ctx *PricingContext, decision *HybridDecision) {
	guardrailsJSON, _ := json.Marshal(decision.GuardrailsApplied)

	event := HybridEvent{
		UserID:         ctx.UserID,
		OrderID:        ctx.OrderID,
		Source:         string(decision.Source),
		PriceCents:     decision.PriceCents,
		PricePercent:   decision.PricePercent,
		Confidence:     decision.Confidence,
		IsExploration:  decision.IsExploration,
		IsShadow:       decision.IsShadow,
		UXVariant:      decision.UXVariant,
		GuardrailsJSON: string(guardrailsJSON),
	}
	db.Create(&event)
}

// ── Hybrid Feedback Processing ────────────────────────────────────────────────────

// ProcessHybridFeedback updates ALL engines (RL + Bandit) with the outcome.
func ProcessHybridFeedback(db *gorm.DB, userID, orderID uuid.UUID, feedback HybridFeedback) error {
	orderUUID, _ := uuid.Parse(feedback.OrderID)

	// 1. Update hybrid event
	db.Model(&HybridEvent{}).
		Where("user_id = ? AND order_id = ?", userID, orderUUID).
		Order("created_at DESC").
		Limit(1).
		Updates(map[string]interface{}{
			"did_buy": feedback.DidBuy,
		})

	// 2. Feed back to Bandit
	claimCost := feedback.ClaimCostCents
	_ = RecordBanditOutcome(db, userID, orderUUID, feedback.DidBuy, claimCost)

	// 3. Feed back to RL
	_ = RLRecordFeedback(db, userID, orderUUID, feedback.DidBuy, feedback.DidClaim, feedback.DidChurn, feedback.ClaimCostCents)

	// 4. Feed back to A/B tracking
	UpdatePricingOutcome(db, userID, orderUUID, feedback.DidBuy, feedback.DidChurn, feedback.DidClaim)

	return nil
}

// ── Hybrid Dashboard ──────────────────────────────────────────────────────────────

func GetHybridDashboard(db *gorm.DB) *HybridDashboard {
	config := loadHybridConfig(db)

	var totalDecisions int64
	db.Model(&HybridEvent{}).Count(&totalDecisions)

	// Decisions by source
	bySource := make(map[string]int64)
	sources := []string{"rules", "rl", "bandit", "blend", "emergency", "session", "shadow"}
	for _, s := range sources {
		var count int64
		db.Model(&HybridEvent{}).Where("source = ?", s).Count(&count)
		if count > 0 {
			bySource[s] = count
		}
	}

	// Average confidence
	var avgConf struct{ Avg float64 }
	db.Model(&HybridEvent{}).Select("COALESCE(AVG(confidence), 0) as avg").Scan(&avgConf)

	// Attach rate
	var boughtCount int64
	db.Model(&HybridEvent{}).Where("did_buy = ?", true).Count(&boughtCount)
	attachRate := 0.0
	if totalDecisions > 0 {
		attachRate = float64(boughtCount) / float64(totalDecisions)
	}

	// Revenue
	var revenue struct{ Total float64 }
	db.Model(&HybridEvent{}).
		Select("COALESCE(SUM(price_cents), 0) as total").
		Where("did_buy = ?", true).Scan(&revenue)

	// Top guardrails
	var guardrailStats []GuardrailStats
	type guardRow struct {
		Name  string
		Count int64
	}
	var rows []guardRow
	db.Raw(`
		SELECT guardrail_name as name, COUNT(*) as count
	 FROM (
	   SELECT json_array_elements_text(guardrails_json::json) as guardrail_name
	   FROM hybrid_events
	   WHERE guardrails_json IS NOT NULL AND guardrails_json != '[]'
	 ) sub
	 GROUP BY name
	 ORDER BY count DESC
	 LIMIT 5`).Scan(&rows)

	for _, r := range rows {
		pct := 0.0
		if totalDecisions > 0 {
			pct = float64(r.Count) / float64(totalDecisions) * 100
		}
		guardrailStats = append(guardrailStats, GuardrailStats{
			Name:    r.Name,
			Count:   r.Count,
			Percent: pct,
		})
	}

	return &HybridDashboard{
		Config:            config,
		TotalDecisions:    totalDecisions,
		DecisionsBySource: bySource,
		AvgConfidence:     avgConf.Avg,
		AttachRate:        attachRate,
		TotalRevenue:      revenue.Total / 100.0,
		EmergencyModeActive: config.EmergencyModeActive,
		RolloutPercent:    config.RolloutPercent,
		TopGuardrails:     guardrailStats,
	}
}

// ── Emergency Mode ────────────────────────────────────────────────────────────────

func ActivateEmergencyMode(db *gorm.DB) {
	db.Model(&HybridConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"emergency_mode_active": true,
			"updated_at":           time.Now(),
		})
	// Also activate RL and Bandit kill switches for safety
	ActivateRLKillSwitch(db, "emergency_mode")
	ActivateBanditKillSwitch(db, "emergency_mode")
}

func DeactivateEmergencyMode(db *gorm.DB) {
	db.Model(&HybridConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"emergency_mode_active": false,
			"updated_at":           time.Now(),
		})
	DeactivateRLKillSwitch(db)
	DeactivateBanditKillSwitch(db)
}
