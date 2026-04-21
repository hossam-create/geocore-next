package aiops

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// Engine is the core AIOps orchestration loop.
// It scans Prometheus metrics every 30 seconds, runs RCA + runbook generation
// on anomalies, stores incidents in-memory, and notifies via Slack/PagerDuty.
type Engine struct {
	detector *Detector
	builder  *ContextBuilder
	rca      *RCAEngine
	runbook  *RunbookGenerator
	notifier *Notifier

	interval   time.Duration
	running    atomic.Bool
	totalScans atomic.Int64

	mu       sync.Mutex
	lastScan time.Time
}

// NewEngine creates a fully wired AIOps engine using environment configuration.
func NewEngine() *Engine {
	detector := NewDetector(nil)
	llm := NewLLMClient()
	return &Engine{
		detector: detector,
		builder:  NewContextBuilder(detector),
		rca:      NewRCAEngine(llm),
		runbook:  NewRunbookGenerator(llm),
		notifier: NewNotifier(),
		interval: 30 * time.Second,
	}
}

// Start begins the background scan loop. No-op when PROMETHEUS_URL is not set.
func (e *Engine) Start(ctx context.Context) {
	if os.Getenv("PROMETHEUS_URL") == "" {
		slog.Info("aiops: PROMETHEUS_URL not configured — engine disabled (set to enable)")
		return
	}
	e.running.Store(true)
	slog.Info("aiops: engine started", "interval", e.interval, "rules", len(defaultRules))
	go e.loop(ctx)
}

func (e *Engine) loop(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer func() {
		ticker.Stop()
		e.running.Store(false)
	}()
	for {
		select {
		case <-ticker.C:
			e.tick(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (e *Engine) tick(ctx context.Context) {
	e.mu.Lock()
	e.lastScan = time.Now()
	e.mu.Unlock()
	e.totalScans.Add(1)

	incidents := e.detector.Scan(ctx)
	if len(incidents) == 0 {
		return
	}

	for _, inc := range incidents {
		slog.Warn("aiops: incident detected",
			"id", inc.ID,
			"severity", inc.Severity,
			"service", inc.Service,
			"metric", inc.Metric,
			"value", inc.Value,
		)

		// Build rich system context for analysis
		sysCtx := e.builder.Build(ctx, inc)

		// Root cause analysis
		inc.RCA = e.rca.Analyze(ctx, sysCtx)

		// Actionable runbook
		inc.Runbook = e.runbook.Generate(ctx, inc, inc.RCA)

		// Human-readable description
		inc.Description = fmt.Sprintf("Metric %s = %.4f (baseline: %.4f)", inc.Metric, inc.Value, inc.Baseline)

		// Persist to in-memory store
		registry.Add(inc)

		// Notify
		if err := e.notifier.Send(ctx, inc); err != nil {
			slog.Error("aiops: slack notification failed", "incident_id", inc.ID, "error", err)
		}
		if err := e.notifier.SendPagerDuty(ctx, inc); err != nil {
			slog.Error("aiops: pagerduty notification failed", "incident_id", inc.ID, "error", err)
		}
	}
}

// Status returns a snapshot of the engine's operational state.
func (e *Engine) Status() map[string]interface{} {
	e.mu.Lock()
	lastScan := e.lastScan
	e.mu.Unlock()

	return map[string]interface{}{
		"running":              e.running.Load(),
		"last_scan":            lastScan,
		"total_scans":          e.totalScans.Load(),
		"open_incidents":       registry.OpenCount(),
		"prometheus_connected": os.Getenv("PROMETHEUS_URL") != "",
		"llm_enabled":          os.Getenv("OPENAI_API_KEY") != "",
		"slack_enabled":        os.Getenv("SLACK_WEBHOOK_URL") != "",
		"pagerduty_enabled":    os.Getenv("PAGERDUTY_KEY") != "",
	}
}
