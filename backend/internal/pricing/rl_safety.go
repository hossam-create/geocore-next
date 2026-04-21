package pricing

import (
	"encoding/json"
	"hash/fnv"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── RL Safety Guardrails ──────────────────────────────────────────────────────────
//
// 1. Price boundaries: min 1%, max 4%
// 2. Kill switch: conversion drops > 8% → fallback to bandit/static
// 3. Shadow mode: RL decides but doesn't execute (log only)
// 4. Safe exploration: ε capped at 5-10% for risky segments
// 5. Rollout control: shadow → 5% → 25% → 50% → full
// 6. Session stickiness: same user, same session → stable price
// 7. Max session steps: cap re-offers at 3 per session

// RLSelectAction is the main entry point for the RL pricing engine.
// It applies all safety guardrails before returning a decision.
func RLSelectAction(db *gorm.DB, ctx *PricingContext) (*RLSelectionResult, error) {
	// 1. Load config
	config := loadRLConfig(db)

	// 2. Check kill switch
	if config.KillSwitchActive {
		return rlFallbackResult(ctx, config), nil
	}

	// 3. Build state
	state := BuildRLState(db, ctx)
	stateKey := state.Discretize()

	// 4. Check session stickiness
	cached := checkRLSessionStickiness(db, ctx.UserID, ctx.OrderID, config)
	if cached != nil {
		return cached, nil
	}

	// 5. Check max session steps
	if state.SessionStep >= config.MaxSessionSteps {
		return rlFallbackResult(ctx, config), nil
	}

	// 6. Determine if this request should use RL (rollout control)
	if !shouldUseRL(db, ctx.UserID, RolloutPhase(config.RolloutPhase)) {
		// Fall back to bandit pricing
		banditResult, err := SelectPrice(db, ctx)
		if err != nil {
			return rlFallbackResult(ctx, config), nil
		}
		return banditToRLResult(banditResult, ctx, true), nil
	}

	// 7. Safe exploration: reduce ε for high-risk segments
	epsilon := config.Epsilon
	if state.RiskScore > 0.7 || state.RefusalCount > 1 {
		epsilon = math.Min(epsilon, 0.05) // cap at 5% for risky users
	}

	// 8. Select action using Q-learning
	actionIdx, isExploration := SelectActionQ(stateKey, epsilon)
	action := GetActionFromIndex(actionIdx)

	// 9. Clamp price
	pricePercent := math.Max(config.MinPricePercent, math.Min(config.MaxPricePercent, action.PricePercent))
	priceCents := int64(float64(ctx.OrderPriceCents) * pricePercent / 100.0)
	if priceCents < 50 {
		priceCents = 50
	}

	// 10. Determine if shadow mode
	isShadow := config.RolloutPhase == string(RolloutShadow)

	// 11. Build result
	result := &RLSelectionResult{
		Action:           action,
		PriceCents:       priceCents,
		PricePercent:     pricePercent,
		UXVariant:        action.UXVariant,
		StateKey:         stateKey,
		Confidence:       computeRLConfidence(stateKey, actionIdx),
		IsExploration:    isExploration,
		IsShadow:         isShadow,
		SessionStep:      state.SessionStep,
		AnchorPrice:      int64(float64(priceCents) * 1.4),
		KillSwitchOn:     false,
		RecommendedLabel: buildRecommendedLabel(action, state),
	}

	// 12. Record transition (s, a) — reward filled later on feedback
	go recordRLTransition(db, ctx, state, stateKey, int(actionIdx), action, priceCents, config)

	return result, nil
}

// RLRecordFeedback records the outcome of an RL action and updates Q-table.
func RLRecordFeedback(db *gorm.DB, userID, orderID uuid.UUID, didBuy, didClaim, didChurn bool, claimCostCents float64) error {
	config := loadRLConfig(db)

	// Find the most recent transition for this user+order
	var transition RLTransition
	if err := db.Where("user_id = ? AND order_id = ? AND reward_total = 0",
		userID, orderID).Order("created_at DESC").First(&transition).Error; err != nil {
		return err
	}

	// Calculate reward
	reward := CalculateRLReward(transition.PriceCents, didBuy, didClaim, didChurn,
		claimCostCents, config.ChurnPenalty)

	// Build next state
	nextState := BuildRLState(db, &PricingContext{UserID: userID, OrderID: orderID})
	nextStateKey := nextState.Discretize()

	// Update Q-table online
	QUpdate(StateKey(transition.StateKey), ActionIndex(transition.ActionIndex),
		reward.Total, nextStateKey, config.LearningRate, config.DiscountFactor)

	// Update transition in DB
	db.Model(&RLTransition{}).Where("id = ?", transition.ID).Updates(map[string]interface{}{
		"reward_revenue":    reward.Revenue,
		"reward_claim_cost": reward.ClaimCost,
		"reward_churn":      reward.Churn,
		"reward_total":      reward.Total,
		"did_buy":           didBuy,
		"did_claim":         didClaim,
		"did_churn":         didChurn,
		"next_state_key":    string(nextStateKey),
		"is_terminal":       didBuy || didChurn || nextState.SessionStep >= config.MaxSessionSteps,
	})

	// Update session
	UpdateSessionAfterAction(db, userID, orderID, transition.ActionIndex, transition.PricePercent, didBuy)

	// Decay epsilon
	newEpsilon := config.Epsilon * config.EpsilonDecay
	if newEpsilon < config.MinEpsilon {
		newEpsilon = config.MinEpsilon
	}
	db.Model(&RLConfig{}).Where("id = ?", config.ID).Update("epsilon", newEpsilon)

	return nil
}

// ── Safety Helpers ────────────────────────────────────────────────────────────────

// shouldUseRL determines if a request should use RL based on rollout phase.
func shouldUseRL(db *gorm.DB, userID uuid.UUID, phase RolloutPhase) bool {
	switch phase {
	case RolloutShadow:
		return true // always decide (but don't execute)
	case RolloutCanary5:
		return hashPercent(userID) < 5
	case RolloutCanary25:
		return hashPercent(userID) < 25
	case RolloutCanary50:
		return hashPercent(userID) < 50
	case RolloutFull:
		return true
	default:
		return false // unknown phase → don't use RL
	}
}

// hashPercent returns a deterministic 0-100 value for a user ID.
func hashPercent(userID uuid.UUID) int {
	h := fnv.New32a()
	h.Write([]byte(userID.String()))
	return int(h.Sum32() % 100)
}

// checkRLSessionStickiness returns cached result if within cooldown.
func checkRLSessionStickiness(db *gorm.DB, userID, orderID uuid.UUID, config RLConfig) *RLSelectionResult {
	cooldown := time.Duration(config.SessionCooldownMinutes) * time.Minute

	var transition RLTransition
	if err := db.Where("user_id = ? AND order_id = ? AND created_at > ?",
		userID, orderID, time.Now().Add(-cooldown)).
		Order("created_at DESC").First(&transition).Error; err != nil {
		return nil
	}

	action := GetActionFromIndex(ActionIndex(transition.ActionIndex))
	return &RLSelectionResult{
		Action:       action,
		PriceCents:   transition.PriceCents,
		PricePercent: transition.PricePercent,
		UXVariant:    transition.UXVariant,
		// Algorithm: session_sticky
		Confidence: 1.0,
	}
}

// rlFallbackResult returns a safe static price when RL is disabled.
func rlFallbackResult(ctx *PricingContext, config RLConfig) *RLSelectionResult {
	priceCents := int64(float64(ctx.OrderPriceCents) * config.FallbackPricePercent / 100.0)
	if priceCents < 50 {
		priceCents = 50
	}
	return &RLSelectionResult{
		PriceCents:       priceCents,
		PricePercent:     config.FallbackPricePercent,
		UXVariant:        "standard",
		KillSwitchOn:     true,
		Confidence:       1.0,
		RecommendedLabel: "Standard protection",
	}
}

// banditToRLResult converts a bandit result to an RL result.
func banditToRLResult(bandit *BanditSelectionResult, ctx *PricingContext, isFallback bool) *RLSelectionResult {
	return &RLSelectionResult{
		PriceCents:   bandit.PriceCents,
		PricePercent: bandit.PricePercent,
		UXVariant:    "standard",
		Confidence:   bandit.Confidence,
		IsShadow:     isFallback,
		AnchorPrice:  bandit.AnchorPrice,
	}
}

// computeRLConfidence estimates confidence based on Q-table coverage.
func computeRLConfidence(stateKey StateKey, action ActionIndex) float64 {
	qVal := GetQValue(stateKey, action)
	// Simple heuristic: higher |Q| → more experience → higher confidence
	confidence := math.Min(math.Abs(qVal)/100.0, 1.0)
	if confidence < 0.1 {
		confidence = 0.1
	}
	return confidence
}

// buildRecommendedLabel creates a UX label based on action and state.
func buildRecommendedLabel(action RLAction, state *RLState) string {
	switch action.UXVariant {
	case "discount_badge":
		return "Special price based on your usage"
	case "social_proof":
		return "98% of users like you choose this protection"
	case "urgency":
		return "Secure your order now — limited protection slots"
	default:
		return "Recommended protection for you"
	}
}

// recordRLTransition stores the (s, a) part of a transition.
func recordRLTransition(db *gorm.DB, ctx *PricingContext, state *RLState,
	stateKey StateKey, actionIdx int, action RLAction, priceCents int64, config RLConfig) {

	stateJSON, _ := json.Marshal(state)
	session := loadOrCreateSession(db, ctx.UserID, ctx.OrderID)

	transition := RLTransition{
		UserID:       ctx.UserID,
		OrderID:      ctx.OrderID,
		SessionID:    session.EpisodeID,
		StateKey:     string(stateKey),
		StateJSON:    string(stateJSON),
		ActionIndex:  actionIdx,
		PricePercent: action.PricePercent,
		UXVariant:    action.UXVariant,
		PriceCents:   priceCents,
		EpisodeID:    session.EpisodeID,
	}
	db.Create(&transition)
}

// loadRLConfig loads the active RL configuration.
func loadRLConfig(db *gorm.DB) RLConfig {
	var config RLConfig
	if err := db.Where("is_active = ?", true).Order("created_at DESC").First(&config).Error; err != nil {
		return RLConfig{
			Algorithm:               "q_learning",
			LearningRate:            0.1,
			DiscountFactor:          0.95,
			Epsilon:                 0.1,
			EpsilonDecay:            0.995,
			MinEpsilon:              0.05,
			ChurnPenalty:            5.0,
			MinPricePercent:         1,
			MaxPricePercent:         4,
			ConversionDropThreshold: 0.08,
			SessionCooldownMinutes:  5,
			MaxSessionSteps:         3,
			RolloutPhase:            string(RolloutShadow),
			FallbackPricePercent:    2,
		}
	}

	// Load Q-table if available
	if config.QTableJSON != "" {
		_ = DeserializeQTable(config.QTableJSON)
	}

	return config
}

// ── RL Kill Switch ────────────────────────────────────────────────────────────────

// CheckRLConversionDrop monitors conversion and activates kill switch if needed.
func CheckRLConversionDrop(db *gorm.DB) (bool, float64) {
	config := loadRLConfig(db)

	now := time.Now()
	oneHourAgo := now.Add(-1 * time.Hour)
	oneDayAgo := now.Add(-24 * time.Hour)

	var recentTotal int64
	var recentBought int64
	db.Model(&RLTransition{}).Where("created_at > ?", oneHourAgo).Count(&recentTotal)
	db.Model(&RLTransition{}).Where("created_at > ? AND did_buy = ?", oneHourAgo, true).Count(&recentBought)

	var baselineTotal int64
	var baselineBought int64
	db.Model(&RLTransition{}).Where("created_at > ? AND created_at <= ?", oneDayAgo, oneHourAgo).Count(&baselineTotal)
	db.Model(&RLTransition{}).Where("created_at > ? AND created_at <= ? AND did_buy = ?", oneDayAgo, oneHourAgo, true).Count(&baselineBought)

	if recentTotal < 20 || baselineTotal < 50 {
		return false, 0
	}

	recentRate := float64(recentBought) / float64(recentTotal)
	baselineRate := float64(baselineBought) / float64(baselineTotal)

	if baselineRate-recentRate > config.ConversionDropThreshold {
		ActivateRLKillSwitch(db, "conversion_drop")
		return true, recentRate
	}

	return false, recentRate
}

// ActivateRLKillSwitch activates the RL kill switch.
func ActivateRLKillSwitch(db *gorm.DB, reason string) {
	db.Model(&RLConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"kill_switch_active": true,
			"updated_at":         time.Now(),
		})
}

