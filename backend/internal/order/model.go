package order

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DeliveryType string

const (
	DeliveryTypeStandard      DeliveryType = "STANDARD"
	DeliveryTypeCrowdshipping DeliveryType = "CROWDSHIPPING"
	DeliveryTypePickup        DeliveryType = "PICKUP"
)

// OrderStatus represents the state of an order
type OrderStatus string

const (
	StatusPending    OrderStatus = "pending"
	StatusConfirmed  OrderStatus = "confirmed"
	StatusProcessing OrderStatus = "processing"
	StatusShipped    OrderStatus = "shipped"
	StatusDelivered  OrderStatus = "delivered"
	StatusCompleted  OrderStatus = "completed"
	StatusCancelled  OrderStatus = "cancelled"
	StatusDisputed   OrderStatus = "disputed"
	StatusRefunded   OrderStatus = "refunded"
)

// Order represents a purchase transaction between buyer and seller
type Order struct {
	ID                        uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	BuyerID                   uuid.UUID      `gorm:"type:uuid;not null;index" json:"buyer_id"`
	SellerID                  uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	PaymentIntentID           string         `gorm:"index" json:"payment_intent_id,omitempty"`
	PaymentID                 *uuid.UUID     `gorm:"type:uuid" json:"payment_id,omitempty"`
	Status                    OrderStatus    `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	StatusHistory             []StatusChange `gorm:"type:jsonb;serializer:json" json:"status_history,omitempty"`
	Subtotal                  float64        `gorm:"type:decimal(15,2);not null" json:"subtotal"`
	PlatformFee               float64        `gorm:"type:decimal(15,2);not null;default:0" json:"platform_fee"`
	PaymentFee                float64        `gorm:"type:decimal(15,2);not null;default:0" json:"payment_fee"`
	Total                     float64        `gorm:"type:decimal(15,2);not null" json:"total"`
	Currency                  string         `gorm:"type:varchar(3);not null;default:'AED'" json:"currency"`
	ShippingAddress           *Address       `gorm:"type:jsonb;serializer:json" json:"shipping_address,omitempty"`
	TrackingNumber            string         `json:"tracking_number,omitempty"`
	Carrier                   string         `json:"carrier,omitempty"`
	ShippedAt                 *time.Time     `json:"shipped_at,omitempty"`
	DeliveredAt               *time.Time     `json:"delivered_at,omitempty"`
	Notes                     string         `json:"notes,omitempty"`
	DisputeReason             string         `json:"dispute_reason,omitempty"`
	DisputeEvidence           string         `json:"dispute_evidence,omitempty"`
	IsGuestOrder              bool           `gorm:"not null;default:false;index" json:"is_guest_order"`
	GuestEmail                *string        `gorm:"type:varchar(255)" json:"guest_email,omitempty"`
	GuestFirstName            *string        `gorm:"type:varchar(120)" json:"guest_first_name,omitempty"`
	GuestLastName             *string        `gorm:"type:varchar(120)" json:"guest_last_name,omitempty"`
	GuestPhone                *string        `gorm:"type:varchar(40)" json:"guest_phone,omitempty"`
	GuestToken                *uuid.UUID     `gorm:"type:uuid;index" json:"guest_token,omitempty"`
	GuestTokenFingerprintHash *string        `gorm:"type:varchar(64)" json:"-"`
	DeliveryType              DeliveryType   `gorm:"type:varchar(20);not null;default:'STANDARD'" json:"delivery_type"`
	ConfirmedAt               *time.Time     `json:"confirmed_at,omitempty"`
	CompletedAt               *time.Time     `json:"completed_at,omitempty"`
	CancelledAt               *time.Time     `json:"cancelled_at,omitempty"`
	CancelledReason           string         `json:"cancelled_reason,omitempty"`
	CreatedAt                 time.Time      `json:"created_at"`
	UpdatedAt                 time.Time      `json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Items  []OrderItem `gorm:"foreignKey:OrderID;references:ID" json:"items,omitempty"`
	Buyer  UserInfo    `gorm:"-"                                json:"buyer,omitempty"`
	Seller UserInfo    `gorm:"-"                                json:"seller,omitempty"`
}

// OrderItem represents a line item in an order
type OrderItem struct {
	ID         uuid.UUID              `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	OrderID    uuid.UUID              `gorm:"type:uuid;not null;index" json:"order_id"`
	ListingID  *uuid.UUID             `gorm:"type:uuid;index" json:"listing_id,omitempty"`
	AuctionID  *uuid.UUID             `gorm:"type:uuid;index" json:"auction_id,omitempty"`
	Title      string                 `gorm:"type:varchar(200);not null" json:"title"`
	Quantity   int                    `gorm:"not null;default:1" json:"quantity"`
	UnitPrice  float64                `gorm:"type:decimal(15,2);not null" json:"unit_price"`
	TotalPrice float64                `gorm:"type:decimal(15,2);not null" json:"total_price"`
	Condition  string                 `json:"condition,omitempty"`
	Attributes map[string]interface{} `gorm:"-" json:"attributes,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`

	// Relationships
	Listing *ListingInfo `gorm:"-" json:"listing,omitempty"`
}

// StatusChange represents a single status transition
type StatusChange struct {
	Status OrderStatus `json:"status"`
	At     time.Time   `json:"at"`
	By     string      `json:"by,omitempty"` // user_id or "system"
	Note   string      `json:"note,omitempty"`
}

// Address represents a shipping address
type Address struct {
	Name    string `json:"name"`
	Line1   string `json:"line1"`
	Line2   string `json:"line2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state,omitempty"`
	Country string `json:"country"`
	Zip     string `json:"zip,omitempty"`
	Phone   string `json:"phone,omitempty"`
}

// UserInfo is a lightweight user representation for order responses
type UserInfo struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Email  string    `json:"email,omitempty"`
	Avatar string    `json:"avatar,omitempty"`
	Rating float64   `json:"rating,omitempty"`
}

// ListingInfo is a lightweight listing representation for order items
type ListingInfo struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Image    string    `json:"image,omitempty"`
	Category string    `json:"category,omitempty"`
}

// TableName returns the table name for Order
func (Order) TableName() string {
	return "orders"
}

// TableName returns the table name for OrderItem
func (OrderItem) TableName() string {
	return "order_items"
}

// BeforeCreate hook
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// BeforeCreate hook for OrderItem
func (oi *OrderItem) BeforeCreate(tx *gorm.DB) error {
	if oi.ID == uuid.Nil {
		oi.ID = uuid.New()
	}
	return nil
}
