package users

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name           string         `gorm:"not null" json:"name"`
	Email          string         `gorm:"uniqueIndex;not null" json:"email"`
	Phone          string         `gorm:"uniqueIndex" json:"phone,omitempty"`
	PasswordHash   string         `gorm:"not null" json:"-"`
	AvatarURL      string         `json:"avatar_url,omitempty"`
	Bio            string         `json:"bio,omitempty"`
	Location       string         `json:"location,omitempty"`
	Rating         float64        `gorm:"default:0" json:"rating"`
	ReviewCount    int            `gorm:"default:0" json:"review_count"`
	IsVerified     bool           `gorm:"default:false" json:"is_verified"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	Role           string         `gorm:"default:user" json:"role"` // user | admin | moderator
	Language       string         `gorm:"default:en" json:"language"`
	Currency       string         `gorm:"default:USD" json:"currency"`
	LastSeenAt     *time.Time     `json:"last_seen_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type PublicUser struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	AvatarURL   string     `json:"avatar_url"`
	Rating      float64    `json:"rating"`
	ReviewCount int        `json:"review_count"`
	IsVerified  bool       `json:"is_verified"`
	Location    string     `json:"location"`
	CreatedAt   time.Time  `json:"member_since"`
}

func (u *User) ToPublic() PublicUser {
	return PublicUser{
		ID:          u.ID,
		Name:        u.Name,
		AvatarURL:   u.AvatarURL,
		Rating:      u.Rating,
		ReviewCount: u.ReviewCount,
		IsVerified:  u.IsVerified,
		Location:    u.Location,
		CreatedAt:   u.CreatedAt,
	}
}
