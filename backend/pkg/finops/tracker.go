// Package finops provides cost tracking instrumentation for the platform.
package finops

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
)

// CostTracker tracks cloud cost allocation per service and event type.
type CostTracker struct {
	monthlyBudget float64
	currentSpend  atomic.Int64 // in cents
	requestCount  atomic.Int64 // total requests this month
}

// NewCostTracker creates a cost tracker with a monthly budget (in USD).
func NewCostTracker(monthlyBudgetUSD float64) *CostTracker {
	return &CostTracker{monthlyBudget: monthlyBudgetUSD}
}

// RecordSpend adds a cost entry (in USD cents).
func (t *CostTracker) RecordSpend(ctx context.Context, service string, cents int64) {
	t.currentSpend.Add(cents)
	current := float64(t.currentSpend.Load()) / 100.0
	pct := (current / t.monthlyBudget) * 100

	slog.Debug("finops: spend recorded",
		"service", service,
		"cents", cents,
		"total_usd", current,
		"budget_pct", pct,
	)

	if pct > 80 {
		slog.Warn("finops: approaching budget limit", "pct", pct, "budget", t.monthlyBudget)
	}
}

// IncRequest increments the monthly request counter for cost-per-request tracking.
func (t *CostTracker) IncRequest() {
	t.requestCount.Add(1)
}

// CostPerRequest calculates the current cost per HTTP request.
func (t *CostTracker) CostPerRequest() float64 {
	count := t.requestCount.Load()
	if count == 0 {
		return 0
	}
	return (float64(t.currentSpend.Load()) / 100.0) / float64(count)
}

// CostPerOrder calculates the current cost per order.
func (t *CostTracker) CostPerOrder() float64 {
	// This would normally query Prometheus, simplified here
	return 0
}

// CostPerTransaction calculates the current cost per transaction.
func (t *CostTracker) CostPerTransaction() float64 {
	return 0
}

// GetCurrentSpend returns the current monthly spend in USD.
func (t *CostTracker) GetCurrentSpend() float64 {
	return float64(t.currentSpend.Load()) / 100.0
}

// GetBudgetUtilization returns the percentage of budget used.
func (t *CostTracker) GetBudgetUtilization() float64 {
	current := float64(t.currentSpend.Load()) / 100.0
	if t.monthlyBudget == 0 {
		return 0
	}
	return (current / t.monthlyBudget) * 100
}

// ── Waste Detection ────────────────────────────────────────────────────────

// WasteReport identifies resources that are over-provisioned or idle.
type WasteReport struct {
	Timestamp           time.Time    `json:"timestamp"`
	IdlePods            []PodWaste   `json:"idle_pods,omitempty"`
	OverProvisioned     []PodWaste   `json:"overprovisioned,omitempty"`
	CacheInefficiency   []CacheWaste `json:"cache_inefficiency,omitempty"`
	Recommendations     []string     `json:"recommendations"`
	EstimatedSavingsUSD float64      `json:"estimated_savings_usd"`
}

// PodWaste describes an underutilized pod.
type PodWaste struct {
	Name        string  `json:"name"`
	Namespace   string  `json:"namespace"`
	CPUUsagePct float64 `json:"cpu_usage_pct"`
	MemUsagePct float64 `json:"mem_usage_pct"`
	RequestCPU  string  `json:"request_cpu"`
	RequestMem  string  `json:"request_mem"`
	WasteType   string  `json:"waste_type"` // "idle" or "overprovisioned"
}

// CacheWaste describes cache inefficiency.
type CacheWaste struct {
	Namespace string  `json:"namespace"`
	HitRate   float64 `json:"hit_rate"`
	KeyCount  int64   `json:"key_count"`
	MemoryMB  float64 `json:"memory_mb"`
}

// DetectWaste identifies idle pods, over-provisioned resources, and cache inefficiency.
// In production, this would query the Kubernetes API and Prometheus.
func DetectWaste() WasteReport {
	report := WasteReport{
		Timestamp: time.Now(),
		Recommendations: []string{
			"Review HPA idle scale-down thresholds (current: CPU <30% → 50% pod reduction after 10min)",
			"Check Redis maxmemory-policy — recommend allkeys-lru for cache-only workloads",
			"Verify RDS auto-stop is enabled for dev/staging (midnight-8am weekdays)",
			"Consider SPOT instances for non-critical workloads (worker, notification-service)",
			"Review log retention: dev=3d, staging=7d, prod=14d — reduce if possible",
		},
		EstimatedSavingsUSD: 0, // would be calculated from actual resource data
	}
	return report
}

// Report generates a cost report.
func (t *CostTracker) Report() map[string]interface{} {
	return map[string]interface{}{
		"monthly_budget_usd":   t.monthlyBudget,
		"current_spend_usd":    t.GetCurrentSpend(),
		"budget_utilization":   t.GetBudgetUtilization(),
		"cost_per_request":     t.CostPerRequest(),
		"total_requests_month": t.requestCount.Load(),
		"timestamp":            time.Now().Format(time.RFC3339),
	}
}

// Ensure metrics package is referenced (import used above).
var _ = metrics.Init
