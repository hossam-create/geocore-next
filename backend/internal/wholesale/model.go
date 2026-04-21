package wholesale

import (
	"time"

	"github.com/google/uuid"
)

// ── Wholesale Listing ─────────────────────────────────────────────────────────
// A wholesale listing is different from a regular listing:
// - Minimum Order Quantity (MOQ)
// - Tiered pricing (price per unit decreases with quantity)
// - Bulk availability tracking
// - Seller must be wholesale-verified

type WholesaleListing struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SellerID       uuid.UUID `gorm:"type:uuid;not null;index" json:"seller_id"`
	Title          string    `gorm:"size:256;not null" json:"title"`
	Description    string    `gorm:"type:text" json:"description,omitempty"`
	CategorySlug   string    `gorm:"size:128;index" json:"category_slug"`
	Images         []string  `gorm:"type:jsonb;serializer:json;default:'[]'" json:"images"`

	// Pricing
	UnitPriceCents     int64   `gorm:"not null" json:"unit_price_cents"`          // base price per unit
	Currency           string  `gorm:"size:3;not null;default:'EGP'" json:"currency"`
	TierPricing        []PriceTier `gorm:"type:jsonb;serializer:json;default:'[]'" json:"tier_pricing"` // quantity-based tiers

	// Quantity
	MOQ               int     `gorm:"not null;default:1" json:"moq"`             // minimum order quantity
	MaxOrderQuantity  int     `gorm:"default:0" json:"max_order_quantity"`       // 0 = unlimited
	AvailableUnits    int     `gorm:"not null;default:0" json:"available_units"`  // total stock
	UnitsPerLot       int     `gorm:"default:1" json:"units_per_lot"`            // e.g. 12 items per carton

	// Shipping
	ShippingPerUnitCents int64 `gorm:"default:0" json:"shipping_per_unit_cents"`
	FreeShippingMOQ      int   `gorm:"default:0" json:"free_shipping_moq"`       // free shipping above this MOQ
	LeadTimeDays         int   `gorm:"default:3" json:"lead_time_days"`           // fulfillment time

	// Status
	Status           string    `gorm:"size:32;not null;default:'active';index" json:"status"` // active, paused, closed
	IsVerified       bool      `gorm:"default:false" json:"is_verified"`          // admin-verified wholesale listing
	ViewsCount       int       `gorm:"default:0" json:"views_count"`
	OrdersCount      int       `gorm:"default:0" json:"orders_count"`

	CreatedAt        time.Time `gorm:"index" json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func (WholesaleListing) TableName() string { return "wholesale_listings" }

// PriceTier defines quantity-based pricing tiers.
type PriceTier struct {
	MinQuantity    int   `json:"min_quantity"`
	MaxQuantity    int   `json:"max_quantity"`    // 0 = unlimited
	UnitPriceCents int64 `json:"unit_price_cents"`
}

// ── Wholesale Order ────────────────────────────────────────────────────────────

type WholesaleOrder struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	BuyerID         uuid.UUID `gorm:"type:uuid;not null;index" json:"buyer_id"`
	SellerID        uuid.UUID `gorm:"type:uuid;not null;index" json:"seller_id"`
	ListingID       uuid.UUID `gorm:"type:uuid;not null;index" json:"listing_id"`
	OrderID         *uuid.UUID `gorm:"type:uuid;index" json:"order_id,omitempty"` // link to main orders table

	// Order details
	Quantity        int       `gorm:"not null" json:"quantity"`
	UnitPriceCents  int64     `gorm:"not null" json:"unit_price_cents"`
	TotalPriceCents int64     `gorm:"not null" json:"total_price_cents"`
	Currency        string    `gorm:"size:3;not null;default:'EGP'" json:"currency"`
	ShippingCents   int64     `gorm:"default:0" json:"shipping_cents"`

	// Status
	Status          string    `gorm:"size:32;not null;default:'pending';index" json:"status"` // pending, confirmed, shipped, delivered, cancelled
	Notes           string    `gorm:"type:text" json:"notes,omitempty"`

	// Seller response
	SellerResponse  string    `gorm:"size:32" json:"seller_response,omitempty"` // accepted, rejected, counter_offer
	CounterOfferCents int64   `gorm:"default:0" json:"counter_offer_cents,omitempty"`

	CreatedAt       time.Time `gorm:"index" json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (WholesaleOrder) TableName() string { return "wholesale_orders" }

// ── Wholesale Seller Profile ───────────────────────────────────────────────────

type WholesaleSeller struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	CompanyName     string    `gorm:"size:256;not null" json:"company_name"`
	TaxID           string    `gorm:"size:128" json:"tax_id,omitempty"`
	BusinessType    string    `gorm:"size:64" json:"business_type,omitempty"` // manufacturer, distributor, importer
	Categories      []string  `gorm:"type:jsonb;serializer:json;default:'[]'" json:"categories"`
	MinOrderValueCents int64  `gorm:"default:0" json:"min_order_value_cents"`

	// Verification
	IsVerified       bool      `gorm:"default:false;index" json:"is_verified"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
	VerifiedBy       *uuid.UUID `gorm:"type:uuid" json:"verified_by,omitempty"`

	// Stats
	TotalListings   int       `gorm:"default:0" json:"total_listings"`
	TotalOrders     int       `gorm:"default:0" json:"total_orders"`
	Rating          float64   `gorm:"type:numeric(3,2);default:0" json:"rating"`

	Status          string    `gorm:"size:32;not null;default:'pending';index" json:"status"` // pending, active, suspended
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (WholesaleSeller) TableName() string { return "wholesale_sellers" }
