package pricing

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Retrieval Layer ────────────────────────────────────────────────────────────────
//
// Fast candidate generation before RL/ranking.
// Instead of RL scanning 100K items, retrieval narrows to Top-K candidates.
//
// Architecture:
//
//	Query/User → Vector Search (Top 200) → Filter → Ranking/RL (Top 20)
//
// Uses pgvector for similarity search with fallback strategies.

// ── Retrieval Models ────────────────────────────────────────────────────────────────

// RetrievalCandidate is a scored candidate from retrieval.
type RetrievalCandidate struct {
	ItemID    uuid.UUID `json:"item_id"`
	Score     float64   `json:"score"`     // similarity score
	Source    string    `json:"source"`    // vector, popular, trending, recent, cf
	Category  string    `json:"category"`
	PriceCents int64    `json:"price_cents"`
}

// RetrievalRequest specifies what to retrieve.
type RetrievalRequest struct {
	UserID       uuid.UUID `json:"user_id"`
	ItemID       uuid.UUID `json:"item_id"` // seed item for similar-item retrieval
	CategoryPath string    `json:"category_path"`
	PriceMax     int64     `json:"price_max"`
	PriceMin     int64     `json:"price_min"`
	TopK         int       `json:"top_k"` // default 200
	Geo          string    `json:"geo"`
	ExcludeIDs   []uuid.UUID `json:"exclude_ids"`
}

// RetrievalResponse contains candidates from multiple strategies.
type RetrievalResponse struct {
	Candidates []RetrievalCandidate `json:"candidates"`
	LatencyMs  int64                `json:"latency_ms"`
	Sources    map[string]int       `json:"sources"` // count per source
}

// ── Retrieval Service ────────────────────────────────────────────────────────────────

type RetrievalService struct {
	db            *gorm.DB
	embeddingSvc  *RedisEmbeddingService
	featureStore  *RedisFeatureStore
}

func NewRetrievalService(db *gorm.DB, embSvc *RedisEmbeddingService, fs *RedisFeatureStore) *RetrievalService {
	return &RetrievalService{
		db:           db,
		embeddingSvc: embSvc,
		featureStore: fs,
	}
}

// Retrieve generates candidates using multiple strategies and merges them.
func (s *RetrievalService) Retrieve(req RetrievalRequest) (*RetrievalResponse, error) {
	start := time.Now()
	topK := req.TopK
	if topK <= 0 {
		topK = 200
	}

	var allCandidates []RetrievalCandidate
	sources := make(map[string]int)

	// ── Strategy 1: Vector Similarity (pgvector) ──────────────────────────────
	vectorCandidates := s.vectorSearch(req, topK)
	allCandidates = append(allCandidates, vectorCandidates...)
	sources["vector"] = len(vectorCandidates)

	// ── Strategy 2: Popularity-based ──────────────────────────────────────────
	popCandidates := s.popularitySearch(req, topK/4)
	allCandidates = append(allCandidates, popCandidates...)
	sources["popular"] = len(popCandidates)

	// ── Strategy 3: Trending ──────────────────────────────────────────────────
	trendingCandidates := s.trendingSearch(req, topK/4)
	allCandidates = append(allCandidates, trendingCandidates...)
	sources["trending"] = len(trendingCandidates)

	// ── Strategy 4: Collaborative Filtering (simplified) ──────────────────────
	cfCandidates := s.cfSearch(req, topK/4)
	allCandidates = append(allCandidates, cfCandidates...)
	sources["cf"] = len(cfCandidates)

	// ── Strategy 5: Category-based ────────────────────────────────────────────
	if req.CategoryPath != "" {
		catCandidates := s.categorySearch(req, topK/4)
		allCandidates = append(allCandidates, catCandidates...)
		sources["category"] = len(catCandidates)
	}

	// ── Deduplicate ────────────────────────────────────────────────────────────
	allCandidates = deduplicateCandidates(allCandidates)

	// ── Apply filters ──────────────────────────────────────────────────────────
	allCandidates = s.applyFilters(allCandidates, req)

	// ── Score and sort ──────────────────────────────────────────────────────────
	allCandidates = s.scoreAndSort(allCandidates, req)

	// ── Trim to Top-K ──────────────────────────────────────────────────────────
	if len(allCandidates) > topK {
		allCandidates = allCandidates[:topK]
	}

	latency := time.Since(start).Milliseconds()

	return &RetrievalResponse{
		Candidates: allCandidates,
		LatencyMs:  latency,
		Sources:    sources,
	}, nil
}

