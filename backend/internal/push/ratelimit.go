package push

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ════════════════════════════════════════════════════════════════════════════
// Per-user + per-type rate limiting using Redis sliding window counters
// ════════════════════════════════════════════════════════════════════════════

const (
	// Per-user global push limits
	pushUserHourMax = 60 // max 60 pushes per user per hour
	pushUserMinMax  = 15 // max 15 pushes per user per minute

	// Redis key patterns
	pushRLUserHour = "push:rl:user:%s:hour:%s"   // push:rl:user:{uid}:hour:{YYYYMMDDHH}
	pushRLUserMin  = "push:rl:user:%s:min:%s"     // push:rl:user:{uid}:min:{YYYYMMDDHHMM}
	pushRLTypeHour = "push:rl:type:%s:%s:hour:%s" // push:rl:type:{uid}:{type}:hour:{YYYYMMDDHH}
	pushRLTypeMin  = "push:rl:type:%s:%s:min:%s"  // push:rl:type:{uid}:{type}:min:{YYYYMMDDHHMM}
)

// checkPushRateLimit returns true if the push is allowed, false if rate-limited.
// Checks: per-user global + per-user-per-type limits.
func checkPushRateLimit(ctx context.Context, rdb *redis.Client, userID, notificationType string) bool {
	if rdb == nil {
		return true
	}

	now := time.Now()
	hourKey := now.Format("2006010215")
	minKey := now.Format("200601021504")

	// ── Per-user global limits ───────────────────────────────────────────────
	userHourKey := fmt.Sprintf(pushRLUserHour, userID, hourKey)
	userMinKey := fmt.Sprintf(pushRLUserMin, userID, minKey)

	if !incrCheck(ctx, rdb, userHourKey, pushUserHourMax, time.Hour) {
		return false
	}
	if !incrCheck(ctx, rdb, userMinKey, pushUserMinMax, time.Minute) {
		return false
	}

	// ── Per-type limits ──────────────────────────────────────────────────────
	typeLimit, ok := DefaultTypeRateLimits[notificationType]
	if !ok {
		typeLimit = DefaultTypeRateLimit
	}

	typeHourKey := fmt.Sprintf(pushRLTypeHour, userID, notificationType, hourKey)
	typeMinKey := fmt.Sprintf(pushRLTypeMin, userID, notificationType, minKey)

	if !incrCheck(ctx, rdb, typeHourKey, typeLimit.MaxPerHour, time.Hour) {
		return false
	}
	if !incrCheck(ctx, rdb, typeMinKey, typeLimit.MaxPerMinute, time.Minute) {
		return false
	}

	return true
}

// incrCheck atomically increments a Redis counter and checks against the limit.
// Returns false if the limit would be exceeded (counter is NOT incremented).
func incrCheck(ctx context.Context, rdb *redis.Client, key string, limit int, window time.Duration) bool {
	// Use Lua script for atomic check-and-increment
	script := redis.NewScript(`
		local current = redis.call('GET', KEYS[1])
		if current and tonumber(current) >= tonumber(ARGV[1]) then
			return 0
		end
		local val = redis.call('INCR', KEYS[1])
		if val == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[2])
		end
		return 1
	`)

	result, err := script.Run(ctx, rdb, []string{key}, limit, int(window.Seconds())).Int()
	if err != nil {
		// On Redis error, allow the push (fail-open)
		return true
	}
	return result == 1
}
