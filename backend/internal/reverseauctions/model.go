package reverseauctions

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RequestStatus string

const (
	RequestOpen      RequestStatus = "open"
	RequestClosed    RequestStatus = "closed"
	RequestFulfilled RequestStatus = "fulfilled"
	RequestExpired   RequestStatus = "expired"
)

type OfferStatus string

const (
	OfferPending   OfferStatus = "pending"
	OfferAccepted  OfferStatus = "accepted"
	OfferRejected  OfferStatus = "rejected"
	OfferWithdrawn OfferStatus = "withdrawn"
	OfferCountered OfferStatus = "countered"
	OfferExpired   OfferStatus = "expired"
)

type ReverseAuctionRequest struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	BuyerID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"buyer_id"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	CategoryID  *uuid.UUID     `gorm:"type:uuid" json:"category_id,omitempty"`
	MaxBudget   *float64       `gorm:"type:numeric(15,2)" json:"max_budget,omitempty"`
	Deadline    time.Time      `gorm:"not null" json:"deadline"`
	Status      RequestStatus  `gorm:"size:20;not null;default:'open';index" json:"status"`
	Images      string         `gorm:"type:jsonb;default:'[]'" json:"images"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	Offers []ReverseAuctionOffer `gorm:"foreignKey:RequestID" json:"offers,omitempty"`
}

func (ReverseAuctionRequest) TableName() string { return "reverse_auction_requests" }

type ReverseAuctionOffer struct {
	ID           uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	RequestID    uuid.UUID   `gorm:"type:uuid;not null;index" json:"request_id"`
	SellerID     uuid.UUID   `gorm:"type:uuid;not null;index" json:"seller_id"`
	Price        float64     `gorm:"type:numeric(15,2);not null" json:"price"`
	Description  string      `gorm:"type:text" json:"description,omitempty"`
	DeliveryDays *int        `json:"delivery_days,omitempty"`
	CounterPrice *float64    `gorm:"type:numeric(15,2)" json:"counter_price,omitempty"`
	Message      string      `gorm:"type:text" json:"message,omitempty"`
	ExpiresAt    *time.Time  `json:"expires_at,omitempty"`
	RespondedAt  *time.Time  `json:"responded_at,omitempty"`
	Status       OfferStatus `gorm:"size:20;not null;default:'pending'" json:"status"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (ReverseAuctionOffer) TableName() string { return "reverse_auction_offers" }
