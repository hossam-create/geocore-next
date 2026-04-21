package featureflags

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Flags provides Redis-backed feature flag evaluation with TTL + versioning.
// Flag format:
//
//	flag:<key>          → "1" or "0" (enabled/disabled)
//	flag:<key>:version  → incrementing counter (detect stale flags)
//	flag:<key>:rollout  → 0-100 percentage
//	flag:<key>:updated  → RFC3339 timestamp of last change
type Flags struct {
	rdb      *redis.Client
	defaults map[string]bool
}

// NewFlags creates a feature flag store backed by Redis.
func NewFlags(rdb *redis.Client) *Flags {
	return &Flags{
		rdb:      rdb,
		defaults: make(map[string]bool),
	}
}

// SetDefault sets the default value for a flag when Redis is unavailable.
func (f *Flags) SetDefault(key string, enabled bool) {
	f.defaults[key] = enabled
}

// IsEnabled checks if a feature flag is enabled.
// Falls back to the configured default, then false.
func (f *Flags) IsEnabled(ctx context.Context, key string) bool {
	if f.rdb != nil {
		val, err := f.rdb.Get(ctx, "flag:"+key).Result()
		if err == nil {
			return val == "1"
		}
		if err != redis.Nil {
			slog.Debug("featureflags: redis error, using default", "key", key, "error", err)
		}
	}

	if def, ok := f.defaults[key]; ok {
		return def
	}
	return false
}

// Set enables or disables a feature flag at runtime (no TTL, persists forever).
func (f *Flags) Set(ctx context.Context, key string, enabled bool) error {
	return f.SetWithTTL(ctx, key, enabled, 0)
}

// SetWithTTL enables or disables a flag with an optional TTL.
// ttl=0 means no expiry. Automatically bumps the version.
func (f *Flags) SetWithTTL(ctx context.Context, key string, enabled bool, ttl time.Duration) error {
	if f.rdb == nil {
		return nil
	}
	val := "0"
	if enabled {
		val = "1"
	}

	pipe := f.rdb.Pipeline()
	pipe.Set(ctx, "flag:"+key, val, ttl)
	pipe.Incr(ctx, "flag:"+key+":version")
	pipe.Set(ctx, "flag:"+key+":updated", time.Now().UTC().Format(time.RFC3339), ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	slog.Info("featureflags: flag set",
		"key", key, "enabled", enabled, "version", f.getVersionQuiet(ctx, key), "ttl", ttl)
	return nil
}

// GetRollout returns the rollout percentage (0-100) for a flag.
// Returns 100 if fully enabled, 0 if disabled, or a partial value.
func (f *Flags) GetRollout(ctx context.Context, key string) int {
	if f.rdb == nil {
		if def, ok := f.defaults[key]; ok && def {
			return 100
		}
		return 0
	}
	val, err := f.rdb.Get(ctx, "flag:"+key+":rollout").Int()
	if err != nil {
		// If flag is enabled without rollout, assume 100%
		if f.IsEnabled(ctx, key) {
			return 100
		}
		return 0
	}
	return val
}

// IsEnabledForUser checks if a flag is enabled for a specific user
// using rollout percentage (hash-based deterministic).
func (f *Flags) IsEnabledForUser(ctx context.Context, key, userID string) bool {
	if !f.IsEnabled(ctx, key) {
		return false
	}
	rollout := f.GetRollout(ctx, key)
	if rollout >= 100 {
		return true
	}
	if rollout <= 0 {
		return false
	}
	// Deterministic hash: same user always gets same result
	hash := simpleHash(userID+key) % 100
	return hash < rollout
}

// FlagInfo holds full metadata about a flag for admin display.
type FlagInfo struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
	Version int    `json:"version"`
	Rollout int    `json:"rollout"`
	Updated string `json:"updated"`
	Stale   bool   `json:"stale"`
}

// GetInfo returns full metadata for a flag.
func (f *Flags) GetInfo(ctx context.Context, key string) FlagInfo {
	return FlagInfo{
		Key:     key,
		Enabled: f.IsEnabled(ctx, key),
		Version: f.GetVersion(ctx, key),
		Rollout: f.GetRollout(ctx, key),
		Updated: f.GetUpdated(ctx, key),
		Stale:   f.IsStale(ctx, key, 30*24*time.Hour), // 30 days
	}
}

// Summary returns a human-readable flag summary (for logging).
func (fi FlagInfo) Summary() string {
	return fmt.Sprintf("%s=%v v%d rollout=%d%% updated=%s stale=%v",
		fi.Key, fi.Enabled, fi.Version, fi.Rollout, fi.Updated, fi.Stale)
}

// GetVersion returns the current version of a flag (incremented on each Set).
// Returns 0 if flag has never been set.
func (f *Flags) GetVersion(ctx context.Context, key string) int {
	if f.rdb == nil {
		return 0
	}
	val, err := f.rdb.Get(ctx, "flag:"+key+":version").Int()
	if err != nil {
		return 0
	}
	return val
}

// GetUpdated returns the last update timestamp of a flag.
func (f *Flags) GetUpdated(ctx context.Context, key string) string {
	if f.rdb == nil {
		return ""
	}
	val, err := f.rdb.Get(ctx, "flag:"+key+":updated").Result()
	if err != nil {
		return ""
	}
	return val
}

// IsStale returns true if a flag hasn't been updated within the given duration.
func (f *Flags) IsStale(ctx context.Context, key string, maxAge time.Duration) bool {
	updated := f.GetUpdated(ctx, key)
	if updated == "" {
		return true // never set = stale
	}
	t, err := time.Parse(time.RFC3339, updated)
	if err != nil {
		return true
	}
	return time.Since(t) > maxAge
}

// List returns all flag keys matching a prefix (for admin dashboards).
func (f *Flags) List(ctx context.Context, prefix string) ([]string, error) {
	if f.rdb == nil {
		return nil, nil
	}
	pattern := "flag:" + prefix + "*"
	keys, err := f.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, err
	}
	// Filter to only base keys (no :version, :rollout, :updated suffixes)
	var result []string
	for _, k := range keys {
		if !isMetaKey(k) {
			result = append(result, k)
		}
	}
	return result, nil
}

func isMetaKey(k string) bool {
	return len(k) > 8 && (k[len(k)-8:] == ":version" || k[len(k)-8:] == ":rollout" || k[len(k)-8:] == ":updated")
}

func (f *Flags) getVersionQuiet(ctx context.Context, key string) int {
	val, _ := f.rdb.Get(ctx, "flag:"+key+":version").Int()
	return val
}

func simpleHash(s string) int {
	h := 0
	for _, c := range s {
		h = h*31 + int(c)
	}
	if h < 0 {
		h = -h
	}
	return h
}
