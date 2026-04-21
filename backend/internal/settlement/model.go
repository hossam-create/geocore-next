package settlement

import (
	"time"

	"github.com/google/uuid"
)

// SettlementStatus represents the lifecycle of a settlement.
type SettlementStatus string

const (
	SettlementStatusPending   SettlementStatus = "pending"
	SettlementStatusMatched   SettlementStatus = "matched"
	SettlementStatusProcessing SettlementStatus = "processing"
	SettlementStatusCompleted SettlementStatus = "completed"
	SettlementStatusFailed    SettlementStatus = "failed"
)

// Settlement records a matched incoming/outgoing transfer that settles funds
// between buyer and seller through the platform.
type Settlement struct {
	ID              uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SellerID        uuid.UUID        `gorm:"type:uuid;not null;index" json:"seller_id"`
	BuyerID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"buyer_id"`
	OrderID         *uuid.UUID       `gorm:"type:uuid;index" json:"order_id,omitempty"`
	PaymentID       *uuid.UUID       `gorm:"type:uuid;index" json:"payment_id,omitempty"`
	EscrowID        *uuid.UUID       `gorm:"type:uuid;index" json:"escrow_id,omitempty"`
	Amount          float64          `gorm:"not null" json:"amount"`
	Currency        string           `gorm:"size:3;default:'AED'" json:"currency"`
	PlatformFee     float64          `gorm:"default:0" json:"platform_fee"`
	PaymentFee      float64          `gorm:"default:0" json:"payment_fee"`
	NetAmount       float64          `gorm:"not null" json:"net_amount"` // amount - fees
	Status          SettlementStatus `gorm:"size:50;default:'pending';index" json:"status"`
	ProcessedAt     *time.Time       `json:"processed_at,omitempty"`
	FailedReason    string           `gorm:"type:text" json:"failed_reason,omitempty"`
	IdempotencyKey  string           `gorm:"size:128;index" json:"idempotency_key,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// PayoutStatus represents the lifecycle of a seller payout.
type PayoutStatus string

const (
	PayoutStatusPending   PayoutStatus = "pending"
	PayoutStatusApproved  PayoutStatus = "approved"
	PayoutStatusProcessing PayoutStatus = "processing"
	PayoutStatusCompleted PayoutStatus = "completed"
	PayoutStatusFailed    PayoutStatus = "failed"
	PayoutStatusCancelled PayoutStatus = "cancelled"
)

// Payout records a seller payout (withdrawal of settled funds).
type Payout struct {
	ID             uuid.UUID    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SellerID       uuid.UUID    `gorm:"type:uuid;not null;index" json:"seller_id"`
	Amount         float64      `gorm:"not null" json:"amount"`
	Currency       string       `gorm:"size:3;default:'AED'" json:"currency"`
	Destination    string       `gorm:"size:255" json:"destination"` // bank account, wallet, etc.
	Method         string       `gorm:"size:50" json:"method"`       // bank_transfer, wallet, paymob
	Status         PayoutStatus `gorm:"size:50;default:'pending';index" json:"status"`
	ApprovedBy     *uuid.UUID   `gorm:"type:uuid;index" json:"approved_by,omitempty"`
	ApprovedAt     *time.Time   `json:"approved_at,omitempty"`
	ProcessedAt    *time.Time   `json:"processed_at,omitempty"`
	ReferenceID    string       `gorm:"size:255" json:"reference_id,omitempty"`
	FailedReason   string       `gorm:"type:text" json:"failed_reason,omitempty"`
	AuditLog       string       `gorm:"type:text" json:"audit_log,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
