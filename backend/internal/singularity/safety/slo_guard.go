package safety

import (
	"github.com/geocore-next/backend/internal/singularity/proposals"
)

// SLOGuard ensures proposals do not degrade SLO targets.
type SLOGuard struct {
	maxLatencyDegradation float64 // max allowed P95 increase (ms)
	maxErrorRateIncrease  float64 // max allowed error rate increase (%)
}

// NewSLOGuard creates an SLO guard with default thresholds.
func NewSLOGuard() *SLOGuard {
	return &SLOGuard{
		maxLatencyDegradation: 50,  // max 50ms P95 increase
		maxErrorRateIncrease:  0.5, // max 0.5% error rate increase
	}
}

// Approve checks if the proposal's simulated impact is within SLO tolerance.
func (g *SLOGuard) Approve(p *proposals.ChangeProposal) bool {
	// Rollback is always SLO-safe (it restores previous state)
	if p.Type == proposals.ProposalRollback {
		return true
	}

	// Check simulated latency impact
	if p.SimulatedLatencyDelta > g.maxLatencyDegradation {
		return false
	}

	// Check simulated error rate impact
	if p.SimulatedErrorDelta > g.maxErrorRateIncrease {
		return false
	}

	return true
}
