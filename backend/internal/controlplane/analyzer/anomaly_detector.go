package analyzer

import (
	"context"
	"log/slog"
)

// Anomaly represents a detected system anomaly.
type Anomaly struct {
	Type        string  `json:"type"` // high_latency, high_error, overprovisioned, kafka_lag, cost_spike
	Target      string  `json:"target"`
	Severity    float64 `json:"severity"` // 0-1
	Value       float64 `json:"value"`
	Threshold   float64 `json:"threshold"`
	Description string  `json:"description"`
}

// Analyzer collects metrics and detects anomalies.
type Analyzer struct {
	Reader *MetricsReader
}

// NewAnalyzer creates an analyzer with a metrics reader.
func NewAnalyzer(reader *MetricsReader) *Analyzer {
	return &Analyzer{Reader: reader}
}

// Collect reads the current metrics snapshot.
func (a *Analyzer) Collect(ctx context.Context) Metrics {
	if a.Reader == nil {
		return Metrics{}
	}
	return a.Reader.Read(ctx)
}

// DetectAnomalies examines metrics and returns detected anomalies.
func (a *Analyzer) DetectAnomalies(m Metrics) []Anomaly {
	var anomalies []Anomaly

	// High latency
	if m.LatencyP95 > 500 {
		anomalies = append(anomalies, Anomaly{
			Type: "high_latency", Target: "api",
			Severity: minF((m.LatencyP95-500)/500, 1.0),
			Value: m.LatencyP95, Threshold: 500,
			Description: "P95 latency exceeds 500ms",
		})
	}

	// High error rate
	if m.ErrorRate > 1.0 {
		anomalies = append(anomalies, Anomaly{
			Type: "high_error", Target: "api",
			Severity: minF(m.ErrorRate/5.0, 1.0),
			Value: m.ErrorRate, Threshold: 1.0,
			Description: "Error rate exceeds 1%",
		})
	}

	// Overprovisioned (low CPU + low RPS)
	if m.CPUUsage < 30 && m.RPS < 50 {
		anomalies = append(anomalies, Anomaly{
			Type: "overprovisioned", Target: "api",
			Severity: 0.3,
			Value: m.CPUUsage, Threshold: 30,
			Description: "Low CPU usage with low RPS — may be overprovisioned",
		})
	}

	// Kafka lag
	if m.KafkaLag > 2000 {
		anomalies = append(anomalies, Anomaly{
			Type: "kafka_lag", Target: "consumers",
			Severity: minF(float64(m.KafkaLag)/10000, 1.0),
			Value: float64(m.KafkaLag), Threshold: 2000,
			Description: "Kafka consumer lag exceeds threshold",
		})
	}

	// Cost spike
	if m.CostPerHour > 60 {
		anomalies = append(anomalies, Anomaly{
			Type: "cost_spike", Target: "infrastructure",
			Severity: minF((m.CostPerHour-60)/40, 1.0),
			Value: m.CostPerHour, Threshold: 60,
			Description: "Hourly cost exceeds budget threshold",
		})
	}

	if len(anomalies) > 0 {
		slog.Info("controlplane: anomalies detected", "count", len(anomalies))
	}

	return anomalies
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
