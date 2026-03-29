package disputes

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DisputeStatus defines dispute lifecycle states
type DisputeStatus string

const (
	StatusOpen       DisputeStatus = "open"
	StatusUnderReview DisputeStatus = "under_review"
	StatusAwaitingResponse DisputeStatus = "awaiting_response"
	StatusEscalated  DisputeStatus = "escalated"
	StatusResolved   DisputeStatus = "resolved"
	StatusClosed     DisputeStatus = "closed"
)

// DisputeReason defines why a dispute was opened
type DisputeReason string

const (
	ReasonItemNotReceived    DisputeReason = "item_not_received"
	ReasonItemNotAsDescribed DisputeReason = "item_not_as_described"
	ReasonItemDamaged        DisputeReason = "item_damaged"
	ReasonWrongItem          DisputeReason = "wrong_item"
	ReasonSellerNotResponding DisputeReason = "seller_not_responding"
	ReasonPaymentIssue       DisputeReason = "payment_issue"
	ReasonFraud              DisputeReason = "fraud"
	ReasonOther              DisputeReason = "other"
)

// ResolutionType defines how a dispute was resolved
type ResolutionType string

const (
	ResolutionFullRefund    ResolutionType = "full_refund"
	ResolutionPartialRefund ResolutionType = "partial_refund"
	ResolutionReplacement   ResolutionType = "replacement"
	ResolutionNoRefund      ResolutionType = "no_refund"
	ResolutionMutualAgreement ResolutionType = "mutual_agreement"
	ResolutionAdminDecision ResolutionType = "admin_decision"
)

// Dispute represents a dispute between buyer and seller
type Dispute struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	OrderID     *uuid.UUID     `gorm:"type:uuid;index" json:"order_id,omitempty"`
	AuctionID   *uuid.UUID     `gorm:"type:uuid;index" json:"auction_id,omitempty"`
	EscrowID    *uuid.UUID     `gorm:"type:uuid;index" json:"escrow_id,omitempty"`
	
	BuyerID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"buyer_id"`
	SellerID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
	AssignedTo  *uuid.UUID     `gorm:"type:uuid;index" json:"assigned_to,omitempty"` // Admin handling the dispute
	
	Reason      DisputeReason  `gorm:"type:varchar(50);not null" json:"reason"`
	Description string         `gorm:"type:text;not null" json:"description"`
	Amount      float64        `gorm:"not null" json:"amount"` // Disputed amount
	Currency    string         `gorm:"default:USD" json:"currency"`
	
	Status      DisputeStatus  `gorm:"type:varchar(30);default:'open';index" json:"status"`
	Priority    int            `gorm:"default:5" json:"priority"` // 1 = highest
	
	// Resolution
	Resolution       *ResolutionType `gorm:"type:varchar(30)" json:"resolution,omitempty"`
	ResolutionAmount *float64        `json:"resolution_amount,omitempty"`
	ResolutionNotes  string          `gorm:"type:text" json:"resolution_notes,omitempty"`
	ResolvedBy       *uuid.UUID      `gorm:"type:uuid" json:"resolved_by,omitempty"`
	ResolvedAt       *time.Time      `json:"resolved_at,omitempty"`
	
	// Deadlines
	ResponseDeadline *time.Time `json:"response_deadline,omitempty"`
	EscalationDate   *time.Time `json:"escalation_date,omitempty"`
	
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	
	// Relations
	Messages  []DisputeMessage  `gorm:"foreignKey:DisputeID" json:"messages,omitempty"`
	Evidence  []DisputeEvidence `gorm:"foreignKey:DisputeID" json:"evidence,omitempty"`
}

// DisputeMessage represents a message in a dispute thread
type DisputeMessage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	DisputeID uuid.UUID `gorm:"type:uuid;not null;index" json:"dispute_id"`
	SenderID  uuid.UUID `gorm:"type:uuid;not null" json:"sender_id"`
	SenderRole string   `gorm:"type:varchar(20);not null" json:"sender_role"` // buyer, seller, admin
	Message   string    `gorm:"type:text;not null" json:"message"`
	IsInternal bool     `gorm:"default:false" json:"is_internal"` // Admin-only notes
	CreatedAt time.Time `json:"created_at"`
}

// DisputeEvidence represents evidence submitted for a dispute
type DisputeEvidence struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	DisputeID   uuid.UUID `gorm:"type:uuid;not null;index" json:"dispute_id"`
	SubmittedBy uuid.UUID `gorm:"type:uuid;not null" json:"submitted_by"`
	Type        string    `gorm:"type:varchar(30);not null" json:"type"` // image, document, screenshot, video
	URL         string    `gorm:"not null" json:"url"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// DisputeActivity logs all actions taken on a dispute
type DisputeActivity struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	DisputeID uuid.UUID `gorm:"type:uuid;not null;index" json:"dispute_id"`
	ActorID   uuid.UUID `gorm:"type:uuid;not null" json:"actor_id"`
	Action    string    `gorm:"type:varchar(50);not null" json:"action"`
	Details   string    `gorm:"type:text" json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
