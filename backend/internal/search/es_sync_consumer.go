package search

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/events"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Elasticsearch Sync Consumer — Syncs listing events to ES via Kafka/Event Bus
// ════════════════════════════════════════════════════════════════════════════

// ESSyncConsumer handles syncing listing events to Elasticsearch
type ESSyncConsumer struct {
	esService *ESService
	db        *gorm.DB
}

// NewESSyncConsumer creates a new ES sync consumer
func NewESSyncConsumer(esService *ESService, db *gorm.DB) *ESSyncConsumer {
	return &ESSyncConsumer{
		esService: esService,
		db:        db,
	}
}

// Start begins consuming listing events
func (c *ESSyncConsumer) Start() {
	if !c.esService.IsEnabled() {
		slog.Warn("Elasticsearch not enabled, skipping sync consumer")
		return
	}

	// Subscribe to listing events
	events.Subscribe(events.EventListingCreated, c.handleListingCreated)
	events.Subscribe(events.EventOrderCreated, c.handleListingUpdated) // Orders can affect listing popularity
	events.Subscribe(events.EventReviewPosted, c.handleListingUpdated) // Reviews affect ranking

	slog.Info("Elasticsearch sync consumer started")
}

// handleListingCreated syncs a new listing to ES
func (c *ESSyncConsumer) handleListingCreated(e events.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listingIDStr, ok := e.Payload["listing_id"].(string)
	if !ok {
		slog.Error("invalid listing_id in event", "event", e.Type)
		return
	}

	listingID, err := uuid.Parse(listingIDStr)
	if err != nil {
		slog.Error("failed to parse listing_id", "listing_id", listingIDStr, "err", err)
		return
	}

	// Fetch listing from Postgres
	doc, err := c.fetchListingDocument(ctx, listingID)
	if err != nil {
		slog.Error("failed to fetch listing for sync", "listing_id", listingID, "err", err)
		return
	}

	// Index in ES
	if err := c.esService.IndexDocument(ctx, doc); err != nil {
		slog.Error("failed to index listing in ES", "listing_id", listingID, "err", err)
		return
	}

	slog.Info("listing synced to ES", "listing_id", listingID)
}

// handleListingUpdated updates an existing listing in ES
func (c *ESSyncConsumer) handleListingUpdated(e events.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listingIDStr, ok := e.Payload["listing_id"].(string)
	if !ok {
		return // Not a listing event
	}

	listingID, err := uuid.Parse(listingIDStr)
	if err != nil {
		slog.Error("failed to parse listing_id", "listing_id", listingIDStr, "err", err)
		return
	}

	// Fetch updated listing from Postgres
	doc, err := c.fetchListingDocument(ctx, listingID)
	if err != nil {
		slog.Error("failed to fetch listing for sync", "listing_id", listingID, "err", err)
		return
	}

	// Re-index in ES
	if err := c.esService.IndexDocument(ctx, doc); err != nil {
		slog.Error("failed to update listing in ES", "listing_id", listingID, "err", err)
		return
	}

	slog.Info("listing updated in ES", "listing_id", listingID)
}

// fetchListingDocument fetches a listing from Postgres and converts to ES document
func (c *ESSyncConsumer) fetchListingDocument(ctx context.Context, listingID uuid.UUID) (*ListingDocument, error) {
	var listing struct {
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

	err := c.db.WithContext(ctx).Table("listings").
		Select("id, title, description, price, currency, category, location, condition, seller_id, status, view_count, search_click_count, search_impression_count, created_at, updated_at, last_searched_at, last_viewed_at, images").
		Where("id = ?", listingID).
		First(&listing).Error

	if err != nil {
		return nil, err
	}

	doc := &ListingDocument{
		ID:                    listing.ID.String(),
		Title:                 listing.Title,
		Description:           listing.Description,
		Price:                 listing.Price,
		Currency:              listing.Currency,
		Category:              listing.Category,
		Location:              listing.Location,
		Condition:             listing.Condition,
		SellerID:              listing.SellerID.String(),
		SellerName:            listing.SellerName,
		Status:                listing.Status,
		ViewCount:             listing.ViewCount,
		SearchClickCount:      listing.SearchClickCount,
		SearchImpressionCount: listing.SearchImpressionCount,
		CreatedAt:             listing.CreatedAt,
		UpdatedAt:             listing.UpdatedAt,
		LastSearchedAt:        listing.LastSearchedAt,
		LastViewedAt:          listing.LastViewedAt,
		Images:                listing.Images,
		PopularityScore:       float64(listing.ViewCount) + float64(listing.SearchClickCount)*2,
		RecencyScore:          c.esService.calculateRecencyScore(listing.CreatedAt),
	}

	doc.RankScore = c.esService.calculateRankScore(doc)

	return doc, nil
}

// SyncListingByID manually syncs a specific listing to ES (for manual reindex)
func (c *ESSyncConsumer) SyncListingByID(ctx context.Context, listingID uuid.UUID) error {
	doc, err := c.fetchListingDocument(ctx, listingID)
	if err != nil {
		return err
	}

	return c.esService.IndexDocument(ctx, doc)
}

// DeleteListingByID removes a listing from ES
func (c *ESSyncConsumer) DeleteListingByID(ctx context.Context, listingID uuid.UUID) error {
	return c.esService.DeleteDocument(ctx, listingID.String())
}

// ════════════════════════════════════════════════════════════════════════════
// Event Payloads
// ════════════════════════════════════════════════════════════════════════════

// ListingEvent represents a listing change event
type ListingEvent struct {
	EventType string    `json:"event_type"`
	ListingID uuid.UUID `json:"listing_id"`
	Timestamp time.Time `json:"timestamp"`
}

// MarshalBinary implements encoding.BinaryMarshaler for Kafka
func (e *ListingEvent) MarshalBinary() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler for Kafka
func (e *ListingEvent) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, e)
}
