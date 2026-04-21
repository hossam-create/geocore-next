package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geocore-next/backend/internal/gitops/diff"
	"github.com/geocore-next/backend/internal/gitops/executor"
	"github.com/geocore-next/backend/internal/gitops/planner"
	"github.com/geocore-next/backend/internal/gitops/policy"
	"github.com/geocore-next/backend/internal/gitops/rollback"
	"github.com/geocore-next/backend/internal/gitops/simulator"
	"github.com/geocore-next/backend/internal/gitops/watcher"
	"github.com/geocore-next/backend/pkg/telemetry"
)

func main() {
	slog.Info("gitops: starting GitOps Autopilot")

	// Configuration from environment
	repoURL := envOr("GIT_REPO_URL", "https://github.com/geocore-next/infra.git")
	branch := envOr("GIT_BRANCH", "main")
	interval := 10 * time.Second
	if v := os.Getenv("POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	// Initialize components
	repoWatcher := watcher.NewRepoWatcher(repoURL, branch)
	diffEngine := diff.NewDiffEngine()
	rolloutPlanner := planner.NewRolloutPlanner()
	policyEngine := policy.NewPolicyEngine()
	canarySim := simulator.NewCanarySimulator()
	k8sExecutor := executor.NewK8sExecutor()
	rollbackEngine := rollback.NewRollbackEngine()
	telemetryCollector := telemetry.NewCollector()

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("gitops: received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Main reconciliation loop
	slog.Info("gitops: autopilot running", "interval", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("gitops: autopilot stopped")
			return
		case <-ticker.C:
			reconcile(ctx, repoWatcher, diffEngine, rolloutPlanner, policyEngine, canarySim, k8sExecutor, rollbackEngine, telemetryCollector)
		}
	}
}

func reconcile(
	ctx context.Context,
	w *watcher.RepoWatcher,
	d *diff.DiffEngine,
	p *planner.RolloutPlanner,
	pol *policy.PolicyEngine,
	sim *simulator.CanarySimulator,
	exec *executor.K8sExecutor,
	rb *rollback.RollbackEngine,
	tel *telemetry.Collector,
) {
	// 1. Watch — detect changes from Git
	changes := w.DetectChanges(ctx)
	if len(changes) == 0 {
		return
	}

	for _, change := range changes {
		// 2. Diff — compute desired vs live state
		stateDiff := d.Compute(change)
		if !stateDiff.Drift {
			continue // no drift, nothing to do
		}

		// 3. Plan — create rollout strategy
		plan := p.BuildPlan(change, stateDiff)

		// 4. Policy — check safety gates
		decision := pol.Allowed(stateDiff, plan)
		if !decision.Allowed {
			slog.Warn("gitops: change blocked by policy",
				"service", change.Service,
				"blocked_by", decision.BlockedBy)
			continue
		}

		// 5. Simulate — canary/shadow test
		simResult := sim.RunCanary(ctx, plan)
		if !simResult.Safe {
			slog.Warn("gitops: change unsafe after simulation",
				"service", change.Service,
				"reason", simResult.Reason)
			continue
		}

		// 6. Execute — apply to Kubernetes
		if err := exec.Apply(ctx, plan); err != nil {
			slog.Error("gitops: rollout failed",
				"service", change.Service, "error", err)
			continue
		}

		// 7. Observe — check metrics post-deploy
		metrics := tel.Collect()
		if rb.ShouldRollback(metrics.ErrorRate, metrics.P95Latency, metrics.KafkaLag, metrics.CPUUsage, metrics.PodRestarts) {
			// 8. Rollback — automatic safety rollback
			_ = rb.Rollback(ctx, plan)
			continue
		}

		// 9. Update — record live state
		d.UpdateLiveState(change.Service, change.Version)
		slog.Info("gitops: deployment successful",
			"service", change.Service,
			"version", change.Version)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
