package listings

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Request / Response types
// ════════════════════════════════════════════════════════════════════════════

type SearchRequest struct {
	Query        string   `form:"q"`
	CategoryID   string   `form:"category_id"`
	CategorySlug string   `form:"category"`
	CategoryPath string   `form:"category_path"` // includes descendants (prefix match)
	MinPrice     *float64 `form:"min_price"`
	MaxPrice     *float64 `form:"max_price"`
	Condition    string   `form:"condition"` // new | used | refurbished
	Type         string   `form:"type"`      // sell | buy | rent | auction | service
	Lat          *float64 `form:"lat"`
	Lng          *float64 `form:"lng"`
	Radius       int      `form:"radius"` // km, default 50
	City         string   `form:"city"`
	Country      string   `form:"country"`
	SortBy       string   `form:"sort_by"` // relevance | price_asc | price_desc | date | distance
	Page         int      `form:"page"`
	PerPage      int      `form:"per_page"`
}

type FacetCategory struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type FacetCondition struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

type PriceRange struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Label string  `json:"label"`
	Count int64   `json:"count"`
}

type SearchFacets struct {
	Categories  []FacetCategory  `json:"categories"`
	Conditions  []FacetCondition `json:"conditions"`
	PriceRanges []PriceRange     `json:"price_ranges"`
}

type SearchResponse struct {
	Results     []Listing    `json:"results"`
	Total       int64        `json:"total"`
	Page        int          `json:"page"`
	PerPage     int          `json:"per_page"`
	Pages       int64        `json:"pages"`
	Facets      SearchFacets `json:"facets"`
	LiveResults []LiveResult `json:"live_results,omitempty"` // Sprint 18: live-session discovery
	Cached      bool         `json:"cached,omitempty"`
}

