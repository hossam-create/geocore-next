package search

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Search Ranking and Analytics Service
// ════════════════════════════════════════════════════════════════════════════

// SearchAnalytics tracks search performance
type SearchAnalytics struct {
	QueryHash         string     `json:"query_hash"`
	ZeroResult        bool       `json:"zero_result"`
	ResultCount       int        `json:"result_count"`
	ClickedListingID  *uuid.UUID `json:"clicked_listing_id,omitempty"`
	PositionInResults int        `json:"position_in_results,omitempty"`
	UserID            *uuid.UUID `json:"user_id,omitempty"`
	IPAddress         string     `json:"ip_address,omitempty"`
	UserAgent         string     `json:"user_agent,omitempty"`
}

// DidYouMeanResult contains spelling correction suggestions
type DidYouMeanResult struct {
	OriginalQuery   string   `json:"original_query"`
	SuggestedQuery  string   `json:"suggested_query"`
	SimilarityScore float64  `json:"similarity_score"`
	MatchedTitles   []string `json:"matched_titles"`
}

// RankingSignals contains factors for ranking
type RankingSignals struct {
	RecencyScore         float64 // Time decay (newer = higher)
	PopularityScore      float64 // View count, click-through rate
	RelevanceScore       float64 // Text/vector similarity
	PersonalizationScore float64 // User-specific (history, favorites)
	LocationScore        float64 // Distance to user
	PriceScore           float64 // Price competitiveness
}

// ── Did You Mean (Spelling Correction) ────────────────────────────────────────

// GetDidYouMeanSuggestions finds similar titles using pg_trgm similarity
func (h *Handler) GetDidYouMeanSuggestions(query string, limit int) (*DidYouMeanResult, error) {
	type similarTitle struct {
		Title      string  `gorm:"column:title"`
		Similarity float64 `gorm:"column:similarity"`
	}

	var titles []similarTitle
	err := h.db.Raw(`
		SELECT title, similarity(title, ?) as similarity
		FROM listings
		WHERE status = 'active'
		  AND similarity(title, ?) > 0.3
		ORDER BY similarity DESC, title
		LIMIT ?
	`, query, query, limit).Scan(&titles).Error

	if err != nil {
		return nil, err
	}

	if len(titles) == 0 {
		return nil, nil
	}

	// Use the most similar title as the suggestion
	suggestion := titles[0]

	matchedTitles := make([]string, len(titles))
	for i, t := range titles {
		matchedTitles[i] = t.Title
	}

	return &DidYouMeanResult{
		OriginalQuery:   query,
		SuggestedQuery:  suggestion.Title,
		SimilarityScore: suggestion.Similarity,
		MatchedTitles:   matchedTitles,
	}, nil
}

// ── Ranking Calculation ──────────────────────────────────────────────────────

// CalculateRankScore computes a combined ranking score (0-100)
func (h *Handler) CalculateRankScore(
	db *gorm.DB,
	listingID uuid.UUID,
	relevanceScore float64,
	userID *uuid.UUID,
	userLocation string,
) (float64, error) {
	var listing struct {
		ViewCount             int        `gorm:"column:view_count"`
		SearchClickCount      int        `gorm:"column:search_click_count"`
		SearchImpressionCount int        `gorm:"column:search_impression_count"`
		Price                 float64    `gorm:"column:price"`
		Location              string     `gorm:"column:location"`
		CreatedAt             time.Time  `gorm:"column:created_at"`
		LastViewedAt          *time.Time `gorm:"column:last_viewed_at"`
	}

	err := db.Table("listings").Where("id = ?", listingID).First(&listing).Error
	if err != nil {
		return relevanceScore, nil // fallback to relevance only
	}

	signals := RankingSignals{
		RelevanceScore: relevanceScore,
	}

	// 1. Recency score (0-20): newer listings get higher scores
	daysSinceCreation := time.Since(listing.CreatedAt).Hours() / 24
	if daysSinceCreation < 1 {
		signals.RecencyScore = 20
	} else if daysSinceCreation < 7 {
		signals.RecencyScore = 15
	} else if daysSinceCreation < 30 {
		signals.RecencyScore = 10
	} else if daysSinceCreation < 90 {
		signals.RecencyScore = 5
	} else {
		signals.RecencyScore = 0
	}

	// 2. Popularity score (0-25): based on views and click-through rate
	signals.PopularityScore = 0

	// View count contribution (capped at 15)
	if listing.ViewCount > 0 {
		viewScore := float64(listing.ViewCount) / 100.0 * 15
		if viewScore > 15 {
			viewScore = 15
		}
		signals.PopularityScore += viewScore
	}

	// Click-through rate contribution (capped at 10)
	if listing.SearchImpressionCount > 0 {
		ctr := float64(listing.SearchClickCount) / float64(listing.SearchImpressionCount)
		ctrScore := ctr * 10
		if ctrScore > 10 {
			ctrScore = 10
		}
		signals.PopularityScore += ctrScore
	}

	// 3. Personalization score (0-20): user-specific signals
	signals.PersonalizationScore = 0
	if userID != nil {
		// Check if user has viewed this listing before
		var viewCount int
		db.Raw(`
			SELECT COUNT(*) FROM listing_views
			WHERE listing_id = ? AND user_id = ?
		`, listingID, userID).Scan(&viewCount)

		if viewCount > 0 {
			signals.PersonalizationScore = 10 // User has viewed before
		}

		// Check if user has favorited this listing
		var favoriteCount int
		db.Raw(`
			SELECT COUNT(*) FROM favorites
			WHERE listing_id = ? AND user_id = ?
		`, listingID, userID).Scan(&favoriteCount)

		if favoriteCount > 0 {
			signals.PersonalizationScore += 10 // User has favorited
		}
	}

	// 4. Location score (0-15): proximity to user
	signals.LocationScore = 0
	if userLocation != "" && listing.Location != "" {
		// Simple string match for now (could be enhanced with geospatial)
		if containsSubstring(listing.Location, userLocation) || containsSubstring(userLocation, listing.Location) {
			signals.LocationScore = 15
		}
	}

	// 5. Price score (0-20): competitive pricing (not too high, not too low)
	signals.PriceScore = 10 // neutral baseline

	// Combine scores with weights
	// Relevance: 30%, Recency: 15%, Popularity: 20%, Personalization: 15%, Location: 10%, Price: 10%
	rankScore := (signals.RelevanceScore * 0.30) +
		(signals.RecencyScore * 0.15) +
		(signals.PopularityScore * 0.20) +
		(signals.PersonalizationScore * 0.15) +
		(signals.LocationScore * 0.10) +
		(signals.PriceScore * 0.10)

	// Normalize to 0-100
	if rankScore > 100 {
		rankScore = 100
	}

	return rankScore, nil
}

