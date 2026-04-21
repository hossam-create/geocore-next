package recommendations

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const cacheTTL = 5 * time.Minute

// Engine provides context-aware product recommendations.
// Uses a rule-based hybrid approach: personalized + trending + similar + recently viewed.
// ML/LLM layer can be plugged in via the Scorer interface later.
type Engine struct {
	db  *gorm.DB
	rdb *redis.Client
}

// NewEngine creates a recommendation engine.
func NewEngine(db *gorm.DB, rdb *redis.Client) *Engine {
	return &Engine{db: db, rdb: rdb}
}

// GetRecommendations returns recommendations based on context.
func (e *Engine) GetRecommendations(ctx context.Context, userID uuid.UUID, rctx Context, referenceID *uuid.UUID, limit int) Response {
	start := time.Now()
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	// Try cache
	cacheKey := fmt.Sprintf("recs:%s:%s:%v", userID, rctx, referenceID)
	if e.rdb != nil {
		if cached, err := e.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var resp Response
			if json.Unmarshal([]byte(cached), &resp) == nil {
				now := time.Now()
				resp.CachedAt = &now
				resp.DurationMs = time.Since(start).Milliseconds()
				return resp
			}
		}
	}

	var items []Recommendation

	switch rctx {
	case ContextHomePage:
		items = e.homePageRecs(ctx, userID, limit)
	case ContextProductPage:
		if referenceID != nil {
			items = e.productPageRecs(ctx, userID, *referenceID, limit)
		} else {
			items = e.trendingRecs(ctx, limit)
		}
	case ContextCartPage, ContextCheckout:
		items = e.cartRecs(ctx, userID, limit)
	case ContextSearch, ContextCategory:
		items = e.trendingRecs(ctx, limit)
	default:
		items = e.trendingRecs(ctx, limit)
	}

	resp := Response{
		Items:      items,
		Context:    rctx,
		Count:      len(items),
		DurationMs: time.Since(start).Milliseconds(),
	}

	// Cache result
	if e.rdb != nil {
		if b, err := json.Marshal(resp); err == nil {
			e.rdb.Set(ctx, cacheKey, b, cacheTTL)
		}
	}

	return resp
}

// homePageRecs: 40% personalized + 30% trending + 20% recently viewed + 10% new
func (e *Engine) homePageRecs(ctx context.Context, userID uuid.UUID, limit int) []Recommendation {
	var results []Recommendation

	personalized := e.personalizedRecs(ctx, userID, max(1, limit*4/10))
	results = append(results, personalized...)

	trending := e.trendingRecs(ctx, max(1, limit*3/10))
	results = append(results, trending...)

	recent := e.recentlyViewedRecs(ctx, userID, max(1, limit*2/10))
	results = append(results, recent...)

	newArrivals := e.newArrivalRecs(ctx, max(1, limit*1/10))
	results = append(results, newArrivals...)

	return dedup(results, limit)
}

// productPageRecs: 50% similar + 30% frequently bought together + 20% personalized
func (e *Engine) productPageRecs(ctx context.Context, userID, listingID uuid.UUID, limit int) []Recommendation {
	var results []Recommendation

	similar := e.similarItemRecs(ctx, listingID, max(1, limit*5/10))
	results = append(results, similar...)

	fbt := e.frequentlyBoughtTogether(ctx, listingID, max(1, limit*3/10))
	results = append(results, fbt...)

	personalized := e.personalizedRecs(ctx, userID, max(1, limit*2/10))
	results = append(results, personalized...)

	return dedup(results, limit)
}

// cartRecs: complementary items for items in cart
func (e *Engine) cartRecs(ctx context.Context, userID uuid.UUID, limit int) []Recommendation {
	// Get user's recent purchases/views for complementary items
	return e.trendingRecs(ctx, limit)
}

// ── Core Algorithms ────────────────────────────────────────────────────────

func (e *Engine) personalizedRecs(ctx context.Context, userID uuid.UUID, limit int) []Recommendation {
	// Get categories user interacted with most
	type catCount struct {
		CategoryID uuid.UUID
		Count      int64
	}
	var cats []catCount
	e.db.WithContext(ctx).
		Table("recommendation_interactions").
		Select("category_id, COUNT(*) as count").
		Where("user_id = ? AND category_id IS NOT NULL", userID).
		Group("category_id").
		Order("count DESC").
		Limit(3).
		Scan(&cats)

	if len(cats) == 0 {
		return e.trendingRecs(ctx, limit)
	}

	catIDs := make([]uuid.UUID, len(cats))
	for i, c := range cats {
		catIDs[i] = c.CategoryID
	}

	type listing struct {
		ID uuid.UUID
	}
	var listings []listing
	e.db.WithContext(ctx).
		Table("listings").
		Select("id").
		Where("category_id IN ? AND status = 'active'", catIDs).
		Order("created_at DESC").
		Limit(limit).
		Scan(&listings)

	results := make([]Recommendation, 0, len(listings))
	for i, l := range listings {
		results = append(results, Recommendation{
			ListingID: l.ID,
			Score:     0.9 - float64(i)*0.03,
			Reason:    "Based on your interests",
			Algorithm: AlgoPersonalized,
		})
	}
	return results
}

