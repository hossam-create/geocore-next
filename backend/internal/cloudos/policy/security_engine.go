package policy

import "github.com/geocore-next/backend/internal/cloudos/resources"

// SecurityEngine ensures proposals do not violate security constraints.
type SecurityEngine struct {
	MaxRiskScore float64
	SystemStress float64 // 0-1
}

// NewSecurityEngine creates a security engine with default thresholds.
func NewSecurityEngine() *SecurityEngine {
	return &SecurityEngine{
		MaxRiskScore: 0.7,
		SystemStress: 0,
	}
}

// Check returns true if the proposal is security-safe.
func (s *SecurityEngine) Check(p resources.Proposal) bool {
	if p.Resource == "wallet" {
		return true // wallet actions are always security-approved (audits are safe)
	}

	effectiveRisk := p.RiskScore + s.SystemStress*0.3
	return effectiveRisk <= s.MaxRiskScore
}

// UpdateStress adjusts the system stress factor based on current metrics.
func (s *SecurityEngine) UpdateStress(errorRate, p95Latency float64) {
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
	s.SystemStress = stress
}