// DeactivateRLKillSwitch deactivates the kill switch.
func DeactivateRLKillSwitch(db *gorm.DB) {
	db.Model(&RLConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"kill_switch_active": false,
			"updated_at":         time.Now(),
		})
}

// ── RL Dashboard ──────────────────────────────────────────────────────────────────

// GetRLDashboard returns the full RL dashboard for admin view.
func GetRLDashboard(db *gorm.DB) *RLDashboard {
	config := loadRLConfig(db)

	var totalTransitions int64
	db.Model(&RLTransition{}).Count(&totalTransitions)

	var totalEpisodes int64
	db.Table("rl_sessions").Where("is_complete = ?", true).Count(&totalEpisodes)

	var avgReward struct{ Avg float64 }
	db.Model(&RLTransition{}).Where("reward_total != 0").
		Select("COALESCE(AVG(reward_total), 0) as avg").Scan(&avgReward)

	var totalBought int64
	var totalChurned int64
	var totalClaimed int64
	db.Model(&RLTransition{}).Where("did_buy = ?", true).Count(&totalBought)
	db.Model(&RLTransition{}).Where("did_churn = ?", true).Count(&totalChurned)
	db.Model(&RLTransition{}).Where("did_claim = ?", true).Count(&totalClaimed)

	attachRate := 0.0
	churnRate := 0.0
	claimRate := 0.0
	if totalTransitions > 0 {
		attachRate = float64(totalBought) / float64(totalTransitions)
		churnRate = float64(totalChurned) / float64(totalTransitions)
		claimRate = float64(totalClaimed) / float64(totalTransitions)
	}

	// Top actions
	var topActions []ActionStats
	db.Model(&RLTransition{}).
		Select("price_percent, ux_variant, COUNT(*) as count, AVG(reward_total) as avg_reward, " +
			"CAST(SUM(CASE WHEN did_buy THEN 1 ELSE 0 END) AS FLOAT) / NULLIF(COUNT(*), 0) as conversion_rate").
		Group("price_percent, ux_variant").
		Order("avg_reward DESC").
		Limit(10).
		Scan(&topActions)

	return &RLDashboard{
		Config:           config,
		TotalTransitions: totalTransitions,
		TotalEpisodes:    totalEpisodes,
		AvgRewardPerStep: avgReward.Avg,
		AttachRate:       attachRate,
		ChurnRate:        churnRate,
		ClaimRate:        claimRate,
		KillSwitchActive: config.KillSwitchActive,
		RolloutPhase:     RolloutPhase(config.RolloutPhase),
		TopActions:       topActions,
	}
}

