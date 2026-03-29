package loyalty

import (
	"time"

	"github.com/google/uuid"
)

// TierLevel defines loyalty tier levels
type TierLevel string

const (
	TierBronze   TierLevel = "bronze"
	TierSilver   TierLevel = "silver"
	TierGold     TierLevel = "gold"
	TierPlatinum TierLevel = "platinum"
	TierDiamond  TierLevel = "diamond"
)

// PointsAction defines actions that earn/spend points
type PointsAction string

const (
	ActionPurchase        PointsAction = "purchase"
	ActionSale            PointsAction = "sale"
	ActionReview          PointsAction = "review"
	ActionReferral        PointsAction = "referral"
	ActionDailyLogin      PointsAction = "daily_login"
	ActionProfileComplete PointsAction = "profile_complete"
	ActionKYCVerified     PointsAction = "kyc_verified"
	ActionFirstPurchase   PointsAction = "first_purchase"
	ActionAuctionWin      PointsAction = "auction_win"
	ActionRedemption      PointsAction = "redemption"
	ActionExpired         PointsAction = "expired"
	ActionAdjustment      PointsAction = "adjustment"
)

// Points multipliers per tier
var TierMultipliers = map[TierLevel]float64{
	TierBronze:   1.0,
	TierSilver:   1.25,
	TierGold:     1.5,
	TierPlatinum: 2.0,
	TierDiamond:  2.5,
}

// Points required for each tier
var TierThresholds = map[TierLevel]int{
	TierBronze:   0,
	TierSilver:   1000,
	TierGold:     5000,
	TierPlatinum: 15000,
	TierDiamond:  50000,
}

// LoyaltyAccount represents a user's loyalty account
type LoyaltyAccount struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	CurrentPoints  int        `gorm:"default:0" json:"current_points"`
	LifetimePoints int        `gorm:"default:0" json:"lifetime_points"`
	Tier           TierLevel  `gorm:"type:varchar(20);default:'bronze'" json:"tier"`
	TierExpiresAt  *time.Time `json:"tier_expires_at,omitempty"`
	ReferralCode   string     `gorm:"type:varchar(20);uniqueIndex" json:"referral_code"`
	ReferredBy     *uuid.UUID `gorm:"type:uuid" json:"referred_by,omitempty"`
	TotalReferrals int        `gorm:"default:0" json:"total_referrals"`
	LastLoginBonus *time.Time `json:"last_login_bonus,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// PointsTransaction represents a points transaction
type PointsTransaction struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AccountID   uuid.UUID    `gorm:"type:uuid;not null;index" json:"account_id"`
	Action      PointsAction `gorm:"type:varchar(30);not null" json:"action"`
	Points      int          `gorm:"not null" json:"points"`  // Positive = earn, Negative = spend
	Balance     int          `gorm:"not null" json:"balance"` // Balance after transaction
	Multiplier  float64      `gorm:"default:1" json:"multiplier"`
	ReferenceID *string      `gorm:"type:varchar(100)" json:"reference_id,omitempty"` // Order ID, etc.
	Description string       `gorm:"type:text" json:"description,omitempty"`
	ExpiresAt   *time.Time   `json:"expires_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// Reward represents a redeemable reward
type Reward struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name               string     `gorm:"not null" json:"name"`
	Description        string     `gorm:"type:text" json:"description"`
	PointsCost         int        `gorm:"not null" json:"points_cost"`
	Type               string     `gorm:"type:varchar(30);not null" json:"type"` // discount, free_shipping, cashback, gift
	Value              float64    `gorm:"not null" json:"value"`                 // Discount %, cashback amount, etc.
	MinTier            TierLevel  `gorm:"type:varchar(20);default:'bronze'" json:"min_tier"`
	MaxRedemptions     *int       `json:"max_redemptions,omitempty"` // nil = unlimited
	CurrentRedemptions int        `gorm:"default:0" json:"current_redemptions"`
	ValidFrom          time.Time  `json:"valid_from"`
	ValidUntil         *time.Time `json:"valid_until,omitempty"`
	IsActive           bool       `gorm:"default:true" json:"is_active"`
	ImageURL           string     `json:"image_url,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// RewardRedemption represents a reward redemption
type RewardRedemption struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AccountID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"account_id"`
	RewardID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"reward_id"`
	PointsSpent int        `gorm:"not null" json:"points_spent"`
	Code        string     `gorm:"type:varchar(50);uniqueIndex" json:"code"`        // Redemption code
	Status      string     `gorm:"type:varchar(20);default:'active'" json:"status"` // active, used, expired
	UsedAt      *time.Time `json:"used_at,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`

	Reward Reward `gorm:"foreignKey:RewardID" json:"reward,omitempty"`
}

// Badge represents an achievement badge
type Badge struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	ImageURL    string    `json:"image_url"`
	Criteria    string    `gorm:"type:varchar(50);not null" json:"criteria"` // first_purchase, 10_reviews, etc.
	PointsBonus int       `gorm:"default:0" json:"points_bonus"`
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserBadge represents a badge earned by a user
type UserBadge struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AccountID uuid.UUID `gorm:"type:uuid;not null;index" json:"account_id"`
	BadgeID   uuid.UUID `gorm:"type:uuid;not null;index" json:"badge_id"`
	EarnedAt  time.Time `json:"earned_at"`

	Badge Badge `gorm:"foreignKey:BadgeID" json:"badge,omitempty"`
}

// Streak tracks user engagement streaks
type Streak struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AccountID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"account_id"`
	CurrentStreak int       `gorm:"default:0" json:"current_streak"`
	LongestStreak int       `gorm:"default:0" json:"longest_streak"`
	LastActivity  time.Time `json:"last_activity"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GenerateReferralCode generates a unique referral code
func GenerateReferralCode() string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 8)
	for i := range code {
		code[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(1 * time.Nanosecond)
	}
	return string(code)
}

// GetTierForPoints returns the tier for a given points total
func GetTierForPoints(points int) TierLevel {
	if points >= TierThresholds[TierDiamond] {
		return TierDiamond
	} else if points >= TierThresholds[TierPlatinum] {
		return TierPlatinum
	} else if points >= TierThresholds[TierGold] {
		return TierGold
	} else if points >= TierThresholds[TierSilver] {
		return TierSilver
	}
	return TierBronze
}

// GetNextTier returns the next tier and points needed
func GetNextTier(currentTier TierLevel, currentPoints int) (TierLevel, int) {
	switch currentTier {
	case TierBronze:
		return TierSilver, TierThresholds[TierSilver] - currentPoints
	case TierSilver:
		return TierGold, TierThresholds[TierGold] - currentPoints
	case TierGold:
		return TierPlatinum, TierThresholds[TierPlatinum] - currentPoints
	case TierPlatinum:
		return TierDiamond, TierThresholds[TierDiamond] - currentPoints
	default:
		return TierDiamond, 0
	}
}
