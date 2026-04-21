package autonomy

import (
	"context"
	"log/slog"
	"time"
)

// ControlLoop runs the autonomous observe→decide→act cycle.
type ControlLoop struct {
	engine   *DecisionEngine
	interval time.Duration
	running  bool
}

// NewControlLoop creates a control loop with the given decision engine.
func NewControlLoop(engine *DecisionEngine, interval time.Duration) *ControlLoop {
	return &ControlLoop{
		engine:   engine,
		interval: interval,
	}
}

// Start begins the autonomous control loop.
func (cl *ControlLoop) Start(ctx context.Context) {
	if cl.running {
		return
	}
	cl.running = true
	slog.Info("autonomy: control loop started", "interval", cl.interval)

	go func() {
		ticker := time.NewTicker(cl.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("autonomy: control loop stopped")
				cl.running = false
				return
			case <-ticker.C:
				cl.tick(ctx)
			}
		}
	}()
}

func (cl *ControlLoop) tick(ctx context.Context) {
	// 1. Observe — collect current metrics
	metrics := CollectMetrics()

	// 2. Decide — evaluate against SLOs and policies
	decision := cl.engine.Evaluate(ctx, metrics)

	// 3. Guard — check if action is safe
	if !AllowAction(decision) {
		slog.Warn("autonomy: decision blocked by guardrails",
			"action", decision.Action,
			"reason", decision.Reason,
		)
		return
	}

	// 4. Act — execute the decision
	if decision.Action != "noop" {
		Execute(decision)
		RecordAction(decision)
	}

	// 5. Log — record for observability
	LogSnapshot()
}

// IsRunning returns whether the control loop is active.
func (cl *ControlLoop) IsRunning() bool {
	return cl.running
}

// TriggerEmergencyMode forces an immediate rollback decision.
func TriggerEmergencyMode(reason string) {
	slog.Error("autonomy: EMERGENCY MODE triggered", "reason", reason)
	d := AutonomyDecision{Action: "rollback", Reason: reason}
	if AllowAction(d) {
		Execute(d)
		RecordAction(d)
	}
}
