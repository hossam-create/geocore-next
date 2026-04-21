package addons

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AddonStatus represents the lifecycle state of an installed addon.
type AddonStatus string

const (
	AddonStatusAvailable AddonStatus = "available" // listed in marketplace, not installed
	AddonStatusInstalled AddonStatus = "installed" // installed but not enabled
	AddonStatusEnabled  AddonStatus = "enabled"   // installed and active
	AddonStatusError    AddonStatus = "error"      // installation or runtime error
)

// Addon represents a marketplace addon/plugin that can be installed on the platform.
type Addon struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Slug          string         `gorm:"size:100;uniqueIndex;not null" json:"slug"`
	Name          string         `gorm:"size:200;not null" json:"name"`
	Description   string         `gorm:"type:text" json:"description"`
	Category      string         `gorm:"size:50;index" json:"category"`
	Tags          string         `gorm:"type:jsonb;default:'[]'" json:"tags"`
	IconURL       string         `gorm:"size:500" json:"icon_url,omitempty"`
	ScreenshotURL string         `gorm:"size:500" json:"screenshot_url,omitempty"`
	Author        string         `gorm:"size:100" json:"author"`
	AuthorURL     string         `gorm:"size:500" json:"author_url,omitempty"`
	Version       string         `gorm:"size:20" json:"version"` // latest version string
	DownloadURL   string         `gorm:"size:500" json:"download_url,omitempty"`
	DownloadCount int            `gorm:"default:0" json:"download_count"`
	AvgRating     float64        `gorm:"type:decimal(3,2);default:0" json:"avg_rating"`
	RatingCount   int            `gorm:"default:0" json:"rating_count"`
	IsFree        bool           `gorm:"default:true" json:"is_free"`
	Price         float64        `gorm:"type:decimal(10,2);default:0" json:"price,omitempty"`
	Currency      string         `gorm:"size:3;default:'AED'" json:"currency,omitempty"`
	IsVerified    bool           `gorm:"default:false" json:"is_verified"`
	IsOfficial    bool           `gorm:"default:false" json:"is_official"`
	Permissions   string         `gorm:"type:jsonb;default:'[]'" json:"permissions"` // required permissions
	Hooks         string         `gorm:"type:jsonb;default:'[]'" json:"hooks"`        // event hooks
	ConfigSchema  string         `gorm:"type:jsonb" json:"config_schema,omitempty"`   // JSON schema for config
	// Installation state
	Status      AddonStatus    `gorm:"size:30;default:'available';index" json:"status"`
	Config      string         `gorm:"type:jsonb" json:"config,omitempty"` // current instance config
	InstalledAt *time.Time      `json:"installed_at,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Addon) TableName() string { return "addons" }

// AddonVersion tracks version history for an addon.
type AddonVersion struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AddonID        uuid.UUID `gorm:"type:uuid;not null;index" json:"addon_id"`
	Version        string    `gorm:"size:20;not null" json:"version"`
	Changelog      string    `gorm:"type:text" json:"changelog,omitempty"`
	DownloadURL    string    `gorm:"size:500" json:"download_url,omitempty"`
	MinCoreVersion string    `gorm:"size:20" json:"min_core_version,omitempty"`
	MaxCoreVersion string    `gorm:"size:20" json:"max_core_version,omitempty"`
	Dependencies   string    `gorm:"type:jsonb;default:'[]'" json:"dependencies"`
	Manifest       string    `gorm:"type:jsonb" json:"manifest,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (AddonVersion) TableName() string { return "addon_versions" }

// AddonReview stores user ratings and reviews for an addon.
type AddonReview struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AddonID   uuid.UUID  `gorm:"type:uuid;not null;index" json:"addon_id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Rating    int        `gorm:"not null" json:"rating"` // 1-5
	Review    string     `gorm:"type:text" json:"review,omitempty"`
	Version   string     `gorm:"size:20" json:"version,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AddonReview) TableName() string { return "addon_reviews" }
