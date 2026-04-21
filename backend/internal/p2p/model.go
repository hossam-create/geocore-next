package p2p

import (
	"time"

	"github.com/google/uuid"
)

type ExchangeStatus string

const (
	StatusOpen      ExchangeStatus = "open"
	StatusMatched   ExchangeStatus = "matched"
	StatusEscrow    ExchangeStatus = "escrow"
	StatusCompleted ExchangeStatus = "completed"
	StatusCancelled ExchangeStatus = "cancelled"
	StatusDisputed  ExchangeStatus = "disputed"
)

type ExchangeRequest struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"user_id"`
	FromCurrency  string         `gorm:"size:10;not null"                                json:"from_currency"`
	ToCurrency    string         `gorm:"size:10;not null"                                json:"to_currency"`
	FromAmount    float64        `gorm:"type:numeric(14,2);not null"                     json:"from_amount"`
	ToAmount      float64        `gorm:"type:numeric(14,2);not null"                     json:"to_amount"`
	DesiredRate   float64        `gorm:"type:numeric(12,6);not null"                     json:"desired_rate"`
	UseEscrow     bool           `gorm:"not null;default:false"                          json:"use_escrow"`
	Notes         string         `gorm:"type:text"                                       json:"notes,omitempty"`
	Status        ExchangeStatus `gorm:"size:50;not null;default:'open';index"           json:"status"`
	MatchedUserID *uuid.UUID     `gorm:"type:uuid;index"                                json:"matched_user_id,omitempty"`
	MatchedAt     *time.Time     `json:"matched_at,omitempty"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (ExchangeRequest) TableName() string { return "exchange_requests" }

type ExchangeMessage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	RequestID uuid.UUID `gorm:"type:uuid;not null;index"                        json:"request_id"`
	SenderID  uuid.UUID `gorm:"type:uuid;not null"                              json:"sender_id"`
	Body      string    `gorm:"type:text;not null"                              json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func (ExchangeMessage) TableName() string { return "exchange_messages" }
