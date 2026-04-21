package livestream

import (
	"fmt"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Sprint 16: Creator Economy HTTP Handlers
// ════════════════════════════════════════════════════════════════════════════

// ── POST /creators/apply ───────────────────────────────────────────────────

func (h *LiveAuctionHandler) ApplyCreatorHandler(c *gin.Context) {
	if !IsCreatorsEnabled() {
		response.BadRequest(c, "creator system disabled")
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	var req struct {
		DisplayName    string `json:"display_name" binding:"required"`
		Niche          string `json:"niche" binding:"required"`
		FollowersCount int    `json:"followers_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	creator, err := ApplyCreator(h.db, userID, req.DisplayName, req.Niche, req.FollowersCount)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, creator)
}

// ── GET /creators/:id ─────────────────────────────────────────────────────

func (h *LiveAuctionHandler) GetCreatorHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid creator id")
		return
	}
	creator, err := GetCreator(h.db, id)
	if err != nil {
		response.NotFound(c, "creator")
		return
	}
	response.OK(c, creator)
}

// ── GET /creators/top ──────────────────────────────────────────────────────

func (h *LiveAuctionHandler) TopCreatorsHandler(c *gin.Context) {
	limit := 20
	if l, err := parseIntQuery(c, "limit"); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	creators, err := GetTopCreators(h.db, limit)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, creators)
}

// ── GET /creators/me ──────────────────────────────────────────────────────

func (h *LiveAuctionHandler) MyCreatorProfileHandler(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	creator, err := GetCreatorByUser(h.db, userID)
	if err != nil {
		response.NotFound(c, "creator profile")
		return
	}
	response.OK(c, creator)
}

// ── GET /creators/:id/analytics ───────────────────────────────────────────

func (h *LiveAuctionHandler) CreatorAnalyticsHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid creator id")
		return
	}
	analytics, err := GetCreatorAnalytics(h.db, id)
	if err != nil {
		response.NotFound(c, "creator")
		return
	}
	response.OK(c, analytics)
}

// ── POST /creators/deals/invite ───────────────────────────────────────────

func (h *LiveAuctionHandler) InviteCreatorHandler(c *gin.Context) {
	if !IsCreatorsEnabled() {
		response.BadRequest(c, "creator system disabled")
		return
	}
	sellerID, _ := uuid.Parse(c.GetString("user_id"))
	var req struct {
		CreatorID      uuid.UUID `json:"creator_id" binding:"required"`
		CommissionRate float64   `json:"commission_rate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	deal, err := InviteCreator(h.db, sellerID, req.CreatorID, req.CommissionRate)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, deal)
}

// ── POST /creators/deals/:id/accept ───────────────────────────────────────

func (h *LiveAuctionHandler) AcceptDealHandler(c *gin.Context) {
	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid deal id")
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	// Resolve creator profile from user
	creator, err := GetCreatorByUser(h.db, userID)
	if err != nil {
		response.Forbidden(c)
		return
	}
	deal, err := AcceptDeal(h.db, dealID, creator.ID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, deal)
}

// ── POST /creators/deals/:id/reject ───────────────────────────────────────

func (h *LiveAuctionHandler) RejectDealHandler(c *gin.Context) {
	dealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid deal id")
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))
	creator, err := GetCreatorByUser(h.db, userID)
	if err != nil {
		response.Forbidden(c)
		return
	}
	if err := RejectDeal(h.db, dealID, creator.ID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"rejected": true})
}

// ── GET /creators/deals ───────────────────────────────────────────────────

func (h *LiveAuctionHandler) MyDealsHandler(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	creator, err := GetCreatorByUser(h.db, userID)
	if err != nil {
		response.NotFound(c, "creator profile")
		return
	}
	deals, err := GetCreatorDeals(h.db, creator.ID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, deals)
}

// ── GET /sellers/deals ────────────────────────────────────────────────────

func (h *LiveAuctionHandler) SellerDealsHandler(c *gin.Context) {
	sellerID, _ := uuid.Parse(c.GetString("user_id"))
	deals, err := GetSellerDeals(h.db, sellerID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, deals)
}

// ── POST /creators/:id/match-items/:itemId ────────────────────────────────

func (h *LiveAuctionHandler) MatchCreatorsForItemHandler(c *gin.Context) {
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		response.BadRequest(c, "invalid item id")
		return
	}
	limit := 10
	if l, err := parseIntQuery(c, "limit"); err == nil && l > 0 && l <= 50 {
		limit = l
	}
	scores, err := FindBestCreatorsForItem(h.db, itemID, limit)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, scores)
}

// ── GET /creators/:id/referral-code ───────────────────────────────────────

func (h *LiveAuctionHandler) CreatorReferralCodeHandler(c *gin.Context) {
	creatorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid creator id")
		return
	}
	sessionIDStr := c.Query("session_id")
	if sessionIDStr == "" {
		response.BadRequest(c, "session_id required")
		return
	}
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		response.BadRequest(c, "invalid session_id")
		return
	}
	var c2 Creator
	if err := h.db.Where("id = ?", creatorID).First(&c2).Error; err != nil {
		response.NotFound(c, "creator")
		return
	}
	code, err := GetCreatorReferralCode(h.db, c2.UserID, sessionID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"referral_code": code})
}

// ── GET /admin/creators/payouts ────────────────────────────────────────────

func (h *LiveAuctionHandler) AdminCreatorPayoutsHandler(c *gin.Context) {
	payouts, err := GetPendingCreatorPayouts(h.db)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, payouts)
}

// ── POST /admin/creators/:id/refresh-trust ────────────────────────────────

func (h *LiveAuctionHandler) AdminRefreshCreatorTrustHandler(c *gin.Context) {
	creatorID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid creator id")
		return
	}
	if err := RefreshCreatorTrust(h.db, creatorID); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	c2, _ := GetCreator(h.db, creatorID)
	response.OK(c, c2)
}

// parseIntQuery parses an integer query parameter.
func parseIntQuery(c *gin.Context, key string) (int, error) {
	val := c.Query(key)
	if val == "" {
		return 0, fmt.Errorf("missing")
	}
	var n int
	_, err := fmt.Sscanf(val, "%d", &n)
	return n, err
}
