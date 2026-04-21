package safety

import (
	"github.com/geocore-next/backend/internal/singularity/intelligence"
	"github.com/geocore-next/backend/internal/singularity/proposals"
)

// BudgetGuard ensures proposals do not exceed cost budget caps.
type BudgetGuard struct {
	costModel *intelligence.CostModel
}

// NewBudgetGuard creates a budget guard with the given cost model.
func NewBudgetGuard(costModel *intelligence.CostModel) *BudgetGuard {
	return &BudgetGuard{costModel: costModel}
}

// Approve checks if the proposal stays within budget.
func (g *BudgetGuard) Approve(p *proposals.ChangeProposal) bool {
	if g.costModel == nil {
		return true // no cost model = no budget enforcement
	}

	// Scale down always saves money
	if p.Type == proposals.ProposalScaleDown {
		return true
	}

	// Rollback doesn't change cost significantly
	if p.Type == proposals.ProposalRollback {
		return true
	}

	// Check if projected cost stays within budget
	projectedSpend := g.costModel.ProjectedMonthlySpend(p.SimulatedCostDelta)
	if !g.costModel.IsWithinBudget(projectedSpend) {
		return false
	}

	// Check if budget utilization would exceed 90%
	projectedUtil := (projectedSpend / g.costModel.MonthlyBudgetCap) * 100
	if projectedUtil > 90 {
		return false
	}

	return true
}
