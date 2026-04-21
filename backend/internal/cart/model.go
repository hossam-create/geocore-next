package cart

import "time"

const (
	cartTTL = 7 * 24 * time.Hour
)

// CartItem represents one listing in a user's cart.
type CartItem struct {
	ListingID string  `json:"listing_id"`
	Title     string  `json:"title"`
	ImageURL  string  `json:"image_url,omitempty"`
	Currency  string  `json:"currency"`
	UnitPrice float64 `json:"unit_price"`
	Quantity  int     `json:"quantity"`
	Subtotal  float64 `json:"subtotal"`
}

// Cart represents a user's current cart state.
type Cart struct {
	Items     []CartItem `json:"items"`
	ItemCount int        `json:"item_count"`
	Total     float64    `json:"total"`
	Currency  string     `json:"currency,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}
