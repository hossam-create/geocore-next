package payments

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// PayMob — MENA payment gateway
// ════════════════════════════════════════════════════════════════════════════

// PayMobOrder represents a PayMob order record for idempotent tracking.
type PayMobOrder struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	PayMobOrderID  int64      `gorm:"uniqueIndex;not null" json:"paymob_order_id"`
	PaymentKey     string     `gorm:"size:512" json:"payment_key,omitempty"`
	AmountCents    int64      `gorm:"not null" json:"amount_cents"`
	Currency       string     `gorm:"size:3;default:'EGP'" json:"currency"`
	Status         PaymentStatus `gorm:"size:50;default:'pending';index" json:"status"`
	IdempotencyKey string     `gorm:"size:128;index" json:"idempotency_key,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// ProcessedPayMobEvent records every PayMob webhook that has been fully handled.
// Prevents double-processing on PayMob retries.
type ProcessedPayMobEvent struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	PayMobTxnID    int64     `gorm:"uniqueIndex;not null" json:"paymob_txn_id"`
	EventType      string    `gorm:"size:100;index" json:"event_type"`
	ResponseCode   int       `gorm:"not null;default:200" json:"response_code"`
	ResponseBody   string    `gorm:"type:text" json:"response_body"`
	ProcessedAt    time.Time `gorm:"not null;index" json:"processed_at"`
}

// PayMobInitReq is the request body for initiating a PayMob payment.
type PayMobInitReq struct {
	AmountCents    int64  `json:"amount_cents" binding:"required,gt=0"`
	Currency       string `json:"currency"`                     // default: EGP
	PaymentMethod  string `json:"payment_method"`               // card, kiosk, mobile_wallet, bank_installments
	IdempotencyKey string `json:"idempotency_key"`
	ListingID      *string `json:"listing_id"`
	AuctionID      *string `json:"auction_id"`
}

// PayMobWebhookPayload represents the HMAC-signed body from PayMob.
type PayMobWebhookPayload struct {
	Type          string `json:"type"`
	Object        PayMobWebhookObject `json:"obj"`
	Pending       bool   `json:"pending"`
}

// PayMobWebhookObject is the nested object in a PayMob webhook.
type PayMobWebhookObject struct {
	ID               int64  `json:"id"`
	OrderID          int64  `json:"order_id"`
	AmountCents      int64  `json:"amount_cents"`
	Currency         string `json:"currency"`
	PaymentKey       string `json:"payment_key"`
	Success          bool   `json:"success"`
	Pending          bool   `json:"pending"`
	ErrorOccurred    bool   `json:"error_occured"` // PayMob typo is intentional
	Is3DSecure       bool   `json:"is_3d_secure"`
	IsRefunded       bool   `json:"is_refunded"`
	IsVoid           bool   `json:"is_void"`
	IsStandalonePayment bool `json:"is_standalone_payment"`
	Data             string `json:"data"`
	TxnResponseCode  string `json:"txn_response_code"`
}
