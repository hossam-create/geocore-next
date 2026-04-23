package email

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	idempotencyTTL = 24 * time.Hour
	rateLimitTTL   = 2 * time.Hour // keep the key alive for 2 hours
)

// rateLimitKey returns the Redis key for a user's hourly send counter.
// Bucketed by UTC hour so it resets automatically.
func rateLimitKey(userID string) string {
	hour := time.Now().UTC().Format("2006010215") // YYYYMMDDhh
	return fmt.Sprintf("email:rl:%s:%s", userID, hour)
}

// idemKey returns the Redis key for an idempotency token.
func idemKey(token string) string {
	return "email:idem:" + token
}

// isRateLimited returns true if the user has reached the hourly limit.
func isRateLimited(ctx context.Context, rdb *redis.Client, userID string, limit int) (bool, error) {
	key := rateLimitKey(userID)
	count, err := rdb.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		return false, err
	}
	return count >= limit, nil
}

// incrementRateLimit atomically increments the per-user hourly counter.
func incrementRateLimit(ctx context.Context, rdb *redis.Client, userID string) error {
	key := rateLimitKey(userID)
	pipe := rdb.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rateLimitTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// isAlreadySent checks whether an idempotency token has been used.
func isAlreadySent(ctx context.Context, rdb *redis.Client, token string) (bool, error) {
	n, err := rdb.Exists(ctx, idemKey(token)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// markSent records an idempotency token so duplicate sends are suppressed.
func markSent(ctx context.Context, rdb *redis.Client, token string) error {
	return rdb.Set(ctx, idemKey(token), "1", idempotencyTTL).Err()
}
