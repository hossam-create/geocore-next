package autonomy

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/chaos"
	"github.com/geocore-next/backend/internal/slo"
)

// AutonomyDecision is a decision made by the autonomous control plane.
type AutonomyDecision struct {
	Action string `json:"action"` // scale_up, scale_down, rollback, throttle, noop
	Reason string `json:"reason"`
}

// DecisionEngine is the brain of the autonomous control plane.
// It evaluates system metrics against SLOs and chaos state to decide actions.
type DecisionEngine struct {
	sloEngine   *slo.Engine
	chaosEngine *chaos.ChaosEngine
	maxReplicas int
}

// NewDecisionEngine creates a decision engine with SLO and chaos references.
func NewDecisionEngine(sloEngine *slo.Engine, chaosEngine *chaos.ChaosEngine) *DecisionEngine {
	return &DecisionEngine{
		sloEngine:   sloEngine,
		chaosEngine: chaosEngine,
		maxReplicas: 10,
	}
}

// Evaluate examines the current metrics snapshot and returns a decision.
func (d *DecisionEngine) Evaluate(ctx context.Context, snap MetricsSnapshot) AutonomyDecision {
	// Critical: error rate breach → immediate rollback
	if snap.ErrorRate > 5.0 {
		slog.Error("autonomy: critical error rate breach",
			"error_rate", snap.ErrorRate,
		)
		return AutonomyDecision{Action: "rollback", Reason: "error_rate_breach"}
	}

	// High latency → scale up
	if snap.P95Latency > 800 {
		slog.Warn("autonomy: latency breach, recommending scale up",
			"p95_ms", snap.P95Latency,
		)
		return AutonomyDecision{Action: "scale_up", Reason: "latency_breach"}
	}

	// SLO burning → throttle to protect error budget
	if d.sloEngine != nil && !d.sloEngine.IsHealthy() {
		slog.Warn("autonomy: SLO burning, recommending throttle")
		return AutonomyDecision{Action: "throttle", Reason: "slo_burn"}
	}

	// High RPS + low latency → consider scaling down to save cost
	if snap.RPS < 10 && snap.P95Latency < 100 && snap.ErrorRate < 0.5 {
		return AutonomyDecision{Action: "scale_down", Reason: "underutilized"}
	}

	// Kafka lag → scale up consumers
	if snap.KafkaLag > 5000 {
		slog.Warn("autonomy: Kafka lag high, recommending scale up",
			"lag", snap.KafkaLag,
		)
		return AutonomyDecision{Action: "scale_up", Reason: "kafka_lag"}
	}

	return AutonomyDecision{Action: "noop", Reason: "healthy"}
}

// SetMaxReplicas configures the maximum replica count for guardrails.
func (d *DecisionEngine) SetMaxReplicas(max int) {
	d.maxReplicas = max
}

// MaxReplicas returns the configured max replica count.
func (d *DecisionEngine) MaxReplicas() int {
	return d.maxReplicas
}
