package livestream

import (
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Admin Live Control Panel (Sprint 10)
//
// Real-time control over live sessions: stop/pause/resume + ban/unban users.
// Feature-flagged via ENABLE_ADMIN_LIVE_CONTROL env var (default: true).
// All actions are audit-logged and broadcast via WebSocket.
// ════════════════════════════════════════════════════════════════════════════

// IsAdminLiveControlEnabled returns true unless explicitly disabled via env.
func IsAdminLiveControlEnabled() bool {
	val := os.Getenv("ENABLE_ADMIN_LIVE_CONTROL")
	if val == "" {
		return true
	}
	return val != "false" && val != "0"
}

// ── Global Panic Mode ────────────────────────────────────────────────────────
// When enabled: ALL live sessions stop, new sessions/bids/deposits blocked.
// Controlled via POST /admin/live/panic and POST /admin/live/recover.

var liveSystemDisabled atomic.Bool

// IsLiveSystemDisabled returns true if the global panic switch is active.
func IsLiveSystemDisabled() bool {
	return liveSystemDisabled.Load()
}

// ── POST /admin/live/panic — kill ALL live sessions globally ──────────────

func (h *LiveAuctionHandler) AdminPanic(c *gin.Context) {
	if !IsAdminLiveControlEnabled() {
		response.Forbidden(c)
		return
	}

	adminID, _ := uuid.Parse(c.GetString("user_id"))

	// Activate global kill switch
	liveSystemDisabled.Store(true)

	// Stop ALL live sessions
	var liveSessions []Session
	h.db.Where("status = ?", StatusLive).Find(&liveSessions)

	totalItemsEnded := 0
	for _, sess := range liveSessions {
		var activeItems []LiveItem
		h.db.Where("session_id = ? AND status = ?", sess.ID, ItemActive).Find(&activeItems)
		for _, item := range activeItems {
			h.cancelTimer(item.ID)
			h.endItem(item.ID)
			totalItemsEnded++
		}

		now := time.Now()
		h.db.Model(&sess).Updates(map[string]interface{}{
			"status":   StatusEnded,
			"ended_at": now,
		})

		h.broadcastLiveEvent(sess.ID, LiveEvent{
			Event:     "live_stopped",
			SessionID: sess.ID.String(),
			Status:    string(StatusEnded),
		})
	}

	freeze.LogAudit(h.db, "admin_panic", adminID, uuid.Nil,
		fmt.Sprintf("sessions_stopped=%d items_ended=%d", len(liveSessions), totalItemsEnded))
	slog.Warn("admin: PANIC MODE ACTIVATED",
		"admin", adminID, "sessions_stopped", len(liveSessions), "items_ended", totalItemsEnded)

	response.OK(c, gin.H{
		"message":          "PANIC MODE ACTIVATED — All live sessions stopped",
		"sessions_stopped": len(liveSessions),
		"items_ended":      totalItemsEnded,
		"system_status":    "disabled",
	})
}

// ── POST /admin/live/recover — disable panic mode ────────────────────────

func (h *LiveAuctionHandler) AdminRecover(c *gin.Context) {
	if !IsAdminLiveControlEnabled() {
		response.Forbidden(c)
		return
	}

	adminID, _ := uuid.Parse(c.GetString("user_id"))

	liveSystemDisabled.Store(false)

	freeze.LogAudit(h.db, "admin_recover", adminID, uuid.Nil, "panic_mode_disabled")
	slog.Info("admin: panic mode deactivated, live system restored", "admin", adminID)

	response.OK(c, gin.H{
		"message":       "Live system restored — new sessions and bidding allowed",
		"system_status": "enabled",
	})
}

// ── POST /admin/live/:id/stop — immediately stop a live session ──────────

func (h *LiveAuctionHandler) AdminStopSession(c *gin.Context) {
	if !IsAdminLiveControlEnabled() {
		response.Forbidden(c)
		return
	}

	adminID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))

	var sess Session
	if err := h.db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		response.NotFound(c, "Session")
		return
	}
	if sess.Status != StatusLive {
		response.BadRequest(c, "Session is not live")
		return
	}

	// End all active items safely
	var activeItems []LiveItem
	h.db.Where("session_id = ? AND status = ?", sessionID, ItemActive).Find(&activeItems)
	for _, item := range activeItems {
		h.cancelTimer(item.ID)
		h.endItem(item.ID)
	}

	// End session
	now := time.Now()
	h.db.Model(&sess).Updates(map[string]interface{}{
		"status":   StatusEnded,
		"ended_at": now,
	})

	// Broadcast
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:     "live_stopped",
		SessionID: sessionID.String(),
		Status:    string(StatusEnded),
	})

	freeze.LogAudit(h.db, "admin_live_stop", adminID, sessionID,
		fmt.Sprintf("items_ended=%d", len(activeItems)))
	slog.Info("admin: live session stopped", "session_id", sessionID, "admin", adminID)

	response.OK(c, gin.H{
		"message":        "Session stopped",
		"items_ended":    len(activeItems),
		"session_status": "ended",
	})
}

