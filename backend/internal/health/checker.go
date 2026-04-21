// Package health provides system health checking and readiness probes.
package health

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Status represents the health status of a component.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// ComponentHealth holds the health of a single component.
type ComponentHealth struct {
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	LatencyMs int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// Report is the overall system health report.
type Report struct {
	Status     Status                     `json:"status"`
	Components map[string]ComponentHealth `json:"components"`
	CheckedAt  time.Time                  `json:"checked_at"`
}

// Checker performs health checks on system components.
type Checker struct {
	mu   sync.RWMutex
	db   *gorm.DB
	rdb  *redis.Client
	last Report
}

// NewChecker creates a health checker.
func NewChecker(db *gorm.DB, rdb *redis.Client) *Checker {
	return &Checker{db: db, rdb: rdb}
}

// Check runs all health checks and returns a report.
func (c *Checker) Check(ctx context.Context) Report {
	c.mu.Lock()
	defer c.mu.Unlock()

	components := make(map[string]ComponentHealth)
	overall := StatusHealthy

	// ── Database ──────────────────────────────────────────────────────────
	start := time.Now()
	sqlDB, err := c.db.DB()
	dbStatus := StatusHealthy
	dbErr := ""
	if err != nil {
		dbStatus = StatusUnhealthy
		dbErr = err.Error()
	} else if err := sqlDB.PingContext(ctx); err != nil {
		dbStatus = StatusUnhealthy
		dbErr = err.Error()
	}
	components["database"] = ComponentHealth{
		Name:      "database",
		Status:    dbStatus,
		LatencyMs: time.Since(start).Milliseconds(),
		Error:     dbErr,
		CheckedAt: time.Now(),
	}
	if dbStatus == StatusUnhealthy {
		overall = StatusUnhealthy
	}

	// ── Redis ─────────────────────────────────────────────────────────────
	start = time.Now()
	redisStatus := StatusHealthy
	redisErr := ""
	if c.rdb != nil {
		if err := c.rdb.Ping(ctx).Err(); err != nil {
			redisStatus = StatusDegraded // Redis down = degraded, not unhealthy
			redisErr = err.Error()
			if overall == StatusHealthy {
				overall = StatusDegraded
			}
		}
	}
	components["redis"] = ComponentHealth{
		Name:      "redis",
		Status:    redisStatus,
		LatencyMs: time.Since(start).Milliseconds(),
		Error:     redisErr,
		CheckedAt: time.Now(),
	}

	report := Report{
		Status:     overall,
		Components: components,
		CheckedAt:  time.Now(),
	}
	c.last = report

	if overall != StatusHealthy {
		slog.Warn("health: system not fully healthy", "status", string(overall))
	}

	return report
}

// Last returns the most recent health report.
func (c *Checker) Last() Report {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.last
}
