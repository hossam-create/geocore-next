package fraud

import (
	"math"
)

// RiskWeights mirrors the mnbara scoring system.
var RiskWeights = map[string]float64{
	"velocity": 0.25,
	"amount":   0.20,
	"device":   0.15,
	"location": 0.15,
	"behavior": 0.15,
	"history":  0.10,
}

// RiskSignal represents a single fraud indicator.
type RiskSignal struct {
	Type        string  `json:"type"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

// RiskResult is the output of a transaction risk assessment.
type RiskResult struct {
	RiskScore      float64      `json:"risk_score"`
	RiskLevel      string       `json:"risk_level"`
	Decision       string       `json:"decision"` // approved, review, declined
	Signals        []RiskSignal `json:"signals"`
	RequiresReview bool         `json:"requires_review"`
}

// AnalyzeTransaction performs a rule-based risk assessment.
// In production this would query Redis/DB for velocity data.
func AnalyzeTransaction(amount float64, userOrders int, avgOrderValue float64, accountAgeHours float64) RiskResult {
	signals := []RiskSignal{}
	total := 0.0

	// ── Amount deviation ────────────────────────────────────────────────
	if avgOrderValue > 0 && userOrders > 3 {
		deviation := amount / avgOrderValue
		if deviation > 5 {
			s := math.Min(1.0, deviation/10)
			total += s * RiskWeights["amount"] * 100
			signals = append(signals, RiskSignal{"amount_deviation", s, "Amount significantly higher than average"})
		} else if deviation > 3 {
			s := 0.4
			total += s * RiskWeights["amount"] * 100
			signals = append(signals, RiskSignal{"amount_deviation", s, "Amount moderately above average"})
		}
	}

	// ── High single amount ──────────────────────────────────────────────
	if amount > 5000 {
		s := math.Min(1.0, amount/10000)
		total += s * RiskWeights["amount"] * 100
		signals = append(signals, RiskSignal{"high_amount", s, "High-value transaction"})
	}

	// ── New account behavior ────────────────────────────────────────────
	if accountAgeHours < 24 && amount > 500 {
		s := 0.6
		total += s * RiskWeights["behavior"] * 100
		signals = append(signals, RiskSignal{"new_account_high_value", s, "New account with high-value order"})
	}
	if accountAgeHours < 1 && amount > 100 {
		s := 0.8
		total += s * RiskWeights["behavior"] * 100
		signals = append(signals, RiskSignal{"instant_high_value", s, "Order within first hour of registration"})
	}

	// ── Low order history ───────────────────────────────────────────────
	if userOrders == 0 && amount > 1000 {
		s := 0.5
		total += s * RiskWeights["history"] * 100
		signals = append(signals, RiskSignal{"no_history", s, "First-time buyer with high-value order"})
	}

	score := math.Max(0, math.Min(100, total))
	level := riskLevel(score)
	decision := "approved"
	if score >= 80 {
		decision = "declined"
	} else if score >= 50 {
		decision = "review"
	}

	return RiskResult{
		RiskScore:      score,
		RiskLevel:      level,
		Decision:       decision,
		Signals:        signals,
		RequiresReview: decision == "review",
	}
}

func riskLevel(score float64) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	case score >= 20:
		return "low"
	default:
		return "very_low"
	}
}