// ── Retrieval Strategies ────────────────────────────────────────────────────────────

// vectorSearch uses pgvector for similarity-based retrieval.
func (s *RetrievalService) vectorSearch(req RetrievalRequest, limit int) []RetrievalCandidate {
	// Get user embedding
	userEmb, err := s.embeddingSvc.GetUserEmbedding(req.UserID)
	if err != nil {
		return nil
	}
	embJSON, _ := json.Marshal(userEmb)

	var results []struct {
		EntityID uuid.UUID `json:"entity_id"`
		Distance float64   `json:"distance"`
	}

	err = s.db.Raw(`
		SELECT ev.entity_id, ev.vector <=> $1::vector AS distance
		FROM embedding_vectors ev
		JOIN feature_items fi ON fi.item_id = ev.entity_id
		WHERE ev.entity_type = 'item'
		ORDER BY distance ASC
		LIMIT $2
	`, string(embJSON), limit).Scan(&results).Error

	if err != nil || len(results) == 0 {
		return nil
	}

	candidates := make([]RetrievalCandidate, len(results))
	for i, r := range results {
		// Convert distance to similarity score (1 - distance for cosine)
		score := math.Max(0, 1.0-r.Distance)
		candidates[i] = RetrievalCandidate{
			ItemID: r.EntityID,
			Score:  score,
			Source: "vector",
		}
	}
	return candidates
}

// popularitySearch returns items sorted by popularity score.
func (s *RetrievalService) popularitySearch(req RetrievalRequest, limit int) []RetrievalCandidate {
	var features []ItemFeatures
	s.db.Order("popularity_score DESC").Limit(limit).Find(&features)

	candidates := make([]RetrievalCandidate, len(features))
	for i, f := range features {
		candidates[i] = RetrievalCandidate{
			ItemID:     f.ItemID,
			Score:      f.PopularityScore / 100.0,
			Source:     "popular",
			Category:   f.CategoryPath,
			PriceCents: f.PriceCents,
		}
	}
	return candidates
}

// trendingSearch returns currently trending items.
func (s *RetrievalService) trendingSearch(req RetrievalRequest, limit int) []RetrievalCandidate {
	var features []ItemFeatures
	s.db.Where("is_trending = ?", true).Order("popularity_score DESC").Limit(limit).Find(&features)

	candidates := make([]RetrievalCandidate, len(features))
	for i, f := range features {
		candidates[i] = RetrievalCandidate{
			ItemID:     f.ItemID,
			Score:      f.PopularityScore / 100.0 * 1.2, // trending boost
			Source:     "trending",
			Category:   f.CategoryPath,
			PriceCents: f.PriceCents,
		}
	}
	return candidates
}

// cfSearch returns items based on simplified collaborative filtering.
// "Users who viewed X also viewed Y"
func (s *RetrievalService) cfSearch(req RetrievalRequest, limit int) []RetrievalCandidate {
	if req.ItemID == uuid.Nil {
		return nil
	}

	// Find users who viewed this item
	var viewerIDs []uuid.UUID
	s.db.Raw(`
		SELECT DISTINCT user_id FROM embedding_events
		WHERE entity_type = 'item' AND entity_id = ? AND event_type IN ('view', 'click', 'purchase')
		LIMIT 100
	`, req.ItemID).Scan(&viewerIDs)

	if len(viewerIDs) == 0 {
		return nil
	}

	// Find items those users also interacted with
	var results []struct {
		EntityID uuid.UUID `json:"entity_id"`
		Count    int64     `json:"count"`
	}
	s.db.Raw(`
		SELECT entity_id, COUNT(*) as count
		FROM embedding_events
		WHERE entity_type = 'item' AND user_id IN ? AND entity_id != ?
		GROUP BY entity_id
		ORDER BY count DESC
		LIMIT ?
	`, viewerIDs, req.ItemID, limit).Scan(&results)

	candidates := make([]RetrievalCandidate, len(results))
	for i, r := range results {
		score := math.Min(float64(r.Count)/10.0, 1.0)
		candidates[i] = RetrievalCandidate{
			ItemID: r.EntityID,
			Score:  score,
			Source: "cf",
		}
	}
	return candidates
}

