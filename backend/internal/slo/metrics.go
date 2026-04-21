package slo

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SLORequestsTotal counts total requests for SLO tracking.
	SLORequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slo_requests_total",
			Help: "Total requests tracked by SLO.",
		},
		[]string{"slo"},
	)

	// SLOErrorsTotal counts errors for SLO tracking.
	SLOErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slo_errors_total",
			Help: "Total errors tracked by SLO.",
		},
		[]string{"slo"},
	)

	// SLOBudgetRemaining tracks the remaining error budget percentage.
	SLOBudgetRemaining = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "slo_error_budget_remaining_percent",
			Help: "Remaining error budget as a percentage.",
		},
		[]string{"slo"},
	)

	// SLOBurning indicates whether an SLO is burning its error budget.
	SLOBurning = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "slo_burning",
			Help: "Whether the SLO error budget is burning (1=burning, 0=healthy).",
		},
		[]string{"slo"},
	)
)

// RecordRequest records a request for the given SLO.
func RecordRequest(sloName string, success bool) {
	SLORequestsTotal.WithLabelValues(sloName).Inc()
	if !success {
		SLOErrorsTotal.WithLabelValues(sloName).Inc()
	}
}

// RecordBudget updates the remaining budget gauge for an SLO.
func RecordBudget(sloName string, remaining float64) {
	SLOBudgetRemaining.WithLabelValues(sloName).Set(remaining)
}

// RecordBurning updates the burning status gauge for an SLO.
func RecordBurning(sloName string, burning bool) {
	val := 0.0
	if burning {
		val = 1.0
	}
	SLOBurning.WithLabelValues(sloName).Set(val)
}
