package simulator

import (
	"context"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/gitops/planner"
)

// SimulationResult represents the outcome of a canary simulation.
type SimulationResult struct {
	Safe          bool    `json:"safe"`
	LatencyDelta  float64 `json:"latency_delta_ms"`
	ErrorDelta    float64 `json:"error_delta_percent"`
	CostDelta     float64 `json:"cost_delta"`
	RiskScore     float64 `json:"risk_score"`
	Reason        string  `json:"reason"`
}

// CanarySimulator runs shadow/canary simulations before production rollout.
type CanarySimulator struct{}

// NewCanarySimulator creates a canary simulator.
func NewCanarySimulator() *CanarySimulator {
	return &CanarySimulator{}
}

// RunCanary simulates a rollout plan in shadow mode.
func (s *CanarySimulator) RunCanary(ctx context.Context, plan planner.RolloutPlan) SimulationResult {
	slog.Info("gitops: running canary simulation",
		"service", plan.Service,
		"from", plan.From,
		"to", plan.To,
		"strategy", string(plan.Strategy))

	// High-risk plans need stricter simulation
	if plan.Risk > 0.7 {
		return SimulationResult{
			Safe:      false,
			RiskScore: plan.Risk,
			Reason:    "risk score exceeds maximum threshold",
		}
	}

	// Simulate impact based on strategy
	switch plan.Strategy {
	case planner.StrategyCanary:
		return s.simulateCanary(plan)
	case planner.StrategyBlueGreen:
		return s.simulateBlueGreen(plan)
	default:
		return s.simulateRolling(plan)
	}
}

func (s *CanarySimulator) simulateCanary(plan planner.RolloutPlan) SimulationResult {
	// Canary: progressive traffic shift, lower blast radius
	return SimulationResult{
		Safe:         true,
		LatencyDelta: -5,   // slight improvement expected
		ErrorDelta:   -0.1, // errors should decrease
		CostDelta:    0.05, // slight cost increase (new pods)
		RiskScore:    plan.Risk,
		Reason:       "canary simulation passed",
	}
}

func (s *CanarySimulator) simulateBlueGreen(plan planner.RolloutPlan) SimulationResult {
	// Blue/Green: instant switch, but verified before
	return SimulationResult{
		Safe:         true,
		LatencyDelta: -10,
		ErrorDelta:   -0.2,
		CostDelta:    0.10, // double resources briefly
		RiskScore:    plan.Risk,
		Reason:       "blue-green simulation passed",
	}
}

func (s *CanarySimulator) simulateRolling(plan planner.RolloutPlan) SimulationResult {
	// Rolling: standard update, moderate risk
	if plan.Risk > 0.4 {
		return SimulationResult{
			Safe:      false,
			RiskScore: plan.Risk,
			Reason:    "rolling update too risky for this change, canary required",
		}
	}

	return SimulationResult{
		Safe:         true,
		LatencyDelta: 0,
		ErrorDelta:   0,
		CostDelta:    0,
		RiskScore:    plan.Risk,
		Reason:       "rolling simulation passed",
	}
}

// ObserveStage simulates observing metrics during a canary stage.
// Returns the simulated error rate at this traffic percentage.
func (s *CanarySimulator) ObserveStage(stage planner.RolloutStage) float64 {
	// In production: read real Prometheus metrics
	// Simulated: assume healthy (0.5% error rate)
	_ = time.Now() // placeholder
	return 0.5
}
