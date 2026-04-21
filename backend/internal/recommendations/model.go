package recommendations

import (
	"time"

	"github.com/google/uuid"
)

// Context defines where recommendations are being requested from.
type Context string

const (
	ContextHomePage     Context = "home_page"
	ContextProductPage  Context = "product_page"
	ContextCartPage     Context = "cart_page"
	ContextCheckout     Context = "checkout"
	ContextSearch       Context = "search_results"
	ContextCategory     Context = "category_page"
)

// Algorithm identifies which algorithm produced a recommendation.
type Algorithm string

const (
	AlgoTrending           Algorithm = "trending"
	AlgoPersonalized       Algorithm = "personalized"
	AlgoSimilarItems       Algorithm = "similar_items"
	AlgoFrequentlyBought   Algorithm = "frequently_bought_together"
	AlgoRecentlyViewed     Algorithm = "recently_viewed"
	AlgoNewArrivals        Algorithm = "new_arrivals"
)

// UserInteraction tracks a user's interaction with a listing.
type UserInteraction struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index"                        json:"user_id"`
	ListingID       uuid.UUID `gorm:"type:uuid;not null;index"                        json:"listing_id"`
	InteractionType string    `gorm:"type:varchar(30);not null;index"                 json:"interaction_type"` // view, click, purchase, wishlist, search
	CategoryID      *uuid.UUID `gorm:"type:uuid"                                      json:"category_id,omitempty"`
	SessionID       string    `gorm:"type:varchar(100)"                               json:"session_id,omitempty"`
	DwellTimeMs     int       `gorm:"default:0"                                       json:"dwell_time_ms"`
	Converted       bool      `gorm:"default:false"                                   json:"converted"`
	CreatedAt       time.Time `json:"created_at"`
}

func (UserInteraction) TableName() string { return "recommendation_interactions" }

// Recommendation is a single recommendation result returned to the client.
type Recommendation struct {
	ListingID  uuid.UUID `json:"listing_id"`
	Score      float64   `json:"score"`
	Reason     string    `json:"reason"`
	Algorithm  Algorithm `json:"algorithm"`
}

// Response is the API response for recommendation requests.
type Response struct {
	Items      []Recommendation `json:"items"`
	Context    Context          `json:"context"`
	Count      int              `json:"count"`
	CachedAt   *time.Time       `json:"cached_at,omitempty"`
	DurationMs int64            `json:"duration_ms"`
}
