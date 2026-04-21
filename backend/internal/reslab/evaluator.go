package reslab

import (
	"math"

	"github.com/geocore-next/backend/internal/stress"
)

// ExperimentScore holds the weighted SLO score (0–100) for one experiment run.
//
// Scoring formula:
//
//	Overall = 0.40*Latency + 0.30*Reliability + 0.20*Recovery + 0.10*Cost
type ExperimentScore struct {
	Overall     float64 `json:"overall"`     // composite 0-100
	Latency     float64 `json:"latency"`     // 40% weight
	Reliability float64 `json:"reliability"` // 30% weight
	Recovery    float64 `json:"recovery"`    // 20% weight
	Cost        float64 `json:"cost"`        // 10% weight
	Grade       string  `json:"grade"`       // A / B / C / D / F
}

// Score computes the weighted experiment score from a completed stress report.
func Score(r stress.StressReport) ExperimentScore {
	m := r.Metrics

	lat := scoreLatency(m.P95LatencyMs)
	rel := scoreReliability(m.ErrorRatePct)
	rec := scoreRecovery(r.Validation)
	cost := scoreCost(m.RPS, m.P95LatencyMs)

	overall := 0.40*lat + 0.30*rel + 0.20*rec + 0.10*cost
	overall = math.Round(overall*100) / 100

	return ExperimentScore{
		Overall:     overall,
		Latency:     math.Round(lat*100) / 100,
		Reliability: math.Round(rel*100) / 100,
		Recovery:    math.Round(rec*100) / 100,
		Cost:        math.Round(cost*100) / 100,
		Grade:       grade(overall),
	}
}

// scoreLatency: 100 at ≤300ms, linear drop to 0 at 2000ms.
func scoreLatency(p95ms float64) float64 {
	if p95ms <= 300 {
		return 100
	}
	if p95ms >= 2000 {
		return 0
	}
	return math.Max(0, 100*(1-(p95ms-300)/1700))
}

// scoreReliability: 100 at 0% errors, linear drop to 0 at 10%.
func scoreReliability(errorPct float64) float64 {
	if errorPct <= 0 {
		return 100
	}
	if errorPct >= 10 {
		return 0
	}
	return math.Max(0, 100*(1-errorPct/10))
}

// scoreRecovery: based on post-test health status + AIOps detection.
func scoreRecovery(v stress.ValidationResult) float64 {
	score := 40.0
	switch v.HealthStatus {
	case "healthy":
		score += 40
	case "degraded":
		score += 15
	}
	if v.AIOpsTriggered {
		score += 20 // AIOps detected = observability working correctly
	}
	return math.Min(score, 100)
}

// scoreCost: efficiency measured as throughput per unit of latency.
func scoreCost(rps, p95ms float64) float64 {
	if rps <= 0 || p95ms <= 0 {
		return 50
	}
	return math.Min(rps/p95ms*100, 100)
}

func grade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}