func (e *Engine) trendingRecs(ctx context.Context, limit int) []Recommendation {
	type listing struct {
		ID uuid.UUID
	}
	var listings []listing
	e.db.WithContext(ctx).
		Table("listings").
		Select("listings.id").
		Joins("LEFT JOIN recommendation_interactions ri ON ri.listing_id = listings.id AND ri.created_at > ?", time.Now().Add(-7*24*time.Hour)).
		Where("listings.status = 'active'").
		Group("listings.id").
		Order("COUNT(ri.id) DESC").
		Limit(limit).
		Scan(&listings)

	results := make([]Recommendation, 0, len(listings))
	for i, l := range listings {
		results = append(results, Recommendation{
			ListingID: l.ID,
			Score:     0.8 - float64(i)*0.03,
			Reason:    "Trending now",
			Algorithm: AlgoTrending,
		})
	}
	return results
}

func (e *Engine) similarItemRecs(ctx context.Context, listingID uuid.UUID, limit int) []Recommendation {
	// Find listings in the same category
	type listing struct {
		ID uuid.UUID
	}

	var catID uuid.UUID
	e.db.WithContext(ctx).Table("listings").Select("category_id").Where("id = ?", listingID).Scan(&catID)

	var listings []listing
	e.db.WithContext(ctx).
		Table("listings").
		Select("id").
		Where("category_id = ? AND id != ? AND status = 'active'", catID, listingID).
		Order("created_at DESC").
		Limit(limit).
		Scan(&listings)

	results := make([]Recommendation, 0, len(listings))
	for i, l := range listings {
		results = append(results, Recommendation{
			ListingID: l.ID,
			Score:     0.95 - float64(i)*0.03,
			Reason:    "Similar to what you're viewing",
			Algorithm: AlgoSimilarItems,
		})
	}
	return results
}

func (e *Engine) frequentlyBoughtTogether(ctx context.Context, listingID uuid.UUID, limit int) []Recommendation {
	// Find listings purchased by users who also purchased this listing
	type listing struct {
		ID uuid.UUID
	}
	var listings []listing
	e.db.WithContext(ctx).Raw(`
		SELECT DISTINCT oi2.listing_id AS id
		FROM order_items oi1
		JOIN order_items oi2 ON oi1.order_id = oi2.order_id AND oi2.listing_id != oi1.listing_id
		WHERE oi1.listing_id = ?
		LIMIT ?
	`, listingID, limit).Scan(&listings)

	results := make([]Recommendation, 0, len(listings))
	for i, l := range listings {
		results = append(results, Recommendation{
			ListingID: l.ID,
			Score:     0.85 - float64(i)*0.05,
			Reason:    "Frequently bought together",
			Algorithm: AlgoFrequentlyBought,
		})
	}
	return results
}

func (e *Engine) recentlyViewedRecs(ctx context.Context, userID uuid.UUID, limit int) []Recommendation {
	type listing struct {
		ListingID uuid.UUID
	}
	var listings []listing
	e.db.WithContext(ctx).
		Table("recommendation_interactions").
		Select("DISTINCT listing_id").
		Where("user_id = ? AND interaction_type = 'view'", userID).
		Order("created_at DESC").
		Limit(limit).
		Scan(&listings)

	results := make([]Recommendation, 0, len(listings))
	for i, l := range listings {
		results = append(results, Recommendation{
			ListingID: l.ListingID,
			Score:     0.7 - float64(i)*0.05,
			Reason:    "Recently viewed",
			Algorithm: AlgoRecentlyViewed,
		})
	}
	return results
}

func (e *Engine) newArrivalRecs(ctx context.Context, limit int) []Recommendation {
	type listing struct {
		ID uuid.UUID
	}
	var listings []listing
	e.db.WithContext(ctx).
		Table("listings").
		Select("id").
		Where("status = 'active' AND created_at > ?", time.Now().Add(-7*24*time.Hour)).
		Order("created_at DESC").
		Limit(limit).
		Scan(&listings)

	results := make([]Recommendation, 0, len(listings))
	for i, l := range listings {
		results = append(results, Recommendation{
			ListingID: l.ID,
			Score:     0.75 - float64(i)*0.05,
			Reason:    "New arrival",
			Algorithm: AlgoNewArrivals,
		})
	}
	return results
}

// TrackInteraction records a user interaction for future recommendations.
func (e *Engine) TrackInteraction(userID, listingID uuid.UUID, interactionType string, categoryID *uuid.UUID, sessionID string, dwellTimeMs int) {
	interaction := UserInteraction{
		UserID:          userID,
		ListingID:       listingID,
		InteractionType: interactionType,
		CategoryID:      categoryID,
		SessionID:       sessionID,
		DwellTimeMs:     dwellTimeMs,
		CreatedAt:       time.Now(),
	}
	if err := e.db.Create(&interaction).Error; err != nil {
		slog.Warn("recommendations: failed to track interaction", "error", err)
	}

	// Invalidate cache for this user
	if e.rdb != nil {
		pattern := fmt.Sprintf("recs:%s:*", userID)
		if keys, err := e.rdb.Keys(context.Background(), pattern).Result(); err == nil && len(keys) > 0 {
			e.rdb.Del(context.Background(), keys...)
		}
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

func dedup(items []Recommendation, limit int) []Recommendation {
	seen := make(map[uuid.UUID]bool)
	var unique []Recommendation
	for _, item := range items {
		if !seen[item.ListingID] {
			seen[item.ListingID] = true
			unique = append(unique, item)
		}
	}
	if len(unique) > limit {
		unique = unique[:limit]
	}
	return unique
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
