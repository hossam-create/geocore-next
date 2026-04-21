package listings

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ParentID     *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	NameEn       string     `gorm:"not null" json:"name_en"`
	NameAr       string     `gorm:"not null" json:"name_ar"`
	Slug         string     `gorm:"uniqueIndex;not null" json:"slug"`
	Description  string     `json:"description,omitempty"`
	Icon         string     `json:"icon"`
	IconURL      string     `json:"icon_url,omitempty"`
	ImageURL     string     `json:"image_url,omitempty"`
	Color        string     `gorm:"size:7" json:"color,omitempty"` // eBay category color hex e.g. "#E53238"
	SortOrder    int        `gorm:"default:0" json:"sort_order"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	IsLeaf       bool       `gorm:"default:false" json:"is_leaf"`   // true = no subcategories
	ListingCount int        `gorm:"default:0" json:"listing_count"` // cached count
	// Sprint 18: Discovery/Tree — computed by BackfillCategoryTree
	Level int    `gorm:"default:0;index" json:"level"`
	Path  string `gorm:"size:500;index" json:"path"` // e.g. "electronics/phones/smartphones"
	// Derived / relations
	Children []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

// SellerInfo holds public seller data embedded in listing responses.
// It maps to the users table (read-only, no AutoMigrate).
type SellerInfo struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string    `json:"name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Rating      float64   `json:"rating"`
	ReviewCount int       `json:"review_count"`
	SoldCount   int       `json:"sold_count"`
	IsVerified  bool      `json:"is_verified"`
	Location    string    `json:"location,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (SellerInfo) TableName() string { return "users" }

// ListingType defines the trading mode for a listing
type ListingType string

const (
	ListingTypeBuyNow      ListingType = "buy_now"
	ListingTypeNegotiation ListingType = "negotiation"
	ListingTypeAuction     ListingType = "auction"
	ListingTypeHybrid      ListingType = "hybrid" // Buy Now + Offer
)

// ListingTradeConfig controls trading behaviour for a listing.
// Stored as JSONB in the trade_config column.
type ListingTradeConfig struct {
	BuyNowEnabled     bool    `json:"buy_now_enabled"`
	OfferEnabled      bool    `json:"offer_enabled"`
	AuctionEnabled    bool    `json:"auction_enabled"`
	MinOfferPercent   float64 `json:"min_offer_percent"`   // e.g. 0.7 = 70% of listing price
	AutoAcceptPercent float64 `json:"auto_accept_percent"` // e.g. 0.95 = 95% of listing price
	OfferExpiryHours  int     `json:"offer_expiry_hours"`  // default 48
}

// DefaultTradeConfig returns sensible defaults based on listing type.
func DefaultTradeConfig(lt ListingType) ListingTradeConfig {
	cfg := ListingTradeConfig{
		MinOfferPercent:   0.7,
		AutoAcceptPercent: 0.95,
		OfferExpiryHours:  48,
	}
	switch lt {
	case ListingTypeBuyNow:
		cfg.BuyNowEnabled = true
	case ListingTypeNegotiation:
		cfg.OfferEnabled = true
	case ListingTypeAuction:
		cfg.AuctionEnabled = true
	case ListingTypeHybrid:
		cfg.BuyNowEnabled = true
		cfg.OfferEnabled = true
	}
	return cfg
}

type Listing struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	CategoryID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"category_id"`
	Title         string         `gorm:"not null" json:"title"`
	Description   string         `gorm:"type:text" json:"description"`
	Price         *float64       `json:"price,omitempty"`
	Currency      string         `gorm:"default:USD" json:"currency"`
	PriceType     string         `gorm:"default:fixed" json:"price_type"`                        // fixed | negotiable | free | contact
	Condition     string         `json:"condition"`                                              // new | used | refurbished
	Status        string         `gorm:"default:active;index" json:"status"`                     // draft | pending | active | sold | expired
	Type          string         `gorm:"default:sell" json:"type"`                               // sell | buy | rent | auction | service
	ListingType   ListingType    `gorm:"type:varchar(20);default:'buy_now'" json:"listing_type"` // buy_now | negotiation | auction | hybrid
	TradeConfig   string         `gorm:"type:jsonb;default:'{}'" json:"trade_config,omitempty"`  // ListingTradeConfig as JSON
	PriceCents    int64          `gorm:"default:0" json:"price_cents"`                           // integer-based price in cents
	Country       string         `gorm:"index" json:"country"`
	City          string         `gorm:"index" json:"city"`
	Address       string         `json:"address,omitempty"`
	Latitude      *float64       `json:"latitude,omitempty"`
	Longitude     *float64       `json:"longitude,omitempty"`
	ViewCount     int            `gorm:"default:0" json:"view_count"`
	FavoriteCount int            `gorm:"default:0" json:"favorite_count"`
	IsWatched     *bool          `gorm:"-" json:"is_watched,omitempty"`
	IsFeatured    bool           `gorm:"default:false;index" json:"is_featured"`
	ExpiresAt     *time.Time     `json:"expires_at,omitempty"`
	SoldAt        *time.Time     `json:"sold_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	CustomFields  string         `gorm:"type:jsonb;default:'{}'" json:"custom_fields,omitempty"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	// Relations
	Images   []ListingImage `gorm:"foreignKey:ListingID" json:"images,omitempty"`
	Category *Category      `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Seller   *SellerInfo    `gorm:"foreignKey:UserID;references:ID" json:"seller,omitempty"`
}

type ListingImage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index" json:"listing_id"`
	URL       string    `gorm:"not null" json:"url"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	IsCover   bool      `gorm:"default:false" json:"is_cover"`
}

// CategoryField defines a dynamic field for a category (e.g. year, mileage for vehicles).
type CategoryField struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	CategoryID  uuid.UUID `gorm:"type:uuid;not null;index" json:"category_id"`
	Name        string    `gorm:"size:100;not null" json:"name"`
	Label       string    `gorm:"size:100;not null" json:"label"`
	LabelEn     string    `gorm:"size:100" json:"label_en"`
	LabelAr     string    `gorm:"size:100" json:"label_ar,omitempty"`
	FieldType   string    `gorm:"size:20;not null" json:"field_type"` // text|number|select|boolean|range|date
	Options     string    `gorm:"type:jsonb;default:'[]'" json:"options"`
	IsRequired  bool      `gorm:"default:false" json:"is_required"`
	Placeholder string    `gorm:"size:200" json:"placeholder,omitempty"`
	Unit        string    `gorm:"size:20" json:"unit,omitempty"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetTradeConfig parses the JSONB trade_config field into a ListingTradeConfig.
// Returns DefaultTradeConfig(ListingTypeBuyNow) if the field is empty or invalid.
func (l *Listing) GetTradeConfig() ListingTradeConfig {
	if l.TradeConfig == "" || l.TradeConfig == "{}" {
		return DefaultTradeConfig(l.ListingType)
	}
	var cfg ListingTradeConfig
	if err := json.Unmarshal([]byte(l.TradeConfig), &cfg); err != nil {
		return DefaultTradeConfig(l.ListingType)
	}
	return cfg
}

type Favorite struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index" json:"listing_id"`
	CreatedAt time.Time `json:"created_at"`
}
