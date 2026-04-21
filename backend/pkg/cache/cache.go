package cache

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/chaos"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/redis/go-redis/v9"
)

const (
	// stampedeLockTTL is how long a single request holds the rebuild lock.
	stampedeLockTTL = 10 * time.Second

	// staleGracePeriod is how long stale data is served while another request rebuilds.
	staleGracePeriod = 30 * time.Second

	// jitterFraction is the ±fraction of TTL added as random jitter to prevent
	// thunder herd when many keys expire at the same instant.
	// 0.15 = ±15% → a 60s TTL becomes 51–69s.
	jitterFraction = 0.15
)

// Cache wraps a Redis client with simple JSON get/set/del helpers.
type Cache struct {
	rdb *redis.Client
}

// New returns a Cache backed by the given Redis client.
func New(rdb *redis.Client) *Cache {
	return &Cache{rdb: rdb}
}

// Get retrieves a cached value and unmarshals it into dst.
// Returns false (miss) when the key is absent or on any error.
func (c *Cache) Get(ctx context.Context, key string, dst any) bool {
	if c == nil || c.rdb == nil || chaos.IsRedisDown() {
		metrics.IncCacheMiss(cacheNamespace(key))
		return false
	}
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		metrics.IncCacheMiss(cacheNamespace(key))
		return false
	}
	if json.Unmarshal(b, dst) == nil {
		metrics.IncCacheHit(cacheNamespace(key))
		return true
	}
	metrics.IncCacheMiss(cacheNamespace(key))
	return false
}

// GetWithStampedeProtection returns cached value like Get, but when the key is
// expired/missing, only ONE caller acquires the rebuild lock (SETNX). Others
// receive stale data from the grace key, or a miss if no stale data exists.
//
// Returns (hit bool, locked bool):
//   - hit=true:  data found in cache or stale grace key
//   - hit=false, locked=true:  caller should rebuild and call SetWithStampede
//   - hit=false, locked=false: another caller is rebuilding; no data available
func (c *Cache) GetWithStampedeProtection(ctx context.Context, key string, dst any) (hit bool, locked bool) {
	if c == nil || c.rdb == nil || chaos.IsRedisDown() {
		metrics.IncCacheMiss(cacheNamespace(key))
		return false, true // allow caller to rebuild from DB
	}

	// Try primary key first
	b, err := c.rdb.Get(ctx, key).Bytes()
	if err == nil && json.Unmarshal(b, dst) == nil {
		metrics.IncCacheHit(cacheNamespace(key))
		return true, false
	}

	// Cache miss — try stale grace key
	staleKey := key + ":stale"
	staleData, staleErr := c.rdb.Get(ctx, staleKey).Bytes()
	if staleErr == nil && json.Unmarshal(staleData, dst) == nil {
		metrics.IncCacheHit(cacheNamespace(key) + ":stale")
		// Serve stale data while someone rebuilds
		return true, false
	}

	// No data at all — try to acquire rebuild lock
	lockKey := key + ":lock"
	acquired, setErr := c.rdb.SetNX(ctx, lockKey, "1", stampedeLockTTL).Result()
	if setErr != nil {
		metrics.IncCacheMiss(cacheNamespace(key))
		return false, true // fallback: let caller rebuild
	}
	if acquired {
		metrics.IncCacheMiss(cacheNamespace(key))
		return false, true // caller is the rebuild leader
	}

	// Another caller is rebuilding — return miss, no lock
	metrics.IncCacheMiss(cacheNamespace(key))
	return false, false
}

// SetWithStampede stores the value and populates the stale grace key,
// then releases the rebuild lock.
func (c *Cache) SetWithStampede(ctx context.Context, key string, v any, ttl time.Duration) {
	if c == nil || c.rdb == nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}

	// Set primary key with TTL (jittered)
	_ = c.rdb.Set(ctx, key, b, jitterTTL(ttl)).Err()

	// Set stale grace key with extended TTL for stampede protection (jittered)
	staleKey := key + ":stale"
	staleTTL := ttl + staleGracePeriod
	_ = c.rdb.Set(ctx, staleKey, b, jitterTTL(staleTTL)).Err()

	// Release rebuild lock
	lockKey := key + ":lock"
	c.rdb.Del(ctx, lockKey)
}

// Set marshals v to JSON and stores it with the given TTL.
// A random jitter of ±15% is added to the TTL to prevent thunder herd
// when many keys would otherwise expire at the same instant.
func (c *Cache) Set(ctx context.Context, key string, v any, ttl time.Duration) {
	if c == nil || c.rdb == nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_ = c.rdb.Set(ctx, key, b, jitterTTL(ttl)).Err()
}

// Del removes one or more cache keys (e.g. on write/invalidation).
func (c *Cache) Del(ctx context.Context, keys ...string) {
	if c == nil || c.rdb == nil {
		return
	}
	if len(keys) == 0 {
		return
	}
	_ = c.rdb.Del(ctx, keys...).Err()
}

// jitterTTL adds ±15% random jitter to the given TTL to prevent
// synchronized cache expiry (thunder herd) across many keys.
func jitterTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return ttl
	}
	jitter := time.Duration(float64(ttl) * jitterFraction)
	return ttl + time.Duration(rand.Int63n(int64(2*jitter+1))-int64(jitter)) //nolint:gosec
}

func cacheNamespace(key string) string {
	if key == "" {
		return "unknown"
	}
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "unknown"
	}
	return parts[0]
}
