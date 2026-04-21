package requests

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RequestStatus represents the state of a product request
type RequestStatus string

const (
	StatusOpen      RequestStatus = "open"
	StatusFulfilled RequestStatus = "fulfilled"
	StatusExpired   RequestStatus = "expired"
	StatusCancelled RequestStatus = "cancelled"
)

// ProductRequest represents a buyer's demand for a product they can't find
type ProductRequest struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	CategoryID  *uuid.UUID     `gorm:"type:uuid;index" json:"category_id,omitempty"`
	Budget      *float64       `gorm:"type:decimal(15,2)" json:"budget,omitempty"`
	Currency    string         `gorm:"size:3;not null;default:'AED'" json:"currency"`
	Status      RequestStatus  `gorm:"type:varchar(20);not null;default:'open'" json:"status"`
	FulfilledBy *uuid.UUID     `gorm:"type:uuid" json:"fulfilled_by,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Associations (populated in responses)
	ResponseCount int    `gorm:"-" json:"response_count,omitempty"`
	UserName      string `gorm:"-" json:"user_name,omitempty"`
	CategoryName  string `gorm:"-" json:"category_name,omitempty"`
}

func (ProductRequest) TableName() string { return "product_requests" }

func (pr *ProductRequest) BeforeCreate(tx *gorm.DB) error {
	if pr.ID == uuid.Nil {
		pr.ID = uuid.New()
	}
	return nil
}

// ProductRequestResponse represents a seller's response to a product request
type ProductRequestResponse struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	RequestID uuid.UUID  `gorm:"type:uuid;not null;index" json:"request_id"`
	SellerID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"seller_id"`
	ListingID *uuid.UUID `gorm:"type:uuid" json:"listing_id,omitempty"`
	Message   string     `gorm:"type:text;not null" json:"message"`
	CreatedAt time.Time  `json:"created_at"`

	// Associations
	SellerName string `gorm:"-" json:"seller_name,omitempty"`
}

func (ProductRequestResponse) TableName() string { return "product_request_responses" }

func (r *ProductRequestResponse) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
