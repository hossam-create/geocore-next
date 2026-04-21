package crowdshipping

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Trip ────────────────────────────────────────────────────────────────────────

type TripStatus string

const (
	TripStatusActive    TripStatus = "active"
	TripStatusMatched   TripStatus = "matched"
	TripStatusInTransit TripStatus = "in_transit"
	TripStatusCompleted TripStatus = "completed"
	TripStatusCancelled TripStatus = "cancelled"
)

type Trip struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TravelerID      uuid.UUID  `gorm:"type:uuid;not null;index"                        json:"traveler_id"`
	OriginCountry   string     `gorm:"size:100;not null"                               json:"origin_country"`
	OriginCity      string     `gorm:"size:100;not null"                               json:"origin_city"`
	OriginAddress   string     `gorm:"type:text"                                       json:"origin_address,omitempty"`
	DestCountry     string     `gorm:"size:100;not null"                               json:"dest_country"`
	DestCity        string     `gorm:"size:100;not null"                               json:"dest_city"`
	DestAddress     string     `gorm:"type:text"                                       json:"dest_address,omitempty"`
	DepartureDate   time.Time  `gorm:"not null"                                        json:"departure_date"`
	ArrivalDate     time.Time  `gorm:"not null"                                        json:"arrival_date"`
	AvailableWeight float64    `gorm:"type:numeric(10,2);default:0"                    json:"available_weight"`
	MaxItems        int        `gorm:"default:5"                                       json:"max_items"`
	PricePerKg      float64    `gorm:"type:numeric(10,2);default:0"                    json:"price_per_kg"`
	BasePrice       float64    `gorm:"type:numeric(10,2);default:0"                    json:"base_price"`
	Currency        string     `gorm:"size:10;not null;default:'AED'"                  json:"currency"`
	Notes           string     `gorm:"type:text"                                       json:"notes,omitempty"`
	Frequency       string     `gorm:"size:20;not null;default:'one-time'"             json:"frequency"`
	Status          TripStatus `gorm:"size:50;not null;default:'active';index"         json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (Trip) TableName() string { return "trips" }

// ── Delivery Request ────────────────────────────────────────────────────────────

type DeliveryStatus string

const DeliveryTypeCrowdshipping = "CROWDSHIPPING"

const (
	DeliveryPending   DeliveryStatus = "pending"
	DeliveryMatched   DeliveryStatus = "matched"
	DeliveryAccepted  DeliveryStatus = "accepted"
	DeliveryLocked    DeliveryStatus = "locked"
	DeliveryPickedUp  DeliveryStatus = "picked_up"
	DeliveryInTransit DeliveryStatus = "in_transit"
	DeliveryDelivered DeliveryStatus = "delivered"
	DeliveryCancelled DeliveryStatus = "cancelled"
	DeliveryDisputed  DeliveryStatus = "disputed"
)

type DeliveryRequest struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	BuyerID         uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"buyer_id"`
	TripID          *uuid.UUID     `gorm:"type:uuid;index"                                 json:"trip_id,omitempty"`
	TravelerID      *uuid.UUID     `gorm:"type:uuid;index"                                 json:"traveler_id,omitempty"`
	ItemName        string         `gorm:"size:255;not null"                               json:"item_name"`
	ItemDescription string         `gorm:"type:text"                                       json:"item_description,omitempty"`
	ItemURL         string         `gorm:"type:text"                                       json:"item_url,omitempty"`
	ItemPrice       float64        `gorm:"type:numeric(12,2);default:0"                    json:"item_price"`
	ItemWeight      *float64       `gorm:"type:numeric(10,2)"                              json:"item_weight,omitempty"`
	PickupCountry   string         `gorm:"size:100;not null"                               json:"pickup_country"`
	PickupCity      string         `gorm:"size:100;not null"                               json:"pickup_city"`
	DeliveryCountry string         `gorm:"size:100;not null"                               json:"delivery_country"`
	DeliveryCity    string         `gorm:"size:100;not null"                               json:"delivery_city"`
	Reward          float64        `gorm:"type:numeric(10,2);not null;default:0"           json:"reward"`
	Currency        string         `gorm:"size:10;not null;default:'AED'"                  json:"currency"`
	DeliveryType    string         `gorm:"size:20;not null;default:'CROWDSHIPPING'"        json:"delivery_type"`
	Deadline        *time.Time     `json:"deadline,omitempty"`
	Status          DeliveryStatus `gorm:"size:50;not null;default:'pending';index"        json:"status"`
	MatchScore      *float64       `gorm:"type:numeric(6,2)"                               json:"match_score,omitempty"`
	ProofImageURL   string         `gorm:"type:text"                                       json:"proof_image_url,omitempty"`
	Notes           string         `gorm:"type:text"                                       json:"notes,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index"                                          json:"-"`
}

func (DeliveryRequest) TableName() string { return "delivery_requests" }
