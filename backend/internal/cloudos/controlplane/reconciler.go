package controlplane

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/cloudos/gitops"
	"github.com/geocore-next/backend/internal/cloudos/intelligence"
	"github.com/geocore-next/backend/internal/cloudos/policy"
	"github.com/geocore-next/backend/internal/cloudos/resources"
	"github.com/geocore-next/backend/internal/cloudos/runtime"
	"github.com/geocore-next/backend/pkg/telemetry"
)

// Reconciler is the heart of the Cloud OS.
// It implements the full observe → diff → plan → validate → simulate → execute → verify → learn loop.
type Reconciler struct {
	Resources   *resources.Manager
	Intel       *intelligence.Hub
	Policy      *policy.Engine
	GitOps      *gitops.Watcher
	Runtime     *runtime.Executor
	Telemetry   *telemetry.Collector
	Learner     *Learner
}

// NewReconciler creates a Cloud OS reconciler with all subsystems.
func NewReconciler(
	res *resources.Manager,
	intel *intelligence.Hub,
	pol *policy.Engine,
	git *gitops.Watcher,
	rt *runtime.Executor,
	tel *telemetry.Collector,
) *Reconciler {
	return &Reconciler{
		Resources: res,
		Intel:     intel,
		Policy:    pol,
		GitOps:    git,
		Runtime:   rt,
		Telemetry: tel,
		Learner:   NewLearner(),
	}
}

// Reconcile runs one full Cloud OS reconciliation cycle.
func (r *Reconciler) Reconcile(ctx context.Context) {
	// 1. Observe — collect current cluster state
	state := r.CollectClusterState(ctx)

	// 2. Diff — compare against desired state from Git
	desired := r.ReadGitDesiredState(ctx)
	diff := r.CalculateDiff(state, desired)

	if len(diff) == 0 {
		return // system is converged
	}

	// 3. Intelligence — AI-driven optimization proposals
	proposals := r.Intel.Analyze(state, diff)

	// 4. Policy — validate all proposals against safety gates
	for i := range proposals {
		p := &proposals[i]

		if !r.Policy.Allowed(*p) {
			slog.Warn("cloudos: proposal blocked by policy",
				"resource", p.Resource, "action", p.Action, "risk", p.RiskScore)
			continue
		}

		// 5. Simulate — shadow test the proposal
		sim := r.Runtime.Simulate(ctx, *p)
		if !sim.Safe {
			slog.Warn("cloudos: proposal unsafe after simulation",
				"resource", p.Resource, "reason", sim.Reason)
			continue
		}

		// 6. Execute — apply the change
		if err := r.Runtime.Execute(ctx, *p); err != nil {
			slog.Error("cloudos: execution failed",
				"resource", p.Resource, "error", err)
			continue
		}

		// 7. Verify — check post-deploy health
		healthy := r.Verify(ctx, *p)
		if !healthy {
			slog.Error("cloudos: post-deploy verification failed — ROLLING BACK",
				"resource", p.Resource)
			r.Runtime.Rollback(ctx, *p)
			continue
		}

		// 8. Learn — record outcome for future optimization
		r.Learner.Record(*p, state, healthy)
		slog.Info("cloudos: change applied successfully",
			"resource", p.Resource, "action", p.Action)
	}
}

// CollectClusterState gathers the current state of all managed resources.
func (r *Reconciler) CollectClusterState(ctx context.Context) resources.ClusterState {
	metrics := r.Telemetry.Collect()
	return r.Resources.CollectState(ctx, metrics)
}

// ReadGitDesiredState reads the desired state from the Git repository.
func (r *Reconciler) ReadGitDesiredState(ctx context.Context) resources.DesiredState {
	changes := r.GitOps.DetectChanges(ctx)
	return r.Resources.DesiredFromChanges(changes)
}

// CalculateDiff computes the diff between current and desired state.
func (r *Reconciler) CalculateDiff(state resources.ClusterState, desired resources.DesiredState) []resources.Proposal {
	return r.Resources.Diff(state, desired)
}

// Verify checks system health after a change.
func (r *Reconciler) Verify(ctx context.Context, p resources.Proposal) bool {
	metrics := r.Telemetry.Collect()
	return metrics.ErrorRate < 2.0 && metrics.P95Latency < 500
}
