package wallet

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Currency supported currencies
type Currency string

const (
	USD Currency = "USD"
	EUR Currency = "EUR"
	GBP Currency = "GBP"
	SAR Currency = "SAR"
	AED Currency = "AED"
	EGP Currency = "EGP"
)

var SupportedCurrencies = []Currency{USD, EUR, GBP, SAR, AED, EGP}

// TransactionType types of wallet transactions
type TransactionType string

const (
	TransactionDeposit    TransactionType = "DEPOSIT"
	TransactionWithdrawal TransactionType = "WITHDRAWAL"
	TransactionTransfer   TransactionType = "TRANSFER"
	TransactionPayment    TransactionType = "PAYMENT"
	TransactionRefund     TransactionType = "REFUND"
	TransactionEscrow     TransactionType = "ESCROW"
	TransactionRelease    TransactionType = "RELEASE"
	TransactionFee        TransactionType = "FEE"
)

// TransactionStatus status of transactions
type TransactionStatus string

const (
	StatusPending   TransactionStatus = "PENDING"
	StatusCompleted TransactionStatus = "COMPLETED"
	StatusFailed    TransactionStatus = "FAILED"
	StatusCancelled TransactionStatus = "CANCELLED"
)

// Wallet user wallet
type Wallet struct {
	ID              uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID       `json:"user_id" gorm:"type:uuid;uniqueIndex;not null"`
	PrimaryCurrency Currency        `json:"primary_currency" gorm:"type:varchar(3);default:'USD'"`
	DailyLimit      decimal.Decimal `json:"daily_limit" gorm:"type:decimal(15,2);default:10000"`
	MonthlyLimit    decimal.Decimal `json:"monthly_limit" gorm:"type:decimal(15,2);default:100000"`
	IsActive        bool            `json:"is_active" gorm:"default:true"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`

	Balances     []WalletBalance     `json:"balances,omitempty" gorm:"foreignKey:WalletID"`
	Transactions []WalletTransaction `json:"transactions,omitempty" gorm:"foreignKey:WalletID"`
}

