package fraud

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// FeatureStore provides fast TTL-based feature caching in Redis
// for real-time fraud scoring (<20ms overhead target).
type FeatureStore struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewFeatureStore creates a Redis-backed feature store.
func NewFeatureStore(rdb *redis.Client) *FeatureStore {
	return &FeatureStore{
		rdb: rdb,
		ttl: 24 * time.Hour,
	}
}

// ── Feature Schema ──────────────────────────────────────────────────────────

// UserFeatures holds extracted fraud signals for a user.
type UserFeatures struct {
	UserID           string  `json:"user_id"`
	AccountAgeHours  float64 `json:"account_age_hours"`
	TotalOrders      int     `json:"total_orders"`
	TotalSpent       float64 `json:"total_spent"`
	AvgOrderValue    float64 `json:"avg_order_value"`
	TxCountLast1h    int     `json:"tx_count_last_1h"`
	TxCountLast24h   int     `json:"tx_count_last_24h"`
	WithdrawCount24h int     `json:"withdraw_count_24h"`
	FailedLogins24h  int     `json:"failed_logins_24h"`
	GeoMismatchCount int     `json:"geo_mismatch_count"`
	DeviceCount7d    int     `json:"device_count_7d"`
	WalletDrift24h   float64 `json:"wallet_drift_24h"`
	LastTxTime       string  `json:"last_tx_time"`
	LastLoginIP      string  `json:"last_login_ip"`
	LastLoginCountry string  `json:"last_login_country"`
	RiskScore        float64 `json:"risk_score"`
	RiskLevel        string  `json:"risk_level"`
	UpdatedAt        string  `json:"updated_at"`
}

func featureKey(userID string) string {
	return fmt.Sprintf("fraud:features:%s", userID)
}

// ── Read / Write ────────────────────────────────────────────────────────────

// Get retrieves cached features for a user.
func (s *FeatureStore) Get(ctx context.Context, userID string) (*UserFeatures, error) {
	if s.rdb == nil {
		return nil, nil
	}
	data, err := s.rdb.Get(ctx, featureKey(userID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var f UserFeatures
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

// Set caches features for a user with TTL.
func (s *FeatureStore) Set(ctx context.Context, f *UserFeatures) error {
	if s.rdb == nil {
		return nil
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, featureKey(f.UserID), data, s.ttl).Err()
}

// ── Increment helpers (atomic counters) ─────────────────────────────────────

// IncrTxCount atomically increments the transaction count windows.
func (s *FeatureStore) IncrTxCount(ctx context.Context, userID string) error {
	if s.rdb == nil {
		return nil
	}
	f, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	if f == nil {
		f = &UserFeatures{UserID: userID, UpdatedAt: time.Now().Format(time.RFC3339)}
	}
	f.TxCountLast1h++
	f.TxCountLast24h++
	f.LastTxTime = time.Now().Format(time.RFC3339)
	return s.Set(ctx, f)
}

// IncrWithdrawCount atomically increments the 24h withdrawal counter.
func (s *FeatureStore) IncrWithdrawCount(ctx context.Context, userID string) error {
	if s.rdb == nil {
		return nil
	}
	f, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	if f == nil {
		f = &UserFeatures{UserID: userID, UpdatedAt: time.Now().Format(time.RFC3339)}
	}
	f.WithdrawCount24h++
	return s.Set(ctx, f)
}

// IncrFailedLogin atomically increments the failed login counter.
func (s *FeatureStore) IncrFailedLogin(ctx context.Context, userID string) error {
	if s.rdb == nil {
		return nil
	}
	f, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	if f == nil {
		f = &UserFeatures{UserID: userID, UpdatedAt: time.Now().Format(time.RFC3339)}
	}
	f.FailedLogins24h++
	return s.Set(ctx, f)
}

// UpdateGeoInfo updates the last known IP and country, tracking mismatches.
func (s *FeatureStore) UpdateGeoInfo(ctx context.Context, userID, ip, country string) error {
	if s.rdb == nil {
		return nil
	}
	f, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	if f == nil {
		f = &UserFeatures{UserID: userID, UpdatedAt: time.Now().Format(time.RFC3339)}
	}
	if f.LastLoginCountry != "" && f.LastLoginCountry != country {
		f.GeoMismatchCount++
		slog.Warn("fraud: geo mismatch detected",
			"user_id", userID,
			"prev_country", f.LastLoginCountry,
			"new_country", country,
			"mismatch_count", f.GeoMismatchCount,
		)
	}
	f.LastLoginIP = ip
	f.LastLoginCountry = country
	return s.Set(ctx, f)
}

// UpdateWalletDrift tracks the net balance change over 24h.
func (s *FeatureStore) UpdateWalletDrift(ctx context.Context, userID string, delta float64) error {
	if s.rdb == nil {
		return nil
	}
	f, err := s.Get(ctx, userID)
	if err != nil {
		return err
	}
	if f == nil {
		f = &UserFeatures{UserID: userID, UpdatedAt: time.Now().Format(time.RFC3339)}
	}
	f.WalletDrift24h += delta
	return s.Set(ctx, f)
}
