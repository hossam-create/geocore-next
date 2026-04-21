package executor

import (
	"context"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/singularity/proposals"
	"github.com/geocore-next/backend/internal/singularity/simulation"
)

// RolloutEngine executes approved proposals via gradual canary rollout.
type RolloutEngine struct {
	canary *simulation.CanarySimulation
}

// NewRolloutEngine creates a rollout engine with standard canary stages.
func NewRolloutEngine() *RolloutEngine {
	return &RolloutEngine{
		canary: simulation.NewCanarySimulation(),
	}
}

// Execute runs a proposal through the canary rollout process.
// Proposal → Canary (1%) → Observe → 10% → 50% → 100%
func (e *RolloutEngine) Execute(ctx context.Context, p *proposals.ChangeProposal) error {
	if !p.IsSafe() {
		slog.Warn("singularity: rejecting unsafe proposal", "type", p.Type)
		p.Reject("safety_check_failed")
		return nil
	}

	slog.Info("singularity: starting canary rollout",
		"type", p.Type, "target", p.Target)

	p.Status = "executing"
	e.canary = simulation.NewCanarySimulation()

	// Simulate gradual rollout steps
	triggers := simulation.DefaultRollbackTriggers()

	for !e.canary.IsComplete() {
		select {
		case <-ctx.Done():
			slog.Info("singularity: rollout cancelled by context")
			return ctx.Err()
		default:
		}

		// In production, observe real metrics at each step.
		// Here we simulate with the proposal's expected values.
		simulatedErrorRate := 0.5 // assume healthy
		if p.SimulatedErrorDelta > 0 {
			simulatedErrorRate = p.SimulatedErrorDelta
		}

		advanced := e.canary.Advance(simulatedErrorRate)
		if !advanced {
			if e.canary.ShouldRollback() {
				slog.Error("singularity: canary failed — ROLLING BACK",
					"type", p.Type, "step", e.canary.Current)
				p.Status = "rolled_back"
				return nil
			}
			p.Status = "rolled_back"
			return nil
		}

		// Check rollback triggers
		if triggers.ShouldTriggerRollback(
			p.SimulatedLatencyDelta,
			p.SimulatedErrorDelta,
			0,   // kafka lag (not applicable to all proposals)
			50,  // db saturation (assumed healthy)
		) {
			slog.Error("singularity: rollback trigger fired — ROLLING BACK")
			p.Status = "rolled_back"
			return nil
		}

		// Wait between steps (in production, this is real observation time)
		time.Sleep(100 * time.Millisecond) // shortened for simulation
	}

	p.Status = "completed"
	now := time.Now().UTC()
	p.AppliedAt = &now
	slog.Info("singularity: rollout COMPLETE", "type", p.Type, "target", p.Target)
	return nil
}
