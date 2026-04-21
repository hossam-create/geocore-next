package aiops

import (
	"context"
	"fmt"
)

// SystemContext holds a snapshot of system state for LLM analysis.
type SystemContext struct {
	Incident   *Incident
	Metrics    map[string]float64
	LogLines   []string
}

// ContextBuilder assembles system state for RCA and runbook generation.
type ContextBuilder struct {
	detector *Detector
}

func NewContextBuilder(d *Detector) *ContextBuilder {
	return &ContextBuilder{detector: d}
}

// contextQueries are the supporting metrics fetched alongside each incident.
var contextQueries = map[string]string{
	"error_rate_pct":     `sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100`,
	"p95_latency_ms":     `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) * 1000`,
	"rps":                `sum(rate(http_requests_total[5m]))`,
	"db_connections_pct": `db_connections_in_use / db_connections_open * 100`,
	"kafka_lag":          `max(kafka_consumer_group_lag)`,
	"redis_hit_ratio_pct": `redis_keyspace_hits_total / (redis_keyspace_hits_total + redis_keyspace_misses_total) * 100`,
	"goroutines":         `go_goroutines`,
	"heap_mb":            `go_memstats_heap_inuse_bytes / 1048576`,
}

// Build collects relevant system metrics for the given incident.
func (b *ContextBuilder) Build(ctx context.Context, inc *Incident) SystemContext {
	metrics := make(map[string]float64, len(contextQueries))
	for name, query := range contextQueries {
		if val, err := b.detector.queryMetric(ctx, query); err == nil {
			metrics[name] = val
		}
	}

	logLines := []string{
		fmt.Sprintf("[INCIDENT] %s — severity=%s service=%s", inc.Title, inc.Severity, inc.Service),
		fmt.Sprintf("[METRIC]   %s = %.4f  (baseline=%.4f)", inc.Metric, inc.Value, inc.Baseline),
	}

	return SystemContext{
		Incident: inc,
		Metrics:  metrics,
		LogLines: logLines,
	}
}
