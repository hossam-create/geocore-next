package plugins

import (
	"time"

	"github.com/google/uuid"
)

type PluginStatus string

const (
	PluginDraft     PluginStatus = "draft"
	PluginPublished PluginStatus = "published"
	PluginDisabled  PluginStatus = "disabled"
	PluginArchived  PluginStatus = "archived"
)

type Plugin struct {
	ID           uuid.UUID    `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AuthorID     uuid.UUID    `gorm:"type:uuid;not null;index" json:"author_id"`
	Name         string       `gorm:"size:200;not null" json:"name"`
	Slug         string       `gorm:"size:200;uniqueIndex;not null" json:"slug"`
	Description  string       `gorm:"type:text" json:"description,omitempty"`
	Version      string       `gorm:"size:20;not null;default:'1.0.0'" json:"version"`
	Category     string       `gorm:"size:100;not null;default:'general'" json:"category"`
	IconURL      string       `gorm:"type:text" json:"icon_url,omitempty"`
	RepoURL      string       `gorm:"type:text" json:"repo_url,omitempty"`
	ConfigSchema string       `gorm:"type:jsonb;default:'{}'" json:"config_schema"`
	Price        float64      `gorm:"type:numeric(10,2);default:0" json:"price"`
	Currency     string       `gorm:"size:10;not null;default:'USD'" json:"currency"`
	IsFree       bool         `gorm:"not null;default:true" json:"is_free"`
	InstallCount int          `gorm:"default:0" json:"install_count"`
	AvgRating    float64      `gorm:"type:numeric(3,2);default:0" json:"avg_rating"`
	Status       PluginStatus `gorm:"size:50;not null;default:'draft'" json:"status"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

func (Plugin) TableName() string { return "plugins" }

type PluginInstall struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	PluginID    uuid.UUID `gorm:"type:uuid;not null" json:"plugin_id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Config      string    `gorm:"type:jsonb;default:'{}'" json:"config"`
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	InstalledAt time.Time `json:"installed_at"`
}

func (PluginInstall) TableName() string { return "plugin_installs" }
