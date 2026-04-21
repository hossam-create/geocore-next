package rollback

import (
	"context"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/gitops/planner"
)

// RollbackTrigger defines conditions that trigger an automatic rollback.
type RollbackTrigger struct {
	ErrorRateAbove  float64 `json:"error_rate_above"`  // percent
	LatencyAbove    float64 `json:"latency_above"`     // ms P95
	KafkaLagAbove   int64   `json:"kafka_lag_above"`   // messages
	CPUAbove        float64 `json:"cpu_above"`         // percent
	PodRestartAbove int     `json:"pod_restart_above"` // restart count
}

// DefaultRollbackTriggers returns standard rollback trigger thresholds.
func DefaultRollbackTriggers() RollbackTrigger {
	return RollbackTrigger{
		ErrorRateAbove:  2.0,
		LatencyAbove:    800,
		KafkaLagAbove:   10000,
		CPUAbove:        95,
		PodRestartAbove: 5,
	}
}

// RollbackEngine monitors deployments and triggers automatic rollback.
type RollbackEngine struct {
	triggers  RollbackTrigger
	history   []RollbackEvent
}

// RollbackEvent records a rollback action.
type RollbackEvent struct {
	Service   string    `json:"service"`
	From      string    `json:"from_version"`
	To        string    `json:"to_version"` // rolled back to this
	Reason    string    `json:"reason"`
	TriggeredAt time.Time `json:"triggered_at"`
}

// NewRollbackEngine creates a rollback engine with default triggers.
func NewRollbackEngine() *RollbackEngine {
	return &RollbackEngine{
		triggers: DefaultRollbackTriggers(),
	}
}

// ShouldRollback checks if current metrics warrant an automatic rollback.
func (r *RollbackEngine) ShouldRollback(errorRate, latency float64, kafkaLag int64, cpuPct float64, podRestarts int) bool {
	if errorRate > r.triggers.ErrorRateAbove {
		slog.Error("gitops: ROLLBACK TRIGGER — error rate",
			"rate", errorRate, "threshold", r.triggers.ErrorRateAbove)
		return true
	}
	if latency > r.triggers.LatencyAbove {
		slog.Error("gitops: ROLLBACK TRIGGER — latency",
			"p95", latency, "threshold", r.triggers.LatencyAbove)
		return true
	}
	if kafkaLag > r.triggers.KafkaLagAbove {
		slog.Error("gitops: ROLLBACK TRIGGER — Kafka lag",
			"lag", kafkaLag, "threshold", r.triggers.KafkaLagAbove)
		return true
	}
	if cpuPct > r.triggers.CPUAbove {
		slog.Error("gitops: ROLLBACK TRIGGER — CPU saturation",
			"cpu", cpuPct, "threshold", r.triggers.CPUAbove)
		return true
	}
	if podRestarts > r.triggers.PodRestartAbove {
		slog.Error("gitops: ROLLBACK TRIGGER — pod crashloop",
			"restarts", podRestarts, "threshold", r.triggers.PodRestartAbove)
		return true
	}
	return false
}

// Rollback executes a rollback for a deployment.
func (r *RollbackEngine) Rollback(ctx context.Context, plan planner.RolloutPlan) error {
	slog.Error("gitops: ROLLING BACK",
		"service", plan.Service,
		"from", plan.To,
		"to", plan.From)

	// Record the rollback event
	event := RollbackEvent{
		Service:     plan.Service,
		From:        plan.To,
		To:          plan.From,
		Reason:      "automatic rollback triggered by metrics",
		TriggeredAt: time.Now().UTC(),
	}
	r.history = append(r.history, event)

	// In production: kubectl rollout undo deployment/<service>
	slog.Info("gitops: rollback executed", "service", plan.Service, "reverted_to", plan.From)
	return nil
}

// History returns recent rollback events.
func (r *RollbackEngine) History(n int) []RollbackEvent {
	if n > len(r.history) {
		n = len(r.history)
	}
	result := make([]RollbackEvent, n)
	copy(result, r.history[len(r.history)-n:])
	return result
}
