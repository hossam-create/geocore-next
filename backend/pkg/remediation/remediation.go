// Package remediation provides auto-remediation capabilities for the GeoCore platform.
//
// Current strategies:
//   - Redis fallback: switch to DB-backed caching when Redis is unavailable
//   - Consumer auto-scale: signal HPA when Kafka lag spikes
//   - DB read-only: enable degraded mode for non-critical endpoints under DB load
package remediation

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisAvailable  atomic.Bool
	dbReadOnlyMode  atomic.Bool
	consumerScaling atomic.Bool
	remediationMu   sync.Mutex
	lastRedisCheck  time.Time
	lastDBLoadCheck time.Time
)

// Init sets the initial state — Redis assumed available, DB not in read-only.
func Init() {
	redisAvailable.Store(true)
	dbReadOnlyMode.Store(false)
	consumerScaling.Store(false)
}

// ── Redis Fallback ─────────────────────────────────────────────────────────

// CheckRedis probes Redis and updates availability state.
// Call this from a periodic health check (e.g. every 10s).
func CheckRedis(rdb *redis.Client) {
	if rdb == nil {
		redisAvailable.Store(false)
		return
	}
	err := rdb.Ping(context.Background()).Err()
	wasAvailable := redisAvailable.Load()
	nowAvailable := err == nil
	redisAvailable.Store(nowAvailable)

	if wasAvailable && !nowAvailable {
		slog.Warn("remediation: Redis UNAVAILABLE — switching to DB fallback cache")
	} else if !wasAvailable && nowAvailable {
		slog.Info("remediation: Redis RECOVERED — switching back to Redis cache")
	}
}

// IsRedisAvailable returns whether Redis is currently reachable.
func IsRedisAvailable() bool {
	return redisAvailable.Load()
}

// ── DB Read-Only Mode ──────────────────────────────────────────────────────

// EnableDBReadOnly puts non-critical endpoints in degraded/read-only mode.
// Critical endpoints (payments, wallet) always remain read-write.
func EnableDBReadOnly(reason string) {
	if !dbReadOnlyMode.Load() {
		slog.Warn("remediation: enabling DB read-only mode", "reason", reason)
		dbReadOnlyMode.Store(true)
	}
}

// DisableDBReadOnly restores normal DB operations.
func DisableDBReadOnly() {
	if dbReadOnlyMode.Load() {
		slog.Info("remediation: disabling DB read-only mode — back to normal")
		dbReadOnlyMode.Store(false)
	}
}

// IsDBReadOnly returns whether the system is in DB read-only/degraded mode.
func IsDBReadOnly() bool {
	return dbReadOnlyMode.Load()
}

// ── Consumer Auto-Scale ────────────────────────────────────────────────────

// SignalConsumerScale signals that Kafka consumer lag is high and more
// consumer pods are needed. The HPA will pick this up via the
// kafka_consumer_lag custom metric.
func SignalConsumerScale(lag int64) {
	if lag > 5000 && !consumerScaling.Load() {
		slog.Warn("remediation: Kafka lag spike — signaling consumer scale-up", "lag", lag)
		consumerScaling.Store(true)
		// The HPA configured in k8s/hpa.yaml handles the actual scaling
		// via the kafka_consumer_lag Prometheus metric.
	} else if lag < 1000 && consumerScaling.Load() {
		slog.Info("remediation: Kafka lag normalized — consumers can scale down", "lag", lag)
		consumerScaling.Store(false)
	}
}

// IsConsumerScaling returns whether consumers are currently being auto-scaled.
func IsConsumerScaling() bool {
	return consumerScaling.Load()
}

// ── System Status ─────────────────────────────────────────────────────────

// Status returns the current remediation state for health/metrics endpoints.
type Status struct {
	RedisAvailable  bool `json:"redis_available"`
	DBReadOnly      bool `json:"db_read_only"`
	ConsumerScaling bool `json:"consumer_scaling"`
}

// GetStatus returns a snapshot of the current remediation state.
func GetStatus() Status {
	return Status{
		RedisAvailable:  redisAvailable.Load(),
		DBReadOnly:      dbReadOnlyMode.Load(),
		ConsumerScaling: consumerScaling.Load(),
	}
}
