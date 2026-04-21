package subscriptions

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionStatus mirrors the DB enum
type SubscriptionStatus string

const (
	StatusActive     SubscriptionStatus = "active"
	StatusCancelled  SubscriptionStatus = "cancelled"
	StatusPastDue    SubscriptionStatus = "past_due"
	StatusTrialing   SubscriptionStatus = "trialing"
	StatusIncomplete SubscriptionStatus = "incomplete"
	StatusUnpaid     SubscriptionStatus = "unpaid"
)

// FreePlanListingLimit is the default limit for users without a paid plan
const FreePlanListingLimit = 5

// Plan represents a subscription tier
type Plan struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name           string    `gorm:"size:50;not null;uniqueIndex" json:"name"`
	DisplayName    string    `gorm:"size:100;not null" json:"display_name"`
	PriceMonthly   float64   `gorm:"type:decimal(10,2);not null;default:0" json:"price_monthly"`
	Currency       string    `gorm:"size:3;not null;default:'AED'" json:"currency"`
	StripePriceID  string    `gorm:"size:128" json:"stripe_price_id,omitempty"`
	ListingLimit   int       `gorm:"not null;default:5" json:"listing_limit"`
	Features       []string  `gorm:"type:jsonb;serializer:json" json:"features"`
	IsActive       bool      `gorm:"default:true" json:"is_active"`
	SortOrder      int       `gorm:"default:0" json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (Plan) TableName() string { return "plans" }

// Subscription represents a user's active subscription
type Subscription struct {
	ID                    uuid.UUID          `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID                uuid.UUID          `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	PlanID                uuid.UUID          `gorm:"type:uuid;not null" json:"plan_id"`
	Status                SubscriptionStatus `gorm:"size:20;not null;default:'active'" json:"status"`
	StripeSubscriptionID  string             `gorm:"size:128;uniqueIndex:idx_subs_stripe_sub,where:stripe_subscription_id <> ''" json:"stripe_subscription_id,omitempty"`
	StripeCustomerID      string             `gorm:"size:128" json:"stripe_customer_id,omitempty"`
	CurrentPeriodStart    *time.Time         `json:"current_period_start,omitempty"`
	CurrentPeriodEnd      *time.Time         `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd     bool               `gorm:"default:false" json:"cancel_at_period_end"`
	CancelledAt           *time.Time         `json:"cancelled_at,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at"`

	Plan *Plan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

func (Subscription) TableName() string { return "subscriptions" }

func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// GetUserPlanLimits returns the active listing limit for a user.
// Falls back to FreePlanListingLimit if the user has no subscription.
func GetUserPlanLimits(db *gorm.DB, userID uuid.UUID) (listingLimit int) {
	var sub Subscription
	err := db.Preload("Plan").Where("user_id = ? AND status = ?", userID, StatusActive).First(&sub).Error
	if err != nil || sub.Plan == nil {
		return FreePlanListingLimit
	}
	return sub.Plan.ListingLimit
}
