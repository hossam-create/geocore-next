package policy

import (
	"github.com/geocore-next/backend/internal/controlplane/planner"
)

// PolicyEngine combines all safety gates into a single allow/deny decision.
type PolicyEngine struct {
	SLO    *SLOGuard
	Budget *BudgetGuard
	Risk   *RiskEngine
}

// NewPolicyEngine creates a policy engine with all safety gates.
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		SLO:    NewSLOGuard(),
		Budget: NewBudgetGuard(),
		Risk:   NewRiskEngine(),
	}
}

// Allow checks a proposal through ALL safety gates.
// A proposal must pass EVERY gate to be approved.
func (pe *PolicyEngine) Allow(p planner.Proposal) bool {
	if !pe.SLO.Allow(p) {
		return false
	}
	if !pe.Budget.Allow(p) {
		return false
	}
	if !pe.Risk.Allow(p) {
		return false
	}
	return true
}
