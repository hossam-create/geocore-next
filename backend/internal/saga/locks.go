package saga

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lock represents a distributed lock acquired via Redis.
type Lock struct {
	rdb   *redis.Client
	key   string
	value string
	ttl   time.Duration
}

// AcquireLock tries to acquire a distributed lock using Redis SETNX.
// Returns the lock if acquired, nil if the key is already locked.
func AcquireLock(ctx context.Context, rdb *redis.Client, key string, ttl time.Duration) *Lock {
	if rdb == nil {
		return &Lock{key: key, ttl: ttl} // no-op lock when Redis unavailable
	}

	value := time.Now().Format(time.RFC3339Nano)
	acquired, err := rdb.SetNX(ctx, "lock:"+key, value, ttl).Result()
	if err != nil || !acquired {
		return nil
	}

	return &Lock{rdb: rdb, key: key, value: value, ttl: ttl}
}

// Release removes the lock from Redis (only if we still hold it).
func (l *Lock) Release(ctx context.Context) bool {
	if l == nil || l.rdb == nil {
		return true
	}

	// Only release if we still hold the lock (compare values)
	script := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`
	result, err := l.rdb.Eval(ctx, script, []string{"lock:" + l.key}, l.value).Int()
	if err != nil {
		return false
	}
	return result == 1
}

// Extend renews the lock TTL.
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) bool {
	if l == nil || l.rdb == nil {
		return true
	}

	script := `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("PEXPIRE", KEYS[1], ARGV[2])
	else
		return 0
	end
	`
	result, err := l.rdb.Eval(ctx, script, []string{"lock:" + l.key}, l.value, int(ttl.Milliseconds())).Int()
	if err != nil {
		return false
	}
	return result == 1
}

// IsHeld checks if the lock is currently held.
func (l *Lock) IsHeld(ctx context.Context) bool {
	if l == nil || l.rdb == nil {
		return false
	}
	val, err := l.rdb.Get(ctx, "lock:"+l.key).Result()
	if err != nil {
		return false
	}
	return val == l.value
}
