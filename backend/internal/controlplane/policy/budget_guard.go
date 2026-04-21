package policy

import (
	"github.com/geocore-next/backend/internal/controlplane/planner"
)

// BudgetGuard ensures proposals do not exceed cost budget caps.
type BudgetGuard struct {
	MonthlyCap     float64
	CurrentSpend   float64
	MaxUtilization float64 // percent
}

// NewBudgetGuard creates a budget guard with default thresholds.
func NewBudgetGuard() *BudgetGuard {
	return &BudgetGuard{
		MonthlyCap:     5000,
		CurrentSpend:   2000,
		MaxUtilization: 90,
	}
}

// Allow checks if the proposal is budget-safe.
func (g *BudgetGuard) Allow(p planner.Proposal) bool {
	if p.Type == planner.ScaleDown || p.Type == planner.Rollback {
		return true // these save money or are cost-neutral
	}
	return g.utilization() < g.MaxUtilization
}

func (g *BudgetGuard) utilization() float64 {
	if g.MonthlyCap == 0 {
		return 0
	}
	return (g.CurrentSpend / g.MonthlyCap) * 100
}
