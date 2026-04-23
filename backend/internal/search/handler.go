package search

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Handler — semantic search using pgvector + OpenAI embeddings
// ════════════════════════════════════════════════════════════════════════════

type Handler struct {
	db     *gorm.DB
	openai *OpenAIClient
	es     *ESService
}

func NewHandler(db *gorm.DB) *Handler {
	es := NewESService(db)
	return &Handler{
		db:     db,
		openai: NewOpenAIClientFromEnv(),
		es:     es,
	}
}

// ── Request/Response types ────────────────────────────────────────────────────

type SearchRequest struct {
	Query    string                 `json:"query" binding:"required,min=2,max=500"`
	Filters  map[string]interface{} `json:"filters"`
	Limit    int                    `json:"limit"`
	Offset   int                    `json:"offset"`
	Language string                 `json:"language"` // en, ar, etc.
	UserID   string                 `json:"user_id,omitempty"`
}

type SearchIntent struct {
	Keywords    []string `json:"keywords"`
	Category    string   `json:"category,omitempty"`
	PriceMin    *float64 `json:"price_min,omitempty"`
	PriceMax    *float64 `json:"price_max,omitempty"`
	Location    string   `json:"location,omitempty"`
	Condition   string   `json:"condition,omitempty"`
	Summary     string   `json:"summary"`
	Suggestions []string `json:"suggestions"`
}

type SearchResult struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Price           float64   `json:"price"`
	Currency        string    `json:"currency"`
	Category        string    `json:"category"`
	Location        string    `json:"location"`
	Condition       string    `json:"condition"`
	Images          []string  `json:"images"`
	SellerID        uuid.UUID `json:"seller_id"`
	SimilarityScore float64   `json:"similarity_score"`
	AIReason        string    `json:"ai_reason"`
	RankScore       float64   `json:"rank_score"` // Combined ranking score
	ViewCount       int       `json:"view_count"`
	CreatedAt       time.Time `json:"created_at"`
}

// ── POST /api/v1/search ───────────────────────────────────────────────────────

func (h *Handler) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	req.Query = security.SanitizeSearchQuery(req.Query)
	if req.Query == "" {
		response.BadRequest(c, "query is required")
		return
	}

	if req.Limit <= 0 || req.Limit > 50 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Default language to English if not specified
	if req.Language == "" {
		req.Language = "en"
	}

	// Get user ID from context if authenticated
	if req.UserID == "" {
		if uid, exists := c.Get("user_id"); exists {
			req.UserID = uid.(string)
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// ── Step 1: Understand query with AI ──────────────────────────────────────
	intent := h.parseIntent(ctx, req.Query, req.Filters)

	// ── Step 2: Search with priority: ES > Vector > Text ────────────────────────
	var results []SearchResult
	var err error
	var searchEngine string

	// Try Elasticsearch first if enabled
	if h.es != nil && h.es.IsEnabled() {
		esDocs, total, esErr := h.es.Search(ctx, req.Query, req.Filters, req.Limit, req.Offset)
		if esErr == nil {
			// Convert ES documents to SearchResults
			results = h.convertESResults(esDocs)
			searchEngine = "elasticsearch"
			slog.Info("Elasticsearch search successful", "results", total, "engine", searchEngine)
		} else {
			slog.Warn("Elasticsearch search failed, falling back to vector/text", "err", esErr)
		}
	}

	// Fallback to vector search if ES failed or disabled
	if len(results) == 0 && h.openai != nil {
		results, err = h.vectorSearch(ctx, req.Query, intent, req.Limit, req.Offset)
		if err != nil {
			slog.Warn("vector search failed, falling back to text search", "err", err)
			results, err = h.textSearch(ctx, intent, req.Limit, req.Offset)
			searchEngine = "text"
		} else {
			searchEngine = "vector"
		}
	} else if len(results) == 0 {
		// Text-based fallback when OpenAI not configured
		results, err = h.textSearch(ctx, intent, req.Limit, req.Offset)
		searchEngine = "text"
	}

	if err != nil {
		slog.Error("search failed", "err", err)
		response.InternalError(c, err)
		return
	}

	// ── Step 3: Calculate ranking scores for results ───────────────────────
	for i := range results {
		var userID *uuid.UUID
		if req.UserID != "" {
			uid, _ := uuid.Parse(req.UserID)
			userID = &uid
		}
		rankScore, _ := h.CalculateRankScore(h.db, results[i].ID, results[i].SimilarityScore, userID, intent.Location)
		results[i].RankScore = rankScore
	}

	// Sort by rank score (descending)
	sortResultsByRank(results)

	// ── Step 4: Did you mean? (if results are poor) ───────────────────────────
	var didYouMean *DidYouMeanResult
	if len(results) < 3 {
		didYouMean, _ = h.GetDidYouMeanSuggestions(req.Query, 5)
	}

	// ── Step 5: Log query + analytics asynchronously ───────────────────────
	queryHash := hashQuery(req.Query)
	var userID *uuid.UUID
	if req.UserID != "" {
		uid, _ := uuid.Parse(req.UserID)
		userID = &uid
	}
	h.TrackSearchQuery(&SearchAnalytics{
		QueryHash:   queryHash,
		ZeroResult:  len(results) == 0,
		ResultCount: len(results),
		UserID:      userID,
		IPAddress:   c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"query":         req.Query,
			"intent":        intent,
			"results":       results,
			"total":         len(results),
			"ai_powered":    h.openai != nil,
			"es_enabled":    h.es != nil && h.es.IsEnabled(),
			"search_engine": searchEngine,
			"did_you_mean":  didYouMean,
		},
	})
}

