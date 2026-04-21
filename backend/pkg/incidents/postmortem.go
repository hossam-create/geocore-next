package incidents

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Postmortem is an automatically generated root cause analysis document.
type Postmortem struct {
	IncidentID    string                 `json:"incident_id"`
	Title         string                 `json:"title"`
	Severity      Severity               `json:"severity"`
	DetectedAt    time.Time              `json:"detected_at"`
	ResolvedAt    time.Time              `json:"resolved_at,omitempty"`
	Duration      string                 `json:"duration,omitempty"`
	RootCause     string                 `json:"root_cause"`
	Timeline      []TimelineEntry        `json:"timeline"`
	AffectedServices []string            `json:"affected_services"`
	MetricsSnapshot map[string]float64   `json:"metrics_snapshot"`
	ActionsTaken  []string               `json:"actions_taken"`
	Prevention    []string               `json:"prevention_recommendations"`
}

// TimelineEntry represents a single event in the incident timeline.
type TimelineEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string   `json:"event"`
	Source    string   `json:"source,omitempty"`
}

// GeneratePostmortem creates an auto-RCA from an incident record.
func GeneratePostmortem(inc Incident) Postmortem {
	pm := Postmortem{
		IncidentID:    inc.ID,
		Title:         fmt.Sprintf("Auto-RCA: %s", inc.Title),
		Severity:      inc.Severity,
		DetectedAt:    inc.DetectedAt,
		RootCause:     inferRootCause(inc),
		AffectedServices: inferAffectedServices(inc),
		ActionsTaken:  inc.Actions,
		MetricsSnapshot: snapshotMetrics(),
		Prevention:    inferPrevention(inc),
	}

	if !inc.Active {
		pm.ResolvedAt = time.Now()
		pm.Duration = pm.ResolvedAt.Sub(pm.DetectedAt).Round(time.Second).String()
	}

	// Build timeline from incident source
	pm.Timeline = buildTimeline(inc)

	return pm
}

// SavePostmortem writes the postmortem to disk for review.
func SavePostmortem(pm Postmortem) error {
	dir := filepath.Join("postmortems")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	filename := filepath.Join(dir, fmt.Sprintf("%s-%s.json",
		pm.IncidentID, pm.DetectedAt.Format("2006-01-02-150405")))
	data, err := json.MarshalIndent(pm, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return err
	}
	slog.Info("incidents: postmortem saved", "file", filename)
	return nil
}

// inferRootCause attempts to determine the root cause from the incident source.
func inferRootCause(inc Incident) string {
	causes := map[string]string{
		"latency":       "Database query saturation — likely missing index or lock contention under high concurrency",
		"kafka_lag":     "Consumer processing bottleneck — handler latency exceeds produce rate, causing backlog",
		"db_pressure":   "Connection pool exhaustion — too many concurrent transactions or slow queries holding connections",
		"redis_eviction":"Redis maxmemory reached — cache hit rate degraded, keys being evicted under memory pressure",
		"error_rate":    "Upstream service degradation — increased 5xx responses from dependent services or DB timeouts",
	}
	if cause, ok := causes[inc.Source]; ok {
		return cause
	}
	return fmt.Sprintf("Unknown root cause for source: %s — manual investigation required", inc.Source)
}

// inferAffectedServices determines which services are impacted.
func inferAffectedServices(inc Incident) []string {
	serviceMap := map[string][]string{
		"latency":        {"api-service", "wallet-service"},
		"kafka_lag":      {"kafka-consumers", "notification-service", "wallet-service"},
		"db_pressure":    {"api-service", "worker", "fraud-engine"},
		"redis_eviction": {"api-service", "cache-layer"},
		"error_rate":     {"api-service", "payment-gateway"},
	}
	if services, ok := serviceMap[inc.Source]; ok {
		return services
	}
	return []string{"unknown"}
}

// inferPrevention recommends actions to prevent recurrence.
func inferPrevention(inc Incident) []string {
	preventionMap := map[string][]string{
		"latency": {
			"Add missing composite indexes for hot queries",
			"Implement adaptive concurrency limiting",
			"Add query timeout at application level",
		},
		"kafka_lag": {
			"Scale consumer group replicas proactively",
			"Optimize handler processing time",
			"Implement backpressure on producer side",
		},
		"db_pressure": {
			"Increase max_connections with monitoring",
			"Add statement_timeout and lock_timeout",
			"Implement connection pool metrics alerting",
		},
		"redis_eviction": {
			"Increase Redis maxmemory or upgrade instance",
			"Review cache TTL policies for stale keys",
			"Implement cache warming for critical keys",
		},
		"error_rate": {
			"Add circuit breakers for upstream dependencies",
			"Implement retry with exponential backoff",
			"Add health check probes for all dependencies",
		},
	}
	if p, ok := preventionMap[inc.Source]; ok {
		return p
	}
	return []string{"Manual investigation required"}
}

// buildTimeline creates a timeline from the incident data.
func buildTimeline(inc Incident) []TimelineEntry {
	entries := []TimelineEntry{
		{
			Timestamp: inc.DetectedAt,
			Event:     fmt.Sprintf("Incident detected: %s", inc.Message),
			Source:    "detector",
		},
	}
	for i, action := range inc.Actions {
		entries = append(entries, TimelineEntry{
			Timestamp: inc.DetectedAt.Add(time.Duration(i+1) * time.Second),
			Event:     fmt.Sprintf("Auto-mitigation: %s", action),
			Source:    "remediation",
		})
	}
	if !inc.Active {
		entries = append(entries, TimelineEntry{
			Timestamp: time.Now(),
			Event:     "Incident resolved",
			Source:    "detector",
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.Before(entries[j].Timestamp)
	})
	return entries
}

// snapshotMetrics captures current system metrics for the postmortem.
func snapshotMetrics() map[string]float64 {
	// In production, this would query Prometheus for the current values.
	// For now, return a structured placeholder that indicates what metrics
	// should be captured.
	return map[string]float64{
		"note": 0, // placeholder — production would fill from Prometheus API
	}
}

// ListPostmortems returns all saved postmortem files.
func ListPostmortems() ([]string, error) {
	dir := "postmortems"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	return files, nil
}
