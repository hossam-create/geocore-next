package region

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// Store provides Redis-backed region status caching and sticky user routing.
type Store struct {
	rdb *redis.Client
}

// NewStore creates a region store backed by Redis.
func NewStore(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

// SetStatus caches the current health of all regions (TTL 10s).
func (s *Store) SetStatus(ctx context.Context, regions []RegionStatus) error {
	if s.rdb == nil {
		return nil
	}
	data, _ := json.Marshal(regions)
	return s.rdb.Set(ctx, "regions:status", data, 10*time.Second).Err()
}

// GetStatus retrieves the cached region health data.
func (s *Store) GetStatus(ctx context.Context) ([]RegionStatus, error) {
	if s.rdb == nil {
		return nil, redis.Nil
	}
	val, err := s.rdb.Get(ctx, "regions:status").Bytes()
	if err != nil {
		return nil, err
	}
	var regions []RegionStatus
	_ = json.Unmarshal(val, &regions)
	return regions, nil
}

// SetUserRegion pins a user to a specific region for 24h (sticky routing).
func (s *Store) SetUserRegion(ctx context.Context, userID, region string) error {
	if s.rdb == nil {
		return nil
	}
	return s.rdb.Set(ctx, "user:region:"+userID, region, 24*time.Hour).Err()
}

// GetUserRegion retrieves the pinned region for a user.
func (s *Store) GetUserRegion(ctx context.Context, userID string) (string, error) {
	if s.rdb == nil {
		return "", redis.Nil
	}
	return s.rdb.Get(ctx, "user:region:"+userID).Result()
}