// ── POST /admin/live/:id/pause — pause bidding (no new bids allowed) ────

func (h *LiveAuctionHandler) AdminPauseSession(c *gin.Context) {
	if !IsAdminLiveControlEnabled() {
		response.Forbidden(c)
		return
	}

	adminID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))

	var sess Session
	if err := h.db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		response.NotFound(c, "Session")
		return
	}
	if sess.Status != StatusLive {
		response.BadRequest(c, "Session is not live")
		return
	}

	// Pause all active items (set status to paused — they keep their timers)
	var activeItems []LiveItem
	h.db.Where("session_id = ? AND status = ?", sessionID, ItemActive).Find(&activeItems)
	for _, item := range activeItems {
		h.db.Model(&item).Update("status", LiveItemStatus("paused"))
		h.cancelTimer(item.ID) // stop timer while paused
	}

	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:     "live_paused",
		SessionID: sessionID.String(),
		Status:    "paused",
	})

	freeze.LogAudit(h.db, "admin_live_pause", adminID, sessionID,
		fmt.Sprintf("items_paused=%d", len(activeItems)))
	response.OK(c, gin.H{"message": "Session paused", "items_paused": len(activeItems)})
}

// ── POST /admin/live/:id/resume — resume a paused session ────────────────

func (h *LiveAuctionHandler) AdminResumeSession(c *gin.Context) {
	if !IsAdminLiveControlEnabled() {
		response.Forbidden(c)
		return
	}

	adminID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))

	var sess Session
	if err := h.db.Where("id = ?", sessionID).First(&sess).Error; err != nil {
		response.NotFound(c, "Session")
		return
	}
	if sess.Status != StatusLive {
		response.BadRequest(c, "Session is not live")
		return
	}

	// Resume paused items
	var pausedItems []LiveItem
	h.db.Where("session_id = ? AND status = ?", sessionID, "paused").Find(&pausedItems)
	for _, item := range pausedItems {
		remaining := 60 * time.Second // default 60s on resume
		if item.EndsAt != nil {
			remaining = time.Until(*item.EndsAt)
			if remaining <= 0 {
				remaining = 60 * time.Second
			}
		}
		newEnd := time.Now().Add(remaining)
		h.db.Model(&item).Updates(map[string]interface{}{
			"status":  ItemActive,
			"ends_at": newEnd,
		})
		h.scheduleItemEnd(item.ID, newEnd)
	}

	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:     "live_resumed",
		SessionID: sessionID.String(),
		Status:    string(StatusLive),
	})

	freeze.LogAudit(h.db, "admin_live_resume", adminID, sessionID,
		fmt.Sprintf("items_resumed=%d", len(pausedItems)))
	response.OK(c, gin.H{"message": "Session resumed", "items_resumed": len(pausedItems)})
}

// ── POST /admin/live/:id/user/:userId/ban — freeze user + block bidding ─

func (h *LiveAuctionHandler) AdminBanUser(c *gin.Context) {
	if !IsAdminLiveControlEnabled() {
		response.Forbidden(c)
		return
	}

	adminID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	targetUserID, _ := uuid.Parse(c.Param("userId"))

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Reason is required")
		return
	}

	// Freeze user (reuse freeze package)
	if err := freeze.FreezeUser(h.db, targetUserID, adminID, "admin_ban:"+req.Reason); err != nil {
		response.InternalError(c, err)
		return
	}

	// Release any reserved funds for active items in this session
	var activeItems []LiveItem
	h.db.Where("session_id = ? AND status = ? AND highest_bidder_id = ?",
		sessionID, ItemActive, targetUserID).Find(&activeItems)
	for _, item := range activeItems {
		if item.CurrentBidCents > 0 {
			if err := wallet.ReleaseReservedFunds(h.db, targetUserID, item.CurrentBidCents); err != nil {
				slog.Warn("admin-ban: failed to release bidder funds",
					"user", targetUserID, "item", item.ID, "error", err)
			}
		}
	}

	// Broadcast
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:     "user_banned",
		SessionID: sessionID.String(),
		ViewerID:  targetUserID.String(),
	})

	freeze.LogAudit(h.db, "admin_user_ban", adminID, targetUserID,
		fmt.Sprintf("session=%s reason=%s items_released=%d", sessionID, req.Reason, len(activeItems)))
	slog.Info("admin: user banned from live session",
		"user_id", targetUserID, "admin", adminID, "session", sessionID)

	response.OK(c, gin.H{
		"message":        "User banned",
		"funds_released": len(activeItems),
	})
}

