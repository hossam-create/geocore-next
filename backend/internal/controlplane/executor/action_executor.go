package executor

import (
	"context"
	"log/slog"
	"os/exec"

	"github.com/geocore-next/backend/internal/controlplane/planner"
)

// Executor carries out approved proposals.
type Executor struct {
	Rollout *RolloutManager
}

// NewExecutor creates an executor with a rollout manager.
func NewExecutor() *Executor {
	return &Executor{
		Rollout: NewRolloutManager(),
	}
}

// Execute applies a proposal's action.
func (e *Executor) Execute(ctx context.Context, p planner.Proposal) {
	slog.Info("controlplane: executing proposal",
		"type", p.Type, "target", p.Target, "action", p.Action)

	switch p.Type {
	case planner.ScaleUp:
		scaleUp(p.Target)
	case planner.ScaleDown:
		scaleDown(p.Target)
	case planner.Rollback:
		rolloutUndo(p.Target)
	case planner.KafkaRebalance:
		rebalanceConsumers(p.Target)
	case planner.Throttle:
		enableThrottle()
	default:
		slog.Warn("controlplane: unknown proposal type", "type", p.Type)
	}
}

func scaleUp(service string) {
	slog.Info("controlplane: scaling up", "service", service)
	_ = exec.Command("kubectl", "scale", "deployment", service, "--replicas=+2").Run()
}

func scaleDown(service string) {
	slog.Info("controlplane: scaling down", "service", service)
	_ = exec.Command("kubectl", "scale", "deployment", service, "--replicas=-1").Run()
}

func rolloutUndo(service string) {
	slog.Error("controlplane: ROLLING BACK", "service", service)
	_ = exec.Command("kubectl", "rollout", "undo", "deployment/"+service).Run()
}

func rebalanceConsumers(group string) {
	slog.Info("controlplane: rebalancing Kafka consumers", "group", group)
}

func enableThrottle() {
	slog.Warn("controlplane: enabling global throttle (50%)")
}
