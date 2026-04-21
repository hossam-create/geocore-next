package settings

import (
	"time"

	"github.com/google/uuid"
)

// AdminSetting represents a single configurable platform setting.
type AdminSetting struct {
	Key         string     `gorm:"primaryKey;size:255" json:"key"`
	Value       string     `gorm:"type:text;not null" json:"value"`
	Type        string     `gorm:"size:50;not null" json:"type"`
	Category    string     `gorm:"size:100;not null;index" json:"category"`
	Label       string     `gorm:"size:255;not null" json:"label"`
	Description string     `gorm:"type:text" json:"description,omitempty"`
	Options     *string    `gorm:"type:jsonb" json:"options,omitempty"`
	IsPublic    bool       `gorm:"default:false" json:"is_public"`
	IsSecret    bool       `gorm:"default:false" json:"is_secret"`
	UpdatedAt   time.Time  `json:"updated_at"`
	UpdatedBy   *uuid.UUID `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (AdminSetting) TableName() string { return "admin_settings" }

// FeatureFlag represents a toggleable platform feature with rollout control.
type FeatureFlag struct {
	Key           string    `gorm:"primaryKey;size:255" json:"key"`
	Enabled       bool      `gorm:"default:false" json:"enabled"`
	RolloutPct    int       `gorm:"default:100" json:"rollout_pct"`
	AllowedGroups []string  `gorm:"type:text[];serializer:json" json:"allowed_groups,omitempty"`
	Description   string    `gorm:"type:text" json:"description,omitempty"`
	Category      string    `gorm:"size:100" json:"category,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (FeatureFlag) TableName() string { return "feature_flags" }

// SupportTicket represents a user support request.
type SupportTicket struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	AssignedTo *uuid.UUID `gorm:"type:uuid" json:"assigned_to,omitempty"`
	Subject    string     `gorm:"size:255;not null" json:"subject"`
	Status     string     `gorm:"size:20;not null;default:'open';index" json:"status"`
	Priority   string     `gorm:"size:20;not null;default:'medium'" json:"priority"`
	Category   string     `gorm:"size:100" json:"category,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	ClosedAt   *time.Time `json:"closed_at,omitempty"`

	// Joined
	UserName string          `gorm:"-" json:"user_name,omitempty"`
	Messages []TicketMessage `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
}

func (SupportTicket) TableName() string { return "support_tickets" }

// TicketMessage is a single message within a support ticket thread.
type TicketMessage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TicketID  uuid.UUID `gorm:"type:uuid;not null;index" json:"ticket_id"`
	SenderID  uuid.UUID `gorm:"type:uuid;not null" json:"sender_id"`
	Body      string    `gorm:"type:text;not null" json:"body"`
	IsAdmin   bool      `gorm:"default:false" json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`

	// Joined
	SenderName string `gorm:"-" json:"sender_name,omitempty"`
}

func (TicketMessage) TableName() string { return "ticket_messages" }

// TrustFlag represents a trust & safety flag for content moderation.
type TrustFlag struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TargetType string     `gorm:"size:50;not null;index:idx_trust_flags_target" json:"target_type"`
	TargetID   uuid.UUID  `gorm:"type:uuid;not null;index:idx_trust_flags_target" json:"target_id"`
	FlagType   string     `gorm:"size:100;not null" json:"flag_type"`
	Severity   string     `gorm:"size:20;not null" json:"severity"`
	Source     string     `gorm:"size:50;not null" json:"source"`
	Status     string     `gorm:"size:50;default:'open';index" json:"status"`
	Notes      string     `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy *uuid.UUID `gorm:"type:uuid" json:"resolved_by,omitempty"`
}

func (TrustFlag) TableName() string { return "trust_flags" }

// CategoryGroup wraps settings for grouped API responses.
type CategoryGroup struct {
	Category string         `json:"category"`
	Settings []AdminSetting `json:"settings"`
}
