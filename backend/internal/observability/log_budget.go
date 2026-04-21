package observability

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

// LogBudget controls log volume to prevent log flooding in production.
type LogBudget struct {
	dailyLimit int64
	used       atomic.Int64
	mu         sync.Mutex
	lastReset  time.Time
}

// NewLogBudget creates a log budget with a daily limit.
func NewLogBudget(dailyLimit int) *LogBudget {
	return &LogBudget{
		dailyLimit: int64(dailyLimit),
		lastReset:  time.Now(),
	}
}

// Allow returns true if the log budget has not been exhausted.
func (b *LogBudget) Allow() bool {
	b.maybeReset()
	used := b.used.Add(1)
	if used > b.dailyLimit {
		b.used.Add(-1) // don't count over-budget
		return false
	}
	return true
}

// AllowLevel checks budget and logs a warning when budget is running low.
func (b *LogBudget) AllowLevel() bool {
	if !b.Allow() {
		return false
	}
	used := b.used.Load()
	if used > b.dailyLimit*90/100 {
		slog.Warn("log_budget: 90% exhausted", "used", used, "limit", b.dailyLimit)
	}
	return true
}

// Remaining returns how many log entries are left in the budget.
func (b *LogBudget) Remaining() int64 {
	b.maybeReset()
	return b.dailyLimit - b.used.Load()
}

// Reset manually resets the budget counter.
func (b *LogBudget) Reset() {
	b.used.Store(0)
	b.lastReset = time.Now()
}

func (b *LogBudget) maybeReset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.Sub(b.lastReset) >= 24*time.Hour {
		b.used.Store(0)
		b.lastReset = now
	}
}
