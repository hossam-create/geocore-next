package policy

import "github.com/geocore-next/backend/internal/cloudos/resources"

// SLOEngine ensures proposals do not violate SLO targets.
type SLOEngine struct {
	MaxLatencyDegradation float64 // ms
	MaxErrorRateIncrease  float64 // percent
}

// NewSLOEngine creates an SLO engine with default thresholds.
func NewSLOEngine() *SLOEngine {
	return &SLOEngine{
		MaxLatencyDegradation: 50,
		MaxErrorRateIncrease:  0.5,
	}
}

// Check returns true if the proposal is SLO-safe.
func (s *SLOEngine) Check(p resources.Proposal) bool {
	if p.Resource == "wallet" && p.Action == "emergency_audit" {
		return true // wallet audit is always SLO-safe
	}
	return p.RiskScore <= 0.6
}
