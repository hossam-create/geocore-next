package remediation

import (
	"context"
	"log/slog"
	"time"
)

// CanaryStepper automatically increases canary weight over time
// when the deployment is healthy. Planned for Sprint 4.
//
// Weight progression: 10% → 25% → 50% → 100%
// Each step requires a healthy interval before advancing.
type CanaryStepper struct {
	steps      []int
	stepIndex  int
	interval   time.Duration
	monitor    *CanaryMonitor
}

// NewCanaryStepper creates a stepper with default progression.
func NewCanaryStepper(monitor *CanaryMonitor) *CanaryStepper {
	return &CanaryStepper{
		steps:     []int{10, 25, 50, 100},
		stepIndex: 0,
		interval:  5 * time.Minute,
		monitor:   monitor,
	}
}

// Start runs the auto-step loop. On each healthy interval, advances weight.
// On unhealthy, signals rollback and stops.
func (s *CanaryStepper) Start(ctx context.Context, setWeight func(weight int) error) {
	slog.Info("canary-stepper: started", "steps", s.steps, "interval", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("canary-stepper: stopped")
			return
		case <-ticker.C:
			health := s.monitor.Health()
			if !health.Healthy {
				slog.Error("canary-stepper: unhealthy — rolling back", "reason", health.Reason)
				SignalRollback("canary_stepper_unhealthy")
				return
			}

			if s.stepIndex >= len(s.steps)-1 {
				slog.Info("canary-stepper: fully promoted to 100%")
				return
			}

			s.stepIndex++
			weight := s.steps[s.stepIndex]
			slog.Info("canary-stepper: advancing weight", "weight", weight)
			if err := setWeight(weight); err != nil {
				slog.Error("canary-stepper: failed to set weight", "error", err)
			}
		}
	}
}

// CurrentWeight returns the current canary weight percentage.
func (s *CanaryStepper) CurrentWeight() int {
	if s.stepIndex < len(s.steps) {
		return s.steps[s.stepIndex]
	}
	return 100
}
