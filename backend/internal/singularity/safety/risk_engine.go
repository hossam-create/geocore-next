package safety

import (
	"github.com/geocore-next/backend/internal/singularity/proposals"
)

// RiskEngine evaluates the overall risk of a proposal.
// Combines proposal risk score with system state to make a decision.
type RiskEngine struct {
	maxRiskScore       float64 // max allowed risk score (0-1)
	systemStressFactor float64 // current system stress (0-1)
}

// NewRiskEngine creates a risk engine with default thresholds.
func NewRiskEngine() *RiskEngine {
	return &RiskEngine{
		maxRiskScore:       0.6,
		systemStressFactor: 0,
	}
}

// Approve checks if the proposal's risk is acceptable.
func (r *RiskEngine) Approve(p *proposals.ChangeProposal) bool {
	// Rollback is always low-risk
	if p.Type == proposals.ProposalRollback {
		return true
	}

	// Effective risk = proposal risk + system stress
	effectiveRisk := p.RiskScore + r.systemStressFactor*0.3

	if effectiveRisk > r.maxRiskScore {
		return false
	}

	// High-risk proposals need simulation to pass
	if p.RiskScore > 0.4 && !p.SimulationPassed {
		return false
	}

	return true
}

// UpdateStressFactor adjusts the system stress based on current metrics.
func (r *RiskEngine) UpdateStressFactor(errorRate, p95Latency float64) {
	stress := 0.0
	if errorRate > 3.0 {
		stress += 0.3
	}
	if p95Latency > 600 {
		stress += 0.3
	}
	if errorRate > 1.0 {
		stress += 0.1
	}
	if p95Latency > 400 {
		stress += 0.1
	}
	if stress > 1.0 {
		stress = 1.0
	}
	r.systemStressFactor = stress
}
