package invite

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Invite holds a unique invite code issued by a trusted user.
type Invite struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	InviterID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"inviter_id"`
	InviteCode string     `gorm:"size:8;uniqueIndex;not null" json:"invite_code"`
	MaxUses    int        `gorm:"not null;default:3" json:"max_uses"`
	UsedCount  int        `gorm:"not null;default:0" json:"used_count"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	IsActive   bool       `gorm:"not null;default:true" json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (Invite) TableName() string { return "invites" }

func (i *Invite) BeforeCreate(_ *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// InviteUsage records each time an invite code is consumed.
type InviteUsage struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	InviteID       uuid.UUID `gorm:"type:uuid;not null;index" json:"invite_id"`
	InvitedUserID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"invited_user_id"`
	UsedAt         time.Time `json:"used_at"`
	ReferralStatus string    `gorm:"size:16;not null;default:'pending'" json:"referral_status"` // pending / qualified / rejected
}

func (InviteUsage) TableName() string { return "invite_usages" }

func (u *InviteUsage) BeforeCreate(_ *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// ReferralReward is a delayed reward granted after the referred user's first transaction.
type ReferralReward struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	ReferredUserID uuid.UUID  `gorm:"type:uuid;not null" json:"referred_user_id"`
	RewardType     string     `gorm:"size:32;not null;default:'fee_discount'" json:"reward_type"` // fee_discount / boost / credits
	Amount         float64    `gorm:"type:numeric(12,4);not null;default:5" json:"amount"`
	Status         string     `gorm:"size:16;not null;default:'pending'" json:"status"` // pending / granted
	GrantedAt      *time.Time `json:"granted_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (ReferralReward) TableName() string { return "referral_rewards" }

func (r *ReferralReward) BeforeCreate(_ *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

const codeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// GenerateInviteCode produces a random 8-char unambiguous code.
func GenerateInviteCode() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = codeCharset[rand.Intn(len(codeCharset))]
	}
	return string(b)
}

// AutoMigrate creates/updates invite-related tables.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Invite{}, &InviteUsage{}, &ReferralReward{})
}
