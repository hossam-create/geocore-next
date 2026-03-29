package auctions

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuctionType defines the type of auction
type AuctionType string

const (
	AuctionTypeStandard AuctionType = "standard" // Normal ascending auction
	AuctionTypeDutch    AuctionType = "dutch"    // Price decreases over time
	AuctionTypeReverse  AuctionType = "reverse"  // Lowest bid wins (for services)
	AuctionTypeSealed   AuctionType = "sealed"   // Hidden bids until end
)

// AuctionStatus defines auction lifecycle states
type AuctionStatus string

const (
	StatusScheduled AuctionStatus = "scheduled" // Not yet started
	StatusActive    AuctionStatus = "active"    // Live and accepting bids
	StatusEnded     AuctionStatus = "ended"     // Time expired
	StatusSold      AuctionStatus = "sold"      // Buy Now or Reserve met
	StatusCancelled AuctionStatus = "cancelled" // Cancelled by seller/admin
)

// Anti-sniping configuration
const (
	AntiSnipeWindow    = 2 * time.Minute // If bid in last 2 minutes
	AntiSnipeExtension = 5 * time.Minute // Extend by 5 minutes
	MaxExtensions      = 10              // Maximum number of extensions
)

type Auction struct {
	ID           uuid.UUID     `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID    uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex" json:"listing_id"`
	SellerID     uuid.UUID     `gorm:"type:uuid;not null;index" json:"seller_id"`
	Type         AuctionType   `gorm:"type:varchar(20);default:'standard'" json:"type"`
	StartPrice   float64       `gorm:"not null" json:"start_price"`
	ReservePrice *float64      `json:"reserve_price,omitempty"`
	BuyNowPrice  *float64      `json:"buy_now_price,omitempty"`
	CurrentBid   float64       `gorm:"default:0" json:"current_bid"`
	BidCount     int           `gorm:"default:0" json:"bid_count"`
	WinnerID     *uuid.UUID    `gorm:"type:uuid" json:"winner_id,omitempty"`
	Status       AuctionStatus `gorm:"type:varchar(20);default:'active';index" json:"status"`
	StartsAt     time.Time     `json:"starts_at"`
	EndsAt       time.Time     `gorm:"index" json:"ends_at"`
	Currency     string        `gorm:"default:USD" json:"currency"`

	// Anti-sniping
	AntiSnipeEnabled bool `gorm:"default:true" json:"anti_snipe_enabled"`
	ExtensionCount   int  `gorm:"default:0" json:"extension_count"`

	// Dutch auction specific
	DutchStartPrice    *float64 `json:"dutch_start_price,omitempty"`     // Initial high price
	DutchEndPrice      *float64 `json:"dutch_end_price,omitempty"`       // Minimum price
	DutchPriceDropRate *float64 `json:"dutch_price_drop_rate,omitempty"` // Price drop per interval
	DutchDropInterval  *int     `json:"dutch_drop_interval,omitempty"`   // Minutes between drops

	// Proxy bidding
	ProxyBidEnabled bool `gorm:"default:true" json:"proxy_bid_enabled"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Bids      []Bid          `gorm:"foreignKey:AuctionID" json:"bids,omitempty"`
}

// GetCurrentDutchPrice calculates current price for Dutch auction
func (a *Auction) GetCurrentDutchPrice() float64 {
	if a.Type != AuctionTypeDutch || a.DutchStartPrice == nil || a.DutchEndPrice == nil {
		return a.StartPrice
	}

	elapsed := time.Since(a.StartsAt)
	intervalMinutes := 5 // default
	if a.DutchDropInterval != nil {
		intervalMinutes = *a.DutchDropInterval
	}

	intervals := int(elapsed.Minutes()) / intervalMinutes
	dropRate := (*a.DutchStartPrice - *a.DutchEndPrice) / float64(int(a.EndsAt.Sub(a.StartsAt).Minutes())/intervalMinutes)
	if a.DutchPriceDropRate != nil {
		dropRate = *a.DutchPriceDropRate
	}

	currentPrice := *a.DutchStartPrice - (dropRate * float64(intervals))
	if currentPrice < *a.DutchEndPrice {
		return *a.DutchEndPrice
	}
	return currentPrice
}

type Bid struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AuctionID      uuid.UUID `gorm:"type:uuid;not null;index" json:"auction_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Amount         float64   `gorm:"not null" json:"amount"`
	IsAuto         bool      `gorm:"default:false" json:"is_auto"`
	MaxAmount      *float64  `json:"max_amount,omitempty"` // for proxy/auto-bidding
	IdempotencyKey *string   `gorm:"type:varchar(100);uniqueIndex" json:"idempotency_key,omitempty"`
	PlacedAt       time.Time `gorm:"index" json:"placed_at"`
}

// ProxyBid represents automatic bidding configuration
type ProxyBid struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AuctionID uuid.UUID `gorm:"type:uuid;not null;index" json:"auction_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	MaxAmount float64   `gorm:"not null" json:"max_amount"`
	Increment float64   `gorm:"default:1" json:"increment"` // Minimum bid increment
	IsActive  bool      `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
