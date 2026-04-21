package aiops

import (
	"context"
	"fmt"
)

// RCAEngine runs root cause analysis using LLM + system context.
type RCAEngine struct {
	llm *LLMClient
}

func NewRCAEngine(llm *LLMClient) *RCAEngine {
	return &RCAEngine{llm: llm}
}

const rcaSystemPrompt = `You are a Senior Site Reliability Engineer (SRE) with deep expertise in distributed systems, PostgreSQL, Kafka, Redis, and Kubernetes.

Analyze the incident and system metrics to determine the ROOT CAUSE.

Rules:
- Be concise and precise (max 150 words total)
- State confidence as a percentage
- List specific evidence from the provided metrics
- Identify blast radius (what else is affected)

Respond in exactly this format:
Root Cause: [one sentence]
Confidence: [X%]
Evidence:
- [metric evidence item]
- [metric evidence item]
Blast Radius: [what services/users are impacted]`

// Analyze generates a root cause analysis for the given incident.
func (e *RCAEngine) Analyze(ctx context.Context, sysCtx SystemContext) string {
	inc := sysCtx.Incident
	m := sysCtx.Metrics

	userPrompt := fmt.Sprintf(`INCIDENT: %s
Severity: %s | Service: %s
Triggered Metric: %s = %.4f (baseline: %.4f, deviation: %.1fx above normal)

CURRENT SYSTEM METRICS:
  error_rate:          %.2f%%
  p95_latency:         %.0f ms
  requests_per_sec:    %.1f
  db_connections:      %.1f%%
  kafka_consumer_lag:  %.0f
  redis_hit_ratio:     %.1f%%
  goroutines:          %.0f
  heap:                %.0f MB`,
		inc.Title,
		inc.Severity, inc.Service,
		inc.Metric, inc.Value, inc.Baseline, safeRatio(inc.Value, inc.Baseline),
		m["error_rate_pct"],
		m["p95_latency_ms"],
		m["rps"],
		m["db_connections_pct"],
		m["kafka_lag"],
		m["redis_hit_ratio_pct"],
		m["goroutines"],
		m["heap_mb"],
	)

	result, err := e.llm.Generate(ctx, rcaSystemPrompt, userPrompt)
	if err != nil {
		return fmt.Sprintf("RCA generation failed: %v", err)
	}
	return result
}

func safeRatio(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}
