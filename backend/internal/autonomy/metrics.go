package autonomy

import (
	"log/slog"
	"sync"
	"time"
)

// MetricsSnapshot is a point-in-time view of system health metrics.
type MetricsSnapshot struct {
	ErrorRate  float64 `json:"error_rate"`
	P95Latency float64 `json:"p95_latency_ms"`
	RPS        float64 `json:"rps"`
	KafkaLag   int     `json:"kafka_lag"`
	Timestamp  time.Time `json:"timestamp"`
}

// MetricsCollector aggregates system metrics for the decision engine.
type MetricsCollector struct {
	mu       sync.RWMutex
	snapshot MetricsSnapshot
}

var globalCollector = &MetricsCollector{}

// CollectMetrics returns the current metrics snapshot.
func CollectMetrics() MetricsSnapshot {
	return globalCollector.Get()
}

// Get returns the current snapshot.
func (c *MetricsCollector) Get() MetricsSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.snapshot
}

// Update sets the current metrics snapshot.
func (c *MetricsCollector) Update(snap MetricsSnapshot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	snap.Timestamp = time.Now().UTC()
	c.snapshot = snap
}

// RecordErrorRate updates the error rate metric.
func RecordErrorRate(rate float64) {
	globalCollector.mu.Lock()
	defer globalCollector.mu.Unlock()
	globalCollector.snapshot.ErrorRate = rate
	globalCollector.snapshot.Timestamp = time.Now().UTC()
}

// RecordP95 updates the p95 latency metric.
func RecordP95(ms float64) {
	globalCollector.mu.Lock()
	defer globalCollector.mu.Unlock()
	globalCollector.snapshot.P95Latency = ms
	globalCollector.snapshot.Timestamp = time.Now().UTC()
}

// RecordRPS updates the requests per second metric.
func RecordRPS(rps float64) {
	globalCollector.mu.Lock()
	defer globalCollector.mu.Unlock()
	globalCollector.snapshot.RPS = rps
	globalCollector.snapshot.Timestamp = time.Now().UTC()
}

// RecordKafkaLag updates the Kafka consumer lag metric.
func RecordKafkaLag(lag int) {
	globalCollector.mu.Lock()
	defer globalCollector.mu.Unlock()
	globalCollector.snapshot.KafkaLag = lag
	globalCollector.snapshot.Timestamp = time.Now().UTC()
}

// LogSnapshot logs the current metrics for observability.
func LogSnapshot() {
	snap := CollectMetrics()
	slog.Info("autonomy: metrics snapshot",
		"error_rate", snap.ErrorRate,
		"p95_ms", snap.P95Latency,
		"rps", snap.RPS,
		"kafka_lag", snap.KafkaLag,
	)
}
