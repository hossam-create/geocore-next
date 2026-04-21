package billing

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
)

type meterKey struct {
	TenantID  string
	EventType EventType
}

// Meter buffers usage events in memory and flushes to the DB periodically.
// This prevents hammering the DB on every API request.
type Meter struct {
	mu       sync.Mutex
	counts   map[meterKey]int64
	db       *gorm.DB
	interval time.Duration
}

// GlobalMeter is the application-wide usage meter, initialised at startup.
var GlobalMeter *Meter

// NewMeter creates a Meter with a 60s flush interval.
func NewMeter(db *gorm.DB) *Meter {
	return &Meter{
		counts:   make(map[meterKey]int64),
		db:       db,
		interval: 60 * time.Second,
	}
}

// Record increments the in-memory counter for a tenant+event (non-blocking).
func (m *Meter) Record(tenantID string, et EventType, qty int64) {
	if tenantID == "" {
		return
	}
	m.mu.Lock()
	m.counts[meterKey{tenantID, et}] += qty
	m.mu.Unlock()
}

// Start begins the background flush loop. Call once at application startup.
func (m *Meter) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.Flush(ctx)
			case <-ctx.Done():
				m.Flush(context.Background()) // final flush on graceful shutdown
				return
			}
		}
	}()
}

// Flush writes all accumulated counts to the database atomically.
func (m *Meter) Flush(ctx context.Context) {
	m.mu.Lock()
	if len(m.counts) == 0 {
		m.mu.Unlock()
		return
	}
	snapshot := m.counts
	m.counts = make(map[meterKey]int64)
	m.mu.Unlock()

	now := time.Now()
	events := make([]UsageEvent, 0, len(snapshot))
	for k, qty := range snapshot {
		events = append(events, UsageEvent{
			TenantID:  k.TenantID,
			EventType: k.EventType,
			Quantity:  qty,
			Metadata:  fmt.Sprintf(`{"source":"meter","ts":"%s"}`, now.Format(time.RFC3339)),
			Ts:        now,
		})
	}

	if err := m.db.WithContext(ctx).Create(&events).Error; err != nil {
		slog.Error("billing: meter flush failed", "error", err, "events", len(events))
		// Re-queue failed events
		m.mu.Lock()
		for k, qty := range snapshot {
			m.counts[k] += qty
		}
		m.mu.Unlock()
	} else {
		slog.Debug("billing: meter flushed", "events", len(events))
	}
}
