package pricing

import (
	"math"
)

// ── RL Reward Function ────────────────────────────────────────────────────────────
//
// reward = r_revenue - r_claim_cost - r_churn
//
// r_revenue:   income from the price (if bought)
// r_claim_cost: expected/paid claim payout
// r_churn:     penalty if user left after refusal
//
// Objective:
//
//	maximize E[Σ γ^t (r_revenue - r_claim_cost - r_churn)]

// RLRewardComponents breaks down the reward into its components.
type RLRewardComponents struct {
	Revenue    float64 `json:"revenue"`     // price * bought
	ClaimCost  float64 `json:"claim_cost"`  // claim payout
	Churn      float64 `json:"churn"`       // churn penalty
	Total      float64 `json:"total"`       // revenue - claim_cost - churn
	Discounted float64 `json:"discounted"`  // γ^t * total
}

// CalculateRLReward computes the full reward for an RL transition.
func CalculateRLReward(priceCents int64, didBuy, didClaim, didChurn bool,
	claimCostCents float64, churnPenalty float64) RLRewardComponents {

	components := RLRewardComponents{}

	// Revenue: only if user bought
	if didBuy {
		components.Revenue = float64(priceCents) / 100.0 // convert to "currency units"
	}

	// Claim cost: if user filed a claim
	if didClaim {
		components.ClaimCost = claimCostCents / 100.0
	}

	// Churn penalty: if user left after refusal
	if didChurn {
		components.Churn = churnPenalty
	}

	// Total reward
	components.Total = components.Revenue - components.ClaimCost - components.Churn

	return components
}

// CalculateDiscountedReward applies discount factor γ over time steps.
func CalculateDiscountedReward(rewards []float64, gamma float64) float64 {
	total := 0.0
	for t, r := range rewards {
		total += math.Pow(gamma, float64(t)) * r
	}
	return total
}

// EstimateRLClaimCost estimates expected claim payout for reward calculation.
func EstimateRLClaimCost(ctx *PricingContext) float64 {
	// Base claim probability
	claimProb := ctx.CancellationRate*0.3 + ctx.DeliveryRiskScore*0.2
	if claimProb > 1.0 {
		claimProb = 1.0
	}

	// Average payout ~15% of order value
	avgPayoutPct := 0.15

	// Abuse adjustment
	if ctx.AbuseFlags > 2 {
		avgPayoutPct += 0.10
	}

	return float64(ctx.OrderPriceCents) * claimProb * avgPayoutPct
}

// EstimateChurnPenalty estimates the long-term cost of a user churning.
// This is the key insight that separates RL from bandit:
// we penalize not just the missed sale, but the lost future revenue.
func EstimateChurnPenalty(ctx *PricingContext) float64 {
	// Estimate user lifetime value based on past behavior
	avgOrderValue := ctx.AvgOrderValue
	if avgOrderValue == 0 {
		avgOrderValue = float64(ctx.OrderPriceCents)
	}

	// Expected future orders (simplified)
	expectedFutureOrders := 3.0 // baseline
	if ctx.InsuranceBuyRate > 0.5 {
		expectedFutureOrders = 5.0 // loyal users buy more
	} else if ctx.InsuranceBuyRate < 0.2 {
		expectedFutureOrders = 1.0 // disengaged users
	}

	// Average insurance revenue per order (2% of avg order value)
	insuranceRevenuePerOrder := avgOrderValue * 0.02

	// Churn penalty = lost future insurance revenue
	churnPenalty := expectedFutureOrders * insuranceRevenuePerOrder

	// Scale down to reasonable range (don't over-penalize)
	if churnPenalty > 50 {
		churnPenalty = 50
	}

	return churnPenalty
}

// ── Episode Reward Tracker ────────────────────────────────────────────────────────

// EpisodeReward tracks cumulative reward across a session.
type EpisodeReward struct {
	Steps    []RLRewardComponents `json:"steps"`
	TotalRaw float64              `json:"total_raw"`
	TotalDiscounted float64       `json:"total_discounted"`
	Gamma    float64              `json:"gamma"`
}

func NewEpisodeReward(gamma float64) *EpisodeReward {
	return &EpisodeReward{Gamma: gamma}
}

func (e *EpisodeReward) AddStep(components RLRewardComponents) {
	e.Steps = append(e.Steps, components)
	e.TotalRaw += components.Total
	e.TotalDiscounted = CalculateDiscountedReward(
		mapRewards(e.Steps), e.Gamma,
	)
}

func mapRewards(steps []RLRewardComponents) []float64 {
	rewards := make([]float64, len(steps))
	for i, s := range steps {
		rewards[i] = s.Total
	}
	return rewards
}
