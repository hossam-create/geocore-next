package stress

import (
	"fmt"
	"strings"
	"time"
)

// StressReport is the complete outcome of a stress test run.
type StressReport struct {
	ID          string           `json:"id"`
	Scenario    string           `json:"scenario"`
	StartedAt   time.Time        `json:"started_at"`
	FinishedAt  time.Time        `json:"finished_at"`
	Duration    string           `json:"duration"`
	Metrics     MetricsSummary   `json:"metrics"`
	Validation  ValidationResult `json:"validation"`
	ChaosEvents []ChaosEvent     `json:"chaos_events"`
	Summary     string           `json:"summary"`
	Status      string           `json:"status"` // passed | failed | aborted
}

func buildReport(
	id string,
	scenario Scenario,
	start time.Time,
	metrics MetricsSummary,
	validation ValidationResult,
	chaosEvents []ChaosEvent,
) StressReport {
	finish := time.Now()
	status := "passed"
	if !validation.Passed {
		status = "failed"
	}

	r := StressReport{
		ID:          id,
		Scenario:    scenario.Name,
		StartedAt:   start,
		FinishedAt:  finish,
		Duration:    finish.Sub(start).Round(time.Millisecond).String(),
		Metrics:     metrics,
		Validation:  validation,
		ChaosEvents: chaosEvents,
		Status:      status,
	}
	r.Summary = formatSummary(r)
	return r
}

func formatSummary(r StressReport) string {
	var sb strings.Builder

	sb.WriteString("🚨 STRESS TEST COMPLETE\n")
	sb.WriteString(fmt.Sprintf("Scenario:  %s\n", r.Scenario))
	sb.WriteString(fmt.Sprintf("Duration:  %s\n", r.Duration))
	sb.WriteString(fmt.Sprintf("Status:    %s\n\n", strings.ToUpper(r.Status)))

	ok := func(b bool) string {
		if b {
			return "✅"
		}
		return "❌"
	}

	sb.WriteString("Results:\n")
	sb.WriteString(fmt.Sprintf("  p50 latency:     %.0f ms\n", r.Metrics.P50LatencyMs))
	sb.WriteString(fmt.Sprintf("  p95 latency:     %.0f ms   %s\n", r.Metrics.P95LatencyMs, ok(r.Validation.P95LatencyOK)))
	sb.WriteString(fmt.Sprintf("  p99 latency:     %.0f ms\n", r.Metrics.P99LatencyMs))
	sb.WriteString(fmt.Sprintf("  error rate:      %.2f %%  %s\n", r.Metrics.ErrorRatePct, ok(r.Validation.ErrorRateOK)))
	sb.WriteString(fmt.Sprintf("  peak RPS:        %.1f\n", r.Metrics.RPS))
	sb.WriteString(fmt.Sprintf("  total requests:  %d\n", r.Metrics.TotalRequests))
	sb.WriteString(fmt.Sprintf("  health status:   %s\n", r.Validation.HealthStatus))

	sb.WriteString("\nAIOps:\n")
	sb.WriteString(fmt.Sprintf("  incidents triggered: %d   %s\n",
		r.Validation.OpenIncidents, ok(r.Validation.AIOpsTriggered)))

	if len(r.ChaosEvents) > 0 {
		sb.WriteString(fmt.Sprintf("\nChaos Injected (%d events):\n", len(r.ChaosEvents)))
		for _, e := range r.ChaosEvents {
			sb.WriteString(fmt.Sprintf("  %s\n", e.Action))
		}
	}

	if len(r.Validation.FailureReasons) > 0 {
		sb.WriteString("\n⚠️  SLO Violations:\n")
		for _, f := range r.Validation.FailureReasons {
			sb.WriteString(fmt.Sprintf("  ❌ %s\n", f))
		}
	}

	if r.Status == "passed" {
		sb.WriteString("\n✅ PASSED — system handles this load within SLO thresholds\n")
	} else {
		sb.WriteString("\n❌ FAILED — system violated SLO thresholds under this load\n")
	}

	return sb.String()
}
