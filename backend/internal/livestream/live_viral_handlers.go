package livestream

import (
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 15: Viral Growth HTTP Handlers
// ════════════════════════════════════════════════════════════════════════════

// ── POST /livestream/:id/invites — create a live invite link ──────────────

func (h *LiveAuctionHandler) CreateLiveInviteHandler(c *gin.Context) {
	if !IsLiveInvitesEnabled() {
		response.BadRequest(c, "live invites disabled")
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	inv, err := CreateLiveInvite(h.db, userID, sessionID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, inv)
}

// ── POST /livestream/invites/:code/track — record invitee join ────────────

func (h *LiveAuctionHandler) TrackLiveInviteHandler(c *gin.Context) {
	if !IsLiveInvitesEnabled() {
		response.BadRequest(c, "live invites disabled")
		return
	}
	inviteeID, _ := uuid.Parse(c.GetString("user_id"))
	code := c.Param("code")
	inv, err := TrackLiveInvite(h.db, code, inviteeID, c.ClientIP())
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, inv)
}

// ── POST /livestream/items/:itemId/win-shares — generate brag share ───────

func (h *LiveAuctionHandler) CreateWinShareHandler(c *gin.Context) {
	if !IsShareRewardsEnabled() {
		response.BadRequest(c, "share rewards disabled")
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		response.BadRequest(c, "invalid item id")
		return
	}
	// Verify user won this item
	var item LiveItem
	if err := h.db.Where("id = ?", itemID).First(&item).Error; err != nil {
		response.NotFound(c, "item")
		return
	}
	if item.HighestBidderID == nil || *item.HighestBidderID != userID || item.Status != ItemSold {
		response.Forbidden(c)
		return
	}
	ws, err := CreateWinShare(h.db, userID, item.SessionID, itemID, item.CurrentBidCents)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, ws)
}

// ── POST /livestream/shares/:code/attribute — new user joined via share ──

func (h *LiveAuctionHandler) AttributeShareJoinHandler(c *gin.Context) {
	if !IsShareRewardsEnabled() {
		response.BadRequest(c, "share rewards disabled")
		return
	}
	newUserID, _ := uuid.Parse(c.GetString("user_id"))
	code := c.Param("code")
	if err := AttributeShareJoin(h.db, code, newUserID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"attributed": true})
}

// ── POST /livestream/:id/group-invites — create group invite ──────────────

func (h *LiveAuctionHandler) CreateGroupInviteHandler(c *gin.Context) {
	if !IsGroupBuyEnabled() {
		response.BadRequest(c, "group buy disabled")
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	g, err := CreateGroupInvite(h.db, userID, sessionID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, g)
}

// ── POST /livestream/group-invites/:code/join ──────────────────────────────

func (h *LiveAuctionHandler) JoinGroupInviteHandler(c *gin.Context) {
	if !IsGroupBuyEnabled() {
		response.BadRequest(c, "group buy disabled")
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	code := c.Param("code")
	g, err := JoinGroupInvite(h.db, code, userID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, g)
}

// ── GET /livestream/streaks/me — user's streak summary ────────────────────

func (h *LiveAuctionHandler) MyStreaksHandler(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	var streaks []UserStreak
	h.db.Where("user_id = ?", userID).Find(&streaks)
	response.OK(c, gin.H{"streaks": streaks})
}

// ── GET /admin/growth/metrics ──────────────────────────────────────────────

func (h *LiveAuctionHandler) AdminGrowthMetrics(c *gin.Context) {
	m := ComputeGrowthMetrics(h.db)
	response.OK(c, m)
}
