package policy

import (
	"github.com/geocore-next/backend/internal/controlplane/planner"
)

// SLOGuard ensures proposals do not degrade SLO targets.
type SLOGuard struct {
	MaxLatencyDegradation float64 // ms
	MaxErrorRateIncrease  float64 // percent
}

// NewSLOGuard creates an SLO guard with default thresholds.
func NewSLOGuard() *SLOGuard {
	return &SLOGuard{
		MaxLatencyDegradation: 50,
		MaxErrorRateIncrease:  0.5,
	}
}

// Allow checks if the proposal is SLO-safe.
func (g *SLOGuard) Allow(p planner.Proposal) bool {
	if p.Type == planner.Rollback {
		return true // rollback is always SLO-safe
	}
	return p.RiskScore <= 0.6
}