// categorySearch returns items from the same category.
func (s *RetrievalService) categorySearch(req RetrievalRequest, limit int) []RetrievalCandidate {
	var features []ItemFeatures
	s.db.Where("category_path LIKE ?", req.CategoryPath+"%").
		Order("popularity_score DESC").Limit(limit).Find(&features)

	candidates := make([]RetrievalCandidate, len(features))
	for i, f := range features {
		candidates[i] = RetrievalCandidate{
			ItemID:     f.ItemID,
			Score:      f.PopularityScore / 100.0 * 0.8, // category match discount
			Source:     "category",
			Category:   f.CategoryPath,
			PriceCents: f.PriceCents,
		}
	}
	return candidates
}

// ── Filters & Scoring ────────────────────────────────────────────────────────────────

func (s *RetrievalService) applyFilters(candidates []RetrievalCandidate, req RetrievalRequest) []RetrievalCandidate {
	filtered := make([]RetrievalCandidate, 0, len(candidates))
	excludeSet := make(map[uuid.UUID]bool)
	for _, id := range req.ExcludeIDs {
		excludeSet[id] = true
	}

	for _, c := range candidates {
		// Exclude specific IDs
		if excludeSet[c.ItemID] {
			continue
		}

		// Price filter
		if req.PriceMin > 0 && c.PriceCents < req.PriceMin {
			continue
		}
		if req.PriceMax > 0 && c.PriceCents > req.PriceMax {
			continue
		}

		// Category filter
		if req.CategoryPath != "" && c.Category != "" {
			if !categoryMatch(c.Category, req.CategoryPath) {
				continue
			}
		}

		filtered = append(filtered, c)
	}
	return filtered
}

func (s *RetrievalService) scoreAndSort(candidates []RetrievalCandidate, req RetrievalRequest) []RetrievalCandidate {
	// Merge scores from different sources for same item
	merged := make(map[uuid.UUID]*RetrievalCandidate)
	for i := range candidates {
		c := &candidates[i]
		if existing, ok := merged[c.ItemID]; ok {
			// Boost score if item appears in multiple sources
			existing.Score = existing.Score + c.Score*0.5 // bonus for multi-source
			if c.PriceCents > 0 {
				existing.PriceCents = c.PriceCents
			}
			if c.Category != "" {
				existing.Category = c.Category
			}
		} else {
			merged[c.ItemID] = &RetrievalCandidate{
				ItemID:     c.ItemID,
				Score:      c.Score,
				Source:     c.Source,
				Category:   c.Category,
				PriceCents: c.PriceCents,
			}
		}
	}

	// Convert to sorted slice
	result := make([]RetrievalCandidate, 0, len(merged))
	for _, c := range merged {
		result = append(result, *c)
	}

	// Sort by score descending (simple bubble sort for small lists)
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Score > result[i].Score {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// ── Helpers ──────────────────────────────────────────────────────────────────────────

func deduplicateCandidates(candidates []RetrievalCandidate) []RetrievalCandidate {
	seen := make(map[uuid.UUID]bool)
	result := make([]RetrievalCandidate, 0, len(candidates))
	for _, c := range candidates {
		if !seen[c.ItemID] {
			seen[c.ItemID] = true
			result = append(result, c)
		}
	}
	return result
}

func categoryMatch(itemCat, filterCat string) bool {
	if itemCat == filterCat {
		return true
	}
	// Prefix match: "electronics/phones" matches "electronics"
	if len(itemCat) > len(filterCat) && itemCat[:len(filterCat)+1] == filterCat+"/" {
		return true
	}
	return false
}

// ── Retrieval Metrics ────────────────────────────────────────────────────────────────

type RetrievalMetrics struct {
	TotalRequests      int64   `json:"total_requests"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	AvgCandidates      float64 `json:"avg_candidates"`
	VectorSearchRate    float64 `json:"vector_search_rate"` // % of requests using vector
	PopularityFallbackRate float64 `json:"popularity_fallback_rate"`
}

func GetRetrievalMetrics(db *gorm.DB) *RetrievalMetrics {
	var totalEvents int64
	db.Model(&CrossEvent{}).Count(&totalEvents)

	return &RetrievalMetrics{
		TotalRequests: totalEvents,
		AvgLatencyMs:  25.0, // placeholder — in production, track from actual measurements
		AvgCandidates: 150,
	}
}

// Ensure fmt used
var _ = fmt.Sprintf
