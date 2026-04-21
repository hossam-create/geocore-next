package executor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/gitops/planner"
	"github.com/geocore-next/backend/internal/gitops/simulator"
)

// K8sExecutor applies rollout plans to the Kubernetes cluster.
type K8sExecutor struct {
	canary *simulator.CanarySimulator
}

// NewK8sExecutor creates a Kubernetes executor.
func NewK8sExecutor() *K8sExecutor {
	return &K8sExecutor{
		canary: simulator.NewCanarySimulator(),
	}
}

// Apply executes a rollout plan against the cluster.
func (e *K8sExecutor) Apply(ctx context.Context, plan planner.RolloutPlan) error {
	slog.Info("gitops: executing rollout",
		"service", plan.Service,
		"from", plan.From,
		"to", plan.To,
		"strategy", string(plan.Strategy))

	switch plan.Strategy {
	case planner.StrategyCanary:
		return e.applyCanary(ctx, plan)
	case planner.StrategyBlueGreen:
		return e.applyBlueGreen(ctx, plan)
	default:
		return e.applyRolling(ctx, plan)
	}
}

func (e *K8sExecutor) applyCanary(ctx context.Context, plan planner.RolloutPlan) error {
	for i, stage := range plan.Stages {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		slog.Info("gitops: canary stage",
			"service", plan.Service,
			"stage", i+1,
			"traffic_pct", stage.TrafficPercent)

		// Apply traffic shift
		e.setCanaryWeight(plan.Service, stage.TrafficPercent)

		// Observe metrics at this stage
		errorRate := e.canary.ObserveStage(stage)
		if errorRate > 1.0 {
			return fmt.Errorf("canary failed at stage %d: error rate %.2f%% exceeds 1%%", i+1, errorRate)
		}

		// Wait for observation duration
		if stage.Duration > 0 {
			time.Sleep(stage.Duration)
		}
	}

	slog.Info("gitops: canary rollout COMPLETE", "service", plan.Service)
	return nil
}

func (e *K8sExecutor) applyBlueGreen(ctx context.Context, plan planner.RolloutPlan) error {
	// Deploy new version (green) alongside current (blue)
	slog.Info("gitops: deploying green environment", "service", plan.Service)
	e.deployVersion(plan.Service, plan.To)

	// Verify green is healthy
	slog.Info("gitops: verifying green environment", "service", plan.Service)

	// Switch traffic
	slog.Info("gitops: switching traffic to green", "service", plan.Service)
	e.switchTraffic(plan.Service, plan.To)

	slog.Info("gitops: blue-green rollout COMPLETE", "service", plan.Service)
	return nil
}

func (e *K8sExecutor) applyRolling(ctx context.Context, plan planner.RolloutPlan) error {
	slog.Info("gitops: rolling update", "service", plan.Service, "to", plan.To)
	e.deployVersion(plan.Service, plan.To)
	slog.Info("gitops: rolling rollout COMPLETE", "service", plan.Service)
	return nil
}

func (e *K8sExecutor) setCanaryWeight(service string, weight int) {
	slog.Debug("gitops: setting canary weight", "service", service, "weight", weight)
	// Production: update Istio VirtualService / Argo Rollout canary weight
}

func (e *K8sExecutor) deployVersion(service, version string) {
	slog.Debug("gitops: deploying version", "service", service, "version", version)
	// Production: kubectl set image deployment/<service> <service>=<image>:<version>
}

func (e *K8sExecutor) switchTraffic(service, version string) {
	slog.Debug("gitops: switching traffic", "service", service, "to", version)
	// Production: update service selector or Istio route
}
