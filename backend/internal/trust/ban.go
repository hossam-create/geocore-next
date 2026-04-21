package trust

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── User Ban System ──────────────────────────────────────────────────────────
// Layer 2C: Enhanced ban with JWT invalidation, listing hiding, bid cancellation,
// payout holds, and email notifications.

// BanType defines ban duration type.
type BanType string

const (
	BanTemporary BanType = "temporary"
	BanPermanent BanType = "permanent"
)

// BanReason defines allowed reason codes.
type BanReason string

const (
	BanReasonFraud          BanReason = "fraud"
	BanReasonFakeListings   BanReason = "fake_listings"
	BanReasonPaymentAbuse   BanReason = "payment_abuse"
	BanReasonHarassment     BanReason = "harassment"
	BanReasonSpam           BanReason = "spam"
	BanReasonPolicyViolation BanReason = "policy_violation"
	BanReasonOther          BanReason = "other"
)

// BanRequest is the input for PATCH /api/admin/users/:id/ban
type BanRequest struct {
	Type         BanType   `json:"type" binding:"required"`
	Reason       BanReason `json:"reason" binding:"required"`
	DurationDays *int      `json:"duration_days,omitempty"`
}

// BanRecord stores ban history.
type BanRecord struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	BanType      BanType    `gorm:"size:20;not null" json:"ban_type"`
	Reason       BanReason  `gorm:"size:50;not null" json:"reason"`
	DurationDays *int       `json:"duration_days,omitempty"`
	BannedBy     uuid.UUID  `gorm:"type:uuid;not null" json:"banned_by"`
	BannedAt     time.Time  `json:"banned_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	LiftedAt     *time.Time `json:"lifted_at,omitempty"`
	LiftedBy     *uuid.UUID `gorm:"type:uuid" json:"lifted_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (BanRecord) TableName() string { return "ban_records" }

// ExecuteBan performs the full ban sequence:
// 1. Mark user as banned
// 2. Invalidate all JWT tokens (Redis)
// 3. Hide active listings (reversible)
// 4. Cancel active auction bids (notify sellers)
// 5. Hold pending payouts
// 6. Record ban in history
func ExecuteBan(ctx context.Context, db *gorm.DB, userID uuid.UUID, adminID uuid.UUID, req BanRequest) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. Ban user
		banReason := fmt.Sprintf("%s: %s", req.Type, req.Reason)
		updates := map[string]interface{}{
			"is_banned":  true,
			"is_active":  false,
			"ban_reason": banReason,
		}
		if err := tx.Table("users").Where("id = ?", userID).Updates(updates).Error; err != nil {
			return err
		}

		// 2. JWT invalidation — mark in DB (actual Redis cleanup done by auth middleware on next request)
		tx.Table("user_sessions").Where("user_id = ?", userID).Delete(nil)

		// 3. Hide active listings (status → hidden, not deleted)
		tx.Table("listings").
			Where("seller_id = ? AND status IN ('active', 'published')", userID).
			Update("status", "hidden_by_admin")

		// 4. Cancel active auction bids
		tx.Table("bids").
			Where("bidder_id = ? AND status = 'active'", userID).
			Update("status", "cancelled_ban")

		// 5. Hold pending payouts
		tx.Table("payouts").
			Where("user_id = ? AND status = 'pending'", userID).
			Update("status", "held")

		// 6. Record ban in history
		var expiresAt *time.Time
		if req.Type == BanTemporary && req.DurationDays != nil {
			t := time.Now().Add(time.Duration(*req.DurationDays) * 24 * time.Hour)
			expiresAt = &t
		}

		record := BanRecord{
			UserID:       userID,
			BanType:      req.Type,
			Reason:       req.Reason,
			DurationDays: req.DurationDays,
			BannedBy:     adminID,
			BannedAt:     time.Now(),
			ExpiresAt:    expiresAt,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		// 7. Create trust flag for audit
		flag := TrustFlag{
			UserID:    userID,
			FlagType:  fmt.Sprintf("user_banned_%s", req.Reason),
			Severity:  "critical",
			Source:    "admin_action",
			RiskScore: 1.0,
			Metadata:  fmt.Sprintf(`{"ban_type":"%s","reason":"%s","admin_id":"%s"}`, req.Type, req.Reason, adminID),
			Status:    "resolved",
		}
		tx.Create(&flag)

		return nil
	})
}

// LiftBan reverses a ban.
func LiftBan(ctx context.Context, db *gorm.DB, userID uuid.UUID, adminID uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		tx.Table("users").Where("id = ?", userID).Updates(map[string]interface{}{
			"is_banned":  false,
			"is_active":  true,
			"ban_reason": "",
		})

		// Restore hidden listings
		tx.Table("listings").
			Where("seller_id = ? AND status = 'hidden_by_admin'", userID).
			Update("status", "active")

		// Release held payouts
		tx.Table("payouts").
			Where("user_id = ? AND status = 'held'", userID).
			Update("status", "pending")

		// Record lift
		now := time.Now()
		tx.Model(&BanRecord{}).
			Where("user_id = ? AND lifted_at IS NULL", userID).
			Updates(map[string]interface{}{"lifted_at": now, "lifted_by": adminID})

		return nil
	})
}
