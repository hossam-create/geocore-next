package policy

import (
	"log/slog"

	"github.com/geocore-next/backend/internal/gitops/diff"
	"github.com/geocore-next/backend/internal/gitops/planner"
)

// PolicyDecision represents the outcome of policy evaluation.
type PolicyDecision struct {
	Allowed     bool     `json:"allowed"`
	Reason      string   `json:"reason"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
}

// PolicyEngine evaluates rollout plans against safety policies.
type PolicyEngine struct {
	maxRiskScore       float64 // max allowed risk
	sloErrorBudgetPct  float64 // remaining SLO error budget %
	sloBurning         bool    // true if SLO budget is exhausted
	deployFreezeActive bool    // true if deployments are frozen
}

// NewPolicyEngine creates a policy engine with default safety thresholds.
func NewPolicyEngine() *PolicyEngine {
	return &PolicyEngine{
		maxRiskScore:      0.7,
		sloErrorBudgetPct: 100,
		sloBurning:        false,
	}
}

// Allowed checks if a rollout plan is permitted by all safety gates.
func (pe *PolicyEngine) Allowed(d diff.StateDiff, plan planner.RolloutPlan) PolicyDecision {
	var blocked []string

	// Gate 1: Risk score
	if d.Risk > pe.maxRiskScore {
		blocked = append(blocked, "risk_score")
		slog.Warn("gitops: blocked by risk gate",
			"service", d.Service, "risk", d.Risk, "max", pe.maxRiskScore)
	}

	// Gate 2: SLO budget burning
	if pe.sloBurning {
		blocked = append(blocked, "slo_budget")
		slog.Warn("gitops: blocked by SLO budget gate",
			"service", d.Service, "slo_remaining", pe.sloErrorBudgetPct)
	}

	// Gate 3: Deploy freeze
	if pe.deployFreezeActive {
		blocked = append(blocked, "deploy_freeze")
		slog.Warn("gitops: blocked by deploy freeze", "service", d.Service)
	}

	// Gate 4: Canary required for high-risk changes
	if d.Risk > 0.5 && plan.Strategy != planner.StrategyCanary {
		blocked = append(blocked, "canary_required")
		slog.Warn("gitops: canary required for high-risk change",
			"service", d.Service, "risk", d.Risk)
	}

	if len(blocked) > 0 {
		return PolicyDecision{
			Allowed:   false,
			Reason:    "blocked by safety gates",
			BlockedBy: blocked,
		}
	}

	return PolicyDecision{Allowed: true, Reason: "all gates passed"}
}

// SetSLOBurning updates the SLO budget state.
func (pe *PolicyEngine) SetSLOBurning(burning bool, remainingPct float64) {
	pe.sloBurning = burning
	pe.sloErrorBudgetPct = remainingPct
}

// SetDeployFreeze enables or disables deployment freeze.
func (pe *PolicyEngine) SetDeployFreeze(active bool) {
	pe.deployFreezeActive = active
}
