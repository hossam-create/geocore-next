package reslab

import "github.com/geocore-next/backend/internal/stress"

// Suggestion is a specific, actionable improvement recommendation.
type Suggestion struct {
	Area     string `json:"area"`
	Action   string `json:"action"`
	Impact   string `json:"impact"`
	Priority string `json:"priority"` // high | medium | low
}

// GenerateSuggestions produces a ranked list of improvement recommendations
// based on experiment metrics, validation results, and the composite score.
func GenerateSuggestions(m stress.MetricsSummary, v stress.ValidationResult, s ExperimentScore) []Suggestion {
	var out []Suggestion

	// ── Latency ──────────────────────────────────────────────────────────────

	if m.P95LatencyMs > 800 {
		out = append(out, Suggestion{
			Area:     "Scaling",
			Action:   "Tighten HPA scale-up: stabilizationWindowSeconds=30, scaleUp.percent=100 in k8s/hpa.yaml",
			Impact:   "Cuts p95 latency spikes during sudden load surges by ~40%",
			Priority: "high",
		})
	} else if m.P95LatencyMs > 500 {
		out = append(out, Suggestion{
			Area:     "Caching",
			Action:   "Add 30s Redis cache on GET /api/v1/listings and GET /api/v1/search — both are read-heavy and idempotent",
			Impact:   "Reduces DB read pressure by ~60%, expected p95 drop to <400ms",
			Priority: "medium",
		})
	}

	// ── Error Rate ───────────────────────────────────────────────────────────

	if m.ErrorRatePct > 2.0 {
		out = append(out, Suggestion{
			Area:     "Resilience",
			Action:   "Implement circuit breaker on DB connection pool: reject at 90% saturation with 503 + Retry-After header",
			Impact:   "Prevents cascading 500 storms under DB overload — converts crash to graceful degradation",
			Priority: "high",
		})
	} else if m.ErrorRatePct > 0.5 {
		out = append(out, Suggestion{
			Area:     "Reliability",
			Action:   "Add retry with exponential backoff (max 2 retries, 100ms/200ms) on transient DB errors",
			Impact:   "Converts ~80% of transient errors to successes without user impact",
			Priority: "medium",
		})
	}

	// ── AIOps / Observability ────────────────────────────────────────────────

	if v.OpenIncidents > 2 {
		out = append(out, Suggestion{
			Area:     "Messaging",
			Action:   "Scale Kafka consumers: kubectl scale deployment wallet-service --replicas=3 + increase partitions to 12",
			Impact:   "Reduces consumer lag and DLQ growth under event burst",
			Priority: "high",
		})
	}

	if !v.AIOpsTriggered && v.OpenIncidents == 0 {
		out = append(out, Suggestion{
			Area:     "Observability",
			Action:   "Set PROMETHEUS_URL env var and OPENAI_API_KEY to enable full AIOps auto-detection pipeline",
			Impact:   "Enables automatic incident detection, RCA, and runbook generation",
			Priority: "medium",
		})
	}

	// ── Architecture ─────────────────────────────────────────────────────────

	if s.Overall < 70 {
		out = append(out, Suggestion{
			Area:     "Architecture",
			Action:   "Enable read replica routing in pkg/db/split.go: direct all GET handlers to dbRead pool",
			Impact:   "Halves primary DB connection pressure, improves overall score by ~15 points",
			Priority: "high",
		})
	}

	// ── Cost ─────────────────────────────────────────────────────────────────

	if s.Cost < 50 {
		out = append(out, Suggestion{
			Area:     "Cost",
			Action:   "Enable request coalescing: deduplicate identical concurrent queries via pkg/idempotency or singleflight",
			Impact:   "Reduces redundant DB calls by up to 70% under burst — direct cloud bill savings",
			Priority: "low",
		})
	}

	// ── Baseline (no issues found) ───────────────────────────────────────────

	if len(out) == 0 {
		out = append(out, Suggestion{
			Area:     "Next Experiment",
			Action:   "System is within SLO under this load — run mixed_failure_scenario to find the true breaking point",
			Impact:   "Discovers failure modes before real users do",
			Priority: "low",
		})
	}

	return out
}
