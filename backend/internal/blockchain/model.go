package blockchain

import (
	"time"

	"github.com/google/uuid"
)

type EscrowStatus string

const (
	EscrowPending  EscrowStatus = "pending"
	EscrowFunded   EscrowStatus = "funded"
	EscrowReleased EscrowStatus = "released"
	EscrowRefunded EscrowStatus = "refunded"
	EscrowDisputed EscrowStatus = "disputed"
)

type EscrowContract struct {
	ID              uuid.UUID    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	OrderID         uuid.UUID    `gorm:"type:uuid;not null;index"   json:"order_id"`
	BuyerID         uuid.UUID    `gorm:"type:uuid;not null;index"   json:"buyer_id"`
	SellerID        uuid.UUID    `gorm:"type:uuid;not null;index"   json:"seller_id"`
	Amount          float64      `gorm:"type:numeric(14,2);not null" json:"amount"`
	Currency        string       `gorm:"size:10;not null;default:'AED'" json:"currency"`
	Chain           string       `gorm:"size:50;not null;default:'ethereum'" json:"chain"`
	ContractAddress string       `gorm:"size:100" json:"contract_address,omitempty"`
	TxHashFund      string       `gorm:"size:100" json:"tx_hash_fund,omitempty"`
	TxHashRelease   string       `gorm:"size:100" json:"tx_hash_release,omitempty"`
	Status          EscrowStatus `gorm:"size:50;not null;default:'pending';index" json:"status"`
	FundedAt        *time.Time   `json:"funded_at,omitempty"`
	ReleasedAt      *time.Time   `json:"released_at,omitempty"`
	ExpiresAt       *time.Time   `json:"expires_at,omitempty"`
	Metadata        string       `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

func (EscrowContract) TableName() string { return "escrow_contracts" }
