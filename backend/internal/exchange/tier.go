package exchange

// tier.go — VIP Community Exchange tier system.
//
// Tier levels:
//   free  — 1 active request, standard fees, standard matching priority
//   vip   — unlimited requests, reduced fees, priority matching
//   pro   — unlimited requests, lowest fees, auto-matching, preferred visibility

import (
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Constants
// ════════════════════════════════════════════════════════════════════════════

const (
	TierFree = "free"
	TierVIP  = "vip"
	TierPro  = "pro"
)

// Tier limits
const (
	FreeMaxActiveRequests = 1
	VIPMaxActiveRequests  = -1 // unlimited
	ProMaxActiveRequests  = -1 // unlimited
)

// Fee multipliers per tier (applied to base fee rates)
const (
	FreeFeeMult = 1.00 // no discount
	VIPFeeMult  = 0.70 // 30% off
	ProFeeMult  = 0.50 // 50% off
)

// Matching priority boost added to composite score before ranking
const (
	FreePriorityBoost = 0.0
	VIPPriorityBoost  = 15.0
	ProPriorityBoost  = 30.0
)

// ════════════════════════════════════════════════════════════════════════════
// Model
// ════════════════════════════════════════════════════════════════════════════

// ExchangeUserTier records the current VIP tier for a user.
type ExchangeUserTier struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex"                  json:"user_id"`
	Tier      string         `gorm:"size:10;not null;default:'free'"                 json:"tier"`
	ExpiresAt *time.Time     `json:"expires_at,omitempty"` // nil = lifetime
	GrantedBy *uuid.UUID     `gorm:"type:uuid"                                       json:"granted_by,omitempty"`
	Note      string         `gorm:"type:text"                                       json:"note,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                           json:"-"`
}

func (ExchangeUserTier) TableName() string { return "exchange_user_tiers" }

// ════════════════════════════════════════════════════════════════════════════
// Tier helpers
// ════════════════════════════════════════════════════════════════════════════

// GetUserTier returns the active tier for a user.
// Returns TierFree for users with no record or an expired record.
func GetUserTier(db *gorm.DB, userID uuid.UUID) string {
	var t ExchangeUserTier
	if err := db.Where("user_id = ?", userID).First(&t).Error; err != nil {
		return TierFree
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return TierFree
	}
	return t.Tier
}

// SetUserTier upserts the tier for a user (admin action).
func SetUserTier(db *gorm.DB, userID uuid.UUID, tier string, expiresAt *time.Time, grantedBy uuid.UUID, note string) (*ExchangeUserTier, error) {
	switch tier {
	case TierFree, TierVIP, TierPro:
	default:
		return nil, errors.New("tier: invalid tier, must be free/vip/pro")
	}
	var record ExchangeUserTier
	db.Where("user_id = ?", userID).First(&record)
	record.UserID = userID
	record.Tier = tier
	record.ExpiresAt = expiresAt
	g := grantedBy
	record.GrantedBy = &g
	record.Note = note

	if record.ID == uuid.Nil {
		record.ID = uuid.New()
		if err := db.Create(&record).Error; err != nil {
			return nil, err
		}
	} else {
		if err := db.Save(&record).Error; err != nil {
			return nil, err
		}
	}
	slog.Info("exchange: tier updated", "user_id", userID, "tier", tier)
	return &record, nil
}

// TierFeeMultiplier returns the fee multiplier for a tier.
func TierFeeMultiplier(tier string) float64 {
	switch tier {
	case TierVIP:
		return VIPFeeMult
	case TierPro:
		return ProFeeMult
	default:
		return FreeFeeMult
	}
}

// TierPriorityBoost returns the matching priority boost for a tier.
func TierPriorityBoost(tier string) float64 {
	switch tier {
	case TierVIP:
		return VIPPriorityBoost
	case TierPro:
		return ProPriorityBoost
	default:
		return FreePriorityBoost
	}
}

// CanCreateRequest checks if a user is within their tier's active request limit.
func CanCreateRequest(db *gorm.DB, userID uuid.UUID) error {
	tier := GetUserTier(db, userID)
	max := activeRequestLimit(tier)
	if max < 0 {
		return nil // unlimited
	}
	var count int64
	db.Model(&ExchangeRequest{}).
		Where("user_id = ? AND status = ?", userID, StatusOpen).
		Count(&count)
	if int(count) >= max {
		return errors.New("tier: active request limit reached — upgrade to VIP for unlimited requests")
	}
	return nil
}

// IsProAutoMatch returns true if the user's tier supports auto-matching.
func IsProAutoMatch(db *gorm.DB, userID uuid.UUID) bool {
	return GetUserTier(db, userID) == TierPro
}

// activeRequestLimit returns max open requests for a tier (-1 = unlimited).
func activeRequestLimit(tier string) int {
	switch tier {
	case TierVIP, TierPro:
		return -1
	default:
		return FreeMaxActiveRequests
	}
}

// TierCapabilities is returned to the client to show what their tier unlocks.
type TierCapabilities struct {
	Tier             string   `json:"tier"`
	MaxActiveReqs    int      `json:"max_active_requests"` // -1 = unlimited
	FeeDiscount      float64  `json:"fee_discount_pct"`
	PriorityMatching bool     `json:"priority_matching"`
	AutoMatching     bool     `json:"auto_matching"`
	FasterDisputes   bool     `json:"faster_disputes"`
	Perks            []string `json:"perks"`
}

// GetTierCapabilities returns the human-readable capability set for a tier.
func GetTierCapabilities(tier string) TierCapabilities {
	switch tier {
	case TierVIP:
		return TierCapabilities{
			Tier:             TierVIP,
			MaxActiveReqs:    -1,
			FeeDiscount:      30,
			PriorityMatching: true,
			AutoMatching:     false,
			FasterDisputes:   true,
			Perks: []string{
				"Unlimited active exchange requests",
				"30% fee discount",
				"Priority position in matching queue",
				"Faster dispute resolution",
			},
		}
	case TierPro:
		return TierCapabilities{
			Tier:             TierPro,
			MaxActiveReqs:    -1,
			FeeDiscount:      50,
			PriorityMatching: true,
			AutoMatching:     true,
			FasterDisputes:   true,
			Perks: []string{
				"Unlimited active exchange requests",
				"50% fee discount",
				"Top priority in matching queue",
				"Automatic matching (no manual trigger needed)",
				"Preferred visibility in request listings",
				"Fastest dispute resolution",
			},
		}
	default:
		return TierCapabilities{
			Tier:             TierFree,
			MaxActiveReqs:    FreeMaxActiveRequests,
			FeeDiscount:      0,
			PriorityMatching: false,
			AutoMatching:     false,
			FasterDisputes:   false,
			Perks: []string{
				"1 active exchange request",
				"Standard fees",
			},
		}
	}
}
