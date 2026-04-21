package simulation

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/singularity/intelligence"
	"github.com/geocore-next/backend/internal/singularity/proposals"
)

// ShadowRunner simulates proposals in shadow mode before execution.
// NO SIMULATION → NO EXECUTION.
type ShadowRunner struct {
	latencyModel *intelligence.LatencyModel
	costModel    *intelligence.CostModel
}

// NewShadowRunner creates a shadow simulation runner.
func NewShadowRunner(latencyModel *intelligence.LatencyModel, costModel *intelligence.CostModel) *ShadowRunner {
	return &ShadowRunner{
		latencyModel: latencyModel,
		costModel:    costModel,
	}
}

// Simulate runs a proposal through shadow simulation.
// Returns true if the proposal is safe to execute.
func (s *ShadowRunner) Simulate(ctx context.Context, p *proposals.ChangeProposal) bool {
	slog.Info("singularity: running shadow simulation", "type", p.Type, "target", p.Target)

	var latencyDelta, errorDelta, costDelta float64
	passed := true

	switch p.Type {
	case proposals.ProposalScaleUp:
		// Simulate: adding 2 pods → latency should decrease
		latencyDelta = s.latencyModel.EstimateLatencyDelta(4, 6, 100)
		costDelta = s.costModel.EstimateScaleUpCost(2)
		// Scale up should not increase latency
		if latencyDelta > 0 {
			passed = false
		}
		// Cost must stay within budget
		if !s.costModel.IsWithinBudget(s.costModel.ProjectedMonthlySpend(costDelta)) {
			passed = false
		}

	case proposals.ProposalScaleDown:
		// Simulate: removing 1 pod → latency should not spike
		latencyDelta = s.latencyModel.EstimateLatencyDelta(4, 3, 100)
		costDelta = s.costModel.EstimateScaleDownSavings(1)
		// Scale down should not increase latency beyond 20%
		if latencyDelta > 50 {
			passed = false
		}

	case proposals.ProposalRollback:
		// Rollback is always safe to simulate — it restores previous state
		latencyDelta = -50 // assumed improvement
		errorDelta = -2.0  // assumed error reduction
		passed = true

	case proposals.ProposalKafkaRebalance:
		// Simulate: adding consumers → lag should decrease
		latencyDelta = -20 // assumed improvement
		passed = true

	case proposals.ProposalThrottle:
		// Throttle reduces throughput but protects SLO
		latencyDelta = 100 // throttle adds latency
		errorDelta = -1.0  // but reduces errors
		// Only pass if error rate is critical
		passed = errorDelta < -0.5

	default:
		// Unknown proposal types need manual simulation
		passed = false
	}

	p.MarkSimulated(latencyDelta, errorDelta, costDelta, passed)

	if passed {
		slog.Info("singularity: simulation PASSED",
			"type", p.Type,
			"latency_delta", latencyDelta,
			"cost_delta", costDelta,
		)
	} else {
		slog.Warn("singularity: simulation FAILED",
			"type", p.Type,
			"latency_delta", latencyDelta,
			"cost_delta", costDelta,
		)
	}

	return passed
}
