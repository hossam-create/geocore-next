// Package idempotency provides request deduplication using Redis.
package idempotency

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// KeyPrefix is the Redis key prefix for idempotency records.
const KeyPrefix = "idempotency:"

// Record stores the result of a previously processed request.
type Record struct {
	Status     int         `json:"status"`
	Body       interface{} `json:"body"`
	ProcessedAt time.Time  `json:"processed_at"`
}

// Store provides Redis-backed request deduplication.
type Store struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewStore creates an idempotency store with the given TTL (default 24h).
func NewStore(rdb *redis.Client, ttl ...time.Duration) *Store {
	d := 24 * time.Hour
	if len(ttl) > 0 {
		d = ttl[0]
	}
	return &Store{rdb: rdb, ttl: d}
}

// Check returns the previous response if this request was already processed.
// Returns nil if this is a new request (caller should proceed).
func (s *Store) Check(ctx context.Context, requestID string) (*Record, error) {
	if s.rdb == nil {
		return nil, nil
	}
	key := KeyPrefix + requestID
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // new request
		}
		return nil, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// Record saves the response of a processed request for future dedup.
func (s *Store) Record(ctx context.Context, requestID string, status int, body interface{}) error {
	if s.rdb == nil {
		return nil
	}
	key := KeyPrefix + requestID
	rec := Record{
		Status:      status,
		Body:        body,
		ProcessedAt: time.Now(),
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, key, data, s.ttl).Err()
}

// Acquire attempts to claim a request for processing (distributed lock).
// Returns true if this instance won the claim.
func (s *Store) Acquire(ctx context.Context, requestID string) (bool, error) {
	if s.rdb == nil {
		return true, nil
	}
	key := fmt.Sprintf("%slock:%s", KeyPrefix, requestID)
	return s.rdb.SetNX(ctx, key, "1", s.ttl).Result()
}
