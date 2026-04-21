package core

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/controlplane/analyzer"
	"github.com/geocore-next/backend/internal/controlplane/executor"
	"github.com/geocore-next/backend/internal/controlplane/planner"
	"github.com/geocore-next/backend/internal/controlplane/policy"
	"github.com/geocore-next/backend/internal/controlplane/simulator"
)

// Reconciler is the heart of the control plane.
// It implements the observe → analyze → plan → simulate → guard → execute loop.
type Reconciler struct {
	Analyzer  *analyzer.Analyzer
	Planner   *planner.Planner
	Simulator *simulator.Simulator
	Executor  *executor.Executor
	Policy    *policy.PolicyEngine
}

// NewReconciler creates a reconciler with all components wired together.
func NewReconciler(
	a *analyzer.Analyzer,
	p *planner.Planner,
	s *simulator.Simulator,
	e *executor.Executor,
	pol *policy.PolicyEngine,
) *Reconciler {
	return &Reconciler{
		Analyzer:  a,
		Planner:   p,
		Simulator: s,
		Executor:  e,
		Policy:    pol,
	}
}

// Reconcile runs one full reconciliation cycle.
func (r *Reconciler) Reconcile(ctx context.Context) {
	// 1. Observe — collect current system metrics
	metrics := r.Analyzer.Collect(ctx)

	// 2. Analyze — detect anomalies
	anomalies := r.Analyzer.DetectAnomalies(metrics)
	if len(anomalies) == 0 {
		return // system is healthy
	}

	// 3. Plan — generate optimization proposals
	proposals := r.Planner.Generate(metrics, anomalies)
	if len(proposals) == 0 {
		return
	}

	// 4. Evaluate each proposal through the full pipeline
	for i := range proposals {
		p := &proposals[i]

		// 4a. Policy gate — is this proposal allowed?
		if !r.Policy.Allow(*p) {
			slog.Warn("controlplane: proposal blocked by policy",
				"type", p.Type, "target", p.Target, "risk", p.RiskScore)
			continue
		}

		// 4b. Simulate — shadow execution to predict impact
		impact := r.Simulator.Run(ctx, *p)
		if !impact.Safe {
			slog.Warn("controlplane: proposal unsafe after simulation",
				"type", p.Type, "latency_delta", impact.LatencyDelta,
				"error_delta", impact.ErrorDelta)
			continue
		}

		// 4c. Execute — apply the change
		slog.Info("controlplane: executing proposal",
			"type", p.Type, "target", p.Target, "action", p.Action)
		r.Executor.Execute(ctx, *p)
	}
}
