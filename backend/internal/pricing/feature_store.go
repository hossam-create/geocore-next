package pricing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Feature Store ─────────────────────────────────────────────────────────────────
//
// Centralized feature storage: Redis (online, low-latency) + Postgres (offline, source of truth).
// Every service reads features from ONE place → consistency + speed.
//
// Architecture:
//
//	Postgres (source of truth) → Feature Sync → Redis (online store) → RL / Ranking / Search

// ── Feature Models ────────────────────────────────────────────────────────────────

// UserFeatures stores precomputed user features for fast access.
type UserFeatures struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	TrustScore       float64   `gorm:"type:numeric(5,2);not null;default:50" json:"trust_score"`
	AvgOrderValue    float64   `gorm:"type:numeric(12,2);not null;default:0" json:"avg_order_value"`
	CancelRate       float64   `gorm:"type:numeric(5,4);not null;default:0" json:"cancel_rate"`
	InsuranceBuyRate float64   `gorm:"type:numeric(5,4);not null;default:0" json:"insurance_buy_rate"`
	TotalOrders      int64     `gorm:"not null;default:0" json:"total_orders"`
	TotalSpent       float64   `gorm:"type:numeric(12,2);not null;default:0" json:"total_spent"`
	AccountAgeDays   float64   `gorm:"type:numeric(8,1);not null;default:0" json:"account_age_days"`
	AbuseFlags       int       `gorm:"not null;default:0" json:"abuse_flags"`
	LastOrderAt      *time.Time `json:"last_order_at"`
	Segment          string    `gorm:"size:20;not null;default:'regular'" json:"segment"` // vip/regular/new/at_risk
	EmbeddingID      uuid.UUID `gorm:"type:uuid" json:"embedding_id"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (UserFeatures) TableName() string { return "feature_users" }

// ItemFeatures stores precomputed item/listing features.
type ItemFeatures struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ItemID          uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"item_id"`
	PriceCents      int64     `gorm:"not null;default:0" json:"price_cents"`
	CategoryPath    string    `gorm:"size:200;not null;default:''" json:"category_path"`
	ViewCount       int64     `gorm:"not null;default:0" json:"view_count"`
	PurchaseCount   int64     `gorm:"not null;default:0" json:"purchase_count"`
	AvgRating       float64   `gorm:"type:numeric(3,2);not null;default:0" json:"avg_rating"`
	ClaimRate       float64   `gorm:"type:numeric(5,4);not null;default:0" json:"claim_rate"`
	DeliveryRisk    float64   `gorm:"type:numeric(5,4);not null;default:0" json:"delivery_risk"`
	PopularityScore float64   `gorm:"type:numeric(8,4);not null;default:0" json:"popularity_score"`
	IsTrending      bool      `gorm:"not null;default:false" json:"is_trending"`
	EmbeddingID     uuid.UUID `gorm:"type:uuid" json:"embedding_id"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (ItemFeatures) TableName() string { return "feature_items" }

// SessionFeatures stores real-time session context.
type SessionFeatures struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	SessionID      string    `gorm:"size:50;not null;uniqueIndex" json:"session_id"`
	Device         string    `gorm:"size:20;not null;default:'desktop'" json:"device"`
	Geo            string    `gorm:"size:5;not null;default:''" json:"geo"`
	SessionStep    int       `gorm:"not null;default:0" json:"session_step"`
	RefusalCount   int       `gorm:"not null;default:0" json:"refusal_count"`
	ItemsViewed    int       `gorm:"not null;default:0" json:"items_viewed"`
	ItemsClicked   int       `gorm:"not null;default:0" json:"items_clicked"`
	DemandScore    float64   `gorm:"type:numeric(5,4);not null;default:0" json:"demand_score"`
	UrgencyScore   float64   `gorm:"type:numeric(5,4);not null;default:0" json:"urgency_score"`
	LastActionAt   *time.Time `json:"last_action_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (SessionFeatures) TableName() string { return "feature_sessions" }

// ── Feature Store Interface ────────────────────────────────────────────────────────

type FeatureStore interface {
	GetUserFeatures(userID uuid.UUID) (*UserFeatures, error)
	GetItemFeatures(itemID uuid.UUID) (*ItemFeatures, error)
	GetSessionFeatures(sessionID string) (*SessionFeatures, error)
	BatchGetUserFeatures(userIDs []uuid.UUID) (map[uuid.UUID]*UserFeatures, error)
	RefreshUserFeatures(userID uuid.UUID) error
	RefreshItemFeatures(itemID uuid.UUID) error
}

// ── Redis-backed Feature Store Implementation ──────────────────────────────────────

type RedisFeatureStore struct {
	db  *gorm.DB
	rdb *redis.Client
	ttl time.Duration
}

func NewFeatureStore(db *gorm.DB, rdb *redis.Client) *RedisFeatureStore {
	return &RedisFeatureStore{
		db:  db,
		rdb: rdb,
		ttl: 15 * time.Minute, // Redis TTL: 15 min
	}
}

// GetUserFeatures fetches user features: Redis first, then Postgres.
func (s *RedisFeatureStore) GetUserFeatures(userID uuid.UUID) (*UserFeatures, error) {
	ctx := context.Background()
	key := fmt.Sprintf("feat:user:%s", userID)

	// Try Redis
	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		var features UserFeatures
		if json.Unmarshal([]byte(val), &features) == nil {
			return &features, nil
		}
	}

	// Fallback to Postgres
	var features UserFeatures
	if err := s.db.Where("user_id = ?", userID).First(&features).Error; err != nil {
		return nil, err
	}

	// Write-through to Redis
	data, _ := json.Marshal(features)
	s.rdb.Set(ctx, key, data, s.ttl)

	return &features, nil
}

