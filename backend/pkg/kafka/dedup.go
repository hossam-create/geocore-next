package kafka

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// DedupStore prevents cross-region duplicate event processing using Redis.
type DedupStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewDedupStore creates a dedup store with 24h TTL (cross-region safety window).
func NewDedupStore(rdb *redis.Client) *DedupStore {
	return &DedupStore{rdb: rdb, ttl: 24 * time.Hour}
}

// IsDuplicate checks if an event has already been processed.
func (d *DedupStore) IsDuplicate(ctx context.Context, key string) (bool, error) {
	if d.rdb == nil {
		return false, nil
	}
	exists, err := d.rdb.Exists(ctx, "evt:"+key).Result()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

// MarkProcessed records an event as processed for 24h.
func (d *DedupStore) MarkProcessed(ctx context.Context, key string) error {
	if d.rdb == nil {
		return nil
	}
	return d.rdb.Set(ctx, "evt:"+key, "1", d.ttl).Err()
}

// CheckAndMark is an atomic check-then-mark. Returns true if duplicate.
func (d *DedupStore) CheckAndMark(ctx context.Context, key string) (isDuplicate bool, err error) {
	if d.rdb == nil {
		return false, nil
	}
	// SETNX is atomic — only succeeds if key doesn't exist
	ok, err := d.rdb.SetNX(ctx, "evt:"+key, "1", d.ttl).Result()
	if err != nil {
		return false, err
	}
	if !ok {
		// Key already existed → duplicate
		slog.Info("kafka: duplicate event skipped", "idempotency_key", key)
		return true, nil
	}
	return false, nil
}