// ════════════════════════════════════════════════════════════════════════════
// GET /api/v1/listings/search
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) Search(c *gin.Context) {
	if !searchFlagEnabled() {
		response.OK(c, SearchResponse{Results: []Listing{}, Total: 0, Page: 1, PerPage: 0})
		return
	}
	var req SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 {
		req.PerPage = 20
	}
	if req.PerPage > 100 {
		req.PerPage = 100
	}
	if req.Radius < 1 {
		req.Radius = 50
	}
	if req.SortBy == "" {
		req.SortBy = "date"
	}

	// ── Redis cache check ─────────────────────────────────────────────────────
	cacheKey := searchCacheKey(req)
	if h.rdb != nil {
		if cached, err := h.rdb.Get(context.Background(), cacheKey).Bytes(); err == nil {
			var sr SearchResponse
			if json.Unmarshal(cached, &sr) == nil {
				sr.Cached = true
				response.OK(c, sr)
				return
			}
		}
	}

	// ── Build base query ──────────────────────────────────────────────────────
	q := h.db.Model(&Listing{}).
		Preload("Images").
		Preload("Category").
		Preload("Seller").
		Where("listings.status = ?", "active").
		Where("(listings.expires_at IS NULL OR listings.expires_at > NOW())")

	// ── Boost score subquery: active boosts add to ranking ────────────────────
	boostSubquery := `(SELECT COALESCE(SUM(lb.boost_score),0) FROM listing_boosts lb WHERE lb.listing_id = listings.id AND lb.expires_at > NOW())`
	// ── Reputation subquery: seller reputation adds to ranking ──────────────
	repSubquery := `(SELECT COALESCE(score,50) FROM user_reputations ur WHERE ur.user_id = listings.user_id AND ur.role='seller' LIMIT 1)`
	// ── Sprint 18.1: Live boost — seller has an active live session boosts all their listings.
	// Takes the max boost_score across concurrent live sessions of the same host.
	liveBoostSubquery := `(SELECT COALESCE(MAX(ls.boost_score), 0) FROM livestream_sessions ls
		WHERE ls.host_id = listings.user_id AND ls.status = 'live' AND ls.deleted_at IS NULL)`
	// ── Sprint 18.1: Urgency score — recent view velocity (last 24h) as a trend signal.
	// Caps contribution at 50 to prevent a single viral item from dominating.
	urgencySubquery := `(SELECT LEAST(COUNT(*), 50) FROM listing_views lv
		WHERE lv.listing_id = listings.id AND lv.viewed_at > NOW() - INTERVAL '24 hours')`
	q = q.Select(
		"listings.*, " +
			boostSubquery + " AS boost_score, " +
			repSubquery + " AS rep_score, " +
			liveBoostSubquery + " AS live_boost, " +
			urgencySubquery + " AS urgency_score",
	)

	// Full-text search using PostgreSQL tsvector
	if req.Query != "" {
		q = q.Where(
			"to_tsvector('english', listings.title || ' ' || COALESCE(listings.description, '')) @@ plainto_tsquery('english', ?)",
			req.Query,
		)
	}

	// Filters
	if req.CategoryID != "" {
		q = q.Where("listings.category_id = ?", req.CategoryID)
	} else if req.CategoryPath != "" {
		// Include descendants: any category whose path starts with req.CategoryPath
		q = q.Where(
			"listings.category_id IN (SELECT id FROM categories WHERE path = ? OR path LIKE ?)",
			req.CategoryPath, req.CategoryPath+"/%",
		)
	} else if req.CategorySlug != "" {
		q = q.Where("listings.category_id = (SELECT id FROM categories WHERE slug = ? LIMIT 1)", req.CategorySlug)
	}
	if req.MinPrice != nil {
		q = q.Where("listings.price >= ?", *req.MinPrice)
	}
	if req.MaxPrice != nil {
		q = q.Where("listings.price <= ?", *req.MaxPrice)
	}
	if req.Condition != "" {
		q = q.Where("listings.condition = ?", req.Condition)
	}
	if req.Type != "" {
		q = q.Where("listings.type = ?", req.Type)
	}
	if req.City != "" {
		q = q.Where("LOWER(listings.city) = LOWER(?)", req.City)
	}
	if req.Country != "" {
		q = q.Where("LOWER(listings.country) = LOWER(?)", req.Country)
	}

	// Geo filter — Haversine distance (no PostGIS required)
	if req.Lat != nil && req.Lng != nil {
		haversineWhere := `(
                        6371 * acos(
                                cos(radians(?)) * cos(radians(listings.latitude)) *
                                cos(radians(listings.longitude) - radians(?)) +
                                sin(radians(?)) * sin(radians(listings.latitude))
                        )
                ) <= ?`
		q = q.Where(
			"listings.latitude IS NOT NULL AND listings.longitude IS NOT NULL AND "+haversineWhere,
			*req.Lat, *req.Lng, *req.Lat, float64(req.Radius),
		)
	}

	// ── Count total ───────────────────────────────────────────────────────────
	var total int64
	q.Count(&total)

	// ── Sorting ───────────────────────────────────────────────────────────────
	// Sprint 18.1 — prefix every sort with boost_score + live_boost + rep_score + urgency_score
	// so monetization + reputation + real-time momentum always win at parity.
	const rankPrefix = "boost_score DESC, live_boost DESC, rep_score DESC, urgency_score DESC"
	switch req.SortBy {
	case "price_asc":
		q = q.Order(rankPrefix + ", listings.price ASC NULLS LAST")
	case "price_desc":
		q = q.Order(rankPrefix + ", listings.price DESC NULLS LAST")
	case "relevance":
		if req.Query != "" {
			rankExpr := fmt.Sprintf(
				"ts_rank(to_tsvector('english', listings.title || ' ' || COALESCE(listings.description, '')), plainto_tsquery('english', '%s')) DESC",
				strings.ReplaceAll(req.Query, "'", "''"),
			)
			q = q.Order(rankPrefix + ", " + rankExpr)
		} else {
			q = q.Order(rankPrefix + ", listings.is_featured DESC, listings.created_at DESC")
		}
	case "distance":
		if req.Lat != nil && req.Lng != nil {
			distExpr := fmt.Sprintf(
				"6371 * acos(cos(radians(%f)) * cos(radians(listings.latitude)) * cos(radians(listings.longitude) - radians(%f)) + sin(radians(%f)) * sin(radians(listings.latitude)))",
				*req.Lat, *req.Lng, *req.Lat,
			)
			q = q.Order(rankPrefix + ", " + distExpr + " ASC NULLS LAST")
		} else {
			q = q.Order(rankPrefix + ", listings.created_at DESC")
		}
	default: // "date"
		q = q.Order(rankPrefix + ", listings.is_featured DESC, listings.created_at DESC")
	}

	// ── Paginate + fetch ──────────────────────────────────────────────────────
	offset := (req.Page - 1) * req.PerPage
	var results []Listing
	q.Offset(offset).Limit(req.PerPage).Find(&results)

	// ── Facets ────────────────────────────────────────────────────────────────
	facets := h.buildFacets(req)

	// ── Store recent search in Redis ──────────────────────────────────────────
	if req.Query != "" && h.rdb != nil {
		userID := c.GetString("user_id")
		if userID != "" {
			go h.storeRecentSearch(userID, req.Query)
		}
	}

	// ── Build response + cache ────────────────────────────────────────────────
	pages := (total + int64(req.PerPage) - 1) / int64(req.PerPage)
	sr := SearchResponse{
		Results: results,
		Total:   total,
		Page:    req.Page,
		PerPage: req.PerPage,
		Pages:   pages,
		Facets:  facets,
	}

	// Sprint 18 — inject matching live sessions at the top of results (non-blocking if fails).
	if req.Page == 1 {
		if live := h.findMatchingLiveSessions(c.Request.Context(), req.Query, 5); len(live) > 0 {
			sr.LiveResults = live
		}
	}

	if h.rdb != nil {
		if data, err := json.Marshal(sr); err == nil {
			h.rdb.Set(context.Background(), cacheKey, data, 2*time.Minute)
		}
	}

	response.OK(c, sr)
}

