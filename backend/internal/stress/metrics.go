package stress

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// requestSample records the outcome of one HTTP request fired during a stress test.
type requestSample struct {
	duration   time.Duration
	statusCode int
	endpoint   string
	isError    bool
}

// MetricsCollector is a thread-safe recorder of request samples.
type MetricsCollector struct {
	mu        sync.Mutex
	samples   []requestSample
	total     atomic.Int64
	errors    atomic.Int64
	startTime time.Time
}

// MetricsSummary is the computed summary of all recorded samples.
type MetricsSummary struct {
	TotalRequests int64   `json:"total_requests"`
	Errors        int64   `json:"errors"`
	ErrorRatePct  float64 `json:"error_rate_pct"`
	RPS           float64 `json:"rps"`
	P50LatencyMs  float64 `json:"p50_latency_ms"`
	P95LatencyMs  float64 `json:"p95_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
	MaxLatencyMs  float64 `json:"max_latency_ms"`
}

func newMetricsCollector() *MetricsCollector {
	return &MetricsCollector{startTime: time.Now()}
}

func (m *MetricsCollector) record(s requestSample) {
	m.total.Add(1)
	if s.isError {
		m.errors.Add(1)
	}
	m.mu.Lock()
	m.samples = append(m.samples, s)
	m.mu.Unlock()
}

// Summary computes percentiles and rates from all recorded samples.
func (m *MetricsCollector) Summary() MetricsSummary {
	m.mu.Lock()
	snap := make([]requestSample, len(m.samples))
	copy(snap, m.samples)
	m.mu.Unlock()

	total := m.total.Load()
	errors := m.errors.Load()
	elapsed := time.Since(m.startTime).Seconds()

	rps := 0.0
	if elapsed > 0 {
		rps = float64(total) / elapsed
	}

	errorRate := 0.0
	if total > 0 {
		errorRate = float64(errors) / float64(total) * 100
	}

	durations := make([]float64, len(snap))
	for i, s := range snap {
		durations[i] = float64(s.duration.Milliseconds())
	}
	sort.Float64s(durations)

	return MetricsSummary{
		TotalRequests: total,
		Errors:        errors,
		ErrorRatePct:  errorRate,
		RPS:           rps,
		P50LatencyMs:  pct(durations, 50),
		P95LatencyMs:  pct(durations, 95),
		P99LatencyMs:  pct(durations, 99),
		MaxLatencyMs:  pct(durations, 100),
	}
}

func pct(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := int(float64(len(sorted)) * float64(p) / 100.0)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
