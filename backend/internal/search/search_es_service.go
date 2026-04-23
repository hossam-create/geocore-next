package search

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Elasticsearch Service — Secondary search engine with fallback to Postgres
// ════════════════════════════════════════════════════════════════════════════

const (
	// Index names
	ListingsIndex = "listings"

	// Query types
	QueryTypeMatch      = "match"
	QueryTypeMultiMatch = "multi_match"
	QueryTypeFuzzy      = "fuzzy"
	QueryTypePrefix     = "prefix"
	QueryTypeWildcard   = "wildcard"
	QueryTypeBool       = "bool"
)

// ESService handles Elasticsearch operations
type ESService struct {
	client *elasticsearch.Client
	db     *gorm.DB
	index  string
}

// NewESService creates a new Elasticsearch service
func NewESService(db *gorm.DB) *ESService {
	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://localhost:9200"
	}

	cfg := elasticsearch.Config{
		Addresses: []string{esURL},
		Username:  os.Getenv("ELASTICSEARCH_USERNAME"),
		Password:  os.Getenv("ELASTICSEARCH_PASSWORD"),
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		slog.Error("failed to create Elasticsearch client", "err", err)
		return nil
	}

	// Verify connection
	res, err := client.Info()
	if err != nil {
		slog.Warn("Elasticsearch connection failed, ES will be disabled", "err", err)
		return nil
	}
	defer res.Body.Close()

	if res.IsError() {
		slog.Warn("Elasticsearch returned error, ES will be disabled", "status", res.Status())
		return nil
	}

	return &ESService{
		client: client,
		db:     db,
		index:  ListingsIndex,
	}
}

// IsEnabled returns true if ES client is available
func (es *ESService) IsEnabled() bool {
	return es != nil && es.client != nil
}

// CreateIndex creates the listings index with the mapping
func (es *ESService) CreateIndex(ctx context.Context) error {
	if !es.IsEnabled() {
		return fmt.Errorf("Elasticsearch is not enabled")
	}

	// Read mapping from file
	mappingFile := "internal/search/es_index_mapping.json"
	mappingData, err := os.ReadFile(mappingFile)
	if err != nil {
		return fmt.Errorf("failed to read mapping file: %w", err)
	}

	var mapping map[string]interface{}
	if err := json.Unmarshal(mappingData, &mapping); err != nil {
		return fmt.Errorf("failed to parse mapping: %w", err)
	}

	// Check if index exists
	req := esapi.IndicesExistsRequest{Index: []string{es.index}}
	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		slog.Info("Elasticsearch index already exists", "index", es.index)
		return nil
	}

	// Create index
	createReq := esapi.IndicesCreateRequest{
		Index: es.index,
		Body:  strings.NewReader(string(mappingData)),
	}
	res, err = createReq.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	slog.Info("Elasticsearch index created successfully", "index", es.index)
	return nil
}