// WalletBalance balance per currency
type WalletBalance struct {
	ID               uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletID         uuid.UUID       `json:"wallet_id" gorm:"type:uuid;not null;index"`
	Currency         Currency        `json:"currency" gorm:"type:varchar(3);not null"`
	Balance          decimal.Decimal `json:"balance" gorm:"type:decimal(15,2);default:0"`
	AvailableBalance decimal.Decimal `json:"available_balance" gorm:"type:decimal(15,2);default:0"`
	PendingBalance   decimal.Decimal `json:"pending_balance" gorm:"type:decimal(15,2);default:0"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// WalletTransaction transaction record
type WalletTransaction struct {
	ID            uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	WalletID      uuid.UUID         `json:"wallet_id" gorm:"type:uuid;not null;index"`
	Type          TransactionType   `json:"type" gorm:"type:varchar(20);not null"`
	Currency      Currency          `json:"currency" gorm:"type:varchar(3);not null"`
	Amount        decimal.Decimal   `json:"amount" gorm:"type:decimal(15,2);not null"`
	BalanceBefore decimal.Decimal   `json:"balance_before" gorm:"type:decimal(15,2)"`
	BalanceAfter  decimal.Decimal   `json:"balance_after" gorm:"type:decimal(15,2)"`
	Fee           decimal.Decimal   `json:"fee" gorm:"type:decimal(15,2);default:0"`
	Status        TransactionStatus `json:"status" gorm:"type:varchar(20);default:'PENDING'"`
	ReferenceID   *string           `json:"reference_id" gorm:"type:varchar(100)"`
	ReferenceType *string           `json:"reference_type" gorm:"type:varchar(50)"`
	Description   string            `json:"description" gorm:"type:text"`
	Metadata      string            `json:"metadata" gorm:"type:jsonb"`
	CreatedAt     time.Time         `json:"created_at"`
	CompletedAt   *time.Time        `json:"completed_at"`
}

// Escrow escrow account for auctions/orders
type Escrow struct {
	ID          uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BuyerID     uuid.UUID         `json:"buyer_id" gorm:"type:uuid;not null;index"`
	SellerID    uuid.UUID         `json:"seller_id" gorm:"type:uuid;not null;index"`
	Amount      decimal.Decimal   `json:"amount" gorm:"type:decimal(15,2);not null"`
	Currency    Currency          `json:"currency" gorm:"type:varchar(3);not null"`
	Fee         decimal.Decimal   `json:"fee" gorm:"type:decimal(15,2);default:0"`
	Status      TransactionStatus `json:"status" gorm:"type:varchar(20);default:'PENDING'"`
	ReferenceID string            `json:"reference_id" gorm:"type:varchar(100);not null"` // auction_id or order_id
	Type        string            `json:"type" gorm:"type:varchar(20);not null"`          // AUCTION, ORDER
	Approval1By *uuid.UUID        `json:"approval_1_by,omitempty" gorm:"type:uuid"`
	Approval1At *time.Time        `json:"approval_1_at,omitempty"`
	Approval2By *uuid.UUID        `json:"approval_2_by,omitempty" gorm:"type:uuid"`
	Approval2At *time.Time        `json:"approval_2_at,omitempty"`
	ReleasedAt  *time.Time        `json:"released_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PricePlan subscription plans for listings
type PricePlan struct {
	ID            uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name          string          `json:"name" gorm:"type:varchar(50);not null"`
	NameAr        string          `json:"name_ar" gorm:"type:varchar(50)"`
	Description   string          `json:"description" gorm:"type:text"`
	DescriptionAr string          `json:"description_ar" gorm:"type:text"`
	Price         decimal.Decimal `json:"price" gorm:"type:decimal(10,2);not null"`
	Currency      Currency        `json:"currency" gorm:"type:varchar(3);default:'USD'"`
	DurationDays  int             `json:"duration_days" gorm:"default:30"`
	ListingsLimit int             `json:"listings_limit" gorm:"default:10"`
	FeaturedLimit int             `json:"featured_limit" gorm:"default:0"`
	BoostDays     int             `json:"boost_days" gorm:"default:0"`
	Priority      int             `json:"priority" gorm:"default:0"` // higher = better visibility
	IsActive      bool            `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// IdempotentRequest deduplicates financial write operations via a client-supplied key.
// Unique on (user_id, idempotency_key) — prevents duplicate credits/debits under retries.
type IdempotentRequest struct {
	ID             uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID         uuid.UUID `json:"user_id" gorm:"type:uuid;not null;uniqueIndex:idx_idempotent_user_key"`
	IdempotencyKey string    `json:"idempotency_key" gorm:"type:varchar(128);not null;uniqueIndex:idx_idempotent_user_key"`
	Path           string    `json:"path" gorm:"type:varchar(255)"`
	ResponseCode   int       `json:"response_code"`
	ResponseBody   string    `json:"response_body" gorm:"type:text"`
	CreatedAt      time.Time `json:"created_at" gorm:"index"`
	ExpiresAt      time.Time `json:"expires_at" gorm:"index"`
}

// UserSubscription user's active subscription
type UserSubscription struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index"`
	PlanID      uuid.UUID  `json:"plan_id" gorm:"type:uuid;not null"`
	StartDate   time.Time  `json:"start_date" gorm:"not null"`
	EndDate     time.Time  `json:"end_date" gorm:"not null;index"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	AutoRenew   bool       `json:"auto_renew" gorm:"default:false"`
	CreatedAt   time.Time  `json:"created_at"`
	CancelledAt *time.Time `json:"cancelled_at"`

	Plan PricePlan `json:"plan,omitempty" gorm:"foreignKey:PlanID"`
}
