package livestream

import (
	"time"

	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Live Item — a product shown during a live session for real-time bidding.
// ════════════════════════════════════════════════════════════════════════════

type LiveItemStatus string

const (
	ItemPending       LiveItemStatus = "pending"
	ItemActive        LiveItemStatus = "active"
	ItemSettling      LiveItemStatus = "settling"
	ItemSold          LiveItemStatus = "sold"
	ItemUnsold        LiveItemStatus = "unsold"
	ItemPaymentFailed LiveItemStatus = "payment_failed"
	ItemCancelled     LiveItemStatus = "cancelled"
)

const (
	maxAntiSnipeExtensions = 5
	settlementTimeout      = 30 * time.Second
	maxSettleRetries       = 3
	idempotencyTTL         = 2 * time.Minute
	platformFeePercent     = 2.5
)

type LiveItem struct {
	ID                   uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SessionID            uuid.UUID      `gorm:"type:uuid;not null;index"                        json:"session_id"`
	ListingID            *uuid.UUID     `gorm:"type:uuid"                                       json:"listing_id,omitempty"`
	Title                string         `gorm:"size:255;not null"                               json:"title"`
	ImageURL             string         `gorm:"type:text"                                       json:"image_url,omitempty"`
	StartPriceCents      int64          `gorm:"not null;default:0"                              json:"start_price_cents"`
	BuyNowPriceCents     *int64         `json:"buy_now_price_cents,omitempty"`
	CurrentBidCents      int64          `gorm:"not null;default:0"                              json:"current_bid_cents"`
	MinIncrementCents    int64          `gorm:"not null;default:100"                            json:"min_increment_cents"`
	HighestBidderID      *uuid.UUID     `gorm:"type:uuid"                                      json:"highest_bidder_id,omitempty"`
	BidCount             int            `gorm:"not null;default:0"                              json:"bid_count"`
	Status               LiveItemStatus `gorm:"size:20;not null;default:'pending';index"        json:"status"`
	EndsAt               *time.Time     `json:"ends_at,omitempty"`
	AntiSnipeEnabled     bool           `gorm:"not null;default:true"                           json:"anti_snipe_enabled"`
	ExtensionCount       int            `gorm:"not null;default:0"                              json:"extension_count"`
	RequiresReview       bool           `gorm:"not null;default:false"                          json:"requires_review"`
	RiskScore            int            `gorm:"not null;default:0"                              json:"risk_score"`
	RequiresEntryDeposit bool           `gorm:"not null;default:false"                          json:"requires_entry_deposit"`
	EntryDepositCents    int64          `gorm:"not null;default:0"                              json:"entry_deposit_cents"`
	IsPinned             bool           `gorm:"not null;default:false;index"                    json:"is_pinned"`
	SettlingStartedAt    *time.Time     `json:"settling_started_at,omitempty"`
	SettleRetries        int            `gorm:"not null;default:0"                              json:"settle_retries"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

func (LiveItem) TableName() string { return "live_items" }

// ════════════════════════════════════════════════════════════════════════════
// Live Bid — a bid placed on a live item during a stream.
// ════════════════════════════════════════════════════════════════════════════

type LiveBid struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ItemID         uuid.UUID `gorm:"type:uuid;not null;index"                        json:"item_id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index"                        json:"user_id"`
	BidAmountCents int64     `gorm:"not null"                                        json:"bid_amount_cents"`
	CreatedAt      time.Time `json:"created_at"`
}

func (LiveBid) TableName() string { return "live_bids" }

// ════════════════════════════════════════════════════════════════════════════
// Live Event — structured WebSocket event for real-time UX.
// ════════════════════════════════════════════════════════════════════════════

type LiveEventType string

const (
	EventNewBid            LiveEventType = "new_bid"
	EventOutbid            LiveEventType = "outbid"
	EventAuctionEnd        LiveEventType = "auction_end"
	EventItemActivated     LiveEventType = "item_activated"
	EventItemSettling      LiveEventType = "item_settling"
	EventItemSold          LiveEventType = "item_sold"
	EventItemUnsold        LiveEventType = "item_unsold"
	EventItemPaymentFailed LiveEventType = "item_payment_failed"
	EventItemSoldBuyNow    LiveEventType = "item_sold_buy_now"
	EventViewerJoin        LiveEventType = "viewer_join"
	EventViewerLeave       LiveEventType = "viewer_leave"
	EventBidExtended       LiveEventType = "bid_extended" // anti-snipe extension

	// ── Sprint 11: Live Conversion Engine ─────────────────────────────────
	EventLiveUrgencyUpdate LiveEventType = "live_urgency_update" // FOMO state change
	EventCountdownPhase    LiveEventType = "countdown_phase"     // timer phase change
	EventBuyNowAlmost      LiveEventType = "buy_now_almost"      // current_bid ≥ 90% buy_now
	EventLiveNudge         LiveEventType = "live_nudge"          // personal nudge to user
	EventToast             LiveEventType = "toast"               // generic real-time toast
	EventItemPinned        LiveEventType = "item_pinned"         // seller pinned item
	EventItemUnpinned      LiveEventType = "item_unpinned"       // seller unpinned item
	EventLiveAISuggestion  LiveEventType = "live_ai_suggestion"  // Sprint 14: AI seller assistant hint
)

// ── Urgency & Countdown States ───────────────────────────────────────────
type UrgencyState string

const (
	UrgencyNormal  UrgencyState = "NORMAL"
	UrgencyHot     UrgencyState = "HOT"      // ≥3 bids in last 10s
	UrgencyVeryHot UrgencyState = "VERY_HOT" // ≥6 bids in last 10s
)

type CountdownPhase string

const (
	PhaseNormal CountdownPhase = "normal" // > 30s left
	PhaseOrange CountdownPhase = "orange" // ≤ 30s left
	PhaseRed    CountdownPhase = "red"    // ≤ 10s left
)

// RecentBidder holds minimal info about a recent bidder for social proof.
type RecentBidder struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	AmountCents int64  `json:"amount_cents"`
	BidAt       string `json:"bid_at"`
}

// LiveEvent is the canonical WebSocket event payload.
type LiveEvent struct {
	Event           LiveEventType `json:"event"`
	SessionID       string        `json:"session_id"`
	ItemID          string        `json:"item_id,omitempty"`
	CurrentBidCents int64         `json:"current_bid_cents,omitempty"`
	HighestBidderID *string       `json:"highest_bidder_id,omitempty"`
	BidCount        int           `json:"bid_count,omitempty"`
	Status          string        `json:"status,omitempty"`
	EndsAt          *string       `json:"ends_at,omitempty"`

	// Outbid-specific: the user who was outbid
	OutbidUserID *string `json:"outbid_user_id,omitempty"`

	// Social proof
	ViewerCount   int            `json:"viewer_count,omitempty"`
	RecentBidders []RecentBidder `json:"recent_bidders,omitempty"`

	// Anti-snipe extension info
	Extended       bool    `json:"extended,omitempty"`
	NewEndsAt      *string `json:"new_ends_at,omitempty"`
	ExtensionCount int     `json:"extension_count,omitempty"`

	// Viewer events
	ViewerID    string `json:"viewer_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`

	// Buy Now
	BuyNowPriceCents *int64 `json:"buy_now_price_cents,omitempty"`

	// ── Sprint 11: Live Conversion Engine ──────────────────────────────────
	// FOMO / Urgency
	Urgency       UrgencyState `json:"urgency,omitempty"`
	BidsLast10s   int          `json:"bids_last_10s,omitempty"`
	BidsLast30s   int          `json:"bids_last_30s,omitempty"`
	ActiveBidders int          `json:"active_bidders,omitempty"`

	// Countdown
	Phase       CountdownPhase `json:"phase,omitempty"`
	SecondsLeft int            `json:"seconds_left,omitempty"`

	// Buy-now trigger
	BuyNowProgress float64 `json:"buy_now_progress,omitempty"` // 0.0-1.0

	// Nudge / Toast
	TargetUserID    string `json:"target_user_id,omitempty"`
	NudgeCode       string `json:"nudge_code,omitempty"`
	Message         string `json:"message,omitempty"`
	Icon            string `json:"icon,omitempty"`
	SuggestedAction string `json:"suggested_action,omitempty"` // Sprint 11.5: quick_bid | buy_now | boost | pay_entry
	ActionLabel     string `json:"action_label,omitempty"`     // button label for the CTA

	// Sprint 14: AI Seller Assistant
	SuggestionID   string  `json:"suggestion_id,omitempty"`   // UUID of LiveAIEvent row for accept tracking
	SuggestionType string  `json:"suggestion_type,omitempty"` // e.g. extend_timer, push_buy_now, boost_session
	Confidence     float64 `json:"confidence,omitempty"`      // 0.0 – 1.0

	// Pinned
	Pinned bool `json:"pinned,omitempty"`
}