// ════════════════════════════════════════════════════════════════════════════
// GET /api/v1/listings/suggestions?q=iphone — autocomplete
// ════════════════════════════════════════════════════════════════════════════

// SuggestCategory is a slim category summary for the autocomplete dropdown.
type SuggestCategory struct {
	ID     string `json:"id"`
	Slug   string `json:"slug"`
	NameEn string `json:"name_en"`
	NameAr string `json:"name_ar,omitempty"`
	Icon   string `json:"icon,omitempty"`
}

// SuggestTrending is a popular recent query.
type SuggestTrending struct {
	Query string `json:"query"`
	Count int    `json:"count"`
}

// SuggestResponse is the Amazon-style autocomplete bundle.
type SuggestResponse struct {
	Listings   []string          `json:"listings"`
	Categories []SuggestCategory `json:"categories"`
	Live       []LiveResult      `json:"live"`
	Trending   []SuggestTrending `json:"trending"`
}

func (h *Handler) Suggestions(c *gin.Context) {
	if !autocompleteFlagEnabled() {
		response.OK(c, SuggestResponse{})
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	if len(q) < 2 {
		response.OK(c, SuggestResponse{})
		return
	}

	cacheKey := "suggest:v2:" + strings.ToLower(q)
	if h.rdb != nil {
		if cached, err := h.rdb.Get(context.Background(), cacheKey).Bytes(); err == nil {
			var out SuggestResponse
			if json.Unmarshal(cached, &out) == nil {
				response.OK(c, out)
				return
			}
		}
	}

	like := "%" + q + "%"
	out := SuggestResponse{
		Listings:   []string{},
		Categories: []SuggestCategory{},
		Live:       []LiveResult{},
		Trending:   []SuggestTrending{},
	}

	// Listings (by title, top-viewed first)
	h.db.Model(&Listing{}).
		Where("status = ? AND title ILIKE ?", "active", like).
		Order("view_count DESC, created_at DESC").
		Limit(10).
		Pluck("DISTINCT title", &out.Listings)

	// Categories (ILIKE on name_en / name_ar / slug)
	var cats []SuggestCategory
	h.db.Table("categories").
		Select("id, slug, name_en, name_ar, icon").
		Where("is_active = true AND (name_en ILIKE ? OR name_ar ILIKE ? OR slug ILIKE ?)", like, like, like).
		Order("sort_order ASC, name_en ASC").
		Limit(5).
		Scan(&cats)
	out.Categories = cats

	// Live sessions (short-timeout, non-blocking)
	if live := h.findMatchingLiveSessions(c.Request.Context(), q, 3); live != nil {
		out.Live = live
	}

	// Trending from search_queries (last 7d), prefix match
	var trending []SuggestTrending
	h.db.Raw(`
		SELECT query, COUNT(*) as count
		FROM search_queries
		WHERE created_at > NOW() - INTERVAL '7 days'
		  AND query ILIKE ?
		GROUP BY query
		ORDER BY count DESC
		LIMIT 5`, like).Scan(&trending)
	out.Trending = trending

	if h.rdb != nil {
		if data, _ := json.Marshal(out); data != nil {
			h.rdb.Set(context.Background(), cacheKey, data, 5*time.Minute)
		}
	}

	response.OK(c, out)
}

// ════════════════════════════════════════════════════════════════════════════
// GET /api/v1/listings/recent-searches — user's recent searches
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) RecentSearches(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" || h.rdb == nil {
		response.OK(c, gin.H{"searches": []string{}})
		return
	}
	key := "user:" + userID + ":recent_searches"
	searches, err := h.rdb.LRange(context.Background(), key, 0, 19).Result()
	if err != nil {
		searches = []string{}
	}
	response.OK(c, gin.H{"searches": searches})
}