// ── Autocomplete suggestions ─────────────────────────────────────────────────

func (h *Handler) Suggest(c *gin.Context) {
	q := security.SanitizeSearchQuery(strings.TrimSpace(c.Query("q")))
	if len(q) < 2 {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"suggestions": []string{}}})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	suggestions := h.getSuggestions(ctx, q)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"suggestions": suggestions,
			"ai_powered":  h.openai != nil,
		},
	})
}

// ── Track listing click (analytics) ─────────────────────────────────────────────
func (h *Handler) TrackClick(c *gin.Context) {
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		response.BadRequest(c, "invalid listing id")
		return
	}

	userID := c.GetString("user_id")
	var uid *uuid.UUID
	if userID != "" {
		parsed, _ := uuid.Parse(userID)
		uid = &parsed
	}

	if err := h.TrackListingClick(listingID, uid); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{"success": true})
}

// ── Track listing view (analytics) ─────────────────────────────────────────────
func (h *Handler) TrackView(c *gin.Context) {
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		response.BadRequest(c, "invalid listing id")
		return
	}

	userID := c.GetString("user_id")
	var uid *uuid.UUID
	if userID != "" {
		parsed, _ := uuid.Parse(userID)
		uid = &parsed
	}

	if err := h.TrackListingView(listingID, uid); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{"success": true})
}

// ── ES Management: ReindexAll — full reindex of all listings to ES ─────────────
func (h *Handler) ReindexAll(c *gin.Context) {
	if h.es == nil || !h.es.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Elasticsearch not enabled"})
		return
	}

	// Check admin role
	userRole := c.GetString("role")
	if userRole != "admin" && userRole != "super_admin" {
		response.Forbidden(c)
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Trigger reindex in background
	go func() {
		if err := h.es.ReindexAll(ctx); err != nil {
			slog.Error("ES reindex failed", "err", err)
		}
	}()

	response.OK(c, gin.H{"success": true, "message": "Reindex started"})
}

// ── ES Management: ESStatus — check ES health and index status ─────────────────
func (h *Handler) ESStatus(c *gin.Context) {
	if h.es == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"enabled": false,
			"status":  "disabled",
		})
		return
	}

	// Check ES health
	health := h.es.IsEnabled()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"enabled": health,
		"status":  map[bool]string{true: "healthy", false: "unhealthy"}[health],
	})
}

// ── ES Management: IndexSingleListing — reindex a single listing ────────────────
func (h *Handler) IndexSingleListing(c *gin.Context) {
	if h.es == nil || !h.es.IsEnabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "Elasticsearch not enabled"})
		return
	}

	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid listing id")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	syncConsumer := NewESSyncConsumer(h.es, h.db)
	if err := syncConsumer.SyncListingByID(ctx, listingID); err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{"success": true, "message": "Listing indexed"})
}

// ── Trending searches ─────────────────────────────────────────────────────────

