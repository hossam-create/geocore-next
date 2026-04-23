package push

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Priority levels
// ════════════════════════════════════════════════════════════════════════════

const (
	PriorityHigh   = "high"   // OTP, payments, bids — immediate delivery
	PriorityMedium = "medium" // messages, offers — best-effort
	PriorityLow    = "low"    // marketing, announcements — bulk/deferred
)

// NotificationTypePriority maps notification types to push priority.
var NotificationTypePriority = map[string]string{
	// HIGH — financial + auth
	"otp":            PriorityHigh,
	"payment_success": PriorityHigh,
	"payment_failed":  PriorityHigh,
	"escrow_released": PriorityHigh,
	"new_bid":        PriorityHigh,
	"outbid":         PriorityHigh,
	"auction_won":    PriorityHigh,
	"buy_now":        PriorityHigh,
	// MEDIUM — social + commerce
	"new_message":       PriorityMedium,
	"offer_created":     PriorityMedium,
	"offer_countered":   PriorityMedium,
	"offer_accepted":    PriorityMedium,
	"offer_rejected":    PriorityMedium,
	"new_review":        PriorityMedium,
	"listing_approved":  PriorityMedium,
	"listing_rejected":  PriorityMedium,
	"auction_ended":     PriorityMedium,
	// LOW — marketing
	"announcement":     PriorityLow,
	"promo":            PriorityLow,
	"recommendation":   PriorityLow,
}

// ════════════════════════════════════════════════════════════════════════════
// Device Registry
// ════════════════════════════════════════════════════════════════════════════

// UserDevice stores a push device token with metadata for token lifecycle management.
type UserDevice struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	DeviceToken string         `gorm:"type:text;not null;uniqueIndex" json:"device_token"`
	Platform    string         `gorm:"size:20;not null" json:"platform"` // ios | android | web
	AppVersion  string         `gorm:"size:20" json:"app_version"`
	IsActive    bool           `gorm:"default:true;index" json:"is_active"`
	LastSeenAt  time.Time      `json:"last_seen_at"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// ════════════════════════════════════════════════════════════════════════════
// Push Delivery Log
// ════════════════════════════════════════════════════════════════════════════

// PushLog records every push attempt for observability and debugging.
type PushLog struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	DeviceToken    string     `gorm:"type:text;not null;index" json:"device_token"`
	Platform       string     `gorm:"size:20" json:"platform"`
	NotificationType string   `gorm:"size:50;not null;index" json:"notification_type"`
	Priority       string     `gorm:"size:20;not null;index" json:"priority"`
	Title          string     `gorm:"size:255" json:"title"`
	Body           string     `gorm:"type:text" json:"body"`
	Data           string     `gorm:"type:jsonb" json:"data,omitempty"`
	Status         string     `gorm:"size:20;not null;index;default:queued" json:"status"` // queued | sent | failed | delivered | bounced
	ProviderMsgID  string     `gorm:"size:200" json:"provider_msg_id"`
	ErrorReason    string     `gorm:"type:text" json:"error_reason"`
	Attempts       int        `gorm:"default:1" json:"attempts"`
	IdempotencyKey string     `gorm:"size:200;uniqueIndex" json:"idempotency_key"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Push status constants
const (
	PushStatusQueued    = "queued"
	PushStatusSent      = "sent"
	PushStatusFailed    = "failed"
	PushStatusDelivered = "delivered"
	PushStatusBounced   = "bounced"
)

// ════════════════════════════════════════════════════════════════════════════
// Push Message (in-memory dispatch unit)
// ════════════════════════════════════════════════════════════════════════════

// PushMessage is the canonical push payload passed through the pipeline.
type PushMessage struct {
	UserID          uuid.UUID
	DeviceToken     string
	Platform        string
	NotificationType string
	Priority        string
	Title           string
	Body            string
	Data            map[string]string
	IdempotencyKey  string
	Silent          bool // data-only push (no visible notification)
}

// ════════════════════════════════════════════════════════════════════════════
// Rate-limit config per notification type
// ════════════════════════════════════════════════════════════════════════════

// TypeRateLimit defines per-type push rate limits.
type TypeRateLimit struct {
	MaxPerMinute int
	MaxPerHour   int
}

// DefaultTypeRateLimits are the production rate limits per notification type.
var DefaultTypeRateLimits = map[string]TypeRateLimit{
	"otp":             {MaxPerMinute: 3, MaxPerHour: 10},
	"payment_success":  {MaxPerMinute: 10, MaxPerHour: 30},
	"payment_failed":   {MaxPerMinute: 10, MaxPerHour: 30},
	"outbid":          {MaxPerMinute: 10, MaxPerHour: 60},
	"new_bid":         {MaxPerMinute: 10, MaxPerHour: 60},
	"auction_won":     {MaxPerMinute: 5, MaxPerHour: 20},
	"new_message":     {MaxPerMinute: 10, MaxPerHour: 60},
	"offer_created":   {MaxPerMinute: 5, MaxPerHour: 30},
	"announcement":    {MaxPerMinute: 1, MaxPerHour: 5},
	"promo":           {MaxPerMinute: 1, MaxPerHour: 3},
}

// DefaultTypeRateLimit is the fallback when a type is not explicitly configured.
var DefaultTypeRateLimit = TypeRateLimit{MaxPerMinute: 10, MaxPerHour: 50}
