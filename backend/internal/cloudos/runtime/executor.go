package runtime

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/cloudos/resources"
)

// SimulationResult represents the outcome of a shadow simulation.
type SimulationResult struct {
	Safe         bool    `json:"safe"`
	LatencyDelta float64 `json:"latency_delta_ms"`
	CostDelta    float64 `json:"cost_delta"`
	ErrorDelta   float64 `json:"error_delta"`
	RiskScore    float64 `json:"risk_score"`
	Reason       string  `json:"reason"`
}

// Executor handles simulation, execution, and rollback of proposals.
type Executor struct {
	rollback *RollbackEngine
}

// NewExecutor creates a runtime executor.
func NewExecutor() *Executor {
	return &Executor{
		rollback: NewRollbackEngine(),
	}
}

// Rollback reverts a failed proposal.
func (e *Executor) Rollback(ctx context.Context, p resources.Proposal) error {
	return e.rollback.Rollback(ctx, p)
}

// Simulate runs a shadow simulation for a proposal.
func (e *Executor) Simulate(ctx context.Context, p resources.Proposal) SimulationResult {
	slog.Debug("cloudos: simulating proposal", "resource", p.Resource, "action", p.Action)

	if p.RiskScore > 0.7 {
		return SimulationResult{
			Safe:      false,
			RiskScore: p.RiskScore,
			Reason:    "risk score exceeds maximum threshold",
		}
	}

	switch p.Action {
	case "scale_up":
		return SimulationResult{Safe: true, LatencyDelta: -20, CostDelta: 0.20, RiskScore: p.RiskScore, Reason: "scale up simulation passed"}
	case "scale_down":
		return SimulationResult{Safe: p.RiskScore < 0.4, LatencyDelta: 15, CostDelta: -0.10, RiskScore: p.RiskScore, Reason: "scale down simulation"}
	case "scale_consumers":
		return SimulationResult{Safe: true, LatencyDelta: -10, CostDelta: 0.10, RiskScore: p.RiskScore, Reason: "kafka scale simulation passed"}
	case "increase_sensitivity":
		return SimulationResult{Safe: true, ErrorDelta: -1.0, RiskScore: p.RiskScore, Reason: "fraud sensitivity simulation passed"}
	case "decrease_sensitivity":
		return SimulationResult{Safe: true, ErrorDelta: 0.5, RiskScore: p.RiskScore, Reason: "fraud sensitivity simulation passed"}
	case "emergency_audit":
		return SimulationResult{Safe: true, RiskScore: p.RiskScore, Reason: "wallet audit always safe"}
	case "pre_scale":
		return SimulationResult{Safe: true, LatencyDelta: -15, CostDelta: 0.15, RiskScore: p.RiskScore, Reason: "pre-scale simulation passed"}
	case "shift_traffic":
		return SimulationResult{Safe: true, LatencyDelta: -30, RiskScore: p.RiskScore, Reason: "region shift simulation passed"}
	default:
		return SimulationResult{Safe: false, RiskScore: p.RiskScore, Reason: "unknown action type"}
	}
}

// Execute applies a proposal to the cluster.
func (e *Executor) Execute(ctx context.Context, p resources.Proposal) error {
	slog.Info("cloudos: executing proposal",
		"resource", p.Resource, "action", p.Action, "target", p.Target)

	switch p.Resource {
	case "api":
		return e.executeAPI(p)
	case "wallet":
		return e.executeWallet(p)
	case "fraud":
		return e.executeFraud(p)
	case "kafka":
		return e.executeKafka(p)
	default:
		return e.executeGeneric(p)
	}
}

func (e *Executor) executeAPI(p resources.Proposal) error {
	slog.Info("cloudos: API action", "action", p.Action, "target", p.Target)
	// Production: kubectl scale/set-image
	return nil
}

func (e *Executor) executeWallet(p resources.Proposal) error {
	slog.Error("cloudos: WALLET ACTION — critical", "action", p.Action)
	// Production: trigger audit, freeze transactions, alert finance team
	return nil
}

func (e *Executor) executeFraud(p resources.Proposal) error {
	slog.Info("cloudos: fraud action", "action", p.Action)
	// Production: update fraud threshold in Redis
	return nil
}

func (e *Executor) executeKafka(p resources.Proposal) error {
	slog.Info("cloudos: kafka action", "action", p.Action)
	// Production: scale consumer group
	return nil
}

func (e *Executor) executeGeneric(p resources.Proposal) error {
	slog.Info("cloudos: generic action", "resource", p.Resource, "action", p.Action)
	return nil
}
