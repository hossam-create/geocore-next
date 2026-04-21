package planner

// ChangePlan is a detailed execution plan for a proposal.
type ChangePlan struct {
	Proposal      Proposal
	Steps         []PlanStep
	EstimatedTime string
}

// PlanStep is a single step in a change plan.
type PlanStep struct {
	Order   int    `json:"order"`
	Action  string `json:"action"`
	Target  string `json:"target"`
	Verify  string `json:"verify"` // how to verify this step succeeded
}

// ChangePlanner creates detailed execution plans from proposals.
type ChangePlanner struct{}

// NewChangePlanner creates a change planner.
func NewChangePlanner() *ChangePlanner {
	return &ChangePlanner{}
}

// Plan creates a detailed execution plan for a proposal.
func (cp *ChangePlanner) Plan(p Proposal) ChangePlan {
	switch p.Type {
	case ScaleUp:
		return ChangePlan{
			Proposal: p,
			Steps: []PlanStep{
				{Order: 1, Action: "scale_deployment", Target: p.Target, Verify: "replicas == current + 2"},
				{Order: 2, Action: "verify_ready", Target: p.Target, Verify: "all pods ready"},
				{Order: 3, Action: "check_latency", Target: p.Target, Verify: "p95 < 500ms"},
			},
			EstimatedTime: "2min",
		}

	case ScaleDown:
		return ChangePlan{
			Proposal: p,
			Steps: []PlanStep{
				{Order: 1, Action: "verify_low_load", Target: p.Target, Verify: "cpu < 50%"},
				{Order: 2, Action: "scale_deployment", Target: p.Target, Verify: "replicas == current - 1"},
				{Order: 3, Action: "verify_healthy", Target: p.Target, Verify: "p95 < 500ms"},
			},
			EstimatedTime: "2min",
		}

	case Rollback:
		return ChangePlan{
			Proposal: p,
			Steps: []PlanStep{
				{Order: 1, Action: "rollout_undo", Target: p.Target, Verify: "previous revision active"},
				{Order: 2, Action: "verify_healthy", Target: p.Target, Verify: "error_rate < 1%"},
			},
			EstimatedTime: "3min",
		}

	default:
		return ChangePlan{Proposal: p, EstimatedTime: "1min"}
	}
}
