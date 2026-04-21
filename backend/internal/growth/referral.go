package growth

import (
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Referral & Invite System
// Viral loop: referral codes, wallet rewards, traveler invites.
// ════════════════════════════════════════════════════════════════════════════

// Referral tracks a referral from one user to another.
type Referral struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ReferrerID   uuid.UUID       `gorm:"type:uuid;not null;index" json:"referrer_id"`
	RefereeID    uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex" json:"referee_id"`
	Code         string          `gorm:"size:16;not null;index" json:"code"`
	Status       string          `gorm:"size:20;not null;default:'pending'" json:"status"` // pending, completed, rewarded
	RewardAmount decimal.Decimal `gorm:"type:decimal(12,2);not null;default:0" json:"reward_amount"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

func (Referral) TableName() string { return "referrals" }

// TravelerInvite tracks invitations sent to potential travelers.
type TravelerInvite struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	InviterID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"inviter_id"`
	InviteeEmail string     `gorm:"size:255;not null" json:"invitee_email"`
	Code         string     `gorm:"size:16;not null;uniqueIndex" json:"code"`
	Status       string     `gorm:"size:20;not null;default:'sent'" json:"status"` // sent, registered, completed, rewarded
	RegisteredID *uuid.UUID `gorm:"type:uuid" json:"registered_id,omitempty"`
	RewardClaimed bool      `gorm:"not null;default:false" json:"reward_claimed"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (TravelerInvite) TableName() string { return "traveler_invites" }

const (
	ReferralStatusPending   = "pending"
	ReferralStatusCompleted = "completed"
	ReferralStatusRewarded  = "rewarded"

	InviteStatusSent       = "sent"
	InviteStatusRegistered = "registered"
	InviteStatusCompleted  = "completed"
	InviteStatusRewarded   = "rewarded"
)

// Referral reward config
var (
	ReferrerReward  = decimal.NewFromFloat(5.00)  // $5 wallet credit
	RefereeDiscount = decimal.NewFromFloat(3.00)   // $3 first-order discount
	TravelerReward  = decimal.NewFromFloat(10.00)  // $10 after first completed delivery
)

// GenerateReferralCode creates a unique 8-character referral code.
func GenerateReferralCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no ambiguous chars
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// CreateReferral creates a referral record when a new user signs up with a code.
func CreateReferral(db *gorm.DB, referrerID, refereeID uuid.UUID, code string) error {
	referral := Referral{
		ID:           uuid.New(),
		ReferrerID:   referrerID,
		RefereeID:    refereeID,
		Code:         code,
		Status:       ReferralStatusPending,
		RewardAmount: ReferrerReward,
	}
	return db.Create(&referral).Error
}

// CompleteReferral marks a referral as completed when the referee completes their first order.
func CompleteReferral(db *gorm.DB, refereeID uuid.UUID, notifSvc *notifications.Service) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var referral Referral
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("referee_id = ? AND status = ?", refereeID, ReferralStatusPending).
			First(&referral).Error; err != nil {
			return nil // no pending referral — not an error
		}

		now := time.Now()
		referral.Status = ReferralStatusCompleted
		referral.CompletedAt = &now

		if err := tx.Save(&referral).Error; err != nil {
			return err
		}

		// Credit referrer wallet
		if err := creditWallet(tx, referral.ReferrerID, ReferrerReward, referral.ID); err != nil {
			slog.Error("referral: failed to credit referrer wallet", "referrer_id", referral.ReferrerID, "error", err)
		}

		// Mark as rewarded
		referral.Status = ReferralStatusRewarded
		tx.Save(&referral)

		// Notify referrer
		if notifSvc != nil {
			go notifSvc.Notify(notifications.NotifyInput{
				UserID: referral.ReferrerID,
				Type:   "referral_reward",
				Title:  "Referral Reward!",
				Body:   fmt.Sprintf("You earned $%s for a successful referral!", ReferrerReward.String()),
				Data:   map[string]string{"referral_id": referral.ID.String(), "reward": ReferrerReward.String()},
			})
		}

		slog.Info("referral: completed", "referral_id", referral.ID, "referrer_id", referral.ReferrerID)
		return nil
	})
}

// InviteTraveler creates a traveler invitation link.
func InviteTraveler(db *gorm.DB, inviterID uuid.UUID, inviteeEmail string) (*TravelerInvite, error) {
	code := GenerateReferralCode()
	invite := TravelerInvite{
		ID:           uuid.New(),
		InviterID:    inviterID,
		InviteeEmail: inviteeEmail,
		Code:         code,
		Status:       InviteStatusSent,
	}
	if err := db.Create(&invite).Error; err != nil {
		return nil, err
	}
	return &invite, nil
}

// CompleteTravelerInvite rewards the inviter when the invited traveler completes a delivery.
func CompleteTravelerInvite(db *gorm.DB, travelerUserID uuid.UUID, notifSvc *notifications.Service) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var invite TravelerInvite
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("registered_id = ? AND status IN ?", travelerUserID,
				[]string{InviteStatusRegistered, InviteStatusCompleted}).
			First(&invite).Error; err != nil {
			return nil // no matching invite
		}

		invite.Status = InviteStatusCompleted

		if !invite.RewardClaimed {
			invite.RewardClaimed = true
			// Credit inviter wallet
			if err := creditWallet(tx, invite.InviterID, TravelerReward, invite.ID); err != nil {
				slog.Error("referral: failed to credit traveler invite wallet", "inviter_id", invite.InviterID, "error", err)
			}

			if notifSvc != nil {
				go notifSvc.Notify(notifications.NotifyInput{
					UserID: invite.InviterID,
					Type:   "traveler_invite_reward",
					Title:  "Traveler Invite Reward!",
					Body:   fmt.Sprintf("You earned $%s for inviting a traveler!", TravelerReward.String()),
					Data:   map[string]string{"invite_id": invite.ID.String()},
				})
			}
		}

		return tx.Save(&invite).Error
	})
}

// RegisterTravelerInvite marks an invite as registered when the invitee signs up.
func RegisterTravelerInvite(db *gorm.DB, code string, newUserID uuid.UUID) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var invite TravelerInvite
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code = ? AND status = ?", code, InviteStatusSent).
			First(&invite).Error; err != nil {
			return nil // no matching invite — not an error
		}

		invite.RegisteredID = &newUserID
		invite.Status = InviteStatusRegistered
		return tx.Save(&invite).Error
	})
}

// GetReferralStats returns referral metrics for a user.
func GetReferralStats(db *gorm.DB, userID uuid.UUID) map[string]interface{} {
	var totalReferrals int64
	var completedReferrals int64
	var totalEarned decimal.Decimal

	db.Model(&Referral{}).Where("referrer_id = ?", userID).Count(&totalReferrals)
	db.Model(&Referral{}).Where("referrer_id = ? AND status = ?", userID, ReferralStatusRewarded).Count(&completedReferrals)
	db.Model(&Referral{}).Where("referrer_id = ? AND status = ?", userID, ReferralStatusRewarded).
		Select("COALESCE(SUM(reward_amount),0)").Scan(&totalEarned)

	return map[string]interface{}{
		"total_referrals":     totalReferrals,
		"completed_referrals": completedReferrals,
		"total_earned":        totalEarned.String(),
	}
}

// creditWallet is a helper to credit a user's wallet balance.
func creditWallet(db *gorm.DB, userID uuid.UUID, amount decimal.Decimal, refID uuid.UUID) error {
	return db.Table("users").Where("id = ?", userID).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
}
