package livestream

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 11: Live Conversion Engine — Pinned Items, Quick Bid, Funnel
//
// Pinned items: sticky sellable card on the video.
// Quick bid: one-tap +10/+50/+100 increments.
// Funnel tracking: view → click → bid → win per user/session/item.
// ════════════════════════════════════════════════════════════════════════════

// ── Feature Flags ──────────────────────────────────────────────────────────

func IsPinnedItemsEnabled() bool {
	val := os.Getenv("ENABLE_PINNED_ITEMS")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

func IsQuickBidEnabled() bool {
	val := os.Getenv("ENABLE_QUICK_BID")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// ════════════════════════════════════════════════════════════════════════════
// Funnel Tracking Model
// ════════════════════════════════════════════════════════════════════════════

// ConversionStage enumerates the funnel stages.
type ConversionStage string

const (
	StageView     ConversionStage = "view"      // user joined live session
	StageClickBid ConversionStage = "click_bid" // user tapped the bid button
	StagePlaceBid ConversionStage = "place_bid" // user successfully placed a bid
	StageBuyNow   ConversionStage = "buy_now"   // user completed buy-now purchase
	StageWin      ConversionStage = "win"       // user won the auction
	StageViewPin  ConversionStage = "view_pin"  // user viewed a pinned item card
	StageQuickBid ConversionStage = "quick_bid" // user used a quick-bid button
)

// LiveConversionEvent is a single funnel data point.
type LiveConversionEvent struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SessionID uuid.UUID       `gorm:"type:uuid;not null;index"                        json:"session_id"`
	ItemID    *uuid.UUID      `gorm:"type:uuid;index"                                 json:"item_id,omitempty"`
	UserID    *uuid.UUID      `gorm:"type:uuid;index"                                 json:"user_id,omitempty"`
	Stage     ConversionStage `gorm:"size:20;not null;index"                          json:"stage"`
	Amount    int64           `gorm:"not null;default:0"                              json:"amount,omitempty"`
	Metadata  string          `gorm:"type:text"                                       json:"metadata,omitempty"`
	CreatedAt time.Time       `gorm:"not null;index"                                   json:"created_at"`
}

func (LiveConversionEvent) TableName() string { return "live_conversion_events" }

// RecordConversionEvent persists a funnel data point. Fire-and-forget.
func RecordConversionEvent(db *gorm.DB, sessionID uuid.UUID, itemID, userID *uuid.UUID, stage ConversionStage, amount int64, meta string) {
	if db == nil {
		return
	}
	evt := LiveConversionEvent{
		SessionID: sessionID,
		ItemID:    itemID,
		UserID:    userID,
		Stage:     stage,
		Amount:    amount,
		Metadata:  meta,
		CreatedAt: time.Now(),
	}
	if err := db.Create(&evt).Error; err != nil {
		slog.Debug("live-conversion: failed to record event", "stage", stage, "error", err)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Pinned Items
// ════════════════════════════════════════════════════════════════════════════

// ── POST /livestream/:id/items/:itemId/pin — pin an item ─────────────────

func (h *LiveAuctionHandler) PinItem(c *gin.Context) {
	if !IsPinnedItemsEnabled() {
		response.BadRequest(c, "Pinned items are not enabled")
		return
	}

	hostID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))

	// Only the session host may pin
	var sess Session
	if err := h.db.Where("id = ? AND host_id = ?", sessionID, hostID).First(&sess).Error; err != nil {
		response.Forbidden(c)
		return
	}

	var item LiveItem
	if err := h.db.Where("id = ? AND session_id = ?", itemID, sessionID).First(&item).Error; err != nil {
		response.NotFound(c, "Item")
		return
	}

	// Atomically unpin any previously-pinned items in this session + pin this one
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&LiveItem{}).
			Where("session_id = ? AND is_pinned = ?", sessionID, true).
			Update("is_pinned", false).Error; err != nil {
			return err
		}
		return tx.Model(&item).Update("is_pinned", true).Error
	})
	if err != nil {
		response.InternalError(c, err)
		return
	}

	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:            EventItemPinned,
		SessionID:        sessionID.String(),
		ItemID:           itemID.String(),
		Pinned:           true,
		CurrentBidCents:  item.CurrentBidCents,
		BuyNowPriceCents: item.BuyNowPriceCents,
	})

	slog.Info("live-pinned: item pinned", "session_id", sessionID, "item_id", itemID, "host", hostID)
	response.OK(c, gin.H{"message": "Item pinned", "item_id": itemID})
}

