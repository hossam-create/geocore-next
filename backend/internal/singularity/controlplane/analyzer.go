package controlplane

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/autonomy"
)

// Inefficiency represents a detected system inefficiency.
type Inefficiency struct {
	Type        string  `json:"type"`         // latency, overprovisioned, hot_partition, kafka_lag, cost_waste
	Target      string  `json:"target"`       // service/endpoint/partition name
	Severity    float64 `json:"severity"`     // 0-1 (how bad)
	Metric      string  `json:"metric"`       // the metric that triggered this
	Current     float64 `json:"current"`      // current value
	Threshold   float64 `json:"threshold"`    // acceptable threshold
	Description string  `json:"description"`  // human-readable
}

// Analyzer detects inefficiencies from the current metrics snapshot.
type Analyzer struct{}

// NewAnalyzer creates a new system analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Analyze examines the current metrics and returns detected inefficiencies.
func (a *Analyzer) Analyze(ctx context.Context, snap autonomy.MetricsSnapshot) []Inefficiency {
	var inefficiencies []Inefficiency

	// High latency detection
	if snap.P95Latency > 500 {
		inefficiencies = append(inefficiencies, Inefficiency{
			Type:        "latency",
			Target:      "api",
			Severity:    min((snap.P95Latency-500)/500, 1.0),
			Metric:      "p95_latency_ms",
			Current:     snap.P95Latency,
			Threshold:   500,
			Description: "P95 latency exceeds 500ms threshold",
		})
	}

	// Overprovisioned detection (low RPS relative to capacity)
	if snap.RPS < 50 && snap.P95Latency < 100 && snap.ErrorRate < 0.5 {
		inefficiencies = append(inefficiencies, Inefficiency{
			Type:        "overprovisioned",
			Target:      "api",
			Severity:    0.3,
			Metric:      "rps",
			Current:     snap.RPS,
			Threshold:   50,
			Description: "Low RPS with healthy latency — may be overprovisioned",
		})
	}

	// Kafka lag detection
	if snap.KafkaLag > 2000 {
		inefficiencies = append(inefficiencies, Inefficiency{
			Type:        "kafka_lag",
			Target:      "consumers",
			Severity:    min(float64(snap.KafkaLag)/10000, 1.0),
			Metric:      "kafka_consumer_lag",
			Current:     float64(snap.KafkaLag),
			Threshold:   2000,
			Description: "Kafka consumer lag exceeds threshold",
		})
	}

	// Error rate detection
	if snap.ErrorRate > 1.0 {
		inefficiencies = append(inefficiencies, Inefficiency{
			Type:        "error_rate",
			Target:      "api",
			Severity:    min(snap.ErrorRate/5.0, 1.0),
			Metric:      "error_rate_percent",
			Current:     snap.ErrorRate,
			Threshold:   1.0,
			Description: "Error rate exceeds 1% threshold",
		})
	}

	if len(inefficiencies) > 0 {
		slog.Info("singularity: inefficiencies detected", "count", len(inefficiencies))
		for _, in := range inefficiencies {
			slog.Debug("singularity: inefficiency",
				"type", in.Type, "target", in.Target, "severity", in.Severity)
		}
	}

	return inefficiencies
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
