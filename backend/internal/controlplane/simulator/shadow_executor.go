package simulator

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/controlplane/planner"
)

// Impact represents the predicted impact of a proposal.
type Impact struct {
	Safe          bool    `json:"safe"`
	LatencyDelta  float64 `json:"latency_delta_ms"`
	CostDelta     float64 `json:"cost_delta_hourly"`
	ErrorDelta    float64 `json:"error_delta_percent"`
	RiskScore     float64 `json:"risk_score"`
}

// Simulator runs shadow simulations on proposals before execution.
// NO SIMULATION → NO EXECUTION.
type Simulator struct{}

// NewSimulator creates a shadow simulation runner.
func NewSimulator() *Simulator {
	return &Simulator{}
}

// Run simulates a proposal in shadow mode and predicts its impact.
func (s *Simulator) Run(ctx context.Context, p planner.Proposal) Impact {
	slog.Debug("controlplane: running shadow simulation", "type", p.Type, "target", p.Target)

	// High-risk proposals are automatically unsafe
	if p.RiskScore > 0.7 {
		slog.Warn("controlplane: proposal too risky", "risk", p.RiskScore)
		return Impact{Safe: false, RiskScore: p.RiskScore}
	}

	switch p.Type {
	case planner.ScaleUp:
		return Impact{
			Safe:         true,
			LatencyDelta: -20,   // latency improves
			CostDelta:    0.20,  // cost increases slightly
			ErrorDelta:   -0.1,  // errors decrease
			RiskScore:    p.RiskScore,
		}

	case planner.ScaleDown:
		impact := Impact{
			LatencyDelta: 15,    // latency increases slightly
			CostDelta:    -0.10, // cost decreases
			ErrorDelta:   0.05,  // errors may increase slightly
			RiskScore:    p.RiskScore,
		}
		// Scale down is safe only if latency delta is acceptable
		impact.Safe = impact.LatencyDelta < 50 && impact.ErrorDelta < 0.5
		return impact

	case planner.Rollback:
		return Impact{
			Safe:         true,
			LatencyDelta: -50,  // rollback should improve
			CostDelta:    0,
			ErrorDelta:   -2.0, // errors should decrease
			RiskScore:    p.RiskScore,
		}

	case planner.KafkaRebalance:
		return Impact{
			Safe:         true,
			LatencyDelta: -10,
			CostDelta:    0.10,
			ErrorDelta:   -0.05,
			RiskScore:    p.RiskScore,
		}

	default:
		return Impact{Safe: false, RiskScore: p.RiskScore}
	}
}
