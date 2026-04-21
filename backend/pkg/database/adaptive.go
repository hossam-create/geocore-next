package database

import (
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
)

// AdaptivePoolController dynamically adjusts the DB connection pool based on
// system saturation. When the system is under stress (high goroutine count,
// high DB pool utilization), it shrinks the pool to prevent DB overload.
// When stress subsides, it restores the pool to its configured maximum.
//
// This implements "adaptive concurrency control" — the system self-regulates
// its DB usage based on real-time pressure signals.
type AdaptivePoolController struct {
	sql       *sql.DB
	maxOpen   int
	maxIdle   int
	mu        sync.Mutex
	reduced   bool
}

// NewAdaptivePoolController creates a controller for the given DB.
func NewAdaptivePoolController(db *sql.DB, maxOpen, maxIdle int) *AdaptivePoolController {
	return &AdaptivePoolController{
		sql:     db,
		maxOpen: maxOpen,
		maxIdle: maxIdle,
	}
}

// ReducePool shrinks the connection pool to 50% of configured maximum.
// Called when system saturation is high (>80%).
func (a *AdaptivePoolController) ReducePool() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.reduced {
		return
	}
	reducedOpen := a.maxOpen / 2
	reducedIdle := reducedOpen / 2
	a.sql.SetMaxOpenConns(reducedOpen)
	a.sql.SetMaxIdleConns(reducedIdle)
	a.reduced = true
	slog.Warn("adaptive pool: REDUCED",
		"max_open", reducedOpen,
		"max_idle", reducedIdle,
		"reason", "high saturation",
	)
}

// RestorePool restores the connection pool to its configured maximum.
// Called when system saturation returns to normal (<50%).
func (a *AdaptivePoolController) RestorePool() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.reduced {
		return
	}
	a.sql.SetMaxOpenConns(a.maxOpen)
	a.sql.SetMaxIdleConns(a.maxIdle)
	a.reduced = false
	slog.Info("adaptive pool: RESTORED",
		"max_open", a.maxOpen,
		"max_idle", a.maxIdle,
		"reason", "saturation normalized",
	)
}

// IsReduced returns whether the pool is currently in reduced mode.
func (a *AdaptivePoolController) IsReduced() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.reduced
}

// Start begins the adaptive control loop. It checks DB pool utilization
// every 10 seconds and adjusts the pool size accordingly.
func (a *AdaptivePoolController) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			stats := a.sql.Stats()
			utilization := float64(stats.InUse) / float64(stats.MaxOpenConnections) * 100
			if stats.MaxOpenConnections == 0 {
				continue
			}

			// Report current pool state
			metrics.DBConnectionsOpen.Set(float64(stats.OpenConnections))
			metrics.DBConnectionsIdle.Set(float64(stats.Idle))
			metrics.DBConnectionsInUse.Set(float64(stats.InUse))
			metrics.DBConnectionsWaitCount.Set(float64(stats.WaitCount))

			// Adaptive decisions based on utilization
			if utilization > 85 && !a.IsReduced() {
				a.ReducePool()
			} else if utilization < 40 && a.IsReduced() {
				a.RestorePool()
			}
		}
	}()
}
