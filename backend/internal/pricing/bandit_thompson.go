package pricing

import (
	"math"
	"math/rand"
)

// ── Thompson Sampling ────────────────────────────────────────────────────────────
//
// Core idea: each arm has a Beta(α, β) posterior distribution.
// α = conversions + 1 (successes)
// β = impressions - conversions + 1 (failures)
//
// To select an arm:
//   1. Sample from Beta(α, β) for each arm
//   2. Pick the arm with the highest sample
//
// This naturally balances exploration vs exploitation:
// - Arms with few impressions → wide distribution → sometimes sampled high → explored
// - Arms with many impressions & high conversion → narrow high distribution → exploited

// SampleArm draws a sample from the Beta(α, β) posterior for an arm.
// This is the core of Thompson Sampling.
func SampleArm(arm *BanditArm) float64 {
	return sampleBeta(arm.Alpha, arm.Beta)
}

// sampleBeta generates a sample from a Beta(α, β) distribution
// using the Gamma distribution method: Beta(a,b) = Gamma(a) / (Gamma(a) + Gamma(b))
func sampleBeta(alpha, beta float64) float64 {
	if alpha <= 0 {
		alpha = 1
	}
	if beta <= 0 {
		beta = 1
	}

	x := sampleGamma(alpha)
	y := sampleGamma(beta)

	if x+y == 0 {
		return 0.5
	}
	return x / (x + y)
}

// sampleGamma generates a sample from Gamma(shape, 1) using Marsaglia and Tsang's method.
func sampleGamma(shape float64) float64 {
	if shape < 1 {
		// Use the transformation: Gamma(shape) = Gamma(shape+1) * U^(1/shape)
		// where U ~ Uniform(0,1)
		return sampleGamma(shape+1) * math.Pow(rand.Float64(), 1.0/shape)
	}

	// Marsaglia and Tsang's method for shape >= 1
	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)

	for {
		var x, v float64
		for {
			x = randNorm()
			v = 1.0 + c*x
			if v > 0 {
				break
			}
		}

		u := rand.Float64()
		v = v * v * v

		xSq := x * x
		if u < 1.0-0.0331*xSq*xSq {
			return d * v
		}
		if math.Log(u) < 0.5*xSq+d*(1.0-v+math.Log(v)) {
			return d * v
		}
	}
}

// randNorm generates a standard normal random variable (Box-Muller).
func randNorm() float64 {
	u1 := rand.Float64()
	u2 := rand.Float64()
	for u1 == 0 {
		u1 = rand.Float64()
	}
	return math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)
}

// ── UCB (Upper Confidence Bound) ─────────────────────────────────────────────────

// UCBScore computes the UCB1 score for an arm.
// UCB = avg_reward + sqrt(2 * ln(total_impressions) / arm_impressions)
// Higher uncertainty → higher score → more exploration.
func UCBScore(arm *BanditArm, totalImpressions int64) float64 {
	if arm.Impressions == 0 {
		return math.Inf(1) // unexplored arms get infinite score
	}

	avgReward := arm.TotalReward / float64(arm.Impressions)
	exploration := math.Sqrt(2.0 * math.Log(float64(totalImpressions)) / float64(arm.Impressions))

	return avgReward + exploration
}

// ── Epsilon-Greedy ────────────────────────────────────────────────────────────────

// EpsilonGreedySelect selects the best arm with probability (1-ε),
// or a random arm with probability ε.
func EpsilonGreedySelect(arms []*BanditArm, epsilon float64) *BanditArm {
	if len(arms) == 0 {
		return nil
	}

	if rand.Float64() < epsilon {
		// Explore: random arm
		return arms[rand.Intn(len(arms))]
	}

	// Exploit: best average reward arm
	return bestArmByAvgReward(arms)
}

// bestArmByAvgReward returns the arm with the highest average reward.
func bestArmByAvgReward(arms []*BanditArm) *BanditArm {
	var best *BanditArm
	bestAvg := math.Inf(-1)

	for _, arm := range arms {
		if arm.Impressions == 0 {
			continue
		}
		avg := arm.TotalReward / float64(arm.Impressions)
		if avg > bestAvg {
			bestAvg = avg
			best = arm
		}
	}

	// If no arm has impressions, pick first
	if best == nil && len(arms) > 0 {
		return arms[0]
	}
	return best
}

// ── Thompson Sampling Selection ──────────────────────────────────────────────────

// ThompsonSelect selects an arm using Thompson Sampling.
// Samples from each arm's Beta posterior and picks the highest.
func ThompsonSelect(arms []*BanditArm) (*BanditArm, float64) {
	if len(arms) == 0 {
		return nil, 0
	}

	var bestArm *BanditArm
	bestSample := math.Inf(-1)

	for _, arm := range arms {
		sample := SampleArm(arm)
		if sample > bestSample {
			bestSample = sample
			bestArm = arm
		}
	}

	return bestArm, bestSample
}

// ── Reward Function ──────────────────────────────────────────────────────────────

// CalculateBanditReward computes the reward for a bandit event.
// reward = (price * bought) - expected_claim_cost
// This is the key insight: we optimize for NET revenue, not just conversion.
func CalculateBanditReward(priceCents int64, didBuy bool, claimCostCents float64) float64 {
	if !didBuy {
		return 0 // no reward if user didn't buy
	}
	return float64(priceCents) - claimCostCents
}

// EstimateClaimCost estimates the expected claim payout for an order.
// Based on user risk, delivery risk, and historical claim rates.
func EstimateClaimCost(ctx *PricingContext) float64 {
	// Base claim probability from cancellation rate + delivery risk
	claimProb := ctx.CancellationRate*0.3 + ctx.DeliveryRiskScore*0.2
	if claimProb > 1.0 {
		claimProb = 1.0
	}

	// Average payout is ~15% of order value
	avgPayoutPct := 0.15

	// Higher abuse flags → higher claim cost
	if ctx.AbuseFlags > 2 {
		avgPayoutPct += 0.10
	} else if ctx.AbuseFlags > 0 {
		avgPayoutPct += 0.03
	}

	return float64(ctx.OrderPriceCents) * claimProb * avgPayoutPct
}
