package stores

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Storefront is a seller's branded store page on GeoCore.
// A seller can have at most one storefront (unique index on user_id).
type Storefront struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Slug        string         `gorm:"size:80;not null;uniqueIndex" json:"slug"`
	Name        string         `gorm:"size:120;not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	WelcomeMsg  string         `gorm:"type:text" json:"welcome_msg"`
	LogoURL     string         `gorm:"type:text" json:"logo_url"`
	BannerURL   string         `gorm:"type:text" json:"banner_url"`
	Views       int            `gorm:"default:0" json:"views"`
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
