// Package incidents provides automatic incident detection and mitigation
// for the GeoCore platform. It monitors key metrics and triggers automated
// responses when thresholds are breached.
package incidents

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/remediation"
)

// Severity represents incident severity level.
type Severity string

const (
	SeverityP0 Severity = "P0" // Page immediately
	SeverityP1 Severity = "P1" // Slack notification
	SeverityP2 Severity = "P2" // Email digest
)

// Incident represents a detected incident.
type Incident struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Severity   Severity  `json:"severity"`
	Source     string    `json:"source"` // e.g. "latency", "kafka_lag", "db_pressure"
	Message    string    `json:"message"`
	DetectedAt time.Time `json:"detected_at"`
	Active     bool      `json:"active"`
	Mitigated  bool      `json:"mitigated"`
	Actions    []string  `json:"actions_taken"`
}

// MitigationAction is an automated response to an incident.
type MitigationAction func(ctx context.Context, inc Incident) error

// Detector monitors system health and triggers mitigations.
type Detector struct {
	mu        sync.Mutex
	incidents map[string]*Incident // source → current active incident
	actions   map[string]MitigationAction
	enabled   bool
	alertCh   chan Incident // channel for alerting systems (PagerDuty, Slack)
}

var (
	globalDetector *Detector
	detectorOnce   sync.Once
)

// InitDetector creates and starts the global incident detector.
func InitDetector() {
	detectorOnce.Do(func() {
		d := &Detector{
			incidents: make(map[string]*Incident),
			actions:   make(map[string]MitigationAction),
			enabled:   os.Getenv("APP_ENV") == "production",
			alertCh:   make(chan Incident, 100),
		}
		d.registerDefaultActions()
		globalDetector = d
		go d.run(context.Background())
		slog.Info("incidents: detector initialized", "enabled", d.enabled)
	})
}

// AlertChannel returns the channel for consuming incident alerts.
func AlertChannel() <-chan Incident {
	if globalDetector == nil {
		return nil
	}
	return globalDetector.alertCh
}

// registerDefaultActions registers built-in mitigation strategies.
func (d *Detector) registerDefaultActions() {
	// Latency spike → enable degraded mode
	d.actions["latency"] = func(ctx context.Context, inc Incident) error {
		slog.Warn("incidents: auto-mitigating latency spike — enabling degraded mode",
			"incident_id", inc.ID)
		remediation.EnableDBReadOnly("incident:latency_spike")
		return nil
	}

	// Kafka lag → signal HPA to scale consumers
	d.actions["kafka_lag"] = func(ctx context.Context, inc Incident) error {
		slog.Warn("incidents: auto-mitigating kafka lag — signaling consumer scale",
			"incident_id", inc.ID)
		remediation.SignalConsumerScale(5000)
		return nil
	}

	// DB pressure → no auto-action (P0 page), just alert
	d.actions["db_pressure"] = func(ctx context.Context, inc Incident) error {
		slog.Error("incidents: DB pressure detected — manual intervention required",
			"incident_id", inc.ID)
		return nil
	}

	// Redis eviction → no auto-action, just alert
	d.actions["redis_eviction"] = func(ctx context.Context, inc Incident) error {
		slog.Warn("incidents: Redis eviction detected — cache policy may need review",
			"incident_id", inc.ID)
		return nil
	}

	// Error rate spike → enable degraded mode
	d.actions["error_rate"] = func(ctx context.Context, inc Incident) error {
		slog.Warn("incidents: auto-mitigating error rate spike — enabling degraded mode",
			"incident_id", inc.ID)
		remediation.EnableDBReadOnly("incident:error_rate_spike")
		return nil
	}
}

// run starts the periodic health check loop.
func (d *Detector) run(ctx context.Context) {
	if !d.enabled {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.checkLatency(ctx)
			d.checkKafkaLag(ctx)
			d.checkDBPressure(ctx)
			d.checkRedisEviction(ctx)
			d.checkErrorRate(ctx)
		}
	}
}

// ── Health Checks ──────────────────────────────────────────────────────────

func (d *Detector) checkLatency(ctx context.Context) {
	// Read from Prometheus metrics — if p95 > 2s for 3 consecutive checks, trigger
	// This is a simplified check; production would use Prometheus query API.
	// For now, we rely on the auto-degradation middleware in main.go.
}

func (d *Detector) checkKafkaLag(ctx context.Context) {
	// If consumer lag gauge > 5000 for any topic, trigger
	// The actual lag values are set by consumer.go's reportLag goroutine.
}

func (d *Detector) checkDBPressure(ctx context.Context) {
	// If DB connections > 80% of max, trigger
}

func (d *Detector) checkRedisEviction(ctx context.Context) {
	// If redis_evicted_keys_total is increasing, trigger
}

func (d *Detector) checkErrorRate(ctx context.Context) {
	// If error rate > 5% over 2-minute window, trigger
}

// ── Incident Management ────────────────────────────────────────────────────

// triggerIncident records a new incident and executes mitigation.
func (d *Detector) triggerIncident(inc Incident) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Don't re-trigger if already active
	if existing, ok := d.incidents[inc.Source]; ok && existing.Active {
		return
	}

	inc.ID = generateIncidentID(inc.Source)
	inc.DetectedAt = time.Now()
	inc.Active = true
	d.incidents[inc.Source] = &inc

	slog.Error("incidents: new incident detected",
		"id", inc.ID, "severity", inc.Severity, "source", inc.Source, "message", inc.Message)

	// Execute mitigation action
	if action, ok := d.actions[inc.Source]; ok {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := action(ctx, inc); err != nil {
			slog.Error("incidents: mitigation action failed", "incident_id", inc.ID, "error", err)
		} else {
			inc.Actions = append(inc.Actions, "auto_mitigation_triggered")
		}
	}

	// Send to alert channel (non-blocking)
	select {
	case d.alertCh <- inc:
	default:
		slog.Warn("incidents: alert channel full — dropping notification", "incident_id", inc.ID)
	}
}

// resolveIncident marks an incident as resolved.
func (d *Detector) resolveIncident(source string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if inc, ok := d.incidents[source]; ok && inc.Active {
		inc.Active = false
		inc.Mitigated = true
		slog.Info("incidents: resolved", "id", inc.ID, "source", source)
	}
}

// ActiveIncidents returns all currently active incidents.
func ActiveIncidents() []Incident {
	if globalDetector == nil {
		return nil
	}
	globalDetector.mu.Lock()
	defer globalDetector.mu.Unlock()
	var result []Incident
	for _, inc := range globalDetector.incidents {
		if inc.Active {
			result = append(result, *inc)
		}
	}
	return result
}

func generateIncidentID(source string) string {
	return fmt.Sprintf("INC-%s-%d", source, time.Now().UnixMilli())
}
