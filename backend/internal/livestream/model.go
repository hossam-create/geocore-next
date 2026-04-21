package livestream

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionStatus string

const (
	StatusScheduled SessionStatus = "scheduled"
	StatusLive      SessionStatus = "live"
	StatusEnded     SessionStatus = "ended"
	StatusCancelled SessionStatus = "cancelled"
)

type Session struct {
	ID           uuid.UUID     `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	AuctionID    *uuid.UUID    `gorm:"type:uuid;index"                                 json:"auction_id,omitempty"`
	HostID       uuid.UUID     `gorm:"type:uuid;not null;index"                        json:"host_id"`
	Title        string        `gorm:"size:255;not null"                               json:"title"`
	Description  string        `gorm:"type:text"                                       json:"description,omitempty"`
	Status       SessionStatus `gorm:"size:50;not null;default:'scheduled';index"      json:"status"`
	RoomName     string        `gorm:"size:255;not null;uniqueIndex"                   json:"room_name"`
	ViewerCount  int           `gorm:"not null;default:0"                              json:"viewer_count"`
	StartedAt    *time.Time    `json:"started_at,omitempty"`
	EndedAt      *time.Time    `json:"ended_at,omitempty"`
	ThumbnailURL string        `gorm:"type:text"                                       json:"thumbnail_url,omitempty"`

	// ── Sprint 12: Monetization ─────────────────────────────────────────────
	BoostTier     string `gorm:"size:20;default:''"                              json:"boost_tier,omitempty"` // '', 'basic', 'premium', 'vip'
	BoostScore    int    `gorm:"not null;default:0;index"                       json:"boost_score"`           // ranking boost
	IsPremium     bool   `gorm:"not null;default:false;index"                    json:"is_premium"`           // featured / priority feed
	EntryFeeCents int64  `gorm:"not null;default:0"                              json:"entry_fee_cents"`      // pay-to-enter VIP
	SellerPlan    string `gorm:"size:20;default:'free'"                          json:"seller_plan"`          // free, pro, elite

	// ── Sprint 13: Revenue Flywheel ─────────────────────────────────────────
	StreamerID        *uuid.UUID `gorm:"type:uuid;index"                             json:"streamer_id,omitempty"` // if != HostID → creator split
	UrgencyMultiplier float64    `gorm:"type:numeric(4,2);not null;default:1.0"      json:"urgency_multiplier"`    // boost-driven amplifier
	IsHot             bool       `gorm:"not null;default:false;index"                json:"is_hot"`                // sticky hot flag from boost
	NotifyMoreUsers   bool       `gorm:"not null;default:false"                      json:"notify_more_users"`     // expands notification radius

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                                          json:"-"`
}

func (Session) TableName() string { return "livestream_sessions" }
