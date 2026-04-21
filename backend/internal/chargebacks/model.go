package chargebacks

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChargebackStatus represents the lifecycle of a chargeback dispute.
type ChargebackStatus string

const (
	ChargebackStatusOpen        ChargebackStatus = "open"
	ChargebackStatusUnderReview ChargebackStatus = "under_review"
	ChargebackStatusWon         ChargebackStatus = "won"
	ChargebackStatusLost        ChargebackStatus = "lost"
)

// Chargeback records a bank-initiated dispute (chargeback) against a payment.
// Unlike platform disputes (buyer-seller), chargebacks originate from the
// buyer's bank (e.g. via Stripe) and require evidence submission.
type Chargeback struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	PaymentID       uuid.UUID        `gorm:"type:uuid;not null;index" json:"payment_id"`
	OrderID         *uuid.UUID       `gorm:"type:uuid;index" json:"order_id,omitempty"`
	StripeDisputeID string           `gorm:"size:255;uniqueIndex" json:"stripe_dispute_id,omitempty"`
	Amount          float64          `gorm:"not null" json:"amount"`
	Currency        string           `gorm:"size:3;default:'AED'" json:"currency"`
	Reason          string           `gorm:"type:text" json:"reason"`
	Status          ChargebackStatus `gorm:"size:50;default:'open';index" json:"status"`
	EvidenceDueBy   *time.Time       `json:"evidence_due_by,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	DeletedAt       gorm.DeletedAt   `gorm:"index" json:"-"`
}

// CreateRequest is the payload for creating a chargeback (usually from webhook).
type CreateRequest struct {
	PaymentID       string  `json:"payment_id" binding:"required"`
	OrderID         string  `json:"order_id,omitempty"`
	StripeDisputeID string  `json:"stripe_dispute_id,omitempty"`
	Amount          float64 `json:"amount" binding:"required"`
	Currency        string  `json:"currency"`
	Reason          string  `json:"reason"`
	EvidenceDueBy   string  `json:"evidence_due_by,omitempty"`
}

// EvidenceRequest is the payload for submitting evidence to Stripe.
type EvidenceRequest struct {
	EvidenceType string `json:"evidence_type" binding:"required"` // e.g. "refund_policy", "customer_communication", etc.
	FileURL      string `json:"file_url,omitempty"`
	Description  string `json:"description,omitempty"`
}
