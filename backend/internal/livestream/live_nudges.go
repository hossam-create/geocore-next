package livestream

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 11: Smart Nudges Engine
//
// Delivers personalized, context-aware micro-messages to maximize conversion.
// Throttled to max 1 nudge per user per 20 seconds (Redis-backed).
//
// Feature-flagged via ENABLE_LIVE_NUDGES env var (default: true).
// ════════════════════════════════════════════════════════════════════════════

const (
	nudgeThrottleDuration = 20 * time.Second
)

// NudgeCode enumerates all supported nudge types.
type NudgeCode string

const (
	NudgeWatcherNotBidding NudgeCode = "watcher_not_bidding" // "💡 Only 2 bids away from winning"
	NudgeOutbid            NudgeCode = "outbid"              // "⚠️ You've been outbid"
	NudgeBuyNowClose       NudgeCode = "buy_now_close"       // "🔥 Buy now almost reached"
	NudgeItemAlmostEnd     NudgeCode = "item_almost_end"     // "⏰ Only 10s left!"
	NudgeNewHotItem        NudgeCode = "new_hot_item"        // "🔥 New hot item is live"
)

// NudgeContext carries data used by SendLiveNudge to compose the message.
type NudgeContext struct {
	SessionID   uuid.UUID
	ItemID      uuid.UUID
	Code        NudgeCode
	CurrentBid  int64
	BuyNow      *int64
	SecondsLeft int
	// Custom override (optional)
	Message string
	Icon    string
}

// IsLiveNudgesEnabled returns true unless explicitly disabled.
func IsLiveNudgesEnabled() bool {
	val := os.Getenv("ENABLE_LIVE_NUDGES")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// SendLiveNudge sends a personalized nudge to a user, subject to throttle.
// Returns true if the nudge was sent, false if throttled or disabled.
func SendLiveNudge(rdb *redis.Client, userID uuid.UUID, ctx NudgeContext) bool {
	if !IsLiveNudgesEnabled() || rdb == nil {
		return false
	}

	// Throttle: one nudge per 20s per user
	throttleKey := "live:nudge:throttle:" + userID.String()
	bg := context.Background()
	set, err := rdb.SetNX(bg, throttleKey, string(ctx.Code), nudgeThrottleDuration).Result()
	if err != nil || !set {
		return false
	}

	// Compose message/icon if not provided
	msg, icon := composeNudge(ctx)

	// Sprint 11.5: attach paid CTA (Monetized Nudges)
	action, label := SuggestedActionFor(ctx.Code)

	evt := LiveEvent{
		Event:           EventLiveNudge,
		SessionID:       ctx.SessionID.String(),
		ItemID:          ctx.ItemID.String(),
		TargetUserID:    userID.String(),
		NudgeCode:       string(ctx.Code),
		Message:         msg,
		Icon:            icon,
		SuggestedAction: action,
		ActionLabel:     label,
	}

	// Publish to user-specific channel so frontend can filter
	channel := "live:nudge:" + userID.String()
	payload := serializeEvent(evt)
	rdb.Publish(bg, channel, payload)

	// Also publish to session channel (so frontend bundled listener picks it up)
	sessionChannel := "live:" + ctx.SessionID.String()
	rdb.Publish(bg, sessionChannel, payload)

	slog.Debug("live-nudge: sent", "user", userID, "code", ctx.Code)
	return true
}

// composeNudge returns the default message + icon for each nudge code.
func composeNudge(ctx NudgeContext) (string, string) {
	if ctx.Message != "" {
		icon := ctx.Icon
		if icon == "" {
			icon = "💡"
		}
		return ctx.Message, icon
	}
	switch ctx.Code {
	case NudgeWatcherNotBidding:
		return "💡 Only 2 bids away from winning", "💡"
	case NudgeOutbid:
		return "⚠️ You've been outbid", "⚠️"
	case NudgeBuyNowClose:
		return "🔥 Buy now almost reached — secure it!", "🔥"
	case NudgeItemAlmostEnd:
		return "⏰ Only seconds left!", "⏰"
	case NudgeNewHotItem:
		return "🔥 A hot new item is live", "🔥"
	default:
		return "💡 Live update", "💡"
	}
}

// ── Toast Broadcasting (Feature 8) ─────────────────────────────────────────

// BroadcastToast publishes a generic real-time toast to all session viewers.
func (h *LiveAuctionHandler) BroadcastToast(sessionID uuid.UUID, message, icon string) {
	if h.rdb == nil {
		return
	}
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:     EventToast,
		SessionID: sessionID.String(),
		Message:   message,
		Icon:      icon,
	})
}

// serializeEvent centralizes JSON marshaling for pub/sub to keep it in one place.
func serializeEvent(evt LiveEvent) string {
	b, err := json.Marshal(evt)
	if err != nil {
		return "{}"
	}
	return string(b)
}
