package livestream

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 11: Live Conversion Engine — FOMO, Urgency, Countdown
//
// Drives high-conversion UX inspired by TikTok Shop & eBay Live:
//  - Bid-velocity driven urgency states (NORMAL / HOT / VERY_HOT)
//  - Countdown phase transitions (normal / orange / red)
//  - Auto buy-now trigger (≥90% progress)
//
// Feature-flagged via ENABLE_LIVE_FOMO env var (default: true).
// All events broadcast via existing WebSocket/Redis pub-sub pipeline.
// ════════════════════════════════════════════════════════════════════════════

const (
	fomoWindow10s = 10 * time.Second
	fomoWindow30s = 30 * time.Second

	urgencyHotThreshold     = 3 // ≥3 bids in last 10s
	urgencyVeryHotThreshold = 6 // ≥6 bids in last 10s

	buyNowAlmostThreshold = 0.90 // 90% progress
)

// IsLiveFomoEnabled returns true unless explicitly disabled via env.
func IsLiveFomoEnabled() bool {
	val := os.Getenv("ENABLE_LIVE_FOMO")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// ── Bid Velocity Tracking ──────────────────────────────────────────────────

// recordBidVelocity stores a bid timestamp in Redis for windowed counting.
// Key pattern: live:bids:{itemID} → sorted set (score=ts, member=bidID).
func recordBidVelocity(rdb *redis.Client, itemID uuid.UUID, bidID uuid.UUID) {
	if rdb == nil {
		return
	}
	key := "live:bids:" + itemID.String()
	now := time.Now().UnixMilli()
	ctx := context.Background()
	rdb.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: bidID.String()})
	// Trim old entries (>30s)
	rdb.ZRemRangeByScore(ctx, key, "-inf", toMilliString(now-30_000))
	// Expire after 5 min of inactivity
	rdb.Expire(ctx, key, 5*time.Minute)
}

// countBidsInWindow counts bids placed within the last `window` for an item.
func countBidsInWindow(rdb *redis.Client, itemID uuid.UUID, window time.Duration) int {
	if rdb == nil {
		return 0
	}
	key := "live:bids:" + itemID.String()
	now := time.Now().UnixMilli()
	cutoff := now - window.Milliseconds()
	ctx := context.Background()
	count, err := rdb.ZCount(ctx, key, toMilliString(cutoff), toMilliString(now)).Result()
	if err != nil {
		return 0
	}
	return int(count)
}

// countActiveBidders counts distinct bidders in the last 30s.
func countActiveBidders(db *gorm.DB, itemID uuid.UUID) int {
	var count int64
	db.Model(&LiveBid{}).
		Where("item_id = ? AND created_at >= ?", itemID, time.Now().Add(-fomoWindow30s)).
		Distinct("user_id").
		Count(&count)
	return int(count)
}

// computeUrgency determines the urgency state from recent bid count.
func computeUrgency(bidsLast10s int) UrgencyState {
	switch {
	case bidsLast10s >= urgencyVeryHotThreshold:
		return UrgencyVeryHot
	case bidsLast10s >= urgencyHotThreshold:
		return UrgencyHot
	default:
		return UrgencyNormal
	}
}

// publishUrgencyUpdate broadcasts the current urgency state for an item.
func (h *LiveAuctionHandler) publishUrgencyUpdate(sessionID, itemID uuid.UUID) {
	if !IsLiveFomoEnabled() || h.rdb == nil {
		return
	}
	bids10 := countBidsInWindow(h.rdb, itemID, fomoWindow10s)
	bids30 := countBidsInWindow(h.rdb, itemID, fomoWindow30s)
	active := countActiveBidders(h.db, itemID)
	urgency := computeUrgency(bids10)

	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:         EventLiveUrgencyUpdate,
		SessionID:     sessionID.String(),
		ItemID:        itemID.String(),
		Urgency:       urgency,
		BidsLast10s:   bids10,
		BidsLast30s:   bids30,
		ActiveBidders: active,
	})

	// Sprint 11.5: Smart Boost Injection — if item is hot and no boost active,
	// nudge the seller to boost the session.
	if urgency != UrgencyNormal {
		var sess Session
		if err := h.db.Select("id", "host_id", "boost_tier").Where("id = ?", sessionID).First(&sess).Error; err == nil {
			h.SuggestBoostToHost(&sess, urgency)
		}
	}
}

// ── Countdown Phase Tracking ──────────────────────────────────────────────

// ComputeCountdownPhase returns the current countdown phase based on seconds left.
func ComputeCountdownPhase(secondsLeft int) CountdownPhase {
	switch {
	case secondsLeft <= 10:
		return PhaseRed
	case secondsLeft <= 30:
		return PhaseOrange
	default:
		return PhaseNormal
	}
}

// publishCountdownPhase broadcasts a countdown phase transition.
// Called from scheduleItemEnd timer checkpoints.
func (h *LiveAuctionHandler) publishCountdownPhase(sessionID, itemID uuid.UUID, secondsLeft int) {
	if !IsLiveFomoEnabled() || h.rdb == nil {
		return
	}
	phase := ComputeCountdownPhase(secondsLeft)
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:       EventCountdownPhase,
		SessionID:   sessionID.String(),
		ItemID:      itemID.String(),
		Phase:       phase,
		SecondsLeft: secondsLeft,
	})
}

// ── Buy-Now Almost Trigger ────────────────────────────────────────────────

// IsSmartBuyNowEnabled returns true unless disabled.
func IsSmartBuyNowEnabled() bool {
	val := os.Getenv("ENABLE_SMART_BUY_NOW")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// ComputeBuyNowProgress returns 0.0-1.0 ratio of current bid vs buy-now price.
// Returns 0 if no buy-now price set.
func ComputeBuyNowProgress(currentBidCents int64, buyNowPriceCents *int64) float64 {
	if buyNowPriceCents == nil || *buyNowPriceCents <= 0 {
		return 0
	}
	progress := float64(currentBidCents) / float64(*buyNowPriceCents)
	if progress > 1.0 {
		return 1.0
	}
	return progress
}

// maybeTriggerBuyNowAlmost broadcasts when bid reaches 90% of buy-now price.
// Uses Redis dedup key so the event fires once per item.
func (h *LiveAuctionHandler) maybeTriggerBuyNowAlmost(sessionID, itemID uuid.UUID, currentBid int64, buyNow *int64) {
	if !IsSmartBuyNowEnabled() || h.rdb == nil || buyNow == nil {
		return
	}
	progress := ComputeBuyNowProgress(currentBid, buyNow)
	if progress < buyNowAlmostThreshold {
		return
	}
	// Dedup: only fire once per item
	ctx := context.Background()
	key := "live:buynow_almost:" + itemID.String()
	set, err := h.rdb.SetNX(ctx, key, "1", 30*time.Minute).Result()
	if err != nil || !set {
		return
	}
	slog.Info("live-fomo: buy-now almost triggered", "item_id", itemID, "progress", progress)
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:            EventBuyNowAlmost,
		SessionID:        sessionID.String(),
		ItemID:           itemID.String(),
		CurrentBidCents:  currentBid,
		BuyNowPriceCents: buyNow,
		BuyNowProgress:   progress,
		Message:          "⚡ Almost yours — Buy Now to secure instantly",
		Icon:             "⚡",
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────

func toMilliString(ms int64) string {
	const digits = "0123456789"
	if ms == 0 {
		return "0"
	}
	neg := ms < 0
	if neg {
		ms = -ms
	}
	var buf [20]byte
	i := len(buf)
	for ms > 0 {
		i--
		buf[i] = digits[ms%10]
		ms /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
