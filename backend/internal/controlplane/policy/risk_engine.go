package policy

import (
	"github.com/geocore-next/backend/internal/controlplane/planner"
)

// RiskEngine evaluates overall proposal risk.
type RiskEngine struct {
	MaxRiskScore float64
	SystemStress float64 // 0-1, increases during incidents
}

// NewRiskEngine creates a risk engine with default thresholds.
func NewRiskEngine() *RiskEngine {
	return &RiskEngine{
		MaxRiskScore: 0.8,
		SystemStress: 0,
	}
}

// Allow checks if the proposal's risk is acceptable.
func (r *RiskEngine) Allow(p planner.Proposal) bool {
	if p.Type == planner.Rollback {
		return true // rollback is always low-risk
	}

	effectiveRisk := p.RiskScore + r.SystemStress*0.3
	return effectiveRisk <= r.MaxRiskScore && p.ExpectedGain >= 0.1
}

// UpdateStress adjusts the system stress factor based on current metrics.
func (r *RiskEngine) UpdateStress(errorRate, p95Latency float64) {
	stress := 0.0
	if errorRate > 3.0 {
		stress += 0.3
	}
	if p95Latency > 600 {
		stress += 0.3
	}
	if stress > 1.0 {
		stress = 1.0
	}
	r.SystemStress = stress
}
