package monetization

import (
        "time"

        "github.com/google/uuid"
        "gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Subscription tiers
// ════════════════════════════════════════════════════════════════════════════

type TierName string

const (
        TierBasic    TierName = "basic"    // free; default for all users
        TierPro      TierName = "pro"      // paid monthly
        TierBusiness TierName = "business" // paid monthly, higher limits
)

// TierLimits defines the per-tier listing and image quotas.
type TierLimits struct {
        MaxListings int // 0 = unlimited
        MaxImages   int // images per listing
}

// Limits returns the limits for a given tier.
// Basic is the default (also used as fallback for unknown tiers).
func Limits(tier TierName) TierLimits {
        switch tier {
        case TierPro:
                return TierLimits{MaxListings: 50, MaxImages: 10}
        case TierBusiness:
                return TierLimits{MaxListings: 0, MaxImages: 20} // unlimited listings
        default: // TierBasic
                return TierLimits{MaxListings: 5, MaxImages: 3}
        }
}

// BoostFee is the fixed charge (in USD) to feature a listing for BoostDays days.
const (
        BoostFee      float64 = 5.00
        BoostCurrency         = "usd"
        BoostDays             = 7 // days the listing stays featured

        // SubscriptionFees in USD per month.
        ProMonthlyFee      float64 = 19.99
        BusinessMonthlyFee float64 = 49.99
)

// ════════════════════════════════════════════════════════════════════════════
// PlatformSettings — single-row global configuration
// ════════════════════════════════════════════════════════════════════════════

// PlatformSettings stores tunable platform-wide values.
// Only one row should ever exist (id = 1); use GetSettings() to read it.
type PlatformSettings struct {
        ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
        CommissionRate  float64   `gorm:"not null;default:0.05" json:"commission_rate"` // 0.05 = 5%
        BoostFeeUSD     float64   `gorm:"not null;default:5.00" json:"boost_fee_usd"`
        PlatformBalance float64   `gorm:"not null;default:0" json:"platform_balance"` // accumulated commission credits
        UpdatedAt       time.Time `json:"updated_at"`
}

// ════════════════════════════════════════════════════════════════════════════
// PlatformCommission — one row per escrow release
// ════════════════════════════════════════════════════════════════════════════

type PlatformCommission struct {
        ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
        EscrowID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"escrow_id"`
        SellerID         uuid.UUID `gorm:"type:uuid;not null;index" json:"seller_id"`
        BuyerID          uuid.UUID `gorm:"type:uuid;not null;index" json:"buyer_id"`
        GrossAmount      float64   `gorm:"not null" json:"gross_amount"`
        CommissionRate   float64   `gorm:"not null" json:"commission_rate"`
        CommissionAmount float64   `gorm:"not null" json:"commission_amount"`
        NetAmount        float64   `gorm:"not null" json:"net_amount"`
        Currency         string    `gorm:"size:3;default:'AED'" json:"currency"`
        CreatedAt        time.Time `json:"created_at"`
}

// ════════════════════════════════════════════════════════════════════════════
// BoostPayment — lightweight view of a boost payment (joins on payments table)
// Used only for admin revenue aggregation; actual record lives in payments.
// ════════════════════════════════════════════════════════════════════════════

// SellerSubscription — tracks active paid subscriptions for sellers.
// Tier is also mirrored on users.subscription_tier for fast lookup during listing creation.
type SellerSubscription struct {
        ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
        UserID         uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
        Tier           TierName       `gorm:"size:20;not null;default:'basic'" json:"tier"`
        StripePriceID  string         `gorm:"size:128" json:"stripe_price_id,omitempty"`
        StripeSubID    string         `gorm:"size:128;index" json:"stripe_sub_id,omitempty"`
        StartsAt       time.Time      `json:"starts_at"`
        ExpiresAt      *time.Time     `json:"expires_at,omitempty"`
        CancelledAt    *time.Time     `json:"cancelled_at,omitempty"`
        CreatedAt      time.Time      `json:"created_at"`
        UpdatedAt      time.Time      `json:"updated_at"`
        DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}
