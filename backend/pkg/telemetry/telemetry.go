package telemetry

import (
	"log/slog"
	"sync"
	"time"
)

// MetricsSnapshot holds a point-in-time telemetry reading.
type MetricsSnapshot struct {
	ErrorRate    float64   `json:"error_rate_percent"`
	P95Latency   float64   `json:"p95_latency_ms"`
	CPUUsage     float64   `json:"cpu_usage_percent"`
	KafkaLag     int64     `json:"kafka_lag"`
	RPS          float64   `json:"rps"`
	PodRestarts  int       `json:"pod_restarts"`
	Timestamp    time.Time `json:"timestamp"`
}

// Collector gathers telemetry from observability sources.
type Collector struct {
	mu     sync.RWMutex
	last   MetricsSnapshot
}

// NewCollector creates a telemetry collector.
func NewCollector() *Collector {
	return &Collector{}
}

// Collect returns the latest metrics snapshot.
func (c *Collector) Collect() MetricsSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.last
}

// Update sets the current metrics snapshot.
func (c *Collector) Update(m MetricsSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	m.Timestamp = time.Now().UTC()
	c.last = m
	slog.Debug("telemetry: metrics updated",
		"error_rate", m.ErrorRate,
		"p95", m.P95Latency,
		"cpu", m.CPUUsage,
		"rps", m.RPS,
	)
}

// IsHealthy returns true if all key metrics are within normal ranges.
func (c *Collector) IsHealthy() bool {
	m := c.Collect()
	return m.ErrorRate < 2.0 && m.P95Latency < 500 && m.CPUUsage < 90
}
