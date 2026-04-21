package deals

import (
	"time"

	"github.com/google/uuid"
)

// DealStatus represents the status of a deal
type DealStatus string

const (
	DealStatusScheduled DealStatus = "scheduled"
	DealStatusActive    DealStatus = "active"
	DealStatusExpired   DealStatus = "expired"
	DealStatusCancelled DealStatus = "cancelled"
)

// Deal represents a promotional deal on a listing
type Deal struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"listing_id"`
	SellerID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"seller_id"`
	OriginalPrice float64    `gorm:"not null" json:"original_price"`
	DealPrice     float64    `gorm:"not null" json:"deal_price"`
	DiscountPct   int        `gorm:"not null" json:"discount_pct"` // Calculated: (original - deal) / original * 100
	StartAt       time.Time  `gorm:"not null" json:"start_at"`
	EndAt         time.Time  `gorm:"not null" json:"end_at"`
	Status        DealStatus `gorm:"type:varchar(20);default:'scheduled'" json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Query-only projection fields (joined from listings/users)
	ListingTitle string `gorm:"column:listing_title;->" json:"listing_title,omitempty"`
	ListingImage string `gorm:"column:listing_image;->" json:"listing_image,omitempty"`
	SellerName   string `gorm:"column:seller_name;->" json:"seller_name,omitempty"`
	Currency     string `gorm:"column:currency;->" json:"currency,omitempty"`

	// Relations
	Listing *ListingInfo `gorm:"foreignKey:ListingID" json:"listing,omitempty"`
}

// ListingInfo contains basic listing details for deal responses
type ListingInfo struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Title      string    `gorm:"not null" json:"title"`
	Slug       string    `json:"slug"`
	Images     []string  `gorm:"type:text[]" json:"images"`
	Currency   string    `gorm:"default:'AED'" json:"currency"`
	Condition  string    `json:"condition"`
	CategoryID uuid.UUID `json:"category_id"`
}

// DealCreateRequest is the payload for creating a new deal
type DealCreateRequest struct {
	ListingID string  `json:"listing_id" binding:"required,uuid"`
	DealPrice float64 `json:"deal_price" binding:"required,gt=0"`
	StartAt   string  `json:"start_at" binding:"required"`
	EndAt     string  `json:"end_at" binding:"required"`
}

// DealPublicResponse is the public-facing deal response
type DealPublicResponse struct {
	ID            string    `json:"id"`
	ListingID     string    `json:"listing_id"`
	ListingTitle  string    `json:"listing_title"`
	ListingImage  string    `json:"listing_image"`
	SellerID      string    `json:"seller_id"`
	SellerName    string    `json:"seller_name"`
	OriginalPrice float64   `json:"original_price"`
	DealPrice     float64   `json:"deal_price"`
	DiscountPct   int       `json:"discount_pct"`
	Currency      string    `json:"currency"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	Status        string    `json:"status"`
	TimeRemaining string    `json:"time_remaining"`
}

// TableName sets the table name for Deal
func (Deal) TableName() string {
	return "deals"
}

// CalculateDiscount computes the discount percentage
func CalculateDiscount(original, dealPrice float64) int {
	if original <= 0 {
		return 0
	}
	discount := (original - dealPrice) / original * 100
	return int(discount)
}

// IsExpired checks if the deal has ended
func (d *Deal) IsExpired() bool {
	return time.Now().After(d.EndAt)
}

// IsActive checks if the deal is currently active
func (d *Deal) IsActive() bool {
	now := time.Now()
	return now.After(d.StartAt) && now.Before(d.EndAt)
}

// IsScheduled checks if the deal is scheduled for the future
func (d *Deal) IsScheduled() bool {
	return time.Now().Before(d.StartAt)
}

// GetStatus returns the computed status based on time
func (d *Deal) GetStatus() DealStatus {
	if d.Status == DealStatusCancelled {
		return DealStatusCancelled
	}
	if d.IsExpired() {
		return DealStatusExpired
	}
	if d.IsActive() {
		return DealStatusActive
	}
	return DealStatusScheduled
}
