package slo

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/remediation"
)

// Engine continuously evaluates SLOs and triggers remediation on breach.
type Engine struct {
	slos     []SLO
	mu       sync.RWMutex
	interval time.Duration
	health   map[string]SLOHealth
}

// SLOHealth holds the current health status of an SLO.
type SLOHealth struct {
	Name            string    `json:"name"`
	Target          float64   `json:"target"`
	CurrentRate     float64   `json:"current_rate"`
	ErrorBudget     float64   `json:"error_budget"`
	BudgetRemaining float64   `json:"budget_remaining"`
	Burning         bool      `json:"burning"`
	Critical        bool      `json:"critical"`
	LastChecked     time.Time `json:"last_checked"`
}

// NewEngine creates an SLO evaluation engine.
func NewEngine(slos []SLO, interval time.Duration) *Engine {
	return &Engine{
		slos:     slos,
		interval: interval,
		health:   make(map[string]SLOHealth),
	}
}

// Start begins the SLO evaluation loop.
func (e *Engine) Start(ctx context.Context) {
	slog.Info("slo: engine started", "interval", e.interval, "slos", len(e.slos))
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	// Initial evaluation
	e.evaluate(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("slo: engine stopped")
			return
		case <-ticker.C:
			e.evaluate(ctx)
		}
	}
}

func (e *Engine) evaluate(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, slo := range e.slos {
		errorRate := e.getErrorRate(slo.Name)
		burning := slo.IsBurning(errorRate)
		remaining := slo.BudgetRemaining(errorRate)

		health := SLOHealth{
			Name:            slo.Name,
			Target:          slo.Target,
			CurrentRate:     100.0 - errorRate,
			ErrorBudget:     slo.ErrorBudget(),
			BudgetRemaining: remaining,
			Burning:         burning,
			Critical:        slo.Critical,
			LastChecked:     time.Now().UTC(),
		}
		e.health[slo.Name] = health

		if burning {
			slog.Error("slo: error budget burning",
				"slo", slo.Name,
				"error_rate", errorRate,
				"budget_remaining", remaining,
				"critical", slo.Critical,
			)

			// Auto-rollback for critical SLOs
			if slo.Critical {
				remediation.SignalRollback("slo_breach:" + slo.Name)
				TriggerEmergencyMode()
			}
		} else {
			slog.Debug("slo: healthy", "slo", slo.Name, "availability", 100.0-errorRate)
		}
	}
}

// getErrorRate reads the current error rate for an SLO from metrics.
// In production, this reads from Prometheus. Returns 0 (healthy) as default.
func (e *Engine) getErrorRate(sloName string) float64 {
	// TODO: Read from Prometheus counters in production
	// For now, return 0 (healthy) — real implementation queries /metrics
	return 0
}

// Health returns the current health of all SLOs.
func (e *Engine) Health() map[string]SLOHealth {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make(map[string]SLOHealth, len(e.health))
	for k, v := range e.health {
		result[k] = v
	}
	return result
}

// IsHealthy returns true if no SLO is burning.
func (e *Engine) IsHealthy() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	for _, h := range e.health {
		if h.Burning {
			return false
		}
	}
	return true
}

// IsBurning returns true if any SLO's error budget is being consumed too fast.
func (e *Engine) IsBurning(errorRate float64) bool {
	for _, slo := range e.slos {
		if slo.IsBurning(errorRate) {
			return true
		}
	}
	return false
}

// TriggerEmergencyMode signals the autonomy layer to enter emergency mode.
// Called when critical SLOs are burning and auto-remediation is needed.
func TriggerEmergencyMode() {
	slog.Error("slo: TRIGGERING EMERGENCY MODE — critical SLO breach")
	// The autonomy control loop will pick this up on its next tick
	// via the DecisionEngine.Evaluate() which checks sloEngine.IsHealthy()
}