// ── Analytics Tracking ─────────────────────────────────────────────────────────

// TrackSearchQuery logs search query analytics
func (h *Handler) TrackSearchQuery(analytics *SearchAnalytics) error {
	// Update popular queries
	h.db.Exec(`
		SELECT update_popular_query(?, ?, ?, ?)
	`, analytics.QueryHash, analytics.ResultCount, "en") // TODO: detect language

	// Log to search_queries
	intentJSON := "{}" // simplified for now
	zeroResult := analytics.ResultCount == 0

	h.db.Exec(`
		INSERT INTO search_queries (query, query_hash, intent_json, result_count, zero_result, clicked_listing_id, position_in_results)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, analytics.QueryHash, analytics.QueryHash, intentJSON, analytics.ResultCount, zeroResult,
		analytics.ClickedListingID, analytics.PositionInResults)

	// Track zero results separately
	if zeroResult {
		h.db.Exec(`
			INSERT INTO search_zero_results (query, query_hash, user_id, ip_address, user_agent)
			VALUES (?, ?, ?, ?, ?)
		`, analytics.QueryHash, analytics.QueryHash, analytics.UserID, analytics.IPAddress, analytics.UserAgent)
	}

	return nil
}

// TrackListingClick updates click analytics for a listing
func (h *Handler) TrackListingClick(listingID uuid.UUID, userID *uuid.UUID) error {
	// Increment search_click_count
	h.db.Exec(`
		UPDATE listings
		SET search_click_count = search_click_count + 1,
		    last_searched_at = NOW()
		WHERE id = ?
	`, listingID)

	// Log click event
	h.db.Exec(`
		INSERT INTO listing_clicks (listing_id, user_id, clicked_at)
		VALUES (?, ?, NOW())
		ON CONFLICT (listing_id, user_id) DO UPDATE SET clicked_at = NOW()
	`, listingID, userID)

	return nil
}

// TrackListingView updates view analytics for a listing
func (h *Handler) TrackListingView(listingID uuid.UUID, userID *uuid.UUID) error {
	// Increment view_count
	h.db.Exec(`
		UPDATE listings
		SET view_count = view_count + 1,
		    last_viewed_at = NOW()
		WHERE id = ?
	`, listingID)

	// Log view event
	h.db.Exec(`
		INSERT INTO listing_views (listing_id, user_id, viewed_at)
		VALUES (?, ?, NOW())
	`, listingID, userID)

	return nil
}

// ── Helper Functions ──────────────────────────────────────────────────────────

// hashQuery creates a SHA256 hash of the query for deduplication
func hashQuery(query string) string {
	h := sha256.Sum256([]byte(query))
	return hex.EncodeToString(h[:])
}

// containsSubstring checks if substr is in str (case-insensitive)
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str[:len(substr)] == substr ||
			str[len(str)-len(substr):] == substr ||
			findSubstring(str, substr))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ── Synonym Expansion ─────────────────────────────────────────────────────────

// ExpandQueryWithSynonyms expands the query using the synonym table
func (h *Handler) ExpandQueryWithSynonyms(query, language string) ([]string, error) {
	var synonyms []string

	err := h.db.Raw(`
		SELECT DISTINCT synonym
		FROM search_synonyms
		WHERE term = ANY(string_to_array(lower(?), ' '))
		AND language = ?
		ORDER BY weight DESC
	`, query, language).Scan(&synonyms).Error

	if err != nil {
		return nil, err
	}

	// Return original query + synonyms
	expanded := []string{query}
	expanded = append(expanded, synonyms...)

	return expanded, nil
}