// ════════════════════════════════════════════════════════════════════════════
// Internal helpers
// ════════════════════════════════════════════════════════════════════════════

// buildFacets runs lightweight count queries to return category/condition/price facets.
func (h *Handler) buildFacets(req SearchRequest) SearchFacets {
	base := h.db.Model(&Listing{}).Where("status = ?", "active")

	// Narrow by same text query if present
	if req.Query != "" {
		base = base.Where(
			"to_tsvector('english', title || ' ' || COALESCE(description, '')) @@ plainto_tsquery('english', ?)",
			req.Query,
		)
	}

	// ── Category facets ────────────────────────────────────────────────────────
	type catRow struct {
		CategoryID string
		NameEn     string
		Cnt        int64
	}
	var catRows []catRow
	base.Select("listings.category_id, categories.name_en, COUNT(*) as cnt").
		Joins("JOIN categories ON categories.id = listings.category_id").
		Group("listings.category_id, categories.name_en").
		Order("cnt DESC").
		Limit(10).
		Scan(&catRows)

	categories := make([]FacetCategory, 0, len(catRows))
	for _, r := range catRows {
		categories = append(categories, FacetCategory{
			ID:    r.CategoryID,
			Name:  r.NameEn,
			Count: r.Cnt,
		})
	}

	// ── Condition facets ──────────────────────────────────────────────────────
	type condRow struct {
		Condition string
		Cnt       int64
	}
	var condRows []condRow
	base.Select("condition, COUNT(*) as cnt").
		Where("condition != ''").
		Group("condition").
		Order("cnt DESC").
		Scan(&condRows)

	conditions := make([]FacetCondition, 0, len(condRows))
	for _, r := range condRows {
		conditions = append(conditions, FacetCondition{Value: r.Condition, Count: r.Cnt})
	}

	// ── Price range facets ────────────────────────────────────────────────────
	priceRanges := []PriceRange{
		{Min: 0, Max: 100, Label: "Under AED 100"},
		{Min: 100, Max: 500, Label: "AED 100 – 500"},
		{Min: 500, Max: 1000, Label: "AED 500 – 1000"},
		{Min: 1000, Max: 5000, Label: "AED 1,000 – 5,000"},
		{Min: 5000, Max: 20000, Label: "AED 5,000 – 20,000"},
		{Min: 20000, Max: 1e9, Label: "Over AED 20,000"},
	}
	for i, pr := range priceRanges {
		var cnt int64
		base.Where("price >= ? AND price < ?", pr.Min, pr.Max).Count(&cnt)
		priceRanges[i].Count = cnt
	}

	return SearchFacets{
		Categories:  categories,
		Conditions:  conditions,
		PriceRanges: priceRanges,
	}
}

// storeRecentSearch prepends a query to the user's Redis recent-searches list.
func (h *Handler) storeRecentSearch(userID, query string) {
	key := "user:" + userID + ":recent_searches"
	ctx := context.Background()
	h.rdb.LPush(ctx, key, query)
	h.rdb.LTrim(ctx, key, 0, 19)            // keep last 20
	h.rdb.Expire(ctx, key, 30*24*time.Hour) // 30-day TTL
}

// searchCacheKey builds a deterministic Redis key from the search request.
func searchCacheKey(req SearchRequest) string {
	raw := fmt.Sprintf("%v", req)
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("search:%x", h[:8])
}

// ApplySearchIndexes creates PostgreSQL indexes needed for fast search.
// Called once during application startup (idempotent).
func ApplySearchIndexes(db *gorm.DB) {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_listings_search
                        ON listings USING GIN(to_tsvector('english', title || ' ' || COALESCE(description, '')))`,
		`CREATE INDEX IF NOT EXISTS idx_listings_price    ON listings(price)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_created  ON listings(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_status   ON listings(status)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_category ON listings(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_geo      ON listings(latitude, longitude)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_city     ON listings(city)`,
		`CREATE INDEX IF NOT EXISTS idx_listings_country  ON listings(country)`,
		// Sprint 18.1 — support ranking subqueries
		`CREATE INDEX IF NOT EXISTS idx_livesessions_host_status ON livestream_sessions(host_id, status) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_listingviews_listing_viewed ON listing_views(listing_id, viewed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_categories_path ON categories(path)`,
	}
	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			slog.Warn("search index creation skipped", "error", err.Error())
		}
	}
	slog.Info("✅ Search indexes ready")
}
