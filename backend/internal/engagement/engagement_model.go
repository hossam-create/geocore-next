package engagement

import (
	"time"

	"github.com/google/uuid"
)

// ── Responsible Engagement Engine Models ──────────────────────────────────────────
//
// Core principle: increase retention WITHOUT spam or dark patterns.
// Value → Feedback → Timing loop, not addiction loop.

// ── Session Momentum ────────────────────────────────────────────────────────────────

// SessionMomentum tracks real-time session engagement quality.
type SessionMomentum struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	SessionID       string    `gorm:"size:50;not null;uniqueIndex" json:"session_id"`

	// ── Momentum metrics ────────────────────────────────────────────────────
	ClickRate       float64 `gorm:"type:numeric(5,4);not null;default:0" json:"click_rate"`       // clicks / views
	BidRate         float64 `gorm:"type:numeric(5,4);not null;default:0" json:"bid_rate"`         // bids / views
	TimeOnItem      float64 `gorm:"type:numeric(8,2);not null;default:0" json:"time_on_item"`    // avg seconds
	ScrollVelocity  float64 `gorm:"type:numeric(8,2);not null;default:0" json:"scroll_velocity"` // items/min
	Friction        float64 `gorm:"type:numeric(5,4);not null;default:0" json:"friction"`         // exits+backs / actions

	// ── Derived ──────────────────────────────────────────────────────────────
	MomentumScore   float64 `gorm:"type:numeric(5,4);not null;default:0" json:"momentum_score"` // 0-1 composite
	FeedIntensity   string  `gorm:"size:20;not null;default:'balanced'" json:"feed_intensity"`   // high/low/balanced

	// ── Session counters ──────────────────────────────────────────────────────
	ViewsCount      int     `gorm:"not null;default:0" json:"views_count"`
	ClicksCount     int     `gorm:"not null;default:0" json:"clicks_count"`
	BidsCount       int     `gorm:"not null;default:0" json:"bids_count"`
	SavesCount      int     `gorm:"not null;default:0" json:"saves_count"`
	PurchasesCount  int     `gorm:"not null;default:0" json:"purchases_count"`
	BacksCount      int     `gorm:"not null;default:0" json:"backs_count"`

	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (SessionMomentum) TableName() string { return "engagement_momentum" }

// ── Notification Models ────────────────────────────────────────────────────────────

// NotificationDecision is the output of the notification AI.
type NotificationDecision struct {
	ShouldSend bool    `json:"should_send"`
	Channel    string  `json:"channel"` // push, email, in_app
	Score      float64 `json:"score"`   // P(action) * value_score
	Reason     string  `json:"reason"`  // why this notification was (not) sent
	Priority   string  `json:"priority"` // high, normal, low
}

// NotificationEvent records a notification for audit.
type NotificationEvent struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	EventType    string    `gorm:"size:30;not null;index" json:"event_type"` // outbid, price_drop, live_hot, saved_match
	Channel      string    `gorm:"size:20;not null;default:'push'" json:"channel"`
	Score        float64   `gorm:"type:numeric(8,4);not null" json:"score"`
	Reason       string    `gorm:"size:200" json:"reason"` // audit: why was this sent?
	Opened       bool      `gorm:"not null;default:false" json:"opened"`
	Acted        bool      `gorm:"not null;default:false" json:"acted"` // clicked through / converted
	OptedOut     bool      `gorm:"not null;default:false" json:"opted_out"`
	SentAt       *time.Time `json:"sent_at"`
	OpenedAt     *time.Time `json:"opened_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

func (NotificationEvent) TableName() string { return "engagement_notifications" }

// ── User Segment ────────────────────────────────────────────────────────────────────

type UserSegmentType string

const (
	SegmentActive    UserSegmentType = "active"    // 0-24h since last activity
	SegmentWarm      UserSegmentType = "warm"      // 1-3 days
	SegmentCold      UserSegmentType = "cold"      // 3-7 days
	SegmentChurnRisk UserSegmentType = "churn_risk" // 7+ days
)

// UserEngagementProfile stores per-user engagement data.
type UserEngagementProfile struct {
	ID                    uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID                uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Segment               UserSegmentType `gorm:"size:20;not null;default:'active'" json:"segment"`
	LastActiveAt          *time.Time     `json:"last_active_at"`
	NotificationsToday    int            `gorm:"not null;default:0" json:"notifications_today"`
	NotificationsThisWeek int            `gorm:"not null;default:0" json:"notifications_this_week"`
	OptOutAll             bool           `gorm:"not null;default:false" json:"opt_out_all"`
	OptOutPush            bool           `gorm:"not null;default:false" json:"opt_out_push"`
	OptOutEmail           bool           `gorm:"not null;default:false" json:"opt_out_email"`
	QuietHoursStart       int            `gorm:"not null;default:22" json:"quiet_hours_start"` // hour (0-23), local time
	QuietHoursEnd         int            `gorm:"not null;default:8" json:"quiet_hours_end"`
	PreferredChannels     string         `gorm:"size:100;not null;default:'push,in_app'" json:"preferred_channels"` // comma-separated
	TotalNotificationsSent int           `gorm:"not null;default:0" json:"total_notifications_sent"`
	TotalOpened           int           `gorm:"not null;default:0" json:"total_opened"`
	TotalActed            int           `gorm:"not null;default:0" json:"total_acted"`
	OpenRate              float64       `gorm:"type:numeric(5,4);not null;default:0" json:"open_rate"`
	ActRate               float64       `gorm:"type:numeric(5,4);not null;default:0" json:"act_rate"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
}

