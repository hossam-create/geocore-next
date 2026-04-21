package users

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string         `gorm:"not null" json:"name"`
	Email        string         `gorm:"uniqueIndex" json:"email"`
	Phone        string         `gorm:"uniqueIndex:idx_users_phone,where:phone <> ''" json:"phone,omitempty"`
	PasswordHash string         `json:"-"`
	AvatarURL    string         `json:"avatar_url,omitempty"`
	Bio          string         `json:"bio,omitempty"`
	Location     string         `json:"location,omitempty"`
	Language     string         `gorm:"size:10;default:'en'" json:"language,omitempty"`
	Currency     string         `gorm:"size:10;default:'USD'" json:"currency,omitempty"`
	Rating       float64        `gorm:"default:0" json:"rating"`
	ReviewCount  int            `gorm:"default:0" json:"review_count"`
	SoldCount    int            `gorm:"default:0" json:"sold_count"`
	IsVerified   bool           `gorm:"default:false" json:"is_verified"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	IsBanned     bool           `gorm:"default:false" json:"is_banned"`
	BanReason    string         `json:"-"`
	Role         string         `gorm:"default:'user'" json:"role"`
	Balance      float64        `gorm:"default:0" json:"balance"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	EmailVerified              bool       `gorm:"default:false" json:"email_verified"`
	VerificationToken          string     `gorm:"size:64;index" json:"-"`
	VerificationTokenExpiresAt *time.Time `json:"-"`

	GoogleID     string `gorm:"size:128;index" json:"-"`
	AppleID      string `gorm:"size:128;index" json:"-"`
	FacebookID   string `gorm:"size:128;index" json:"-"`
	AuthProvider string `gorm:"size:32;default:'email'" json:"auth_provider"`

	PasswordResetToken     string     `gorm:"size:64;index" json:"-"`
	PasswordResetExpiresAt *time.Time `json:"-"`
	PasswordChangedAt      *time.Time `json:"-"`

	StripeCustomerID string `gorm:"size:64;uniqueIndex:idx_users_stripe_customer_id,where:stripe_customer_id <> ''" json:"-"`
	ReferralCode     string `gorm:"size:16;uniqueIndex:idx_users_referral_code,where:referral_code <> ''" json:"referral_code,omitempty"`

	// Sprint 20: Private Invite Network
	IsPrivateMember bool `gorm:"default:false" json:"is_private_member"`
}

type PublicUser struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Bio         string    `json:"bio,omitempty"`
	Location    string    `json:"location,omitempty"`
	Rating      float64   `json:"rating"`
	ReviewCount int       `json:"review_count"`
	SoldCount   int       `json:"sold_count"`
	IsVerified  bool      `json:"is_verified"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

// BeforeCreate generates a UUID for the user ID if not already set.
// This ensures compatibility with both PostgreSQL and SQLite (used in tests).
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == (uuid.UUID{}) {
		u.ID = uuid.New()
	}
	return nil
}

func (u *User) ToPublic() PublicUser {
	return PublicUser{
		ID:          u.ID,
		Name:        u.Name,
		AvatarURL:   u.AvatarURL,
		Bio:         u.Bio,
		Location:    u.Location,
		Rating:      u.Rating,
		ReviewCount: u.ReviewCount,
		SoldCount:   u.SoldCount,
		IsVerified:  u.IsVerified,
		Role:        u.Role,
		CreatedAt:   u.CreatedAt,
	}
}
