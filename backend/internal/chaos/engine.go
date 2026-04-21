package chaos

import (
	"context"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
)

// ChaosEngine runs controlled, percentage-based failure injection.
// Injection rates are stored in Redis as chaos:<key> → int (0-100).
// This enables dynamic, remote-controlled chaos without redeployment.
type ChaosEngine struct {
	rdb *redis.Client
}

// NewChaosEngine creates a chaos engine backed by Redis.
func NewChaosEngine(rdb *redis.Client) *ChaosEngine {
	return &ChaosEngine{rdb: rdb}
}

// ShouldInject returns true if chaos should be injected for the given key.
// Reads the injection percentage from Redis key "chaos:<key>".
// Example: chaos:api_latency = 30 → 30% of requests get latency injected.
func (e *ChaosEngine) ShouldInject(ctx context.Context, key string) bool {
	pct := e.getRate(ctx, key)
	if pct <= 0 {
		return false
	}
	return rand.Intn(100) < pct
}

// ShouldInjectN returns true with the given fixed percentage (no Redis lookup).
func ShouldInjectN(pct int) bool {
	if pct <= 0 {
		return false
	}
	return rand.Intn(100) < pct
}

// SetRate sets the injection percentage for a chaos key (0-100).
func (e *ChaosEngine) SetRate(ctx context.Context, key string, pct int) error {
	if e.rdb == nil {
		return nil
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return e.rdb.Set(ctx, "chaos:"+key, pct, 0).Err()
}

// GetRate reads the current injection percentage for a key.
func (e *ChaosEngine) GetRate(ctx context.Context, key string) int {
	return e.getRate(ctx, key)
}

func (e *ChaosEngine) getRate(ctx context.Context, key string) int {
	if e.rdb == nil {
		return 0
	}
	val, err := e.rdb.Get(ctx, "chaos:"+key).Int()
	if err != nil {
		return 0
	}
	return val
}

// AllRates returns all active chaos injection rates.
func (e *ChaosEngine) AllRates(ctx context.Context) map[string]int {
	if e.rdb == nil {
		return nil
	}
	keys, err := e.rdb.Keys(ctx, "chaos:*").Result()
	if err != nil {
		return nil
	}
	rates := make(map[string]int, len(keys))
	for _, k := range keys {
		val, _ := e.rdb.Get(ctx, k).Int()
		rates[k] = val
	}
	return rates
}

// ResetAll clears all chaos injection rates.
func (e *ChaosEngine) ResetAll(ctx context.Context) error {
	if e.rdb == nil {
		return nil
	}
	keys, err := e.rdb.Keys(ctx, "chaos:*").Result()
	if err != nil {
		return nil
	}
	if len(keys) > 0 {
		return e.rdb.Del(ctx, keys...).Err()
	}
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
