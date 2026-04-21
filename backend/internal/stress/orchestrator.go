package stress

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// RunStatus is the live status polled during an active test.
type RunStatus struct {
	ID        string     `json:"id"`
	Scenario  string     `json:"scenario"`
	Phase     string     `json:"phase"`    // idle | ramp_N_of_M | validating | complete | aborted
	Progress  int        `json:"progress"` // 0–100
	StartedAt *time.Time `json:"started_at,omitempty"`
}

// Orchestrator coordinates load generation, chaos injection, validation, and reporting.
type Orchestrator struct {
	chaos     *ChaosEngine
	validator *Validator

	running    atomic.Bool
	mu         sync.RWMutex
	status     RunStatus
	cancelRun  context.CancelFunc
	lastReport *StressReport
}

func newOrchestrator() *Orchestrator {
	return &Orchestrator{
		chaos:     newChaosEngine(),
		validator: newValidator(),
		status:    RunStatus{Phase: "idle"},
	}
}

// NewOrchestrator creates an independent orchestrator for programmatic use (e.g. reslab).
func NewOrchestrator() *Orchestrator { return newOrchestrator() }

// RunSync runs a scenario synchronously, blocking until completion or ctx cancellation.
// Returns the final StressReport and nil error on success.
func (o *Orchestrator) RunSync(ctx context.Context, scenario Scenario) (*StressReport, error) {
	if !o.running.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("orchestrator busy: %s already running", o.status.Scenario)
	}
	innerCtx, cancel := context.WithCancel(ctx)
	o.mu.Lock()
	o.cancelRun = cancel
	o.mu.Unlock()
	defer func() {
		cancel()
		o.running.Store(false)
	}()
	o.run(innerCtx, scenario)
	return o.LastReport(), nil
}

// Status returns a snapshot of the current run state.
func (o *Orchestrator) Status() RunStatus {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.status
}

// LastReport returns the most recent completed test report, or nil if none.
func (o *Orchestrator) LastReport() *StressReport {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.lastReport
}

// IsRunning returns true if a test is currently active.
func (o *Orchestrator) IsRunning() bool { return o.running.Load() }

// StartAsync begins a stress test in the background and returns immediately.
// Returns an error if a test is already running.
func (o *Orchestrator) StartAsync(scenario Scenario) error {
	if !o.running.CompareAndSwap(false, true) {
		return fmt.Errorf("a stress test is already running: %s", o.status.Scenario)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	o.mu.Lock()
	o.cancelRun = cancel
	o.mu.Unlock()

	go func() {
		defer cancel()
		defer o.running.Store(false)
		o.run(ctx, scenario)
	}()
	return nil
}

// Stop aborts the currently running test.
func (o *Orchestrator) Stop() bool {
	o.mu.Lock()
	cancel := o.cancelRun
	o.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

// run is the internal blocking test loop.
func (o *Orchestrator) run(ctx context.Context, scenario Scenario) {
	id := uuid.New().String()
	start := time.Now()

	collector := newMetricsCollector()
	loader := newLoadGenerator(collector)
	o.chaos.reset()

	now := time.Now()
	o.setStatus(RunStatus{
		ID:        id,
		Scenario:  scenario.Name,
		Phase:     "initializing",
		Progress:  0,
		StartedAt: &now,
	})

	slog.Info("stress: test started", "id", id, "scenario", scenario.Name)

	total := len(scenario.LoadRamp)
	for i, ramp := range scenario.LoadRamp {
		if ctx.Err() != nil {
			o.setPhase("aborted", 0)
			slog.Warn("stress: aborted", "id", id)
			return
		}

		progress := int(float64(i) / float64(total) * 80)
		o.setPhase(fmt.Sprintf("ramp_%d_of_%d_%dusers", i+1, total, ramp.Users), progress)
		slog.Info("stress: load ramp", "step", i+1, "users", ramp.Users, "duration", ramp.Duration)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			loader.Wave(ctx, ramp.Users, ramp.Duration)
		}()

		go func(step int) {
			defer wg.Done()
			// Delay chaos to let the load ramp up first (1/3 into the ramp window)
			select {
			case <-time.After(ramp.Duration / 3):
			case <-ctx.Done():
				return
			}
			o.chaos.InjectForScenario(ctx, scenario.ChaosTypes)
		}(i)

		wg.Wait()
	}

	// Validation phase
	o.setPhase("validating", 85)
	slog.Info("stress: validating system health", "id", id)
	metrics := collector.Summary()
	validation := o.validator.Evaluate(ctx, metrics)

	// Build final report
	o.setPhase("generating_report", 95)
	report := buildReport(id, scenario, start, metrics, validation, o.chaos.Events())

	o.mu.Lock()
	o.lastReport = &report
	o.mu.Unlock()

	o.setPhase("complete", 100)
	slog.Info("stress: test complete",
		"id", id,
		"status", report.Status,
		"p95_ms", metrics.P95LatencyMs,
		"error_rate_pct", metrics.ErrorRatePct,
		"total_requests", metrics.TotalRequests,
		"aiops_incidents", validation.OpenIncidents,
	)
}

func (o *Orchestrator) setPhase(phase string, progress int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.status.Phase = phase
	o.status.Progress = progress
}

func (o *Orchestrator) setStatus(s RunStatus) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.status = s
}
