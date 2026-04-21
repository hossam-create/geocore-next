package policy

import "github.com/geocore-next/backend/internal/cloudos/resources"

// BudgetEngine ensures proposals do not exceed cost budget caps.
type BudgetEngine struct {
	MonthlyCap     float64
	CurrentSpend   float64
	MaxUtilization float64 // percent
}

// NewBudgetEngine creates a budget engine with default thresholds.
func NewBudgetEngine() *BudgetEngine {
	return &BudgetEngine{
		MonthlyCap:     5000,
		CurrentSpend:   2000,
		MaxUtilization: 90,
	}
}

// Check returns true if the proposal is budget-safe.
func (b *BudgetEngine) Check(p resources.Proposal) bool {
	if p.Action == "scale_down" || p.Action == "decrease_sensitivity" {
		return true // saves money or is cost-neutral
	}
	return b.utilization() < b.MaxUtilization
}

func (b *BudgetEngine) utilization() float64 {
	if b.MonthlyCap == 0 {
		return 0
	}
	return (b.CurrentSpend / b.MonthlyCap) * 100
}
