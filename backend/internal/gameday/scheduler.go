package gameday

import (
	"context"
	"log/slog"
	"time"
)

// GameDay represents a single resilience test scenario.
type GameDay struct {
	Name     string
	Scenario func(ctx context.Context)
}

// Scheduler runs GameDay scenarios on a weekly schedule.
type Scheduler struct {
	scenarios []GameDay
	interval  time.Duration
	running   bool
}

// NewScheduler creates a weekly GameDay scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		scenarios: defaultScenarios(),
		interval:  7 * 24 * time.Hour,
	}
}

// AddScenario adds a custom scenario to the rotation.
func (s *Scheduler) AddScenario(g GameDay) {
	s.scenarios = append(s.scenarios, g)
}

// Start begins the weekly GameDay loop.
func (s *Scheduler) Start(ctx context.Context) {
	if s.running {
		return
	}
	s.running = true
	slog.Info("gameday: scheduler started", "interval", s.interval, "scenarios", len(s.scenarios))

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		// Run immediately on start (for testing)
		s.runAll(ctx)

		for {
			select {
			case <-ctx.Done():
				slog.Info("gameday: scheduler stopped")
				s.running = false
				return
			case <-ticker.C:
				s.runAll(ctx)
			}
		}
	}()
}

func (s *Scheduler) runAll(ctx context.Context) {
	slog.Info("gameday: running all scenarios", "count", len(s.scenarios))
	for _, g := range s.scenarios {
		slog.Info("gameday: executing scenario", "name", g.Name)
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("gameday: scenario panicked", "name", g.Name, "panic", r)
				}
			}()
			g.Scenario(ctx)
		}()
		slog.Info("gameday: scenario completed", "name", g.Name)
	}
	slog.Info("gameday: all scenarios completed")
}

// IsRunning returns whether the scheduler is active.
func (s *Scheduler) IsRunning() bool {
	return s.running
}

// Scenarios returns the list of registered scenarios.
func (s *Scheduler) Scenarios() []string {
	var names []string
	for _, g := range s.scenarios {
		names = append(names, g.Name)
	}
	return names
}
