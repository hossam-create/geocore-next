package chaos

import (
	"sync"
	"sync/atomic"
	"time"
)

// State holds the current chaos injection state (thread-safe).
// Only active when APP_ENV != "production".
type State struct {
	mu         sync.RWMutex
	redisDown  atomic.Bool
	kafkaDown  atomic.Bool
	dbLatency  atomic.Int64 // nanoseconds
	regions    map[string]bool // region name → down?
}

var global State

func init() {
	global.regions = make(map[string]bool)
}

// Enabled returns true if chaos is allowed (non-production only).
func Enabled() bool {
	return true // caller checks APP_ENV before using
}

// ── Redis ──────────────────────────────────────────────────────────────────

func SetRedisDown(down bool) { global.redisDown.Store(down) }
func IsRedisDown() bool      { return global.redisDown.Load() }

// ── Kafka ──────────────────────────────────────────────────────────────────

func SetKafkaDown(down bool) { global.kafkaDown.Store(down) }
func IsKafkaDown() bool      { return global.kafkaDown.Load() }

// ── DB Latency ─────────────────────────────────────────────────────────────

func SetDBLatency(d time.Duration) { global.dbLatency.Store(int64(d)) }
func DBLatency() time.Duration     { return time.Duration(global.dbLatency.Load()) }

// ── Region ─────────────────────────────────────────────────────────────────

func SetRegionDown(name string, down bool) {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.regions[name] = down
}

func IsRegionDown(name string) bool {
	global.mu.RLock()
	defer global.mu.RUnlock()
	return global.regions[name]
}

func DownRegions() []string {
	global.mu.RLock()
	defer global.mu.RUnlock()
	var names []string
	for r, down := range global.regions {
		if down {
			names = append(names, r)
		}
	}
	return names
}

// Reset clears all chaos state.
func Reset() {
	global.redisDown.Store(false)
	global.kafkaDown.Store(false)
	global.dbLatency.Store(0)
	global.mu.Lock()
	defer global.mu.Unlock()
	global.regions = make(map[string]bool)
}