// ── POST /livestream/:id/items/:itemId/unpin — unpin an item ─────────────

func (h *LiveAuctionHandler) UnpinItem(c *gin.Context) {
	if !IsPinnedItemsEnabled() {
		response.BadRequest(c, "Pinned items are not enabled")
		return
	}

	hostID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))

	var sess Session
	if err := h.db.Where("id = ? AND host_id = ?", sessionID, hostID).First(&sess).Error; err != nil {
		response.Forbidden(c)
		return
	}

	if err := h.db.Model(&LiveItem{}).
		Where("id = ? AND session_id = ?", itemID, sessionID).
		Update("is_pinned", false).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:     EventItemUnpinned,
		SessionID: sessionID.String(),
		ItemID:    itemID.String(),
		Pinned:    false,
	})

	response.OK(c, gin.H{"message": "Item unpinned"})
}

// ════════════════════════════════════════════════════════════════════════════
// Quick Bid
// ════════════════════════════════════════════════════════════════════════════

// QuickBidIncrement is an allowed preset increment in cents (10/50/100 EGP).
var QuickBidIncrements = map[int64]bool{
	1_000:  true, // +10 EGP
	5_000:  true, // +50 EGP
	10_000: true, // +100 EGP
}

// ── POST /livestream/:id/items/:itemId/quick-bid — one-click bid ─────────
// Body: {"increment_cents": 1000}

func (h *LiveAuctionHandler) QuickBid(c *gin.Context) {
	if !IsQuickBidEnabled() {
		response.BadRequest(c, "Quick bid is not enabled")
		return
	}

	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))

	var req struct {
		IncrementCents int64 `json:"increment_cents" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !QuickBidIncrements[req.IncrementCents] {
		response.BadRequest(c, "Invalid quick-bid increment. Allowed: 1000, 5000, 10000 cents")
		return
	}

	// Load item to compute target amount
	var item LiveItem
	if err := h.db.Where("id = ? AND session_id = ?", itemID, sessionID).First(&item).Error; err != nil {
		response.NotFound(c, "Item")
		return
	}

	base := item.CurrentBidCents
	if base == 0 {
		base = item.StartPriceCents
	}
	targetAmount := base + req.IncrementCents

	// Record funnel event
	uid := userID
	iid := itemID
	RecordConversionEvent(h.db, sessionID, &iid, &uid, StageQuickBid, targetAmount,
		fmt.Sprintf("increment=%d", req.IncrementCents))

	// Rewrite request body to the standard bid amount and forward to PlaceBid
	// by setting it on context and calling PlaceBid directly.
	// Simpler: just run the equivalent bid logic inline by copying the amount
	// to a synthetic body — but easier: return structured info for client or
	// call PlaceBid via re-binding. Here we re-use PlaceBid by setting a hint.
	c.Set("quick_bid_amount_cents", targetAmount)
	h.PlaceBid(c)
}

// ════════════════════════════════════════════════════════════════════════════
// Funnel Analytics — GET /live/analytics/funnel
// ════════════════════════════════════════════════════════════════════════════

// shortUserLabel returns a short 6-char user label for toast display.
// Keeps user identity obscured for privacy while still looking social.
func shortUserLabel(userID uuid.UUID) string {
	s := userID.String()
	if len(s) < 6 {
		return "User"
	}
	return "User" + s[:6]
}

// AdminFunnelAnalytics returns counts per stage for a session (or global).
func (h *LiveAuctionHandler) AdminFunnelAnalytics(c *gin.Context) {
	sessionIDStr := c.Query("session_id")
	var results []struct {
		Stage ConversionStage `json:"stage"`
		Count int64           `json:"count"`
	}
	q := h.db.Table("live_conversion_events").
		Select("stage, COUNT(*) as count").
		Group("stage")
	if sessionIDStr != "" {
		q = q.Where("session_id = ?", sessionIDStr)
	}
	if err := q.Scan(&results).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"funnel": results})
}
