package saga

import (
	"log/slog"
)

// StepFn is a saga step that can fail.
type StepFn func() error

// CompensateFn is a compensation action that undoes a completed step.
type CompensateFn func()

// Saga orchestrates a multi-step transaction with compensation on failure.
type Saga struct {
	name         string
	steps        []StepFn
	compensations []CompensateFn
	completed    int
}

// New creates a new saga with the given name.
func New(name string) *Saga {
	return &Saga{name: name}
}

// AddStep adds a step and its corresponding compensation.
func (s *Saga) AddStep(step StepFn, compensate CompensateFn) *Saga {
	s.steps = append(s.steps, step)
	s.compensations = append(s.compensations, compensate)
	return s
}

// Execute runs all steps in order. On failure, rolls back completed steps.
func (s *Saga) Execute() error {
	for i, step := range s.steps {
		if err := step(); err != nil {
			slog.Error("saga: step failed, starting rollback",
				"saga", s.name,
				"step", i,
				"error", err,
			)
			s.rollback(i - 1)
			return err
		}
		s.completed = i + 1
		slog.Debug("saga: step completed", "saga", s.name, "step", i)
	}
	slog.Info("saga: completed successfully", "saga", s.name, "steps", len(s.steps))
	return nil
}

// Completed returns the number of successfully completed steps.
func (s *Saga) Completed() int {
	return s.completed
}

func (s *Saga) rollback(failedAt int) {
	if failedAt < 0 {
		return
	}
	slog.Info("saga: rolling back", "saga", s.name, "from_step", failedAt)
	for i := failedAt; i >= 0; i-- {
		if i < len(s.compensations) && s.compensations[i] != nil {
			slog.Debug("saga: running compensation", "saga", s.name, "step", i)
			s.compensations[i]()
		}
	}
	slog.Info("saga: rollback complete", "saga", s.name)
}