func (h *Handler) Trending(c *gin.Context) {
	var trending []struct {
		Query string `json:"query" gorm:"column:query"`
		Count int    `json:"count" gorm:"column:count"`
	}
	h.db.Raw(`
          SELECT query, COUNT(*) as count
          FROM search_queries
          WHERE created_at > NOW() - INTERVAL '7 days'
          GROUP BY query
          ORDER BY count DESC
          LIMIT 10
      `).Scan(&trending)

	if len(trending) == 0 {
		trending = []struct {
			Query string `json:"query" gorm:"column:query"`
			Count int    `json:"count" gorm:"column:count"`
		}{
			{"iPhone 15 Pro", 1240}, {"Toyota Land Cruiser", 980},
			{"شقة دبي", 875}, {"PS5", 760}, {"MacBook Pro M3", 710},
			{"Rolex", 620}, {"سيارة للبيع", 590}, {"DJI Drone", 480},
		}
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"trending": trending}})
}

// ── EmbedListing — generate and store embedding for a listing ─────────────────

func (h *Handler) EmbedListing(c *gin.Context) {
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid listing id")
		return
	}

	var listing struct {
		Title       string `gorm:"column:title"`
		Description string `gorm:"column:description"`
		Category    string `gorm:"column:category"`
		Location    string `gorm:"column:location"`
		Condition   string `gorm:"column:condition"`
	}
	if err := h.db.Table("listings").Where("id = ?", listingID).First(&listing).Error; err != nil {
		response.NotFound(c, "listing")
		return
	}

	if h.openai == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "OpenAI not configured"})
		return
	}

	content := fmt.Sprintf("%s. %s. Category: %s. Location: %s. Condition: %s.",
		listing.Title, listing.Description, listing.Category, listing.Location, listing.Condition)

	contentHash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	embedding, err := h.openai.Embed(ctx, content)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	vecStr := floatsToVectorStr(embedding)
	h.db.Exec(`
          INSERT INTO listing_embeddings (listing_id, embedding, content_hash, model)
          VALUES (?, ?, ?, 'text-embedding-3-small')
          ON CONFLICT (listing_id) DO UPDATE
          SET embedding = EXCLUDED.embedding, content_hash = EXCLUDED.content_hash, updated_at = NOW()
      `, listingID, vecStr, contentHash)

	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"listing_id": listingID, "embedded": true}})
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (h *Handler) parseIntent(ctx context.Context, query string, filters map[string]interface{}) *SearchIntent {
	intent := &SearchIntent{
		Keywords:    strings.Fields(query),
		Summary:     fmt.Sprintf("Searching for: \"%s\"", query),
		Suggestions: []string{},
	}

	if h.openai == nil {
		lq := strings.ToLower(query)
		if strings.Contains(lq, "iphone") || strings.Contains(lq, "samsung") || strings.Contains(lq, "laptop") {
			intent.Category = "Electronics"
		} else if strings.Contains(lq, "car") || strings.Contains(lq, "سيارة") {
			intent.Category = "Vehicles"
		} else if strings.Contains(lq, "villa") || strings.Contains(lq, "apartment") || strings.Contains(lq, "شقة") {
			intent.Category = "Real Estate"
		}
		return intent
	}

	systemPrompt := `You are a GCC marketplace search assistant. Return ONLY valid JSON:
  {"keywords":["kw1","kw2"],"category":"Electronics|Vehicles|Real Estate|Clothing|Furniture|Watches|Other|null",
  "price_min":null,"price_max":null,"location":"city_or_null","condition":"New|Like New|Good|Fair|null",
  "summary":"human readable summary","suggestions":["rel1","rel2","rel3"]}`

	resp, err := h.openai.ChatComplete(ctx, systemPrompt,
		fmt.Sprintf("Query: \"%s\"\nFilters: %v", query, filters), 400)
	if err != nil {
		return intent
	}

	var parsed struct {
		Keywords    []string `json:"keywords"`
		Category    string   `json:"category"`
		PriceMin    *float64 `json:"price_min"`
		PriceMax    *float64 `json:"price_max"`
		Location    string   `json:"location"`
		Condition   string   `json:"condition"`
		Summary     string   `json:"summary"`
		Suggestions []string `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(resp), &parsed); err == nil {
		if len(parsed.Keywords) > 0 {
			intent.Keywords = parsed.Keywords
		}
		if parsed.Category != "" && parsed.Category != "null" {
			intent.Category = parsed.Category
		}
		intent.PriceMin = parsed.PriceMin
		intent.PriceMax = parsed.PriceMax
		if parsed.Location != "" && parsed.Location != "null" {
			intent.Location = parsed.Location
		}
		if parsed.Condition != "" && parsed.Condition != "null" {
			intent.Condition = parsed.Condition
		}
		if parsed.Summary != "" {
			intent.Summary = parsed.Summary
		}
		intent.Suggestions = parsed.Suggestions
	}
	return intent
}

func (h *Handler) vectorSearch(ctx context.Context, query string, intent *SearchIntent, limit, offset int) ([]SearchResult, error) {
	embedding, err := h.openai.Embed(ctx, query)
	if err != nil {
		return nil, err
	}
	vecStr := floatsToVectorStr(embedding)

	type row struct {
		ID              uuid.UUID `gorm:"column:id"`
		Title           string    `gorm:"column:title"`
		Description     string    `gorm:"column:description"`
		Price           float64   `gorm:"column:price"`
		Currency        string    `gorm:"column:currency"`
		Category        string    `gorm:"column:category"`
		Location        string    `gorm:"column:location"`
		Condition       string    `gorm:"column:condition"`
		SellerID        uuid.UUID `gorm:"column:seller_id"`
		SimilarityScore float64   `gorm:"column:similarity"`
	}
	var rows []row
	err = h.db.WithContext(ctx).Raw(`
          SELECT l.id, l.title, l.description, l.price, l.currency,
                 l.category, l.location, l.condition, l.seller_id,
                 1 - (le.embedding <=> ?::vector) AS similarity
          FROM listings l
          JOIN listing_embeddings le ON le.listing_id = l.id
          WHERE l.status = 'active'
          ORDER BY le.embedding <=> ?::vector
          LIMIT ? OFFSET ?
      `, vecStr, vecStr, limit, offset).Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(rows))
	for i, r := range rows {
		reason := "Possible match"
		if r.SimilarityScore >= 0.85 {
			reason = "Excellent semantic match"
		} else if r.SimilarityScore >= 0.70 {
			reason = "Strong match for your search"
		} else if r.SimilarityScore >= 0.55 {
			reason = "Good match"
		}
		results[i] = SearchResult{
			ID: r.ID, Title: r.Title, Description: r.Description,
			Price: r.Price, Currency: r.Currency, Category: r.Category,
			Location: r.Location, Condition: r.Condition, SellerID: r.SellerID,
			SimilarityScore: r.SimilarityScore, AIReason: reason,
		}
	}
	return results, nil
}

func (h *Handler) textSearch(ctx context.Context, intent *SearchIntent, limit, offset int) ([]SearchResult, error) {
	// Use trigram similarity for fuzzy matching
	query := strings.Join(intent.Keywords, " ")
	type row struct {
		ID          uuid.UUID `gorm:"column:id"`
		Title       string    `gorm:"column:title"`
		Description string    `gorm:"column:description"`
		Price       float64   `gorm:"column:price"`
		Currency    string    `gorm:"column:currency"`
		Category    string    `gorm:"column:category"`
		Location    string    `gorm:"column:location"`
		Condition   string    `gorm:"column:condition"`
		SellerID    uuid.UUID `gorm:"column:seller_id"`
		ViewCount   int       `gorm:"column:view_count"`
		CreatedAt   time.Time `gorm:"column:created_at"`
		Similarity  float64   `gorm:"column:similarity"`
	}
	var rows []row
	db := h.db.WithContext(ctx).Raw(`
		SELECT l.id, l.title, l.description, l.price, l.currency,
		       l.category, l.location, l.condition, l.seller_id,
		       l.view_count, l.created_at,
		       GREATEST(
			   similarity(l.title, ?),
			   similarity(l.description, ?)
		       ) as similarity
		FROM listings l
		WHERE l.status = 'active'
		  AND (l.title % ? OR l.description % ?)
	`, query, query, query, query)

	if intent.Category != "" {
		db = db.Where("category ILIKE ?", "%"+intent.Category+"%")
	}
	if intent.Location != "" {
		db = db.Where("location ILIKE ?", "%"+intent.Location+"%")
	}
	if intent.PriceMax != nil {
		db = db.Where("price <= ?", *intent.PriceMax)
	}

	err := db.Order("similarity DESC").Limit(limit).Offset(offset).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(rows))
	for i, r := range rows {
		reason := "Text match"
		if r.Similarity >= 0.7 {
			reason = "Strong text match"
		} else if r.Similarity >= 0.5 {
			reason = "Good text match"
		}
		results[i] = SearchResult{
			ID: r.ID, Title: r.Title, Description: r.Description,
			Price: r.Price, Currency: r.Currency, Category: r.Category,
			Location: r.Location, Condition: r.Condition, SellerID: r.SellerID,
			SimilarityScore: r.Similarity, AIReason: reason,
			ViewCount: r.ViewCount, CreatedAt: r.CreatedAt,
		}
	}
	return results, nil
}

func (h *Handler) convertESResults(docs []*ListingDocument) []SearchResult {
	results := make([]SearchResult, len(docs))
	for i, doc := range docs {
		id, _ := uuid.Parse(doc.ID)
		sellerID, _ := uuid.Parse(doc.SellerID)

		reason := "ES match"
		if doc.ES_SCORE >= 5.0 {
			reason = "Excellent ES match"
		} else if doc.ES_SCORE >= 3.0 {
			reason = "Strong ES match"
		} else if doc.ES_SCORE >= 1.0 {
			reason = "Good ES match"
		}

		results[i] = SearchResult{
			ID:              id,
			Title:           doc.Title,
			Description:     doc.Description,
			Price:           doc.Price,
			Currency:        doc.Currency,
			Category:        doc.Category,
			Location:        doc.Location,
			Condition:       doc.Condition,
			Images:          doc.Images,
			SellerID:        sellerID,
			SimilarityScore: doc.ES_SCORE,
			AIReason:        reason,
			RankScore:       doc.RankScore,
			ViewCount:       doc.ViewCount,
			CreatedAt:       doc.CreatedAt,
		}
	}
	return results
}

func (h *Handler) getSuggestions(ctx context.Context, q string) []string {
	defaults := []string{q + " for sale", q + " Dubai", "cheap " + q, "used " + q, q + " new"}
	if h.openai == nil {
		return defaults
	}

	resp, err := h.openai.ChatComplete(ctx,
		"You are a GCC marketplace autocomplete engine. Given a partial query, return 5 suggestions as a JSON array of strings (2-5 words each). Match the language of the input.",
		"Partial query: \""+q+"\"", 150)
	if err != nil {
		return defaults
	}

	var suggestions []string
	if err := json.Unmarshal([]byte(resp), &suggestions); err == nil && len(suggestions) > 0 {
		return suggestions
	}
	return defaults
}

func (h *Handler) logQuery(query string, intent *SearchIntent, resultCount int) {
	intentJSON, _ := json.Marshal(intent)
	h.db.Exec(`
          INSERT INTO search_queries (query, intent_json, result_count)
          VALUES (?, ?, ?)
      `, query, string(intentJSON), resultCount)
}

func (h *Handler) enqueueSearchAnalytics(query string, intent *SearchIntent, resultCount int, userID string) {
	_ = jobs.EnqueueDefault(&jobs.Job{
		Type:        jobs.JobTypeAnalytics,
		Priority:    6,
		MaxAttempts: 3,
		Payload: map[string]interface{}{
			"event":   "search.query",
			"user_id": userID,
			"properties": map[string]interface{}{
				"query":        query,
				"result_count": resultCount,
				"category":     intent.Category,
			},
		},
	})

	go h.logQuery(query, intent, resultCount)
}

func floatsToVectorStr(v []float32) string {
	if len(v) == 0 {
		return "[]"
	}
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%.6f", f)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// sortResultsByRank sorts search results by rank score (descending)
func sortResultsByRank(results []SearchResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].RankScore < results[j].RankScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// ── OpenAI client (stdlib HTTP only) ─────────────────────────────────────────

type OpenAIClient struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewOpenAIClientFromEnv() *OpenAIClient {
	base := os.Getenv("OPENAI_API_BASE")
	key := os.Getenv("OPENAI_API_KEY")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	if key == "" {
		slog.Warn("OPENAI_API_KEY not set — AI search degraded to text-only mode")
		return nil
	}
	return &OpenAIClient{baseURL: base, apiKey: key, client: &http.Client{Timeout: 15 * time.Second}}
}

func (c *OpenAIClient) Embed(ctx context.Context, text string) ([]float32, error) {
	payload := map[string]interface{}{"input": text, "model": "text-embedding-3-small"}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", strings.NewReader(string(b)))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return result.Data[0].Embedding, nil
}

func (c *OpenAIClient) ChatComplete(ctx context.Context, system, user string, maxTokens int) (string, error) {
	payload := map[string]interface{}{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"max_tokens": maxTokens,
	}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", strings.NewReader(string(b)))
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}
	return result.Choices[0].Message.Content, nil
}
