package stress

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/aiops"
)

// ValidationResult holds the outcome of post-test system health checks.
type ValidationResult struct {
	P95LatencyOK   bool     `json:"p95_latency_ok"`
	ErrorRateOK    bool     `json:"error_rate_ok"`
	P95LatencyMs   float64  `json:"p95_latency_ms"`
	ErrorRatePct   float64  `json:"error_rate_pct"`
	HealthStatus   string   `json:"health_status"`
	AIOpsTriggered bool     `json:"aiops_triggered"`
	OpenIncidents  int      `json:"open_incidents"`
	Passed         bool     `json:"passed"`
	FailureReasons []string `json:"failure_reasons,omitempty"`
}

// Validator evaluates system health after a stress run.
type Validator struct {
	targetURL   string
	client      *http.Client
	maxP95Ms    float64 // SLO threshold — p95 must be ≤ this
	maxErrorPct float64 // SLO threshold — error rate must be ≤ this
}

func newValidator() *Validator {
	target := os.Getenv("STRESS_TARGET_URL")
	if target == "" {
		target = "http://localhost:8080"
	}
	return &Validator{
		targetURL:   target,
		client:      &http.Client{Timeout: 5 * time.Second},
		maxP95Ms:    800.0,
		maxErrorPct: 2.0,
	}
}

// Evaluate checks metrics against SLO thresholds and queries AIOps + health endpoint.
func (v *Validator) Evaluate(ctx context.Context, m MetricsSummary) ValidationResult {
	r := ValidationResult{
		P95LatencyMs: m.P95LatencyMs,
		ErrorRatePct: m.ErrorRatePct,
		Passed:       true,
	}

	if m.P95LatencyMs <= v.maxP95Ms {
		r.P95LatencyOK = true
	} else {
		r.Passed = false
		r.FailureReasons = append(r.FailureReasons,
			fmt.Sprintf("p95 latency %.0fms > SLO threshold %.0fms", m.P95LatencyMs, v.maxP95Ms))
	}

	if m.ErrorRatePct <= v.maxErrorPct {
		r.ErrorRateOK = true
	} else {
		r.Passed = false
		r.FailureReasons = append(r.FailureReasons,
			fmt.Sprintf("error rate %.2f%% > SLO threshold %.2f%%", m.ErrorRatePct, v.maxErrorPct))
	}

	r.HealthStatus = v.checkHealth(ctx)
	openCount := aiops.GetOpenCount()
	r.AIOpsTriggered = openCount > 0
	r.OpenIncidents = openCount

	return r
}

func (v *Validator) checkHealth(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.targetURL+"/health", nil)
	if err != nil {
		return "unknown"
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return "unreachable"
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return "healthy"
	case 206:
		return "degraded"
	default:
		return fmt.Sprintf("status_%d", resp.StatusCode)
	}
}
