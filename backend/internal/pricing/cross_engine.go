package pricing

import (
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Cross-System RL Engine ─────────────────────────────────────────────────────────
//
// Decision Pipeline:
//
//	1. Global guardrails (hard rules)
//	2. Build cross-state + encode
//	3. Multi-head inference (pricing + ranking + recs)
//	4. Consistency rules (don't boost + raise price simultaneously)
//	5. Per-head fallbacks (bandit / heuristic / CF)
//	6. Final clamp
//	7. Record transition + event

// CrossSelect is the main entry point for the cross-system RL engine.
func CrossSelect(db *gorm.DB, ctx *PricingContext) (*BundleAction, error) {
	config := loadCrossConfig(db)
	var guardrailsApplied []string

	// ── Step 1: Global guardrails ────────────────────────────────────────────
	if config.EmergencyModeActive {
		return crossEmergencyResult(ctx, config), nil
	}

	// Risk override
	if 1.0-ctx.TrustScore/100.0 > 0.9 {
		guardrailsApplied = append(guardrailsApplied, "high_risk_override")
		pct := config.MaxPricePercent
		return &BundleAction{
			PriceCents:    int64(float64(ctx.OrderPriceCents) * pct / 100.0),
			PricePercent:  pct,
			BoostScore:    20, // low boost for risky users
			RecIDs:        []string{},
			RecStrategy:   "popular",
			Source:        "rules",
			SourcePricing: "rules",
			SourceRanking: "heuristic",
			SourceRecs:    "popular",
			Confidence:    1.0,
		}, nil
	}

	// ── Step 2: Build cross state ────────────────────────────────────────────
	crossState := BuildCrossState(db, ctx)
	stateKey := crossState.Discretize()

	// ── Step 3: Cold start check ────────────────────────────────────────────
	if ctx.PastInsuranceUsage < 3 {
		guardrailsApplied = append(guardrailsApplied, "cold_start")
		return crossColdStartResult(db, ctx, crossState, config, guardrailsApplied)
	}

	// ── Step 4: Rollout check ────────────────────────────────────────────────
	isInRollout := hashPercent(ctx.UserID) < config.RolloutPercent
	isShadow := !isInRollout

	// ── Step 5: Multi-head inference ──────────────────────────────────────────
	epsilon := config.Epsilon
	if crossState.RiskScore > 0.7 || crossState.RefusalCount > 1 {
		epsilon = math.Min(epsilon, 0.05)
	}

	// Pricing head
	priceIdx, pricePct, priceExploration := PricingHeadPredict(stateKey, epsilon)
	priceCents := int64(float64(ctx.OrderPriceCents) * pricePct / 100.0)
	sourcePricing := "rl"

	// Ranking head
	boostIdx, boostScore, boostExploration := RankingHeadPredict(stateKey, epsilon)
	sourceRanking := "rl"

	// Recs head
	recsIdx, recStrategy, recsExploration := RecsHeadPredict(stateKey, epsilon)
	sourceRecs := "rl"

	isExploration := priceExploration || boostExploration || recsExploration

	// ── Step 6: Confidence check → per-head fallbacks ────────────────────────
	pricingConfidence := computeCrossConfidence(stateKey, "p", priceIdx)
	rankingConfidence := computeCrossConfidence(stateKey, "r", boostIdx)
	recsConfidence := computeCrossConfidence(stateKey, "c", recsIdx)

	if pricingConfidence < config.ConfidenceThreshold {
		priceCents, pricePct, sourcePricing = FallbackPricing(db, ctx)
		guardrailsApplied = append(guardrailsApplied, "pricing_fallback")
	}

	if rankingConfidence < config.ConfidenceThreshold {
		boostScore, sourceRanking = FallbackRanking(crossState)
		guardrailsApplied = append(guardrailsApplied, "ranking_fallback")
	}

	if recsConfidence < config.ConfidenceThreshold {
		_, sourceRecs = FallbackRecs(db, crossState.CategoryPath, 5)
		recStrategy = sourceRecs
		guardrailsApplied = append(guardrailsApplied, "recs_fallback")
	}

	// ── Step 7: Consistency rules ────────────────────────────────────────────
	// Don't raise price AND boost simultaneously (inconsistent signal to user)
	if pricePct > config.HighPriceThreshold && boostScore > config.MaxBoostWithHighPrice {
		boostScore = config.MaxBoostWithHighPrice
		guardrailsApplied = append(guardrailsApplied, "consistency_price_boost")
	}

	// Don't show urgency nudge if user already refused twice
	if crossState.RefusalCount >= 2 {
		recStrategy = "popular" // safe, non-aggressive
		guardrailsApplied = append(guardrailsApplied, "consistency_refusal_recs")
	}

	// ── Step 8: Price clamp ──────────────────────────────────────────────────
	clamped := false
	if pricePct < config.MinPricePercent {
		pricePct = config.MinPricePercent
		priceCents = int64(float64(ctx.OrderPriceCents) * pricePct / 100.0)
		clamped = true
	}
	if pricePct > config.MaxPricePercent {
		pricePct = config.MaxPricePercent
		priceCents = int64(float64(ctx.OrderPriceCents) * pricePct / 100.0)
		clamped = true
	}
	if priceCents < 50 {
		priceCents = 50
		clamped = true
	}
	if clamped {
		guardrailsApplied = append(guardrailsApplied, "price_clamped")
	}

	// ── Step 9: Build bundle action ──────────────────────────────────────────
	recIDs := []string{} // In production: query from DB based on strategy
	overallConfidence := (pricingConfidence + rankingConfidence + recsConfidence) / 3.0

	source := "rl"
	if sourcePricing != "rl" && sourceRanking != "rl" && sourceRecs != "rl" {
		source = "fallback"
	} else if sourcePricing != "rl" || sourceRanking != "rl" || sourceRecs != "rl" {
		source = "blend"
	}

	action := &BundleAction{
		PriceCents:    priceCents,
		PricePercent:  pricePct,
		BoostScore:    boostScore,
		RecIDs:        recIDs,
		RecStrategy:   recStrategy,
		Source:        source,
		Confidence:    overallConfidence,
		IsExploration: isExploration,
		IsShadow:      isShadow,
		SourcePricing: sourcePricing,
		SourceRanking: sourceRanking,
		SourceRecs:    sourceRecs,
		UXVariant:     selectUXVariant(crossState, pricePct),
		AnchorPrice:   int64(float64(priceCents) * 1.4),
	}

	// ── Step 10: Record ──────────────────────────────────────────────────────
	go recordCrossTransition(db, ctx, crossState, stateKey, action, config)
	go recordCrossEvent(db, ctx, action, guardrailsApplied)

	return action, nil
}

// ── Cross Feedback ──────────────────────────────────────────────────────────────────

// CrossRecordFeedback processes feedback and updates ALL Q-tables.
func CrossRecordFeedback(db *gorm.DB, userID, orderID uuid.UUID, fb CrossFeedback) error {
	config := loadCrossConfig(db)

	// Find most recent transition
	var transition CrossTransition
	if err := db.Where("user_id = ? AND order_id = ? AND reward_total = 0",
		userID, orderID).Order("created_at DESC").First(&transition).Error; err != nil {
		return err
	}

	// Calculate reward components
	reward := CalculateCrossReward(CrossRewardInput{
		PriceCents:    transition.PriceCents,
		DidBuy:        fb.DidBuy,
		DidClick:      fb.DidClick,
		DidClaim:      fb.DidClaim,
		DidChurn:      fb.DidChurn,
		ClaimCostCents: fb.ClaimCostCents,
	}, config)

	// Build next state key
	nextState := BuildCrossState(db, &PricingContext{UserID: userID, OrderID: orderID})
	nextKey := nextState.Discretize()

	// Update per-head Q-tables
	stateKey := CrossStateKey(transition.StateKey)
	CrossPricingQUpdate(stateKey, 0, reward.RewardTotal, config.LearningRate, config.DiscountFactor, nextKey)
	CrossRankingQUpdate(stateKey, 0, reward.RewardTotal, config.LearningRate, config.DiscountFactor, nextKey)
	CrossRecsQUpdate(stateKey, 0, reward.RewardTotal, config.LearningRate, config.DiscountFactor, nextKey)

	// Update transition in DB
	db.Model(&CrossTransition{}).Where("id = ?", transition.ID).Updates(map[string]interface{}{
		"reward_gmv":        reward.RewardGMV,
		"reward_ctr":        reward.RewardCTR,
		"reward_claim_cost": reward.RewardClaimCost,
		"reward_churn":      reward.RewardChurn,
		"reward_total":      reward.RewardTotal,
		"did_buy":           fb.DidBuy,
		"did_click":         fb.DidClick,
		"did_claim":         fb.DidClaim,
		"did_churn":         fb.DidChurn,
		"next_state_key":    string(nextKey),
		"is_terminal":       fb.DidBuy || fb.DidChurn,
	})

	// Also feed back to hybrid engine
	_ = ProcessHybridFeedback(db, userID, orderID, HybridFeedback{
		OrderID:        fb.OrderID,
		DidBuy:         fb.DidBuy,
		DidClaim:       fb.DidClaim,
		DidChurn:       fb.DidChurn,
		ClaimCostCents: fb.ClaimCostCents,
	})

	return nil
}

// ── Cross Reward ──────────────────────────────────────────────────────────────────

type CrossRewardInput struct {
	PriceCents     int64
	DidBuy         bool
	DidClick       bool
	DidClaim       bool
	DidChurn       bool
	ClaimCostCents float64
}

type CrossRewardOutput struct {
	RewardGMV       float64 `json:"reward_gmv"`
	RewardCTR       float64 `json:"reward_ctr"`
	RewardClaimCost float64 `json:"reward_claim_cost"`
	RewardChurn     float64 `json:"reward_churn"`
	RewardTotal     float64 `json:"reward_total"`
}

// CalculateCrossReward computes the multi-objective reward.
// r = α·GMV + β·CTR - γ·ClaimCost - δ·Churn
func CalculateCrossReward(input CrossRewardInput, config CrossConfig) CrossRewardOutput {
	out := CrossRewardOutput{}

	// GMV: price × purchase
	if input.DidBuy {
		out.RewardGMV = float64(input.PriceCents) / 100.0
	}

	// CTR: click signal
	if input.DidClick {
		out.RewardCTR = 1.0 // normalized click value
	}

	// Claim cost
	if input.DidClaim {
		out.RewardClaimCost = input.ClaimCostCents / 100.0
	}

	// Churn penalty
	if input.DidChurn {
		out.RewardChurn = 5.0 // lifetime value penalty
	}

	// Weighted total
	out.RewardTotal = config.WeightGMV*out.RewardGMV +
		config.WeightCTR*out.RewardCTR -
		config.WeightClaimCost*out.RewardClaimCost -
		config.WeightChurn*out.RewardChurn

	return out
}

// ── Helpers ────────────────────────────────────────────────────────────────────────

func crossEmergencyResult(ctx *PricingContext, config CrossConfig) *BundleAction {
	pct := config.FallbackPricePercent
	return &BundleAction{
		PriceCents:    int64(float64(ctx.OrderPriceCents) * pct / 100.0),
		PricePercent:  pct,
		BoostScore:    config.FallbackBoostScore,
		RecIDs:        []string{},
		RecStrategy:   config.FallbackRecStrategy,
		Source:        "emergency",
		SourcePricing: "rules",
		SourceRanking: "heuristic",
		SourceRecs:    "popular",
		Confidence:    1.0,
	}
}

func crossColdStartResult(db *gorm.DB, ctx *PricingContext, state *CrossState, config CrossConfig, guardrails []string) (*BundleAction, error) {
	priceCents, pricePct, sourcePricing := FallbackPricing(db, ctx)
	boostScore, sourceRanking := FallbackRanking(state)
	recIDs, sourceRecs := FallbackRecs(db, state.CategoryPath, 5)

	return &BundleAction{
		PriceCents:     priceCents,
		PricePercent:   pricePct,
		BoostScore:     boostScore,
		RecIDs:         recIDs,
		RecStrategy:    sourceRecs,
		Source:         "fallback",
		SourcePricing:  sourcePricing,
		SourceRanking:  sourceRanking,
		SourceRecs:     sourceRecs,
		Confidence:     0.5,
		AnchorPrice:    int64(float64(priceCents) * 1.4),
	}, nil
}

func computeCrossConfidence(stateKey CrossStateKey, head string, actionIdx int) float64 {
	crossQMu.RLock()
	defer crossQMu.RUnlock()

	var table QTable
	switch head {
	case "p":
		table = crossPricingQ
	case "r":
		table = crossRankingQ
	case "c":
		table = crossRecsQ
	}

	sk := StateKey(string(stateKey) + "_" + head)
	if actions, ok := table[sk]; ok {
		if v, ok2 := actions[ActionIndex(actionIdx)]; ok2 {
			conf := math.Min(math.Abs(v)/100.0, 1.0)
			if conf < 0.1 {
				return 0.1
			}
			return conf
		}
	}
	return 0.1 // low confidence for unseen states
}

func selectUXVariant(state *CrossState, pricePct float64) string {
	if pricePct > 3.0 {
		return "discount_badge"
	}
	if state.DemandScore > 0.7 {
		return "urgency"
	}
	if state.UserSegment == "vip" {
		return "social_proof"
	}
	return "standard"
}

func recordCrossTransition(db *gorm.DB, ctx *PricingContext, state *CrossState,
	stateKey CrossStateKey, action *BundleAction, config CrossConfig) {

	stateJSON, _ := json.Marshal(state)
	recIDsJSON, _ := json.Marshal(action.RecIDs)
	session := loadOrCreateSession(db, ctx.UserID, ctx.OrderID)

	transition := CrossTransition{
		UserID:       ctx.UserID,
		OrderID:      ctx.OrderID,
		SessionID:    session.EpisodeID,
		StateKey:     string(stateKey),
		StateJSON:    string(stateJSON),
		PriceCents:   action.PriceCents,
		PricePercent: action.PricePercent,
		BoostScore:   action.BoostScore,
		RecIDsJSON:   string(recIDsJSON),
		RecStrategy:  action.RecStrategy,
		EpisodeID:    session.EpisodeID,
	}
	db.Create(&transition)
}

func recordCrossEvent(db *gorm.DB, ctx *PricingContext, action *BundleAction, guardrails []string) {
	guardrailsJSON, _ := json.Marshal(guardrails)
	recIDsJSON, _ := json.Marshal(action.RecIDs)

	event := CrossEvent{
		UserID:         ctx.UserID,
		OrderID:        ctx.OrderID,
		SourcePricing:  action.SourcePricing,
		SourceRanking:  action.SourceRanking,
		SourceRecs:     action.SourceRecs,
		PriceCents:     action.PriceCents,
		PricePercent:   action.PricePercent,
		BoostScore:     action.BoostScore,
		RecIDsJSON:     string(recIDsJSON),
		Confidence:     action.Confidence,
		IsShadow:       action.IsShadow,
		GuardrailsJSON: string(guardrailsJSON),
	}
	db.Create(&event)
}

// loadCrossConfig loads the active cross-system config.
func loadCrossConfig(db *gorm.DB) CrossConfig {
	var config CrossConfig
	if err := db.Where("is_active = ?", true).Order("created_at DESC").First(&config).Error; err != nil {
		return CrossConfig{
			WeightGMV:              0.4,
			WeightCTR:              0.2,
			WeightClaimCost:        0.2,
			WeightChurn:            0.2,
			LearningRate:           0.1,
			DiscountFactor:         0.95,
			Epsilon:                0.1,
			ConfidenceThreshold:    0.6,
			MinPricePercent:        1,
			MaxPricePercent:        4,
			MaxBoostWithHighPrice:  30,
			HighPriceThreshold:     3.0,
			ConversionDropThreshold: 0.08,
			SessionCooldownMinutes: 5,
			MaxSessionSteps:        3,
			AnomalyDetectionEnabled: true,
			RolloutPercent:         5,
			FallbackPricePercent:   2,
			FallbackBoostScore:     50,
			FallbackRecStrategy:    "popular",
		}
	}

	if config.QTableJSON != "" {
		_ = DeserializeCrossQTables(config.QTableJSON)
	}

	return config
}

// SaveCrossQTables persists Q-tables to DB.
func SaveCrossQTables(db *gorm.DB) error {
	json, err := SerializeCrossQTables()
	if err != nil {
		return err
	}
	db.Model(&CrossConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{
			"q_table_json": json,
			"updated_at":   time.Now(),
		})
	return nil
}
