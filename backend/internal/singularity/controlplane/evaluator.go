package controlplane

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/singularity/proposals"
	"github.com/geocore-next/backend/internal/singularity/safety"
)

// Evaluator runs proposals through safety gates (SLO, budget, risk).
type Evaluator struct {
	sloGuard    *safety.SLOGuard
	budgetGuard *safety.BudgetGuard
	riskEngine  *safety.RiskEngine
}

// NewEvaluator creates an evaluator with all safety gates.
func NewEvaluator(sloGuard *safety.SLOGuard, budgetGuard *safety.BudgetGuard, riskEngine *safety.RiskEngine) *Evaluator {
	return &Evaluator{
		sloGuard:    sloGuard,
		budgetGuard: budgetGuard,
		riskEngine:  riskEngine,
	}
}

// Evaluate runs a proposal through all safety gates.
// A proposal must pass ALL gates to be approved for execution.
func (e *Evaluator) Evaluate(ctx context.Context, p *proposals.ChangeProposal) bool {
	slog.Info("singularity: evaluating proposal",
		"type", p.Type, "target", p.Target, "risk", p.RiskScore)

	// Gate 1: SLO — must not degrade SLO
	if e.sloGuard != nil {
		p.SloApproved = e.sloGuard.Approve(p)
		if !p.SloApproved {
			slog.Warn("singularity: proposal REJECTED by SLO guard", "type", p.Type)
			p.Reject("slo_guard")
			return false
		}
	} else {
		p.SloApproved = true
	}

	// Gate 2: Budget — must not exceed cost cap
	if e.budgetGuard != nil {
		p.BudgetApproved = e.budgetGuard.Approve(p)
		if !p.BudgetApproved {
			slog.Warn("singularity: proposal REJECTED by budget guard", "type", p.Type)
			p.Reject("budget_guard")
			return false
		}
	} else {
		p.BudgetApproved = true
	}

	// Gate 3: Risk — overall risk must be acceptable
	if e.riskEngine != nil {
		p.RiskApproved = e.riskEngine.Approve(p)
		if !p.RiskApproved {
			slog.Warn("singularity: proposal REJECTED by risk engine", "type", p.Type)
			p.Reject("risk_engine")
			return false
		}
	} else {
		p.RiskApproved = true
	}

	// All gates passed
	p.AllApproved = true
	p.Status = "approved"
	slog.Info("singularity: proposal APPROVED", "type", p.Type, "target", p.Target)
	return true
}
