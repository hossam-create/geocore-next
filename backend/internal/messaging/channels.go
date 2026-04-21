package messaging

import (
	"time"

	"github.com/google/uuid"
)

// ── Messaging Channels ────────────────────────────────────────────────────────────
//
// Supports: push, email, in_app (WebSocket)
// Anti-spam: max 3 messages/hour/user, cooldown per type

// Channel types
const (
	ChannelPush  = "push"
	ChannelEmail = "email"
	ChannelInApp = "in_app"
)

// Message represents a message to be sent to a user.
type Message struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Type        string     `gorm:"size:30;not null;index" json:"type"` // nudge, reminder, win, loss, promo
	Title       string     `gorm:"size:200;not null" json:"title"`
	Body        string     `gorm:"type:text;not null" json:"body"`
	Priority    string     `gorm:"size:10;not null;default:'normal'" json:"priority"` // high, normal, low
	Channel     string     `gorm:"size:20;not null;default:'push'" json:"channel"`
	Metadata    string     `gorm:"type:text" json:"metadata"`                        // JSON
	Status      string     `gorm:"size:20;not null;default:'pending'" json:"status"` // pending, sent, delivered, failed
	SentAt      *time.Time `json:"sent_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	OpenedAt    *time.Time `json:"opened_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (Message) TableName() string { return "messaging_messages" }

// MessageCooldown tracks per-type cooldowns to prevent spam.
type MessageCooldown struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	MsgType    string    `gorm:"size:30;not null;index" json:"msg_type"`
	LastSentAt time.Time `json:"last_sent_at"`
}

func (MessageCooldown) TableName() string { return "messaging_cooldowns" }

// UserMessagingPrefs stores per-user messaging preferences.
type UserMessagingPrefs struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	OptOutPush      bool      `gorm:"not null;default:false" json:"opt_out_push"`
	OptOutEmail     bool      `gorm:"not null;default:false" json:"opt_out_email"`
	OptOutAll       bool      `gorm:"not null;default:false" json:"opt_out_all"`
	QuietHoursStart int       `gorm:"not null;default:22" json:"quiet_hours_start"`
	QuietHoursEnd   int       `gorm:"not null;default:8" json:"quiet_hours_end"`
	MaxPerHour      int       `gorm:"not null;default:3" json:"max_per_hour"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (UserMessagingPrefs) TableName() string { return "messaging_user_prefs" }

// ── Anti-spam Constants ────────────────────────────────────────────────────────────

// CooldownDurations defines per-type cooldown (how long before same type can be sent again).
var CooldownDurations = map[string]time.Duration{
	"outbid":   10 * time.Second, // outbid alerts can be frequent
	"win":      30 * time.Second,
	"loss":     5 * time.Minute,
	"nudge":    30 * time.Minute,
	"reminder": 1 * time.Hour,
	"promo":    4 * time.Hour,
}

// DefaultMaxPerHour is the default maximum messages per hour per user.
const DefaultMaxPerHour = 3