// IndexDocument indexes a single listing document
func (es *ESService) IndexDocument(ctx context.Context, listing *ListingDocument) error {
	if !es.IsEnabled() {
		return fmt.Errorf("Elasticsearch is not enabled")
	}

	doc, err := json.Marshal(listing)
	if err != nil {
		return fmt.Errorf("failed to marshal document: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      es.index,
		DocumentID: listing.ID,
		Body:       strings.NewReader(string(doc)),
		Refresh:    "true",
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	return nil
}

// BulkIndex indexes multiple listings in bulk
func (es *ESService) BulkIndex(ctx context.Context, listings []*ListingDocument) error {
	if !es.IsEnabled() {
		return fmt.Errorf("Elasticsearch is not enabled")
	}

	if len(listings) == 0 {
		return nil
	}

	var body strings.Builder
	for _, listing := range listings {
		doc, err := json.Marshal(listing)
		if err != nil {
			slog.Error("failed to marshal document for bulk index", "id", listing.ID, "err", err)
			continue
		}

		meta := fmt.Sprintf(`{"index": {"_id": "%s"}}`+"\n", listing.ID)
		body.WriteString(meta)
		body.WriteString(string(doc) + "\n")
	}

	req := esapi.BulkRequest{
		Body:    strings.NewReader(body.String()),
		Refresh: "true",
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to bulk index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	return nil
}

// DeleteDocument removes a listing from the index
func (es *ESService) DeleteDocument(ctx context.Context, listingID string) error {
	if !es.IsEnabled() {
		return fmt.Errorf("Elasticsearch is not enabled")
	}

	req := esapi.DeleteRequest{
		Index:      es.index,
		DocumentID: listingID,
		Refresh:    "true",
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() && res.StatusCode != 404 {
		return fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	return nil
}

// Search performs a search query in Elasticsearch
func (es *ESService) Search(ctx context.Context, query string, filters map[string]interface{}, limit, offset int) ([]*ListingDocument, int, error) {
	if !es.IsEnabled() {
		return nil, 0, fmt.Errorf("Elasticsearch is not enabled")
	}

	// Build the search query
	esQuery := es.buildSearchQuery(query, filters)

	// Add pagination
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	esQuery["size"] = limit
	esQuery["from"] = offset

	// Add ranking/sorting
	esQuery["sort"] = []map[string]interface{}{
		{
			"_score": map[string]interface{}{
				"order": "desc",
			},
		},
		{
			"rank_score": map[string]interface{}{
				"order": "desc",
			},
		},
		{
			"created_at": map[string]interface{}{
				"order": "desc",
			},
		},
	}

	queryJSON, err := json.Marshal(esQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal query: %w", err)
	}

	req := esapi.SearchRequest{
		Index: []string{es.index},
		Body:  strings.NewReader(string(queryJSON)),
	}

	res, err := req.Do(ctx, es.client)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, 0, fmt.Errorf("Elasticsearch error: %s", res.String())
	}

	var result struct {
		Hits struct {
			Total struct {
				Value int `json:"value"`
			} `json:"total"`
			Hits []struct {
				ID     string          `json:"_id"`
				Score  float64         `json:"_score"`
				Source ListingDocument `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("failed to decode response: %w", err)
	}

	documents := make([]*ListingDocument, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		doc := hit.Source
		doc.ID = hit.ID
		doc.ES_SCORE = hit.Score
		documents[i] = &doc
	}

	return documents, result.Hits.Total.Value, nil
}

// buildSearchQuery constructs the Elasticsearch query
func (es *ESService) buildSearchQuery(query string, filters map[string]interface{}) map[string]interface{} {
	esQuery := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []interface{}{},
			},
		},
	}

	boolQuery := esQuery["query"].(map[string]interface{})["bool"].(map[string]interface{})

	// Add main search query with fuzzy matching
	if query != "" {
		boolQuery["must"] = append(boolQuery["must"].([]interface{}), map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query": query,
				"fields": []string{
					"title^3",
					"title.ngram^2",
					"title.phonetic^1.5",
					"description",
					"description.ngram",
					"category.text^2",
					"location.text^2",
					"tags^2",
				},
				"type":                 "best_fields",
				"fuzziness":            "AUTO",
				"prefix_length":        2,
				"operator":             "or",
				"minimum_should_match": "75%",
			},
		})
	}

	// Add filters
	if filters != nil {
		filterClauses := []interface{}{}

		// Status filter (only active listings)
		filterClauses = append(filterClauses, map[string]interface{}{
			"term": map[string]interface{}{
				"status": "active",
			},
		})

		// Category filter
		if category, ok := filters["category"].(string); ok && category != "" {
			filterClauses = append(filterClauses, map[string]interface{}{
				"term": map[string]interface{}{
					"category": category,
				},
			})
		}

		// Location filter
		if location, ok := filters["location"].(string); ok && location != "" {
			filterClauses = append(filterClauses, map[string]interface{}{
				"term": map[string]interface{}{
					"location": location,
				},
			})
		}

		// Price range filter
		if priceMin, ok := filters["price_min"].(float64); ok && priceMin > 0 {
			filterClauses = append(filterClauses, map[string]interface{}{
				"range": map[string]interface{}{
					"price": map[string]interface{}{
						"gte": priceMin,
					},
				},
			})
		}
		if priceMax, ok := filters["price_max"].(float64); ok && priceMax > 0 {
			filterClauses = append(filterClauses, map[string]interface{}{
				"range": map[string]interface{}{
					"price": map[string]interface{}{
						"lte": priceMax,
					},
				},
			})
		}

		// Condition filter
		if condition, ok := filters["condition"].(string); ok && condition != "" {
			filterClauses = append(filterClauses, map[string]interface{}{
				"term": map[string]interface{}{
					"condition": condition,
				},
			})
		}

		boolQuery["filter"] = filterClauses
	}

	return esQuery
}

// ReindexAll performs a full reindex of all active listings
func (es *ESService) ReindexAll(ctx context.Context) error {
	if !es.IsEnabled() {
		return fmt.Errorf("Elasticsearch is not enabled")
	}

	slog.Info("Starting full reindex of listings")

	// Fetch all active listings from Postgres
	var listings []struct {
		ID                    uuid.UUID  `gorm:"column:id"`
		Title                 string     `gorm:"column:title"`
		Description           string     `gorm:"column:description"`
		Price                 float64    `gorm:"column:price"`
		Currency              string     `gorm:"column:currency"`
		Category              string     `gorm:"column:category"`
		Location              string     `gorm:"column:location"`
		Condition             string     `gorm:"column:condition"`
		SellerID              uuid.UUID  `gorm:"column:seller_id"`
		SellerName            string     `gorm:"column:seller_name"`
		Status                string     `gorm:"column:status"`
		ViewCount             int        `gorm:"column:view_count"`
		SearchClickCount      int        `gorm:"column:search_click_count"`
		SearchImpressionCount int        `gorm:"column:search_impression_count"`
		CreatedAt             time.Time  `gorm:"column:created_at"`
		UpdatedAt             time.Time  `gorm:"column:updated_at"`
		LastSearchedAt        *time.Time `gorm:"column:last_searched_at"`
		LastViewedAt          *time.Time `gorm:"column:last_viewed_at"`
		Images                []string   `gorm:"column:images"`
	}

	err := es.db.WithContext(ctx).Table("listings").
		Select("id, title, description, price, currency, category, location, condition, seller_id, status, view_count, search_click_count, search_impression_count, created_at, updated_at, last_searched_at, last_viewed_at, images").
		Where("status = ?", "active").
		Scan(&listings).Error

	if err != nil {
		return fmt.Errorf("failed to fetch listings: %w", err)
	}

	slog.Info("Fetched listings for reindexing", "count", len(listings))

	// Convert to ListingDocument
	documents := make([]*ListingDocument, 0, len(listings))
	for _, l := range listings {
		doc := &ListingDocument{
			ID:                    l.ID.String(),
			Title:                 l.Title,
			Description:           l.Description,
			Price:                 l.Price,
			Currency:              l.Currency,
			Category:              l.Category,
			Location:              l.Location,
			Condition:             l.Condition,
			SellerID:              l.SellerID.String(),
			SellerName:            l.SellerName,
			Status:                l.Status,
			ViewCount:             l.ViewCount,
			SearchClickCount:      l.SearchClickCount,
			SearchImpressionCount: l.SearchImpressionCount,
			CreatedAt:             l.CreatedAt,
			UpdatedAt:             l.UpdatedAt,
			LastSearchedAt:        l.LastSearchedAt,
			LastViewedAt:          l.LastViewedAt,
			Images:                l.Images,
			PopularityScore:       float64(l.ViewCount) + float64(l.SearchClickCount)*2,
			RecencyScore:          es.calculateRecencyScore(l.CreatedAt),
		}

		doc.RankScore = es.calculateRankScore(doc)

		documents = append(documents, doc)
	}

	// Bulk index in batches of 100
	batchSize := 100
	for i := 0; i < len(documents); i += batchSize {
		end := i + batchSize
		if end > len(documents) {
			end = len(documents)
		}

		batch := documents[i:end]
		if err := es.BulkIndex(ctx, batch); err != nil {
			slog.Error("failed to bulk index batch", "batch_start", i, "err", err)
			continue
		}

		slog.Info("Bulk indexed batch", "batch_start", i, "batch_size", len(batch))
	}

	slog.Info("Full reindex completed", "total", len(documents))
	return nil
}

// calculateRecencyScore calculates a recency score (0-1)
func (es *ESService) calculateRecencyScore(createdAt time.Time) float64 {
	daysSinceCreation := time.Since(createdAt).Hours() / 24

	if daysSinceCreation < 1 {
		return 1.0
	} else if daysSinceCreation < 7 {
		return 0.8
	} else if daysSinceCreation < 30 {
		return 0.6
	} else if daysSinceCreation < 90 {
		return 0.4
	} else {
		return 0.2
	}
}

// calculateRankScore calculates a combined rank score
func (es *ESService) calculateRankScore(doc *ListingDocument) float64 {
	// Combine popularity and recency
	// Popularity: 0-100 (normalized)
	// Recency: 0-1
	popularity := doc.PopularityScore
	if popularity > 100 {
		popularity = 100
	}

	// Weighted combination: 70% popularity, 30% recency
	return (popularity/100.0)*0.7 + doc.RecencyScore*0.3
}

// ════════════════════════════════════════════════════════════════════════════
// ListingDocument represents a listing in Elasticsearch
// ════════════════════════════════════════════════════════════════════════════

type ListingDocument struct {
	ID                    string     `json:"id"`
	Title                 string     `json:"title"`
	Description           string     `json:"description"`
	Price                 float64    `json:"price"`
	Currency              string     `json:"currency"`
	Category              string     `json:"category"`
	Location              string     `json:"location"`
	GeoLocation           *GeoPoint  `json:"geo_location,omitempty"`
	Condition             string     `json:"condition"`
	SellerID              string     `json:"seller_id"`
	SellerName            string     `json:"seller_name,omitempty"`
	Status                string     `json:"status"`
	ViewCount             int        `json:"view_count"`
	SearchClickCount      int        `json:"search_click_count"`
	SearchImpressionCount int        `json:"search_impression_count"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastSearchedAt        *time.Time `json:"last_searched_at,omitempty"`
	LastViewedAt          *time.Time `json:"last_viewed_at,omitempty"`
	Images                []string   `json:"images,omitempty"`
	Tags                  []string   `json:"tags,omitempty"`
	Language              string     `json:"language,omitempty"`
	PopularityScore       float64    `json:"popularity_score"`
	RecencyScore          float64    `json:"recency_score"`
	RankScore             float64    `json:"rank_score"`
	ES_SCORE              float64    `json:"-"` // Elasticsearch score, not indexed
}

// GeoPoint represents a geographic location
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}