// GetItemFeatures fetches item features: Redis first, then Postgres.
func (s *RedisFeatureStore) GetItemFeatures(itemID uuid.UUID) (*ItemFeatures, error) {
	ctx := context.Background()
	key := fmt.Sprintf("feat:item:%s", itemID)

	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		var features ItemFeatures
		if json.Unmarshal([]byte(val), &features) == nil {
			return &features, nil
		}
	}

	var features ItemFeatures
	if err := s.db.Where("item_id = ?", itemID).First(&features).Error; err != nil {
		return nil, err
	}

	data, _ := json.Marshal(features)
	s.rdb.Set(ctx, key, data, s.ttl)

	return &features, nil
}

// GetSessionFeatures fetches session features: Redis first, then Postgres.
func (s *RedisFeatureStore) GetSessionFeatures(sessionID string) (*SessionFeatures, error) {
	ctx := context.Background()
	key := fmt.Sprintf("feat:session:%s", sessionID)

	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		var features SessionFeatures
		if json.Unmarshal([]byte(val), &features) == nil {
			return &features, nil
		}
	}

	var features SessionFeatures
	if err := s.db.Where("session_id = ?", sessionID).First(&features).Error; err != nil {
		return nil, err
	}

	data, _ := json.Marshal(features)
	s.rdb.Set(ctx, key, data, s.ttl)

	return &features, nil
}

// BatchGetUserFeatures fetches multiple user features in one Redis MGET + DB fallback.
func (s *RedisFeatureStore) BatchGetUserFeatures(userIDs []uuid.UUID) (map[uuid.UUID]*UserFeatures, error) {
	ctx := context.Background()
	result := make(map[uuid.UUID]*UserFeatures, len(userIDs))

	// Build Redis keys
	keys := make([]string, len(userIDs))
	for i, id := range userIDs {
		keys[i] = fmt.Sprintf("feat:user:%s", id)
	}

	// MGET
	vals, err := s.rdb.MGet(ctx, keys...).Result()
	if err == nil {
		var missingIDs []uuid.UUID
		for i, val := range vals {
			if val == nil {
				missingIDs = append(missingIDs, userIDs[i])
				continue
			}
			var features UserFeatures
			strVal, ok := val.(string)
			if ok && json.Unmarshal([]byte(strVal), &features) == nil {
				result[userIDs[i]] = &features
			} else {
				missingIDs = append(missingIDs, userIDs[i])
			}
		}

		// Fetch missing from DB
		if len(missingIDs) > 0 {
			var dbFeatures []UserFeatures
			s.db.Where("user_id IN ?", missingIDs).Find(&dbFeatures)
			for i := range dbFeatures {
				f := &dbFeatures[i]
				result[f.UserID] = f
				// Write-through
				data, _ := json.Marshal(f)
				s.rdb.Set(ctx, fmt.Sprintf("feat:user:%s", f.UserID), data, s.ttl)
			}
		}
	}

	return result, nil
}

// RefreshUserFeatures recomputes user features from raw data and updates both stores.
func (s *RedisFeatureStore) RefreshUserFeatures(userID uuid.UUID) error {
	// Compute features from raw order/user data
	var totalOrders int64
	var totalSpent float64
	var cancelCount int64

	s.db.Table("orders").Where("buyer_id = ?", userID).Count(&totalOrders)
	s.db.Table("orders").Where("buyer_id = ? AND status = ?", userID, "cancelled").Count(&cancelCount)
	s.db.Raw("SELECT COALESCE(SUM(total_cents), 0) FROM orders WHERE buyer_id = ?", userID).Scan(&totalSpent)

	cancelRate := 0.0
	if totalOrders > 0 {
		cancelRate = float64(cancelCount) / float64(totalOrders)
	}

	// Upsert
	var features UserFeatures
	if err := s.db.Where("user_id = ?", userID).First(&features).Error; err != nil {
		features = UserFeatures{UserID: userID}
	}

	features.TotalOrders = totalOrders
	features.TotalSpent = totalSpent / 100.0
	features.CancelRate = cancelRate
	features.AvgOrderValue = 0
	if totalOrders > 0 {
		features.AvgOrderValue = features.TotalSpent / float64(totalOrders)
	}
	features.Segment = classifyUserSegment(features.TrustScore, features.InsuranceBuyRate)
	features.UpdatedAt = time.Now()

	s.db.Save(&features)

	// Invalidate Redis cache
	ctx := context.Background()
	s.rdb.Del(ctx, fmt.Sprintf("feat:user:%s", userID))

	return nil
}

// RefreshItemFeatures recomputes item features from raw data.
func (s *RedisFeatureStore) RefreshItemFeatures(itemID uuid.UUID) error {
	var viewCount, purchaseCount int64
	s.db.Table("listing_views").Where("listing_id = ?", itemID).Count(&viewCount)
	s.db.Table("order_items").Where("listing_id = ?", itemID).Count(&purchaseCount)

	popularity := 0.0
	if viewCount > 0 {
		popularity = float64(purchaseCount) / float64(viewCount) * 100
	}

	var features ItemFeatures
	if err := s.db.Where("item_id = ?", itemID).First(&features).Error; err != nil {
		features = ItemFeatures{ItemID: itemID}
	}

	features.ViewCount = viewCount
	features.PurchaseCount = purchaseCount
	features.PopularityScore = popularity
	features.IsTrending = popularity > 5.0
	features.UpdatedAt = time.Now()

	s.db.Save(&features)

	ctx := context.Background()
	s.rdb.Del(ctx, fmt.Sprintf("feat:item:%s", itemID))

	return nil
}
