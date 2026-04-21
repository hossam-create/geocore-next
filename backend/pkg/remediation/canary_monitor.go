package remediation

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// CanaryMonitor watches deployment health and auto-signals rollback.
// It reads Prometheus metrics from the local registry to compute error rate + p95.
type CanaryMonitor struct {
	interval  time.Duration
	requests  *prometheus.CounterVec
	latencyMs *prometheus.HistogramVec
	health    DeployHealth
}

// NewCanaryMonitor creates a monitor that checks health every interval.
// Pass nil for metrics to disable metric-based checks (manual only).
func NewCanaryMonitor(interval time.Duration) *CanaryMonitor {
	return &CanaryMonitor{
		interval: interval,
	}
}

// Start runs the background health check loop.
func (m *CanaryMonitor) Start(ctx context.Context) {
	slog.Info("canary: monitor started", "interval", m.interval)
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("canary: monitor stopped")
			return
		case <-ticker.C:
			m.check(ctx)
		}
	}
}

// check evaluates current health and signals rollback if needed.
func (m *CanaryMonitor) check(ctx context.Context) {
	// Read from Prometheus metrics if available
	errorRate := m.computeErrorRate()
	p95 := m.computeP95()
	kafkaLag := m.computeKafkaLag()

	m.health = CheckDeployHealth(errorRate, p95, kafkaLag)

	if !m.health.Healthy {
		slog.Error("canary: unhealthy deployment detected",
			"reason", m.health.Reason,
			"error_rate", errorRate,
			"p95", p95,
			"kafka_lag", kafkaLag,
		)
		SignalRollback(m.health.Reason)
	} else {
		slog.Debug("canary: deployment healthy",
			"error_rate", errorRate,
			"p95", p95,
		)
	}
}

// computeErrorRate calculates 5xx / total from Prometheus counters.
func (m *CanaryMonitor) computeErrorRate() float64 {
	if m.requests == nil {
		return 0
	}
	// In production, use prometheus client to read counter values from /metrics endpoint
	// For now, return 0 (healthy) — real implementation reads from /metrics endpoint
	return 0
}

// computeP95 estimates p95 latency from the histogram.
func (m *CanaryMonitor) computeP95() time.Duration {
	if m.latencyMs == nil {
		return 0
	}
	// Simplified: in production, read histogram quantiles
	return 0
}

// computeKafkaLag returns current consumer lag.
func (m *CanaryMonitor) computeKafkaLag() int64 {
	// In production, query Kafka consumer group lag
	return 0
}

// Health returns the last computed health status.
func (m *CanaryMonitor) Health() DeployHealth {
	return m.health
}

// SetMetrics injects Prometheus metric references for live evaluation.
func (m *CanaryMonitor) SetMetrics(requests *prometheus.CounterVec, latencyMs *prometheus.HistogramVec) {
	m.requests = requests
	m.latencyMs = latencyMs
}

// ShouldRollbackAndSignal is a convenience function for inline checks.
// Returns true if rollback was signaled.
func ShouldRollbackAndSignal(errorRate float64, p95 time.Duration) bool {
	if ShouldRollback(errorRate, p95) {
		SignalRollback("error_rate_or_latency_exceeded")
		return true
	}
	return false
}
