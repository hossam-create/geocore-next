package listings

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListingVariant represents a purchasable variant of a listing (color, size, etc.)
type ListingVariant struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"listing_id"`
	SKU         string         `gorm:"size:100" json:"sku,omitempty"`
	Title       string         `gorm:"size:200;not null" json:"title"` // e.g. "Red / Large"
	Price       float64        `json:"price"`
	CompareAt   float64        `json:"compare_at_price,omitempty"` // original price for strikethrough
	Stock       int            `gorm:"default:0" json:"stock"`
	IsDefault   bool           `gorm:"default:false" json:"is_default"`
	Attributes  string         `gorm:"type:jsonb;default:'{}'" json:"attributes"` // {"color":"Red","size":"L"}
	ImageURL    string         `gorm:"size:500" json:"image_url,omitempty"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ListingVariant) TableName() string { return "listing_variants" }

// ListingQA stores questions and answers about a listing.
type ListingQA struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"listing_id"`
	Question   string         `gorm:"type:text;not null" json:"question"`
	Answer     string         `gorm:"type:text" json:"answer,omitempty"`
	AskedBy    uuid.UUID      `gorm:"type:uuid;not null" json:"asked_by"`
	AnsweredBy *uuid.UUID     `gorm:"type:uuid" json:"answered_by,omitempty"`
	IsPublic   bool           `gorm:"default:true" json:"is_public"`
	HelpfulYes int            `gorm:"default:0" json:"helpful_yes"`
	HelpfulNo  int            `gorm:"default:0" json:"helpful_no"`
	CreatedAt  time.Time      `json:"created_at"`
	AnsweredAt *time.Time     `json:"answered_at,omitempty"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ListingQA) TableName() string { return "listing_qa" }

// ListingFeedback stores buyer reviews/feedback for a listing.
type ListingFeedback struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"listing_id"`
	SellerID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	BuyerID     uuid.UUID      `gorm:"type:uuid;not null" json:"buyer_id"`
	OrderID     uuid.UUID      `gorm:"type:uuid" json:"order_id,omitempty"`
	Rating      int            `gorm:"not null" json:"rating"` // 1-5
	Title       string         `gorm:"size:200" json:"title,omitempty"`
	Review      string         `gorm:"type:text" json:"review,omitempty"`
	IsAnonymous bool           `gorm:"default:false" json:"is_anonymous"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ListingFeedback) TableName() string { return "listing_feedback" }
