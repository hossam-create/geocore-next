package ads

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Placement controls where the ad appears on the public site.
type Placement string

const (
	PlacementHero          Placement = "hero"
	PlacementSidebar       Placement = "sidebar"
	PlacementCategory      Placement = "category"
	PlacementListingFooter Placement = "listing_footer"
)

// Ad represents a banner advertisement managed by admins.
type Ad struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Title      string         `gorm:"size:255;not null" json:"title"`
	ImageURL   string         `gorm:"type:text;not null" json:"image_url"`
	LinkURL    string         `gorm:"type:text" json:"link_url"`
	Placement  Placement      `gorm:"size:50;not null;index" json:"placement"`
	Position   int            `gorm:"default:0" json:"position"`
	Enabled    bool           `gorm:"default:true;index" json:"enabled"`
	StartDate  *time.Time     `json:"start_date,omitempty"`
	EndDate    *time.Time     `json:"end_date,omitempty"`
	ClickCount int64          `gorm:"default:0" json:"click_count"`
	ViewCount  int64          `gorm:"default:0" json:"view_count"`
	CreatedBy  *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Ad) TableName() string { return "ads" }

// AdCreateRequest is the payload for creating a new ad.
type AdCreateRequest struct {
	Title     string    `json:"title" binding:"required,min=1,max=255"`
	ImageURL  string    `json:"image_url" binding:"required,url"`
	LinkURL   string    `json:"link_url" binding:"omitempty,url"`
	Placement Placement `json:"placement" binding:"required,oneof=hero sidebar category listing_footer"`
	Position  int       `json:"position"`
	Enabled   *bool     `json:"enabled"`
	StartDate *string   `json:"start_date"`
	EndDate   *string   `json:"end_date"`
}

// AdUpdateRequest is the payload for updating an ad.
type AdUpdateRequest struct {
	Title     *string    `json:"title" binding:"omitempty,min=1,max=255"`
	ImageURL  *string    `json:"image_url" binding:"omitempty,url"`
	LinkURL   *string    `json:"link_url" binding:"omitempty,url"`
	Placement *Placement `json:"placement" binding:"omitempty,oneof=hero sidebar category listing_footer"`
	Position  *int       `json:"position"`
	Enabled   *bool      `json:"enabled"`
	StartDate *string    `json:"start_date"`
	EndDate   *string    `json:"end_date"`
}