// ── POST /admin/live/:id/user/:userId/unban — unfreeze user ──────────────

func (h *LiveAuctionHandler) AdminUnbanUser(c *gin.Context) {
	if !IsAdminLiveControlEnabled() {
		response.Forbidden(c)
		return
	}

	adminID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	targetUserID, _ := uuid.Parse(c.Param("userId"))

	if err := freeze.UnfreezeUser(h.db, targetUserID, adminID); err != nil {
		response.InternalError(c, err)
		return
	}

	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:     "user_unbanned",
		SessionID: sessionID.String(),
		ViewerID:  targetUserID.String(),
	})

	freeze.LogAudit(h.db, "admin_user_unban", adminID, targetUserID,
		fmt.Sprintf("session=%s", sessionID))
	response.OK(c, gin.H{"message": "User unbanned"})
}

// ── GET /admin/live/dashboard — admin overview ───────────────────────────

func (h *LiveAuctionHandler) AdminDashboard(c *gin.Context) {
	if !IsAdminLiveControlEnabled() {
		response.Forbidden(c)
		return
	}

	// Active sessions
	var activeSessions []Session
	h.db.Where("status = ?", StatusLive).Find(&activeSessions)

	// Flagged items (sorted by risk_score DESC)
	var flaggedItems []LiveItem
	h.db.Where("requires_review = ?", true).
		Order("risk_score DESC").
		Find(&flaggedItems)

	// Banned users in active sessions
	var bannedUsers []struct {
		UserID    uuid.UUID `json:"user_id"`
		Reason    string    `json:"reason"`
		CreatedAt time.Time `json:"created_at"`
	}
	h.db.Table("user_freezes").
		Where("is_frozen = ?", true).
		Select("user_id, reason, created_at").
		Scan(&bannedUsers)

	// Deposits collected
	var depositStats []struct {
		ItemID        uuid.UUID `json:"item_id"`
		TotalDeposits int       `json:"total_deposits"`
		TotalCents    int64     `json:"total_cents"`
	}
	h.db.Table("auction_deposits").
		Select("item_id, count(*) as total_deposits, sum(deposit_cents) as total_cents").
		Where("status = ?", "held").
		Group("item_id").
		Scan(&depositStats)

	// Total escrow locked in live auctions
	var escrowLocked struct {
		TotalCents int64 `json:"total_cents"`
	}
	h.db.Table("live_items").
		Select("COALESCE(SUM(current_bid_cents), 0) as total_cents").
		Where("status IN ?", []LiveItemStatus{ItemActive, ItemSettling, ItemSold}).
		Scan(&escrowLocked)

	// Active bidders count per session
	type SessionBidders struct {
		SessionID     uuid.UUID `json:"session_id"`
		ActiveBidders int       `json:"active_bidders"`
	}
	var sessionBidders []SessionBidders
	h.db.Table("live_items").
		Select("session_id, COUNT(DISTINCT highest_bidder_id) as active_bidders").
		Where("status = ? AND highest_bidder_id IS NOT NULL", ItemActive).
		Group("session_id").
		Scan(&sessionBidders)

	// Suspicious activity count (high risk_score items + recent freezes)
	var suspiciousCount struct {
		Count int64 `json:"count"`
	}
	h.db.Table("live_items").
		Where("risk_score >= ? AND status IN ?", RiskScoreReview, []LiveItemStatus{ItemPending, ItemActive}).
		Count(&suspiciousCount.Count)

	// Top risky users (users with most flagged/blocked items)
	var riskyUsers []struct {
		UserID       uuid.UUID `json:"user_id"`
		FlaggedCount int       `json:"flagged_count"`
		MaxRiskScore int       `json:"max_risk_score"`
	}
	h.db.Table("live_items").
		Select("session_id as user_id, COUNT(*) as flagged_count, MAX(risk_score) as max_risk_score").
		Where("risk_score >= ? AND requires_review = ?", RiskScoreReview, true).
		Group("session_id").
		Order("max_risk_score DESC").
		Limit(10).
		Scan(&riskyUsers)

	// System status
	systemStatus := "enabled"
	if IsLiveSystemDisabled() {
		systemStatus = "disabled"
	}

	response.OK(c, gin.H{
		"system_status":       systemStatus,
		"active_sessions":     len(activeSessions),
		"sessions":            activeSessions,
		"flagged_items":       len(flaggedItems),
		"flagged":             flaggedItems,
		"banned_users":        len(bannedUsers),
		"banned":              bannedUsers,
		"deposit_stats":       depositStats,
		"escrow_locked_cents": escrowLocked.TotalCents,
		"session_bidders":     sessionBidders,
		"suspicious_count":    suspiciousCount.Count,
		"top_risky_users":     riskyUsers,
	})
}