func (UserEngagementProfile) TableName() string { return "engagement_profiles" }

// ── Re-engagement Plan ──────────────────────────────────────────────────────────────

// PlannedTouch is a scheduled re-engagement action.
type PlannedTouch struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Segment     UserSegmentType `gorm:"size:20;not null" json:"segment"`
	Channel     string    `gorm:"size:20;not null;default:'push'" json:"channel"`
	MessageType string    `gorm:"size:30;not null" json:"message_type"` // opportunity, social_proof, incentive, discovery
	ScheduledAt time.Time `gorm:"not null;index" json:"scheduled_at"`
	SentAt      *time.Time `json:"sent_at"`
	Opened      bool      `gorm:"not null;default:false" json:"opened"`
	Acted       bool      `gorm:"not null;default:false" json:"acted"`
	Status      string    `gorm:"size:20;not null;default:'planned'" json:"status"` // planned, sent, cancelled
	CreatedAt   time.Time `json:"created_at"`
}

func (PlannedTouch) TableName() string { return "engagement_planned_touches" }

// ── Timing Model ────────────────────────────────────────────────────────────────────

// UserActivityHour records when a user is typically active.
type UserActivityHour struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Hour      int       `gorm:"not null" json:"hour"` // 0-23 (local time)
	DayOfWeek int       `gorm:"not null;default:-1" json:"day_of_week"` // 0-6 or -1 for any day
	Count     int       `gorm:"not null;default:0" json:"count"` // how many times active at this hour
	Score     float64   `gorm:"type:numeric(5,4);not null;default:0" json:"score"` // normalized 0-1
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserActivityHour) TableName() string { return "engagement_activity_hours" }

// ── Config ──────────────────────────────────────────────────────────────────────────

type EngagementConfig struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	MaxNotificationsPerDay  int       `gorm:"not null;default:3" json:"max_notifications_per_day"`
	MaxNotificationsPerWeek int       `gorm:"not null;default:12" json:"max_notifications_per_week"`
	NotificationScoreThreshold float64 `gorm:"type:numeric(5,4);not null;default:0.3" json:"notification_score_threshold"`
	QuietHoursDefaultStart int       `gorm:"not null;default:22" json:"quiet_hours_default_start"`
	QuietHoursDefaultEnd   int       `gorm:"not null;default:8" json:"quiet_hours_default_end"`
	ExplorationPercent      int       `gorm:"not null;default:10" json:"exploration_percent"` // 10% novel content
	MomentumHighThreshold   float64   `gorm:"type:numeric(5,4);not null;default:0.7" json:"momentum_high_threshold"`
	MomentumLowThreshold    float64   `gorm:"type:numeric(5,4);not null;default:0.3" json:"momentum_low_threshold"`
	KillSwitchActive        bool      `gorm:"not null;default:false" json:"kill_switch_active"`
	IsActive                bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (EngagementConfig) TableName() string { return "engagement_configs" }

// ── Dashboard ────────────────────────────────────────────────────────────────────────

type EngagementDashboard struct {
	TotalNotificationsSent int64              `json:"total_notifications_sent"`
	OpenRate               float64            `json:"open_rate"`
	ActRate                float64            `json:"act_rate"`
	OptOutRate             float64            `json:"opt_out_rate"`
	UsersBySegment         map[string]int64   `json:"users_by_segment"`
	AvgMomentumScore       float64            `json:"avg_momentum_score"`
	KillSwitchActive       bool               `json:"kill_switch_active"`
	TopEventTypes          []EventTypeStats   `json:"top_event_types"`
}

type EventTypeStats struct {
	EventType string  `json:"event_type"`
	Count     int64   `json:"count"`
	OpenRate  float64 `json:"open_rate"`
	ActRate   float64 `json:"act_rate"`
}
