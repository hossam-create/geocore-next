package controlplane

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/remediation"
)

// IncidentState holds the current incident detection state.
type IncidentState struct {
	Active      bool      `json:"active"`
	ErrorRate   float64   `json:"error_rate"`
	LatencyMs   float64   `json:"latency_ms"`
	DetectedAt  time.Time `json:"detected_at,omitempty"`
	Reason      string    `json:"reason,omitempty"`
}

var lastIncident IncidentState

// DetectIncident evaluates error rate and latency to detect incidents.
// Returns true if an incident is detected.
func DetectIncident(errorRate float64, latencyMs float64) bool {
	return errorRate > 5.0 || latencyMs > 1000
}

// EvaluateIncident checks metrics and updates incident state.
// Triggers auto-remediation when an incident is detected.
func EvaluateIncident(errorRate float64, latencyMs float64) IncidentState {
	if DetectIncident(errorRate, latencyMs) {
		reason := "error_rate_exceeded"
		if latencyMs > 1000 {
			reason = "latency_exceeded"
		}

		lastIncident = IncidentState{
			Active:     true,
			ErrorRate:  errorRate,
			LatencyMs:  latencyMs,
			DetectedAt: time.Now().UTC(),
			Reason:     reason,
		}

		slog.Error("controlplane: INCIDENT DETECTED",
			"error_rate", errorRate,
			"latency_ms", latencyMs,
			"reason", reason,
		)

		// Auto-remediate: signal rollback
		remediation.SignalRollback(reason)

		return lastIncident
	}

	if lastIncident.Active {
		slog.Info("controlplane: incident resolved", "previous_reason", lastIncident.Reason)
	}

	lastIncident = IncidentState{
		Active:    false,
		ErrorRate: errorRate,
		LatencyMs: latencyMs,
	}

	return lastIncident
}

// CurrentIncident returns the last known incident state.
func CurrentIncident() IncidentState {
	return lastIncident
}
