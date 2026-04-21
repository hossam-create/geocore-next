package exchange

// Sprint 19 — Non-Custodial FX + P2P Exchange Layer.
//
// LEGAL SAFETY MODEL:
//   Platform NEVER holds funds.
//   Platform NEVER executes FX.
//   Platform ONLY matches users, enforces trust, handles disputes.
//   All monetary transfers happen peer-to-peer via external providers
//   (Instapay, bank transfer, PayPal, etc.).

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Status constants
// ════════════════════════════════════════════════════════════════════════════

const (
	// ExchangeRequest statuses
	StatusOpen      = "OPEN"
	StatusMatched   = "MATCHED"
	StatusSettling  = "SETTLING"
	StatusCompleted = "COMPLETED"
	StatusCancelled = "CANCELLED"

	// ExchangeMatch statuses
	MatchPending  = "PENDING"
	MatchVerified = "VERIFIED"
	MatchSettled  = "SETTLED"
	MatchDisputed = "DISPUTED"

	// ExchangeSettlement statuses
	SettlementWaitingProof = "WAITING_PROOF"
	SettlementVerified     = "VERIFIED"
	SettlementReleased     = "RELEASED"

	// ExchangeDispute statuses
	DisputeOpen     = "OPEN"
	DisputeResolved = "RESOLVED"
	DisputeClosed   = "CLOSED"

	// Fee types
	FeeTypeMatch    = "match"
	FeeTypePriority = "priority"
	FeeTypeProtect  = "protection"

	// Trust gate for exchange actions
	TrustGateExchange = "exchange"
)

// ════════════════════════════════════════════════════════════════════════════
// ExchangeRequest — a user's intent to exchange FromCurrency → ToCurrency
// ════════════════════════════════════════════════════════════════════════════

type ExchangeRequest struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID            uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"user_id"`
	FromCurrency      string         `gorm:"size:10;not null;index"                          json:"from_currency"`
	ToCurrency        string         `gorm:"size:10;not null;index"                          json:"to_currency"`
	Amount            float64        `gorm:"not null"                                        json:"amount"`
	PreferredRate     *float64       `json:"preferred_rate,omitempty"`
	PaymentMethod     string         `gorm:"size:20;index"                                   json:"payment_method,omitempty"`
	Status            string         `gorm:"size:20;not null;default:'OPEN';index"           json:"status"`
	ExpiresAt         *time.Time     `json:"expires_at,omitempty"`
	IsSystemGenerated bool           `gorm:"not null;default:false;index"                    json:"is_system_generated"`
	IsInfluencerSeed  bool           `gorm:"not null;default:false;index"                    json:"is_influencer_seed"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index"                                           json:"-"`
}

func (ExchangeRequest) TableName() string { return "exchange_requests" }

// ════════════════════════════════════════════════════════════════════════════
// ExchangeMatch — a confirmed pairing between two opposite requests
// ════════════════════════════════════════════════════════════════════════════

type ExchangeMatch struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	RequestAID uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"request_a_id"`
	RequestBID uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"request_b_id"`
	AgreedRate float64        `gorm:"not null"                                        json:"agreed_rate"`
	Status     string         `gorm:"size:20;not null;default:'PENDING';index"        json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index"                                           json:"-"`
}

func (ExchangeMatch) TableName() string { return "exchange_matches" }

// ════════════════════════════════════════════════════════════════════════════
// ExchangeSettlement — proof collection for a matched pair
//
// NO wallet. NO escrow. NO balance updates.
// Users send to each other directly and upload payment receipts.
// ════════════════════════════════════════════════════════════════════════════

type ExchangeSettlement struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	MatchID     uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"                  json:"match_id"`
	UserAProof  string     `gorm:"type:text"                                       json:"user_a_proof,omitempty"` // URL to receipt/screenshot
	UserBProof  string     `gorm:"type:text"                                       json:"user_b_proof,omitempty"`
	UserAAmount *float64   `json:"user_a_amount,omitempty"` // amount on the proof
	UserBAmount *float64   `json:"user_b_amount,omitempty"`
	VerifiedA   bool       `gorm:"not null;default:false"                          json:"verified_a"`
	VerifiedB   bool       `gorm:"not null;default:false"                          json:"verified_b"`
	ProofAAt    *time.Time `json:"proof_a_at,omitempty"`
	ProofBAt    *time.Time `json:"proof_b_at,omitempty"`
	Status      string     `gorm:"size:30;not null;default:'WAITING_PROOF';index"  json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (ExchangeSettlement) TableName() string { return "exchange_settlements" }

// ════════════════════════════════════════════════════════════════════════════
// ExchangeDispute — dispute raised during settlement
// ════════════════════════════════════════════════════════════════════════════

type ExchangeDispute struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	MatchID    uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"match_id"`
	RaisedBy   uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"raised_by"`
	Reason     string         `gorm:"type:text;not null"                              json:"reason"`
	Resolution string         `gorm:"type:text"                                       json:"resolution,omitempty"`
	Status     string         `gorm:"size:20;not null;default:'OPEN';index"           json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index"                                           json:"-"`
}

func (ExchangeDispute) TableName() string { return "exchange_disputes" }

// ════════════════════════════════════════════════════════════════════════════
// ExchangeFee — fee record per match (charged off-ledger / external)
// ════════════════════════════════════════════════════════════════════════════

type ExchangeFee struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	MatchID   uuid.UUID `gorm:"type:uuid;not null;index"                        json:"match_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"                        json:"user_id"`
	FeeType   string    `gorm:"size:20;not null"                                json:"fee_type"`
	Amount    float64   `gorm:"not null"                                        json:"amount"`
	Currency  string    `gorm:"size:10;not null"                                json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

func (ExchangeFee) TableName() string { return "exchange_fees" }

// ════════════════════════════════════════════════════════════════════════════
// AutoMigrate — run at startup to create tables
// ════════════════════════════════════════════════════════════════════════════

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&ExchangeRequest{},
		&ExchangeMatch{},
		&ExchangeSettlement{},
		&ExchangeDispute{},
		&ExchangeFee{},
		&ExchangeUserTier{},
		&ExchangeLiquidityProfile{},
		&ExchangeRiskFlag{},
	)
}
