package payments

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Payment
// ════════════════════════════════════════════════════════════════════════════

// PaymentStatus represents the lifecycle states of a payment.
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSucceeded PaymentStatus = "succeeded"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// PaymentKind categorises every payment record at creation time.
// This is set server-side only and must never be derived from description text.
type PaymentKind string

const (
	PaymentKindPurchase       PaymentKind = "purchase"
	PaymentKindAuctionPayment PaymentKind = "auction_payment"
	PaymentKindWalletTopUp    PaymentKind = "wallet_topup"
	PaymentKindRefund         PaymentKind = "refund"
)

// Payment records every payment transaction in the system.
type Payment struct {
	ID                    uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID                uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	ListingID             *uuid.UUID     `gorm:"type:uuid;index" json:"listing_id,omitempty"`
	AuctionID             *uuid.UUID     `gorm:"type:uuid;index" json:"auction_id,omitempty"`
	Kind                  PaymentKind    `gorm:"size:50;not null;default:'purchase';index" json:"kind"`
	StripePaymentIntentID string         `gorm:"size:255;uniqueIndex" json:"stripe_payment_intent_id"`
	StripeClientSecret    string         `gorm:"size:512" json:"client_secret,omitempty"`
	Amount                float64        `gorm:"not null" json:"amount"`
	Currency              string         `gorm:"size:3;default:'AED'" json:"currency"`
	Status                PaymentStatus  `gorm:"size:50;default:'pending';index" json:"status"`
	PaymentMethod         string         `gorm:"size:50" json:"payment_method,omitempty"`
	Description           string         `gorm:"type:text" json:"description,omitempty"`
	FailureReason         string         `gorm:"type:text" json:"failure_reason,omitempty"`
	RefundedAt            *time.Time     `json:"refunded_at,omitempty"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`

	// Associations (eager-loaded when needed)
	Escrow *EscrowAccount `gorm:"foreignKey:PaymentID" json:"escrow,omitempty"`
}

// ════════════════════════════════════════════════════════════════════════════
// EscrowAccount
// ════════════════════════════════════════════════════════════════════════════

// EscrowStatus represents the lifecycle states of an escrow account.
type EscrowStatus string

const (
	EscrowStatusHeld     EscrowStatus = "held"
	EscrowStatusReleased EscrowStatus = "released"
	EscrowStatusRefunded EscrowStatus = "refunded"
	EscrowStatusDisputed EscrowStatus = "disputed"
)

// EscrowAccount holds payment funds between a buyer and seller until the
// transaction is confirmed. Funds are released after buyer confirmation
// or automatically after 7 days.
type EscrowAccount struct {
	ID         uuid.UUID    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	PaymentID  uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex" json:"payment_id"`
	SellerID   uuid.UUID    `gorm:"type:uuid;not null;index" json:"seller_id"`
	BuyerID    uuid.UUID    `gorm:"type:uuid;not null;index" json:"buyer_id"`
	Amount     float64      `gorm:"not null" json:"amount"`
	Currency   string       `gorm:"size:3;default:'AED'" json:"currency"`
	Status     EscrowStatus `gorm:"size:50;default:'held';index" json:"status"`
	ReleasedAt *time.Time   `json:"released_at,omitempty"`
	Notes      string       `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt  time.Time    `json:"created_at"`
	UpdatedAt  time.Time    `json:"updated_at"`

	// Association
	Payment *Payment `gorm:"foreignKey:PaymentID" json:"payment,omitempty"`
}

// ════════════════════════════════════════════════════════════════════════════
// SavedPaymentMethod — Stripe payment methods cached locally
// ════════════════════════════════════════════════════════════════════════════

type SavedPaymentMethod struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	StripeMethodID string    `gorm:"size:128;uniqueIndex" json:"stripe_method_id"`
	Brand          string    `gorm:"size:30" json:"brand"` // visa, mastercard, amex
	Last4          string    `gorm:"size:4" json:"last4"`
	ExpMonth       int       `json:"exp_month"`
	ExpYear        int       `json:"exp_year"`
	IsDefault      bool      `gorm:"default:false" json:"is_default"`
	CreatedAt      time.Time `json:"created_at"`
}

// ════════════════════════════════════════════════════════════════════════════
// ProcessedStripeEvent — webhook idempotency
// ════════════════════════════════════════════════════════════════════════════

// ProcessedStripeEvent records every Stripe event that has been fully handled.
// The uniqueIndex on StripeEventID prevents double-processing on Stripe retries.
type ProcessedStripeEvent struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	StripeEventID string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"stripe_event_id"`
	EventType     string    `gorm:"type:varchar(100);index" json:"event_type"`
	ResponseCode  int       `gorm:"not null;default:200" json:"response_code"`
	ResponseBody  string    `gorm:"type:text" json:"response_body"`
	ProcessedAt   time.Time `gorm:"not null;index" json:"processed_at"`
}
