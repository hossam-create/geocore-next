package proposals

import (
	"time"
)

// ProposalType defines the category of optimization.
type ProposalType string

const (
	ProposalScaleUp        ProposalType = "scale_up"
	ProposalScaleDown      ProposalType = "scale_down"
	ProposalRollback       ProposalType = "rollback"
	ProposalThrottle       ProposalType = "throttle"
	ProposalKafkaRebalance ProposalType = "kafka_rebalance"
	ProposalCachePromote   ProposalType = "cache_promote"
	ProposalTrafficReroute ProposalType = "traffic_reroute"
	ProposalConfigTune     ProposalType = "config_tune"
)

// ChangeProposal represents a proposed optimization change.
// Every proposal must pass simulation and safety gates before execution.
type ChangeProposal struct {
	ID           string      `json:"id"`
	Type         ProposalType `json:"type"`
	Target       string      `json:"target"`
	Action       string      `json:"action"`
	CurrentState string      `json:"current_state"`
	DesiredState string      `json:"desired_state"`
	ExpectedGain float64     `json:"expected_gain"` // % improvement expected
	RiskScore    float64     `json:"risk_score"`    // 0-1 (0=safe, 1=dangerous)
	RollbackPlan string      `json:"rollback_plan"`
	Reason       string      `json:"reason"`

	// Simulation results (filled by shadow runner)
	SimulatedLatencyDelta float64 `json:"simulated_latency_delta"`
	SimulatedErrorDelta   float64 `json:"simulated_error_delta"`
	SimulatedCostDelta    float64 `json:"simulated_cost_delta"`
	SimulationPassed      bool    `json:"simulation_passed"`

	// Safety gate results (filled by evaluator)
	SloApproved   bool `json:"slo_approved"`
	BudgetApproved bool `json:"budget_approved"`
	RiskApproved  bool `json:"risk_approved"`
	AllApproved   bool `json:"all_approved"`

	// Execution state
	Status    string     `json:"status"` // proposed, simulating, approved, executing, completed, rejected
	CreatedAt time.Time  `json:"created_at"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
}

// NewChangeProposal creates a new proposal with generated ID and status.
func NewChangeProposal(pType ProposalType, target, action string) *ChangeProposal {
	return &ChangeProposal{
		Type:     pType,
		Target:   target,
		Action:   action,
		Status:   "proposed",
		CreatedAt: time.Now().UTC(),
	}
}

// IsSafe returns true if the proposal passed simulation and all safety gates.
func (c *ChangeProposal) IsSafe() bool {
	return c.SimulationPassed && c.AllApproved
}

// Reject marks the proposal as rejected with a reason.
func (c *ChangeProposal) Reject(reason string) {
	c.Status = "rejected"
}

// Approve marks the proposal as approved for execution.
func (c *ChangeProposal) Approve() {
	c.SloApproved = true
	c.BudgetApproved = true
	c.RiskApproved = true
	c.AllApproved = true
	c.Status = "approved"
}

// MarkSimulated records simulation results.
func (c *ChangeProposal) MarkSimulated(latencyDelta, errorDelta, costDelta float64, passed bool) {
	c.SimulatedLatencyDelta = latencyDelta
	c.SimulatedErrorDelta = errorDelta
	c.SimulatedCostDelta = costDelta
	c.SimulationPassed = passed
	c.Status = "simulating"
}