// SaveQTable persists the current Q-table to the active RL config.
func SaveQTable(db *gorm.DB) error {
	json, err := SerializeQTable()
	if err != nil {
		return err
	}
	db.Model(&RLConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"q_table_json": json,
			"updated_at":   time.Now(),
		})
	return nil
}

// AdvanceRolloutPhase moves to the next rollout phase.
func AdvanceRolloutPhase(db *gorm.DB) RolloutPhase {
	config := loadRLConfig(db)
	current := RolloutPhase(config.RolloutPhase)

	var next RolloutPhase
	switch current {
	case RolloutShadow:
		next = RolloutCanary5
	case RolloutCanary5:
		next = RolloutCanary25
	case RolloutCanary25:
		next = RolloutCanary50
	case RolloutCanary50:
		next = RolloutFull
	default:
		next = current
	}

	db.Model(&RLConfig{}).Where("id = ?", config.ID).
		Updates(map[string]interface{}{
			"rollout_phase": string(next),
			"updated_at":    time.Now(),
		})

	return next
}

// RollbackRolloutPhase moves back to the previous rollout phase.
func RollbackRolloutPhase(db *gorm.DB) RolloutPhase {
	config := loadRLConfig(db)
	current := RolloutPhase(config.RolloutPhase)

	var prev RolloutPhase
	switch current {
	case RolloutFull:
		prev = RolloutCanary50
	case RolloutCanary50:
		prev = RolloutCanary25
	case RolloutCanary25:
		prev = RolloutCanary5
	case RolloutCanary5:
		prev = RolloutShadow
	default:
		prev = current
	}

	db.Model(&RLConfig{}).Where("id = ?", config.ID).
		Updates(map[string]interface{}{
			"rollout_phase": string(prev),
			"updated_at":    time.Now(),
		})

	return prev
}
