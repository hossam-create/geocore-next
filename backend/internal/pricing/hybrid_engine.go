package pricing

import (
	"encoding/json"
	"hash/fnv"
	"math"

	"gorm.io/gorm"
)

// ── Hybrid Pricing Engine ─────────────────────────────────────────────────────────
//
// Decision Pipeline:
//
//	1. Guardrails (hard rules) → override if any fires
//	2. RL model → predict price + confidence
//	3. Confidence gate → if < threshold, fallback to bandit
//	4. Soft blend (optional) → weighted average of RL + Bandit
//	5. Final clamp → enforce min/max price
//	6. Record event → full observability
//
// Feedback:
//
//	Both RL and Bandit learn from every outcome simultaneously.

// HybridSelect is the main entry point for the hybrid pricing engine.
func HybridSelect(db *gorm.DB, ctx *PricingContext) (*HybridDecision, error) {
	config := loadHybridConfig(db)
	var guardrailsApplied []string

	// ── Step 1: Guardrails ──────────────────────────────────────────────────
	rulesResult := ApplyHardRules(db, ctx)
	if rulesResult != nil && rulesResult.Override {
		guardrailsApplied = append(guardrailsApplied, rulesResult.RuleName)

		priceCents := rulesResult.PriceCents
		if priceCents < 50 {
			priceCents = 50
		}

		_ = json.Marshal

		decision := &HybridDecision{
			PriceCents:        priceCents,
			PricePercent:      rulesResult.PricePercent,
			AnchorPrice:       int64(float64(priceCents) * 1.4),
			Source:            SourceRules,
			Confidence:        1.0,
			GuardrailsApplied: guardrailsApplied,
			UXVariant:         "standard",
			RecommendedLabel:  "Standard protection",
			RulesOutput: &RulesSubDecision{
				PriceCents:   priceCents,
				PricePercent: rulesResult.PricePercent,
				RuleName:     rulesResult.RuleName,
			},
		}

		// Record even guardrail decisions
		go recordHybridEvent(db, ctx, decision)

		return decision, nil
	}

	// ── Step 2: Build RL State ──────────────────────────────────────────────
	rlState := BuildRLState(db, ctx)
	stateKey := rlState.Discretize()

	// ── Step 3: Cold start check ────────────────────────────────────────────
	// New users (< 3 insurance interactions) → skip RL, use bandit
	isColdStart := ctx.PastInsuranceUsage < 3

	// ── Step 4: Rollout check ────────────────────────────────────────────────
	// Only use hybrid for X% of traffic
	isInRollout := hashPercent(ctx.UserID) < config.RolloutPercent
	isShadow := !isInRollout // if not in rollout, still decide but mark as shadow

	// ── Step 5: RL Decision ──────────────────────────────────────────────────
	var rlOutput *RLSubDecision
	rlConfidence := 0.0

	if !isColdStart {
		actionIdx, isExploration := SelectActionQ(stateKey, loadRLConfig(db).Epsilon)
		action := GetActionFromIndex(actionIdx)
		qVal := GetQValue(stateKey, actionIdx)

		priceCents := int64(float64(ctx.OrderPriceCents) * action.PricePercent / 100.0)
		rlConfidence = computeRLConfidence(stateKey, actionIdx)

		rlOutput = &RLSubDecision{
			PriceCents:   priceCents,
			PricePercent: action.PricePercent,
			Confidence:   rlConfidence,
			QValue:       qVal,
			StateKey:     string(stateKey),
			ActionIndex:  int(actionIdx),
			UXVariant:    action.UXVariant,
		}

		_ = isExploration
	}

	// ── Step 6: Bandit Decision ──────────────────────────────────────────────
	var banditOutput *BanditSubDecision
	banditResult, err := SelectPrice(db, ctx)
	if err == nil && banditResult != nil {
		banditOutput = &BanditSubDecision{
			PriceCents:   banditResult.PriceCents,
			PricePercent: banditResult.PricePercent,
			SampleValue:  banditResult.SampleValue,
			Segment:      banditResult.Segment,
			ArmID:        banditResult.ArmID,
		}
	}

	// ── Step 7: Decision Logic ────────────────────────────────────────────────
	var finalPriceCents int64
	var finalPricePercent float64
	var source PricingSource
	var confidence float64
	var uxVariant string

	switch {
	// Cold start → bandit only
	case isColdStart:
		if banditOutput != nil {
			finalPriceCents = banditOutput.PriceCents
			finalPricePercent = banditOutput.PricePercent
			confidence = 0.5 // moderate confidence for bandit
			uxVariant = "standard"
		}
		source = SourceBandit
		guardrailsApplied = append(guardrailsApplied, "cold_start")

	// RL confidence is high enough → use RL
	case rlOutput != nil && rlConfidence >= config.RLConfidenceThreshold:
		if config.EnableSoftBlend && banditOutput != nil {
			// Soft blend: weighted average of RL + Bandit
			rlWeight := config.BlendWeightRL
			banditWeight := 1.0 - rlWeight

			finalPricePercent = rlWeight*rlOutput.PricePercent + banditWeight*banditOutput.PricePercent
			finalPriceCents = int64(float64(ctx.OrderPriceCents) * finalPricePercent / 100.0)
			confidence = rlWeight*rlOutput.Confidence + banditWeight*0.5
			source = SourceBlend
		} else {
			// Hard switch: use RL directly
			finalPriceCents = rlOutput.PriceCents
			finalPricePercent = rlOutput.PricePercent
			confidence = rlOutput.Confidence
			source = SourceRL
		}
		uxVariant = rlOutput.UXVariant

	// RL confidence too low → fallback to bandit
	case banditOutput != nil:
		finalPriceCents = banditOutput.PriceCents
		finalPricePercent = banditOutput.PricePercent
		confidence = 0.5
		source = SourceBandit
		uxVariant = "standard"
		guardrailsApplied = append(guardrailsApplied, "rl_low_confidence")

	// No bandit either → rule-based fallback
	default:
		ruleResult := CalculateRuleBasedPrice(ctx, nil)
		finalPriceCents = ruleResult.PriceCents
		finalPricePercent = ruleResult.PricePercent
		confidence = 0.3
		source = SourceRules
		uxVariant = "standard"
		guardrailsApplied = append(guardrailsApplied, "no_ai_available")
	}

	// ── Step 8: Final Clamp ──────────────────────────────────────────────────
	clamped := false
	if finalPricePercent < config.MinPricePercent {
		finalPricePercent = config.MinPricePercent
		finalPriceCents = int64(float64(ctx.OrderPriceCents) * finalPricePercent / 100.0)
		clamped = true
	}
	if finalPricePercent > config.MaxPricePercent {
		finalPricePercent = config.MaxPricePercent
		finalPriceCents = int64(float64(ctx.OrderPriceCents) * finalPricePercent / 100.0)
		clamped = true
	}
	if finalPriceCents < 50 {
		finalPriceCents = 50
		clamped = true
	}
	if clamped {
		guardrailsApplied = append(guardrailsApplied, "price_clamped")
	}

	// ── Step 9: Build Decision ────────────────────────────────────────────────
	decision := &HybridDecision{
		PriceCents:        finalPriceCents,
		PricePercent:      finalPricePercent,
		AnchorPrice:       int64(float64(finalPriceCents) * 1.4),
		Source:            source,
		Confidence:        confidence,
		IsShadow:          isShadow,
		GuardrailsApplied: guardrailsApplied,
		Clamped:           clamped,
		UXVariant:         uxVariant,
		RecommendedLabel:  buildHybridLabel(source, uxVariant, rlState),
		SessionStep:       rlState.SessionStep,
		RLOutput:          rlOutput,
		BanditOutput:      banditOutput,
	}

	// ── Step 10: Record ──────────────────────────────────────────────────────
	go recordHybridEvent(db, ctx, decision)

	// Also record to RL transitions if source was RL or blend
	if source == SourceRL || source == SourceBlend {
		if rlOutput != nil {
			go recordRLTransition(db, ctx, rlState, stateKey, rlOutput.ActionIndex,
				GetActionFromIndex(ActionIndex(rlOutput.ActionIndex)), finalPriceCents, loadRLConfig(db))
		}
	}

	return decision, nil
}

// buildHybridLabel creates a UX label based on the decision source.
func buildHybridLabel(source PricingSource, uxVariant string, state *RLState) string {
	switch uxVariant {
	case "discount_badge":
		return "Special price based on your usage"
	case "social_proof":
		return "98% of users like you choose this protection"
	case "urgency":
		return "Secure your order now — limited protection slots"
	}

	if source == SourceRL {
		return "Recommended protection for you"
	} else if source == SourceBandit {
		return "Smart protection price for you"
	} else if source == SourceBlend {
		return "Optimized protection price for you"
	}
	return "Standard protection"
}

// Ensure imports used
var _ = json.Marshal
var _ = math.Abs
var _ = fnv.New32a
