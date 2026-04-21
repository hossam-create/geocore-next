package slo

import (
	"time"
)

// SLO defines a reliability target for a service.
type SLO struct {
	Name       string        // e.g. "api_availability", "checkout_latency"
	Target     float64       // e.g. 99.9 (percent)
	Window     time.Duration // measurement window (e.g. 30 days)
	Critical   bool          // if true, breach triggers rollback
}

// ErrorBudget returns the allowed error percentage within the SLO window.
// For a 99.9% SLO, the error budget is 0.1%.
func (s *SLO) ErrorBudget() float64 {
	return 100.0 - s.Target
}

// IsBurning returns true if the current error rate exceeds the error budget.
func (s *SLO) IsBurning(errorRate float64) bool {
	return errorRate > s.ErrorBudget()
}

// BudgetRemaining returns how much error budget is left (percentage).
func (s *SLO) BudgetRemaining(errorRate float64) float64 {
	remaining := s.ErrorBudget() - errorRate
	if remaining < 0 {
		return 0
	}
	return remaining
}

// BurnRate returns how fast the error budget is being consumed.
// burnRate > 1 means budget will be exhausted within the window.
func (s *SLO) BurnRate(errorRate float64, elapsed time.Duration) float64 {
	if elapsed <= 0 || s.Window <= 0 {
		return 0
	}
	// Expected errors over full window
	expectedBudget := s.ErrorBudget()
	// Actual errors so far (extrapolated)
	actualRate := errorRate * (float64(s.Window) / float64(elapsed))
	if expectedBudget == 0 {
		return 0
	}
	return actualRate / expectedBudget
}

// Predefined SLOs for the Geocore platform.
var (
	APIAvailability = SLO{
		Name:     "api_availability",
		Target:   99.9,
		Window:   30 * 24 * time.Hour,
		Critical: true,
	}
	CheckoutLatency = SLO{
		Name:     "checkout_p99_latency",
		Target:   95.0, // 95% of checkouts under 2s
		Window:   7 * 24 * time.Hour,
		Critical: true,
	}
	PaymentSuccess = SLO{
		Name:     "payment_success_rate",
		Target:   99.5,
		Window:   7 * 24 * time.Hour,
		Critical: true,
	}
	SearchLatency = SLO{
		Name:     "search_p95_latency",
		Target:   99.0,
		Window:   7 * 24 * time.Hour,
		Critical: false,
	}
)

// AllSLOs returns the list of platform SLOs.
func AllSLOs() []SLO {
	return []SLO{APIAvailability, CheckoutLatency, PaymentSuccess, SearchLatency}
}
