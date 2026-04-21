package cms

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Hero Slider — banner slides on homepage
// ════════════════════════════════════════════════════════════════════════════

type HeroSlide struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Title     string         `gorm:"size:200" json:"title"`
	Subtitle  string         `gorm:"size:400" json:"subtitle"`
	ImageURL  string         `gorm:"size:500;not null" json:"image_url"`
	LinkURL   string         `gorm:"size:500" json:"link_url,omitempty"`
	LinkLabel string         `gorm:"size:100" json:"link_label,omitempty"`
	Badge     string         `gorm:"size:50" json:"badge,omitempty"`  // e.g. "NEW", "SALE", "HOT"
	Position  int            `gorm:"default:0" json:"position"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	StartDate *time.Time     `json:"start_date,omitempty"` // scheduled visibility
	EndDate   *time.Time     `json:"end_date,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (HeroSlide) TableName() string { return "hero_slides" }

// ════════════════════════════════════════════════════════════════════════════
// Content Block — reusable editable content sections
// ════════════════════════════════════════════════════════════════════════════

type ContentBlockType string

const (
	ContentBlockHTML      ContentBlockType = "html"
	ContentBlockMarkdown  ContentBlockType = "markdown"
	ContentBlockImage     ContentBlockType = "image"
	ContentBlockHero      ContentBlockType = "hero"
	ContentBlockCTA       ContentBlockType = "cta"       // call-to-action
	ContentBlockFAQ       ContentBlockType = "faq"
	ContentBlockTestimonial ContentBlockType = "testimonial"
	ContentBlockFeatures  ContentBlockType = "features"
)

type ContentBlock struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Slug        string          `gorm:"size:100;uniqueIndex;not null" json:"slug"` // e.g. "homepage_hero", "footer_about"
	Title       string          `gorm:"size:200" json:"title"`
	Type        ContentBlockType `gorm:"size:30;default:'html'" json:"type"`
	Content     string          `gorm:"type:text" json:"content"`          // main content (HTML/MD/image URL)
	Content2    string          `gorm:"type:text" json:"content2,omitempty"` // secondary (subtitle, CTA label, etc.)
	ImageURL    string          `gorm:"size:500" json:"image_url,omitempty"`
	LinkURL     string          `gorm:"size:500" json:"link_url,omitempty"`
	Metadata    string          `gorm:"type:jsonb;default:'{}'" json:"metadata"` // extra data (FAQ items, feature list, etc.)
	Position    int             `gorm:"default:0" json:"position"`
	IsActive    bool            `gorm:"default:true" json:"is_active"`
	Page        string          `gorm:"size:100;index" json:"page"` // which page this block belongs to: "home", "about", etc.
	Section     string          `gorm:"size:100;index" json:"section"` // section within page: "hero", "features", "footer"
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	DeletedAt   gorm.DeletedAt  `gorm:"index" json:"-"`
}

func (ContentBlock) TableName() string { return "content_blocks" }

// ════════════════════════════════════════════════════════════════════════════
// Media Library — uploaded files management
// ════════════════════════════════════════════════════════════════════════════

type MediaType string

const (
	MediaTypeImage  MediaType = "image"
	MediaTypeVideo  MediaType = "video"
	MediaTypeDocument MediaType = "document"
)

type MediaFile struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	FileName   string     `gorm:"size:255;not null" json:"file_name"`
	FilePath   string     `gorm:"size:500;not null" json:"file_path"`
	URL        string     `gorm:"size:500;not null" json:"url"`
	MimeType   string     `gorm:"size:100" json:"mime_type"`
	SizeBytes  int64      `json:"size_bytes"`
	Type       MediaType  `gorm:"size:20;default:'image'" json:"type"`
	Alt        string     `gorm:"size:200" json:"alt,omitempty"`
	Width      int        `json:"width,omitempty"`
	Height     int        `json:"height,omitempty"`
	Folder     string     `gorm:"size:100;index" json:"folder,omitempty"` // e.g. "banners", "products", "logos"
	UploadedBy uuid.UUID  `gorm:"type:uuid" json:"uploaded_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (MediaFile) TableName() string { return "media_files" }

// ════════════════════════════════════════════════════════════════════════════
// Site Settings — global configurable settings (logo, colors, contact, social)
// ════════════════════════════════════════════════════════════════════════════

type SiteSetting struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Key       string    `gorm:"size:100;uniqueIndex;not null" json:"key"` // e.g. "site_logo", "primary_color", "contact_email"
	Value     string    `gorm:"type:text" json:"value"`
	Group     string    `gorm:"size:50;index" json:"group"` // "branding", "contact", "social", "seo", "general"
	Label     string    `gorm:"size:200" json:"label"` // human-readable label for admin UI
	Type      string    `gorm:"size:30;default:'text'" json:"type"` // "text", "textarea", "color", "image", "url", "email", "number", "boolean"
	UpdatedAt time.Time `json:"updated_at"`
}

func (SiteSetting) TableName() string { return "site_settings" }

// ════════════════════════════════════════════════════════════════════════════
// Navigation Menu — drag-and-drop menu builder
// ════════════════════════════════════════════════════════════════════════════

type NavMenu struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Location    string         `gorm:"size:50;uniqueIndex;not null" json:"location"` // "header", "footer", "mobile", "sidebar"
	Label       string         `gorm:"size:100;not null" json:"label"`
	URL         string         `gorm:"size:500" json:"url"`
	Icon        string         `gorm:"size:50" json:"icon,omitempty"` // lucide icon name
	ParentID    *uuid.UUID     `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	Position    int            `gorm:"default:0" json:"position"`
	IsExternal  bool           `gorm:"default:false" json:"is_external"` // open in new tab
	IsActive    bool           `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (NavMenu) TableName() string { return "nav_menus" }
