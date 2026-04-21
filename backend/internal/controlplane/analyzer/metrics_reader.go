package analyzer

import (
	"context"
	"sync"
	"time"
)

// Metrics is a point-in-time snapshot of system health.
type Metrics struct {
	LatencyP95  float64   `json:"latency_p95_ms"`
	ErrorRate   float64   `json:"error_rate_percent"`
	CPUUsage    float64   `json:"cpu_usage_percent"`
	KafkaLag    int64     `json:"kafka_lag"`
	CostPerHour float64   `json:"cost_per_hour"`
	RPS         float64   `json:"rps"`
	PodCount    int       `json:"pod_count"`
	Timestamp   time.Time `json:"timestamp"`
}

// MetricsReader collects system metrics from telemetry sources.
type MetricsReader struct {
	mu     sync.RWMutex
	last   Metrics
}

// NewMetricsReader creates a metrics reader.
func NewMetricsReader() *MetricsReader {
	return &MetricsReader{}
}

// Read returns the current metrics snapshot.
func (r *MetricsReader) Read(ctx context.Context) Metrics {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.last
}

// Update sets the current metrics snapshot.
func (r *MetricsReader) Update(m Metrics) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m.Timestamp = time.Now().UTC()
	r.last = m
}
