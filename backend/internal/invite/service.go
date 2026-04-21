package invite

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/reputation"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotEligible   = errors.New("invite: trust score too low to create invites")
	ErrQuotaExceeded = errors.New("invite: invite quota exhausted")
	ErrCodeNotFound  = errors.New("invite: code not found")
	ErrCodeExpired   = errors.New("invite: code has expired")
	ErrCodeExhausted = errors.New("invite: code has reached max uses")
	ErrCodeInactive  = errors.New("invite: code is inactive")
	ErrSelfReferral  = errors.New("invite: self-referral not allowed")
)

// AllowedInviteCount returns quota based on trust score (requires ≥60).
func AllowedInviteCount(score float64) int {
	switch {
	case score >= 85:
		return 10
	case score >= 70:
		return 5
	case score >= 60:
		return 2
	default:
		return 0
	}
}

// CreateInvite creates a trust-gated invite for a user.
func CreateInvite(db *gorm.DB, inviterID uuid.UUID, ttlDays int) (*Invite, error) {
	score := reputation.GetOverallScore(db, inviterID)
	quota := AllowedInviteCount(score)
	if quota == 0 {
		return nil, ErrNotEligible
	}
	var existing int64
	db.Model(&Invite{}).Where("inviter_id = ? AND is_active = true", inviterID).Count(&existing)
	if int(existing) >= quota {
		return nil, ErrQuotaExceeded
	}
	code := GenerateInviteCode()
	for i := 0; i < 5; i++ {
		var check Invite
		if db.Where("invite_code = ?", code).First(&check).Error != nil {
			break
		}
		code = GenerateInviteCode()
	}
	inv := Invite{InviterID: inviterID, InviteCode: code, MaxUses: 3, IsActive: true}
	if ttlDays > 0 {
		t := time.Now().Add(time.Duration(ttlDays) * 24 * time.Hour)
		inv.ExpiresAt = &t
	}
	if err := db.Create(&inv).Error; err != nil {
		return nil, err
	}
	slog.Info("invite: created", "inviter_id", inviterID, "code", code)
	return &inv, nil
}

// ValidateInviteCode checks the code without consuming it.
func ValidateInviteCode(db *gorm.DB, code string) (*Invite, error) {
	var inv Invite
	if err := db.Where("invite_code = ?", code).First(&inv).Error; err != nil {
		return nil, ErrCodeNotFound
	}
	if !inv.IsActive {
		return nil, ErrCodeInactive
	}
	if inv.ExpiresAt != nil && inv.ExpiresAt.Before(time.Now()) {
		return nil, ErrCodeExpired
	}
	if inv.UsedCount >= inv.MaxUses {
		return nil, ErrCodeExhausted
	}
	return &inv, nil
}

// UseInvite atomically consumes one invite use and promotes user to private member.
func UseInvite(db *gorm.DB, code string, newUserID uuid.UUID) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var inv Invite
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("invite_code = ?", code).First(&inv).Error; err != nil {
			return ErrCodeNotFound
		}
		if !inv.IsActive || (inv.ExpiresAt != nil && inv.ExpiresAt.Before(time.Now())) || inv.UsedCount >= inv.MaxUses {
			return ErrCodeExhausted
		}
		if inv.InviterID == newUserID {
			return ErrSelfReferral
		}
		inv.UsedCount++
		if inv.UsedCount >= inv.MaxUses {
			inv.IsActive = false
		}
		if err := tx.Save(&inv).Error; err != nil {
			return err
		}
		usage := InviteUsage{InviteID: inv.ID, InvitedUserID: newUserID, UsedAt: time.Now()}
		if err := tx.Create(&usage).Error; err != nil {
			return err
		}
		tx.Exec("UPDATE users SET is_private_member = true WHERE id = ?", newUserID)
		return nil
	})
}

// CheckRapidSignup returns true if >3 signups from same IP in 1 hour.
func CheckRapidSignup(ctx context.Context, rdb *redis.Client, ip string) bool {
	if rdb == nil || ip == "" {
		return false
	}
	key := fmt.Sprintf("signup:ip:%s", ip)
	n, _ := rdb.Incr(ctx, key).Result()
	if n == 1 {
		rdb.Expire(ctx, key, time.Hour)
	}
	return n > 3
}

// FlagAbuser applies a -15 trust penalty and logs the event.
func FlagAbuser(db *gorm.DB, userID uuid.UUID, reason string) {
	for _, role := range []string{"buyer", "seller", "traveler"} {
		reputation.ApplyScoreDelta(db, userID, role, -15, reason)
	}
	slog.Warn("invite: abuse flagged", "user_id", userID, "reason", reason)
}

// GetInviterForUser returns the inviter's ID from the invite usage record.
func GetInviterForUser(db *gorm.DB, newUserID uuid.UUID) (uuid.UUID, bool) {
	var usage InviteUsage
	if err := db.Where("invited_user_id = ?", newUserID).First(&usage).Error; err != nil {
		return uuid.Nil, false
	}
	var inv Invite
	if err := db.First(&inv, "id = ?", usage.InviteID).Error; err != nil {
		return uuid.Nil, false
	}
	return inv.InviterID, true
}

// CreatePendingReward records a pending referral reward for the inviter.
func CreatePendingReward(db *gorm.DB, inviterID, referredUserID uuid.UUID) error {
	r := ReferralReward{
		UserID: inviterID, ReferredUserID: referredUserID,
		RewardType: "fee_discount", Amount: 5.0, Status: "pending",
	}
	return db.Create(&r).Error
}

// GrantReward marks the pending reward as granted after first transaction.
func GrantReward(db *gorm.DB, referredUserID uuid.UUID) error {
	now := time.Now()
	return db.Model(&ReferralReward{}).
		Where("referred_user_id = ? AND status = 'pending'", referredUserID).
		Updates(map[string]interface{}{"status": "granted", "granted_at": now}).Error
}

// QualifyReferral marks an InviteUsage as qualified when the referred user
// completes signup AND performs a first meaningful action (called externally).
// It also applies the inviter's referral boost as a trusted referral (+5 bonus).
func QualifyReferral(db *gorm.DB, referredUserID uuid.UUID) error {
	result := db.Model(&InviteUsage{}).
		Where("invited_user_id = ? AND referral_status = 'pending'", referredUserID).
		Update("referral_status", "qualified")
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return nil // already qualified or no record — not an error
	}
	slog.Info("invite: referral qualified", "referred_user_id", referredUserID)
	// Immediately grant the pending reward as the action is now confirmed.
	return GrantReward(db, referredUserID)
}

// RejectReferral marks an InviteUsage as rejected (abuse or fake account detected).
// Voids any pending reward and logs the event.
func RejectReferral(db *gorm.DB, referredUserID uuid.UUID, reason string) {
	db.Model(&InviteUsage{}).
		Where("invited_user_id = ? AND referral_status = 'pending'", referredUserID).
		Update("referral_status", "rejected")
	db.Model(&ReferralReward{}).
		Where("referred_user_id = ? AND status = 'pending'", referredUserID).
		Update("status", "voided")
	slog.Warn("invite: referral rejected", "referred_user_id", referredUserID, "reason", reason)
}
