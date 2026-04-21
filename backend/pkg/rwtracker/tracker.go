package rwtracker

import (
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// RecentWriteTTL is the duration after a write during which reads for that
	// user are forced to the primary DB to avoid stale-replica reads.
	RecentWriteTTL = 5 * time.Second

	recentWritePrefix = "rw:"
)

// RecentWriteTracker records which users have performed writes recently,
// so that subsequent reads within RecentWriteTTL are routed to the primary DB
// instead of the replica (read-after-write consistency).
type RecentWriteTracker struct {
	rdb *redis.Client
	mu  sync.RWMutex
	// In-process fallback when Redis is unavailable
	local map[string]time.Time
}

// NewRecentWriteTracker creates a tracker backed by the given Redis client.
func NewRecentWriteTracker(rdb *redis.Client) *RecentWriteTracker {
	return &RecentWriteTracker{
		rdb:   rdb,
		local: make(map[string]time.Time),
	}
}

// MarkWrite records that the given user performed a write. Subsequent
// ShouldReadPrimary calls for the same user will return true for
// RecentWriteTTL seconds.
func (t *RecentWriteTracker) MarkWrite(userID string) {
	if t.rdb != nil {
		key := recentWritePrefix + userID
		if err := t.rdb.Set(nil, key, "1", RecentWriteTTL).Err(); err != nil {
			// Fallback to in-process map
			t.mu.Lock()
			t.local[userID] = time.Now().Add(RecentWriteTTL)
			t.mu.Unlock()
		}
	} else {
		t.mu.Lock()
		t.local[userID] = time.Now().Add(RecentWriteTTL)
		t.mu.Unlock()
	}
}

// ShouldReadPrimary returns true when the given user has performed a write
// within the last RecentWriteTTL seconds, meaning reads should go to the
// primary DB to avoid stale replica data.
func (t *RecentWriteTracker) ShouldReadPrimary(userID string) bool {
	if t.rdb != nil {
		key := recentWritePrefix + userID
		val, err := t.rdb.Exists(nil, key).Result()
		if err == nil && val > 0 {
			return true
		}
	}
	// Check in-process fallback
	t.mu.RLock()
	expiry, ok := t.local[userID]
	t.mu.RUnlock()
	if ok && time.Now().Before(expiry) {
		return true
	}
	// Clean up expired local entry
	if ok {
		t.mu.Lock()
		delete(t.local, userID)
		t.mu.Unlock()
	}
	return false
}
