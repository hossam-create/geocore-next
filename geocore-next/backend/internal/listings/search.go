package listings

import (
        "context"
        "crypto/sha256"
        "encoding/json"
        "fmt"
        "log/slog"
        "math"
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
        MinPrice     *float64 `form:"min_price"`
        MaxPrice     *float64 `form:"max_price"`
        Condition    string   `form:"condition"` // new | like-new | good | fair | for-parts
        Type         string   `form:"type"`      // sell | buy | rent | auction | service
        Status       string   `form:"status"`    // active | sold | reserved | expired | draft
        Lat          *float64 `form:"lat"`
        Lng          *float64 `form:"lng"`
        Radius       int      `form:"radius"` // km, default 50
        City         string   `form:"city"`
        Country      string   `form:"country"`
        SortBy       string   `form:"sort_by"` // relevance | price_asc | price_desc | date | distance | most_bids
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
        Results []Listing    `json:"results"`
        Total   int64        `json:"total"`
        Page    int          `json:"page"`
        PerPage int          `json:"per_page"`
        Pages   int64        `json:"pages"`
        Facets  SearchFacets `json:"facets"`
        Cached  bool         `json:"cached,omitempty"`
}

// ════════════════════════════════════════════════════════════════════════════
// GET /api/v1/listings/search
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) Search(c *gin.Context) {
        var req SearchRequest
        if err := c.ShouldBindQuery(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        // Validate filter values
        if err := req.Validate(); err != nil {
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

        // Default status to "active" when not explicitly requested
        if req.Status == "" {
                req.Status = "active"
        }

        // ── Build base query ──────────────────────────────────────────────────────
        q := h.db.Model(&Listing{}).
                Preload("Images").
                Preload("Category").
                Preload("Seller").
                Where("listings.status = ?", req.Status).
                Where("(listings.expires_at IS NULL OR listings.expires_at > NOW())")

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

        // Geo filter — bounding-box pre-filter (uses lat/lng index) followed by exact Haversine.
        // The bounding box eliminates the vast majority of rows before the Haversine is computed.
        if req.Lat != nil && req.Lng != nil {
                radius := float64(req.Radius)
                // 1 degree latitude ≈ 111 km; longitude degrees shrink with cos(lat).
                // Clamp lat to ±89° before computing lngDelta to avoid extreme values near the poles
                // (cos approaches 0 at ±90°, causing lngDelta → ∞).
                clampedLat := *req.Lat
                if clampedLat > 89.0 {
                        clampedLat = 89.0
                } else if clampedLat < -89.0 {
                        clampedLat = -89.0
                }
                latDelta := radius / 111.0
                lngDelta := radius / (111.0 * math.Cos(clampedLat * math.Pi / 180.0))

                // Bounding box — fast index scan
                q = q.Where(
                        "listings.latitude IS NOT NULL AND listings.longitude IS NOT NULL AND "+
                                "listings.latitude  BETWEEN ? AND ? AND "+
                                "listings.longitude BETWEEN ? AND ?",
                        *req.Lat-latDelta, *req.Lat+latDelta,
                        *req.Lng-lngDelta, *req.Lng+lngDelta,
                )

                // Exact Haversine on the already-reduced row set
                haversineWhere := `(
                        6371 * acos(
                                LEAST(1.0, cos(radians(?)) * cos(radians(listings.latitude)) *
                                cos(radians(listings.longitude) - radians(?)) +
                                sin(radians(?)) * sin(radians(listings.latitude)))
                        )
                ) <= ?`
                q = q.Where(haversineWhere, *req.Lat, *req.Lng, *req.Lat, radius)
        }

        // ── Count total ───────────────────────────────────────────────────────────
        var total int64
        q.Count(&total)

        // ── Sorting ───────────────────────────────────────────────────────────────
        switch req.SortBy {
        case "price_asc":
                q = q.Order("listings.price ASC NULLS LAST")
        case "price_desc":
                q = q.Order("listings.price DESC NULLS LAST")
        case "relevance":
                if req.Query != "" {
                        rankExpr := fmt.Sprintf(
                                "ts_rank(to_tsvector('english', listings.title || ' ' || COALESCE(listings.description, '')), plainto_tsquery('english', '%s')) DESC",
                                strings.ReplaceAll(req.Query, "'", "''"),
                        )
                        q = q.Order(rankExpr)
                } else {
                        q = q.Order("CASE WHEN listings.is_featured AND (listings.featured_until IS NULL OR listings.featured_until > NOW()) THEN 1 ELSE 0 END DESC, listings.created_at DESC")
                }
        case "distance":
                if req.Lat != nil && req.Lng != nil {
                        distExpr := fmt.Sprintf(
                                "6371 * acos(cos(radians(%f)) * cos(radians(listings.latitude)) * cos(radians(listings.longitude) - radians(%f)) + sin(radians(%f)) * sin(radians(listings.latitude)))",
                                *req.Lat, *req.Lng, *req.Lat,
                        )
                        q = q.Order(distExpr + " ASC NULLS LAST")
                } else {
                        q = q.Order("listings.created_at DESC")
                }
        case "most_bids":
                // Sort listings by auction bid count (descending). Non-auction listings
                // (no matching row in auctions) sort to the end with NULLS LAST.
                q = q.Joins("LEFT JOIN auctions ON auctions.listing_id = listings.id AND auctions.deleted_at IS NULL").
                        Order("auctions.bid_count DESC NULLS LAST, listings.created_at DESC")
        default: // "date"
                q = q.Order("CASE WHEN listings.is_featured AND (listings.featured_until IS NULL OR listings.featured_until > NOW()) THEN 1 ELSE 0 END DESC, listings.created_at DESC")
        }

        // ── Paginate + fetch ──────────────────────────────────────────────────────
        offset := (req.Page - 1) * req.PerPage
        results := make([]Listing, 0)
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

        if h.rdb != nil {
                if data, err := json.Marshal(sr); err == nil {
                        h.rdb.Set(context.Background(), cacheKey, data, 5*time.Minute)
                }
        }

        response.OK(c, sr)
}

// ════════════════════════════════════════════════════════════════════════════
// GET /api/v1/listings/suggestions?q=iphone — autocomplete
// ════════════════════════════════════════════════════════════════════════════

func (h *Handler) Suggestions(c *gin.Context) {
        q := strings.TrimSpace(c.Query("q"))
        if len(q) < 2 {
                response.OK(c, gin.H{"suggestions": []string{}})
                return
        }

        cacheKey := "suggest:" + q
        if h.rdb != nil {
                if cached, err := h.rdb.Get(context.Background(), cacheKey).Bytes(); err == nil {
                        var out []string
                        if json.Unmarshal(cached, &out) == nil {
                                response.OK(c, gin.H{"suggestions": out})
                                return
                        }
                }
        }

        titles := make([]string, 0)
        h.db.Model(&Listing{}).
                Where("status = ? AND title ILIKE ?", "active", "%"+q+"%").
                Order("view_count DESC, created_at DESC").
                Limit(10).
                Pluck("DISTINCT title", &titles)

        if h.rdb != nil {
                if data, _ := json.Marshal(titles); data != nil {
                        h.rdb.Set(context.Background(), cacheKey, data, 5*time.Minute)
                }
        }

        response.OK(c, gin.H{"suggestions": titles})
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
                // Used when sort_by=most_bids — JOIN auctions.listing_id with bid_count ordering.
                `CREATE INDEX IF NOT EXISTS idx_auctions_listing_bid_count ON auctions(listing_id, bid_count DESC) WHERE deleted_at IS NULL`,
        }
        for _, idx := range indexes {
                if err := db.Exec(idx).Error; err != nil {
                        slog.Warn("search index creation skipped", "error", err.Error())
                }
        }
        slog.Info("✅ Search indexes ready")
}
