package listings

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Category struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ParentID  *uuid.UUID `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	NameEn    string     `gorm:"not null" json:"name_en"`
	NameAr    string     `json:"name_ar"`
	Slug      string     `gorm:"uniqueIndex;not null" json:"slug"`
	Icon      string     `json:"icon"`
	SortOrder int        `gorm:"default:0" json:"sort_order"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	Children  []Category `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

type Listing struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	CategoryID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"category_id"`
	Title        string         `gorm:"not null" json:"title"`
	Description  string         `gorm:"type:text" json:"description"`
	Price        *float64       `json:"price,omitempty"`
	Currency     string         `gorm:"default:USD" json:"currency"`
	PriceType    string         `gorm:"default:fixed" json:"price_type"` // fixed | negotiable | free | contact
	Condition    string         `json:"condition"`                        // new | used | refurbished
	Status       string         `gorm:"default:active;index" json:"status"` // draft | pending | active | sold | expired
	Type         string         `gorm:"default:sell" json:"type"`            // sell | buy | rent | auction | service
	Country      string         `gorm:"index" json:"country"`
	City         string         `gorm:"index" json:"city"`
	Address      string         `json:"address,omitempty"`
	Latitude     *float64       `json:"latitude,omitempty"`
	Longitude    *float64       `json:"longitude,omitempty"`
	ViewCount    int            `gorm:"default:0" json:"view_count"`
	FavoriteCount int           `gorm:"default:0" json:"favorite_count"`
	IsFeatured   bool           `gorm:"default:false;index" json:"is_featured"`
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"`
	SoldAt       *time.Time     `json:"sold_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	// Relations
	Images   []ListingImage `gorm:"foreignKey:ListingID" json:"images,omitempty"`
	Category *Category      `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
}

type ListingImage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index" json:"listing_id"`
	URL       string    `gorm:"not null" json:"url"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	IsCover   bool      `gorm:"default:false" json:"is_cover"`
}

type Favorite struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index" json:"listing_id"`
	CreatedAt time.Time `json:"created_at"`
}
