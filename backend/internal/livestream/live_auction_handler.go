package livestream

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/fraud"
	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ════════════════════════════════════════════════════════════════════════════
// Live Auction Handler — manages items + bidding within a live session.
// Reuses existing WebSocket infrastructure via Redis pub/sub.
// ════════════════════════════════════════════════════════════════════════════

const (
	antiSnipeWindow    = 10 * time.Second
	antiSnipeExtension = 30 * time.Second
	maxBidRatePerMin   = 20
	antiBotMinDelayMs  = 500 // minimum 500ms between bids
)

type LiveAuctionHandler struct {
	db     *gorm.DB
	rdb    *redis.Client
	timers map[uuid.UUID]context.CancelFunc
	mu     sync.Mutex
}

func NewLiveAuctionHandler(db *gorm.DB, rdb *redis.Client) *LiveAuctionHandler {
	h := &LiveAuctionHandler{db: db, rdb: rdb, timers: make(map[uuid.UUID]context.CancelFunc)}
	h.restoreActiveItems()
	return h
}

// restoreActiveItems restarts timers for items that were active when server restarted.
func (h *LiveAuctionHandler) restoreActiveItems() {
	var items []LiveItem
	h.db.Where("status = ? AND ends_at > ?", ItemActive, time.Now()).Find(&items)
	for _, item := range items {
		if item.EndsAt != nil {
			h.scheduleItemEnd(item.ID, *item.EndsAt)
		}
	}
	if len(items) > 0 {
		slog.Info("live-auction: restored active items", "count", len(items))
	}
}

// ── POST /api/v1/livestream/:id/items — add item to session ──────────────

type AddItemReq struct {
	ListingID         *string `json:"listing_id"`
	Title             string  `json:"title" binding:"required,min=3"`
	ImageURL          string  `json:"image_url"`
	StartPriceCents   int64   `json:"start_price_cents" binding:"required,min=1"`
	BuyNowPriceCents  *int64  `json:"buy_now_price_cents"`
	MinIncrementCents int64   `json:"min_increment_cents"`
	DurationSeconds   int     `json:"duration_seconds" binding:"required,min=10,max=3600"`
}

func (h *LiveAuctionHandler) AddItem(c *gin.Context) {
	hostID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))

	// ── Global panic check ──────────────────────────────────────────────────
	if IsLiveSystemDisabled() {
		response.BadRequest(c, "Live system temporarily disabled")
		return
	}

	// ── Freeze check: frozen users cannot create items ──────────────────────
	if freeze.IsUserFrozen(h.db, hostID) {
		response.Forbidden(c)
		return
	}

	// Verify host owns the session
	var sess Session
	if err := h.db.Where("id = ? AND host_id = ?", sessionID, hostID).First(&sess).Error; err != nil {
		response.Forbidden(c)
		return
	}
	if sess.Status != StatusLive && sess.Status != StatusScheduled {
		response.BadRequest(c, "session is not active")
		return
	}

	var req AddItemReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var listingID *uuid.UUID
	if req.ListingID != nil && *req.ListingID != "" {
		id, _ := uuid.Parse(*req.ListingID)
		listingID = &id
	}

	minInc := req.MinIncrementCents
	if minInc <= 0 {
		minInc = 100
	}

	// ── Prohibited items check ────────────────────────────────────────────────
	compliance := EnforceCompliance(h.db, hostID, req.Title, req.ImageURL, "")
	if compliance.Verdict == VerdictBlocked {
		response.Forbidden(c)
		return
	}

	item := LiveItem{
		SessionID:         sessionID,
		ListingID:         listingID,
		Title:             req.Title,
		ImageURL:          req.ImageURL,
		StartPriceCents:   req.StartPriceCents,
		BuyNowPriceCents:  req.BuyNowPriceCents,
		CurrentBidCents:   0,
		MinIncrementCents: minInc,
		Status:            ItemPending,
		AntiSnipeEnabled:  true,
		RequiresReview:    compliance.Verdict == VerdictFlagged,
		RiskScore:         compliance.RiskScore,
	}

	// Auto-apply deposit rules for high-value items
	ApplyDepositRules(&item)
	if err := h.db.Create(&item).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	freeze.LogAudit(h.db, "live_item_added", hostID, item.ID, fmt.Sprintf("session=%s title=%s", sessionID, req.Title))
	response.Created(c, item)
}

// ── POST /api/v1/livestream/:id/items/:itemId/activate — start bidding ───

func (h *LiveAuctionHandler) ActivateItem(c *gin.Context) {
	hostID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))

	var sess Session
	if err := h.db.Where("id = ? AND host_id = ?", sessionID, hostID).First(&sess).Error; err != nil {
		response.Forbidden(c)
		return
	}
	if sess.Status != StatusLive {
		response.BadRequest(c, "session must be live to activate items")
		return
	}

	var req struct {
		DurationSeconds int `json:"duration_seconds" binding:"required,min=10,max=3600"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var item LiveItem
	if err := h.db.Where("id = ? AND session_id = ? AND status = ?", itemID, sessionID, ItemPending).First(&item).Error; err != nil {
		response.NotFound(c, "Item")
		return
	}

	// ── Double-check compliance before going live ────────────────────────────
	if IsProhibitedCheckEnabled() {
		compliance := CheckItemCompliance(item.Title, item.ImageURL, "")
		if compliance.Verdict == VerdictBlocked {
			// Item was allowed at creation but flagged — block before activation
			h.db.Model(&item).Update("status", ItemCancelled)
			freeze.LogAudit(h.db, "prohibited_item_cancelled", hostID, item.ID,
				"reason="+compliance.ReasonCode)
			response.BadRequest(c, "Item blocked: "+compliance.ReasonCode)
			return
		}
		if compliance.Verdict == VerdictFlagged && !item.RequiresReview {
			// Update requires_review flag if not already set
			h.db.Model(&item).Update("requires_review", true)
		}
	}

	endsAt := time.Now().Add(time.Duration(req.DurationSeconds) * time.Second)
	h.db.Model(&item).Updates(map[string]interface{}{
		"status":  ItemActive,
		"ends_at": endsAt,
	})
	item.Status = ItemActive
	item.EndsAt = &endsAt

	h.scheduleItemEnd(item.ID, endsAt)
	h.broadcastItemUpdate(sessionID, &item, "item_activated")

	// Sprint 11: toast for new item starting
	h.BroadcastToast(sessionID, "🆕 New item starting: "+item.Title, "🆕")

	freeze.LogAudit(h.db, "live_item_activated", hostID, item.ID, fmt.Sprintf("duration=%ds", req.DurationSeconds))
	response.OK(c, item)
}

// ── POST /api/v1/livestream/:id/items/:itemId/bid — place bid ────────────

func (h *LiveAuctionHandler) PlaceBid(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))

	// ── Global panic check ──────────────────────────────────────────────────
	if IsLiveSystemDisabled() {
		response.BadRequest(c, "Live system temporarily disabled")
		return
	}

	// ── Fix 1: Freeze check ───────────────────────────────────────────────
	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	// ── Fix 1: Idempotency ──────────────────────────────────────────────────
	idemKey := c.GetHeader("X-Idempotency-Key")
	if idemKey != "" && h.rdb != nil {
		cacheKey := "bid_idem:" + idemKey
		cached, err := h.rdb.Get(context.Background(), cacheKey).Result()
		if err == nil && cached != "" {
			// Return cached result — prevents double reserve / duplicate bid
			c.Data(201, "application/json", []byte(cached))
			return
		}
	}

	// Rate limit + anti-bot delay
	if h.rdb != nil {
		if !h.checkBidRateLimit(userID, itemID) {
			response.RateLimited(c, "Too many bids or bidding too fast. Please wait.")
			return
		}
	}

	var req struct {
		AmountCents int64 `json:"amount_cents" binding:"required,min=1"`
	}
	// Quick-bid hint: amount may be pre-computed by QuickBid handler
	if hint, ok := c.Get("quick_bid_amount_cents"); ok {
		if amt, ok2 := hint.(int64); ok2 && amt > 0 {
			req.AmountCents = amt
		}
	}
	if req.AmountCents == 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	// ── Sprint 11.5: Bidder Quality Filter ──────────────────────────────────
	qr := CheckBidderQuality(h.db, userID, req.AmountCents)
	if !qr.Allowed {
		response.Forbidden(c)
		freeze.LogAudit(h.db, "live_bid_blocked_low_trust", userID, itemID,
			fmt.Sprintf("trust_score=%.1f amount_cents=%d", qr.UserTrustScore, req.AmountCents))
		return
	}
	// High-value bids (≥20k EGP) require deposit — emit signal for the frontend,
	// but only enforce if item is configured with RequiresEntryDeposit.
	if qr.RequireDeposit && !HasUserPaidDeposit(h.db, userID, itemID) {
		response.BadRequest(c,
			"High-value bid requires entry deposit — POST /livestream/:id/items/:itemId/deposit first")
		return
	}

	// ── Fix 5: Fraud check ──────────────────────────────────────────────────
	fraudResult := fraud.CheckRiskBeforeBuyNow(context.Background(), h.db, userID, uuid.Nil, float64(req.AmountCents)/100.0, c.ClientIP())
	if fraudResult.Action == "block" {
		response.Forbidden(c)
		return
	}

	// ── Deposit-gated check: user must have paid deposit for high-value items ──
	if IsAuctionDepositEnabled() {
		var targetItem LiveItem
		if err := h.db.Where("id = ? AND session_id = ?", itemID, sessionID).First(&targetItem).Error; err == nil {
			if targetItem.RequiresEntryDeposit && !HasUserPaidDeposit(h.db, userID, itemID) {
				response.BadRequest(c, "Deposit required to bid on this item. Please pay the entry deposit first.")
				return
			}
			// Dynamic deposit: re-check coverage against current highest bid
			if targetItem.RequiresEntryDeposit && targetItem.CurrentBidCents > 0 {
				covered, shortfall := ValidateDepositCoverage(h.db, userID, itemID, targetItem.StartPriceCents, targetItem.CurrentBidCents)
				if !covered {
					response.BadRequest(c, fmt.Sprintf("Deposit insufficient for current bid level. Please top up by %d cents.", shortfall))
					return
				}
			}
		}
	}

	// ── Fix 2: Pre-flight balance check (available_balance only) ────────────
	if !wallet.HasSufficientBalance(h.db, userID, req.AmountCents) {
		response.BadRequest(c, "Insufficient balance to place this bid")
		return
	}

	var item LiveItem
	var extended bool
	var prevBidderID *uuid.UUID
	var prevBidAmount int64

	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND session_id = ? AND status = ?", itemID, sessionID, ItemActive).
			First(&item).Error; err != nil {
			return fmt.Errorf("item_not_found")
		}

		// Check time
		if item.EndsAt != nil && time.Now().After(*item.EndsAt) {
			return fmt.Errorf("item_ended")
		}

		// Validate bid amount
		minBid := item.CurrentBidCents + item.MinIncrementCents
		if item.BidCount == 0 {
			minBid = item.StartPriceCents
		}
		if req.AmountCents < minBid {
			return fmt.Errorf("bid_too_low:%d", minBid)
		}

		// Cannot bid on own item
		var sess Session
		tx.Where("id = ?", sessionID).First(&sess)
		if sess.HostID == userID {
			return fmt.Errorf("cannot_bid_own")
		}

		// Capture previous highest bidder for fund release
		prevBidderID = item.HighestBidderID
		prevBidAmount = item.CurrentBidCents

		// ── Fix 2: Re-validate balance inside TX (with lock) ────────────
		// ReserveFunds already checks available_balance >= amount under lock.
		// If it fails here, the pre-flight was stale — correct behavior.
		if err := wallet.ReserveFunds(tx, userID, req.AmountCents); err != nil {
			return fmt.Errorf("reserve_failed:%s", err.Error())
		}

		// Release previous bidder's funds (pending → available)
		if prevBidderID != nil && *prevBidderID != userID {
			if err := wallet.ReleaseReservedFunds(tx, *prevBidderID, prevBidAmount); err != nil {
				slog.Warn("live-bid: failed to release prev bidder funds",
					"prev_bidder", prevBidderID, "amount", prevBidAmount, "error", err)
			}
		}

		// Anti-sniping with cap
		if item.AntiSnipeEnabled && item.EndsAt != nil && item.ExtensionCount < maxAntiSnipeExtensions {
			if time.Until(*item.EndsAt) <= antiSnipeWindow {
				newEnd := item.EndsAt.Add(antiSnipeExtension)
				item.EndsAt = &newEnd
				item.ExtensionCount++
				extended = true
			}
		}

		bid := LiveBid{
			ItemID:         itemID,
			UserID:         userID,
			BidAmountCents: req.AmountCents,
		}
		if err := tx.Create(&bid).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"current_bid_cents": req.AmountCents,
			"highest_bidder_id": userID,
			"bid_count":         gorm.Expr("bid_count + 1"),
		}
		if extended {
			updates["ends_at"] = item.EndsAt
			updates["extension_count"] = item.ExtensionCount
		}
		return tx.Model(&item).Updates(updates).Error
	})

	if dbErr != nil {
		msg := dbErr.Error()
		switch {
		case msg == "item_not_found":
			response.NotFound(c, "Item")
		case msg == "item_ended":
			response.BadRequest(c, "Bidding has ended for this item")
		case msg == "cannot_bid_own":
			response.BadRequest(c, "Cannot bid on your own item")
		case len(msg) > 15 && msg[:15] == "reserve_failed:":
			response.BadRequest(c, "Insufficient balance: "+msg[15:])
		default:
			response.BadRequest(c, msg)
		}
		return
	}

	// Reschedule timer if extended
	if extended {
		h.cancelTimer(itemID)
		h.scheduleItemEnd(itemID, *item.EndsAt)
	}

	// Broadcast new_bid event with enriched data
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:           EventNewBid,
		SessionID:       sessionID.String(),
		ItemID:          itemID.String(),
		CurrentBidCents: req.AmountCents,
		HighestBidderID: strPtr(userID.String()),
		BidCount:        item.BidCount + 1,
		Status:          string(ItemActive),
		EndsAt:          timeToStr(item.EndsAt),
		Extended:        extended,
		NewEndsAt:       timeToStr(item.EndsAt),
		ExtensionCount:  item.ExtensionCount,
	})

	// Fetch recent bidders for social proof
	recentBidders := h.fetchRecentBidders(itemID, 5)
	h.broadcastSocialProof(sessionID, itemID, item.BidCount+1, recentBidders)

	// Send dedicated outbid event to previous highest bidder
	if prevBidderID != nil {
		h.broadcastOutbid(sessionID, itemID, prevBidderID, req.AmountCents)
	}

	// ── Sprint 11: Live Conversion Engine integration ──────────────────────
	// 1. Record bid velocity + broadcast urgency state
	if IsLiveFomoEnabled() {
		// We don't have the bid's ID here (the bid row is created inside the tx).
		// For velocity, a synthetic ID is sufficient since we only count timestamps.
		recordBidVelocity(h.rdb, itemID, uuid.New())
		h.publishUrgencyUpdate(sessionID, itemID)
	}
	// 2. Auto buy-now trigger at ≥90% of buy-now price
	h.maybeTriggerBuyNowAlmost(sessionID, itemID, req.AmountCents, item.BuyNowPriceCents)
	// 3. Funnel event
	{
		uid := userID
		iid := itemID
		RecordConversionEvent(h.db, sessionID, &iid, &uid, StagePlaceBid, req.AmountCents, "")
	}
	// 4. Nudge the outbid user
	if prevBidderID != nil {
		SendLiveNudge(h.rdb, *prevBidderID, NudgeContext{
			SessionID:  sessionID,
			ItemID:     itemID,
			Code:       NudgeOutbid,
			CurrentBid: req.AmountCents,
			BuyNow:     item.BuyNowPriceCents,
		})
	}
	// 5. Real-time toast
	h.BroadcastToast(sessionID, fmt.Sprintf("%s placed a bid", shortUserLabel(userID)), "💥")

	// Sprint 14: AI assistant evaluates the new state and may emit a suggestion
	go h.EvaluateAndSuggest(sessionID, itemID)

	// Sprint 15: Viral growth hooks (non-blocking)
	go func() {
		GrantFirstBidCashback(h.db, userID)         // +1 EGP on first-ever live bid
		UpdateStreak(h.db, userID, "bid")           // daily-bid streak
		RewardInviterOnBid(h.db, userID, sessionID) // invitee's inviter reward
	}()

	// Audit (include fraud flags if any)
	auditDetails := fmt.Sprintf("amount_cents=%d session=%s extended=%v reserved=true", req.AmountCents, sessionID, extended)
	if len(fraudResult.Flags) > 0 {
		auditDetails += fmt.Sprintf(" fraud_flags=%v fraud_score=%d", fraudResult.Flags, fraudResult.RiskScore)
	}
	freeze.LogAudit(h.db, "live_bid_placed", userID, itemID, auditDetails)

	result := gin.H{"bid_accepted": true, "current_bid_cents": req.AmountCents, "bid_count": item.BidCount + 1}
	if extended {
		result["extended"] = true
		result["new_ends_at"] = item.EndsAt
	}

	// ── Fix 1: Cache idempotent result ──────────────────────────────────────
	if idemKey != "" && h.rdb != nil {
		resultJSON, _ := json.Marshal(gin.H{"data": result})
		h.rdb.Set(context.Background(), "bid_idem:"+idemKey, string(resultJSON), idempotencyTTL)
	}

	response.Created(c, result)
}

// checkBidRateLimit enforces rate limit (20/min) + anti-bot delay (500ms min between bids).
func (h *LiveAuctionHandler) checkBidRateLimit(userID uuid.UUID, itemID uuid.UUID) bool {
	ctx := context.Background()
	key := fmt.Sprintf("live_bid_rate:%s:%s", userID, itemID)

	// Rate limit
	cnt, _ := h.rdb.Incr(ctx, key).Result()
	if cnt == 1 {
		h.rdb.Expire(ctx, key, time.Minute)
	}
	if cnt > maxBidRatePerMin {
		return false
	}

	// Anti-bot: minimum delay between bids
	lastKey := key + ":last"
	lastStr, _ := h.rdb.Get(ctx, lastKey).Result()
	if lastStr != "" {
		lastMs, _ := strconv.ParseInt(lastStr, 10, 64)
		if time.Now().UnixMilli()-lastMs < antiBotMinDelayMs {
			return false
		}
	}
	h.rdb.Set(ctx, lastKey, time.Now().UnixMilli(), time.Minute)
	return true
}

// ── POST /api/v1/livestream/:id/items/:itemId/buy-now — instant purchase ─

func (h *LiveAuctionHandler) BuyNow(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	sessionID, _ := uuid.Parse(c.Param("id"))
	itemID, _ := uuid.Parse(c.Param("itemId"))

	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	// Fraud check
	fraudResult := fraud.CheckRiskBeforeBuyNow(context.Background(), h.db, userID, uuid.Nil, 0, c.ClientIP())
	if fraudResult.Action == "block" {
		response.Forbidden(c)
		return
	}

	var item LiveItem
	var prevBidderID *uuid.UUID
	var prevBidAmount int64

	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND session_id = ? AND status = ?", itemID, sessionID, ItemActive).
			First(&item).Error; err != nil {
			return fmt.Errorf("item_not_found")
		}
		if item.BuyNowPriceCents == nil {
			return fmt.Errorf("no_buy_now")
		}
		var sess Session
		tx.Where("id = ?", sessionID).First(&sess)
		if sess.HostID == userID {
			return fmt.Errorf("cannot_buy_own")
		}

		buyPrice := *item.BuyNowPriceCents

		// Pre-flight balance check
		if !wallet.HasSufficientBalance(h.db, userID, buyPrice) {
			return fmt.Errorf("insufficient_balance")
		}

		// Reserve buyer's funds (available → pending)
		if err := wallet.ReserveFunds(tx, userID, buyPrice); err != nil {
			return fmt.Errorf("reserve_failed:%s", err.Error())
		}

		// Release previous highest bidder's reserved funds
		prevBidderID = item.HighestBidderID
		prevBidAmount = item.CurrentBidCents
		if prevBidderID != nil && *prevBidderID != userID && prevBidAmount > 0 {
			if err := wallet.ReleaseReservedFunds(tx, *prevBidderID, prevBidAmount); err != nil {
				slog.Warn("live-buy-now: failed to release prev bidder funds",
					"prev_bidder", prevBidderID, "amount", prevBidAmount, "error", err)
			}
		}

		return tx.Model(&item).Updates(map[string]interface{}{
			"status":            ItemSold,
			"current_bid_cents": buyPrice,
			"highest_bidder_id": userID,
			"ends_at":           time.Now(),
		}).Error
	})

	if dbErr != nil {
		msg := dbErr.Error()
		switch {
		case msg == "item_not_found":
			response.NotFound(c, "Item")
		case msg == "no_buy_now":
			response.BadRequest(c, "Buy Now not available")
		case msg == "cannot_buy_own":
			response.BadRequest(c, "Cannot buy your own item")
		case msg == "insufficient_balance":
			response.BadRequest(c, "Insufficient balance for Buy Now")
		case len(msg) > 15 && msg[:15] == "reserve_failed:":
			response.BadRequest(c, "Insufficient balance: "+msg[15:])
		default:
			response.InternalError(c, dbErr)
		}
		return
	}

	h.cancelTimer(itemID)

	// Broadcast sold event
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:            EventItemSoldBuyNow,
		SessionID:        sessionID.String(),
		ItemID:           itemID.String(),
		CurrentBidCents:  *item.BuyNowPriceCents,
		HighestBidderID:  strPtr(userID.String()),
		BuyNowPriceCents: item.BuyNowPriceCents,
		Status:           string(ItemSold),
	})

	// Sprint 11: funnel + toast
	{
		uid := userID
		iid := itemID
		RecordConversionEvent(h.db, sessionID, &iid, &uid, StageBuyNow, *item.BuyNowPriceCents, "")
		RecordConversionEvent(h.db, sessionID, &iid, &uid, StageWin, *item.BuyNowPriceCents, "buy_now")
	}
	h.BroadcastToast(sessionID, "🎉 Item sold via Buy Now!", "🎉")

	// Settlement: convert reserve → escrow (async like PlaceBid flow)
	go h.finalizeItem(&item)

	freeze.LogAudit(h.db, "live_buy_now", userID, itemID,
		fmt.Sprintf("price_cents=%d reserved=true escrow=pending", *item.BuyNowPriceCents))
	response.OK(c, gin.H{"message": "Purchase successful!", "price_cents": *item.BuyNowPriceCents})
}

// ── GET /api/v1/livestream/:id/items — list items in session ─────────────

func (h *LiveAuctionHandler) ListItems(c *gin.Context) {
	sessionID := c.Param("id")
	var items []LiveItem
	h.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&items)
	response.OK(c, items)
}

// ── GET /api/v1/livestream/:id/items/:itemId/bids — bid history ──────────

func (h *LiveAuctionHandler) ListBids(c *gin.Context) {
	itemID := c.Param("itemId")
	var bids []LiveBid
	h.db.Where("item_id = ?", itemID).Order("bid_amount_cents DESC").Limit(50).Find(&bids)
	response.OK(c, bids)
}

// ════════════════════════════════════════════════════════════════════════════
// Timer + Broadcast
// ════════════════════════════════════════════════════════════════════════════

func (h *LiveAuctionHandler) scheduleItemEnd(itemID uuid.UUID, endsAt time.Time) {
	dur := time.Until(endsAt)
	if dur <= 0 {
		go h.endItem(itemID)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.mu.Lock()
	h.timers[itemID] = cancel
	h.mu.Unlock()

	go func() {
		select {
		case <-time.After(dur):
			h.endItem(itemID)
		case <-ctx.Done():
		}
	}()
}

func (h *LiveAuctionHandler) cancelTimer(itemID uuid.UUID) {
	h.mu.Lock()
	if cancel, ok := h.timers[itemID]; ok {
		cancel()
		delete(h.timers, itemID)
	}
	h.mu.Unlock()
}

func (h *LiveAuctionHandler) endItem(itemID uuid.UUID) {
	h.cancelTimer(itemID)

	// Phase 1: Mark as "ended" (no more bids accepted)
	var item LiveItem
	if err := h.db.Where("id = ? AND status = ?", itemID, ItemActive).First(&item).Error; err != nil {
		return
	}

	h.db.Model(&item).Update("status", "ended")
	h.broadcastItemUpdate(item.SessionID, &item, "item_ended")
	slog.Info("live-auction: item ended, starting settlement", "item_id", itemID)

	// Phase 2: Settlement (async)
	go h.finalizeItem(&item)
}

// finalizeItem implements the state machine: ENDED → SETTLING → SOLD / PAYMENT_FAILED / UNSOLD
// All DB mutations are atomic within a single transaction with deadlock retry.
func (h *LiveAuctionHandler) finalizeItem(item *LiveItem) {
	// No bids → unsold (no funds to manage)
	if item.BidCount == 0 || item.HighestBidderID == nil {
		h.db.Model(item).Update("status", ItemUnsold)
		h.broadcastItemUpdate(item.SessionID, item, "item_unsold")
		freeze.LogAudit(h.db, "live_item_unsold", uuid.Nil, item.ID,
			fmt.Sprintf("bid_count=%d", item.BidCount))
		slog.Info("live-auction: item unsold (no bids)", "item_id", item.ID)
		return
	}

	buyerID := *item.HighestBidderID

	// ── Atomic settlement with deadlock retry ──────────────────────────────
	var orderID uuid.UUID
	var sellerID uuid.UUID
	var amountFloat, platformFee, sellerAmount float64
	// Sprint 12: commission tracking
	var commissionCents int64
	var commissionRate float64
	var commissionDynamicBonus float64
	var commissionTier CommissionTier
	// Sprint 13: flywheel breakdown for audit
	var commissionSurgeBonus float64
	var commissionWhaleBonus float64
	var sessionStreamerID *uuid.UUID

	settleErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		// Re-read item under lock to confirm state
		var locked LiveItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", item.ID).First(&locked).Error; err != nil {
			return err
		}
		if locked.Status != ItemSold && locked.Status != "ended" && locked.Status != ItemSettling {
			return fmt.Errorf("item_not_settleable:%s", locked.Status)
		}

		// Mark settling with timestamp
		now := time.Now()
		if err := tx.Model(&locked).Updates(map[string]interface{}{
			"status":              ItemSettling,
			"settling_started_at": now,
			"settle_retries":      gorm.Expr("settle_retries + 1"),
		}).Error; err != nil {
			return err
		}

		// Get session host (seller)
		var sess Session
		if err := tx.Where("id = ?", item.SessionID).First(&sess).Error; err != nil {
			// Release winner's reserved funds — session gone
			_ = wallet.ReleaseReservedFunds(tx, buyerID, item.CurrentBidCents)
			tx.Model(&locked).Update("status", ItemPaymentFailed)
			return fmt.Errorf("session_not_found")
		}
		sellerID = sess.HostID
		sessionStreamerID = sess.StreamerID // Sprint 13: capture for creator split

		// ── Settlement balance check: verify winner can still cover the bid ──
		// If reserved funds were released (e.g. by admin ban), the winner
		// may not have sufficient balance anymore.
		if !wallet.HasSufficientBalance(tx, buyerID, item.CurrentBidCents) {
			_ = wallet.ReleaseReservedFunds(tx, buyerID, item.CurrentBidCents)
			// Forfeit deposit if applicable
			if item.RequiresEntryDeposit {
				_ = ForfeitDeposit(tx, buyerID, item.ID, uuid.Nil)
			}
			tx.Model(&locked).Update("status", ItemPaymentFailed)
			freeze.LogAudit(h.db, "insufficient_settlement_balance", buyerID, item.ID,
				fmt.Sprintf("bid_cents=%d deposit_forfeited=%v", item.CurrentBidCents, item.RequiresEntryDeposit))
			return fmt.Errorf("insufficient_settlement_balance")
		}

		// Convert reserve → escrow hold
		refID := item.ID.String()
		_, err := wallet.ConvertReserveToHold(tx, buyerID, sellerID, item.CurrentBidCents, "live_auction", refID)
		if err != nil {
			// Release winner's reserved funds — escrow failed
			_ = wallet.ReleaseReservedFunds(tx, buyerID, item.CurrentBidCents)
			tx.Model(&locked).Update("status", ItemPaymentFailed)
			return fmt.Errorf("escrow_failed:%s", err.Error())
		}

		// Calculate commission (Sprint 12: tiered + dynamic) or fall back to static fee
		amountFloat = float64(item.CurrentBidCents) / 100.0
		if IsLiveFeesEnabled() {
			bids10 := countBidsInWindow(h.rdb, item.ID, fomoWindow10s)
			commissionTier = DetermineTier(tx, sellerID, h.isItemHot(item.ID) || sess.IsHot)
			// Sprint 13: full flywheel bonus (viewer + bid-rate + surge + whale + urgency mult)
			urgencyMult := sess.UrgencyMultiplier
			if urgencyMult <= 0 {
				urgencyMult = 1.0
			}
			commissionDynamicBonus = ComputeFlywheelBonus(FlywheelBonusInputs{
				ViewerCount:       sess.ViewerCount,
				BidsLast10s:       bids10,
				FinalPriceCents:   item.CurrentBidCents,
				UrgencyMultiplier: urgencyMult,
				Urgency:           computeUrgency(bids10),
			})
			// Track surge/whale components separately for audit (unmultiplied)
			commissionSurgeBonus = ComputeSurgeBonus(bids10)
			commissionWhaleBonus = ComputeWhaleBonus(item.CurrentBidCents)
			var sellerCents int64
			commissionCents, sellerCents, commissionRate = CalculateCommission(item.CurrentBidCents, commissionTier, commissionDynamicBonus)
			platformFee = float64(commissionCents) / 100.0
			sellerAmount = float64(sellerCents) / 100.0
		} else {
			platformFee = amountFloat * platformFeePercent / 100.0
			sellerAmount = amountFloat - platformFee
			commissionCents = int64(platformFee * 100)
			commissionRate = platformFeePercent
			commissionTier = TierBase
		}

		// Create order with fee tracking
		orderID = uuid.New()
		orderNow := time.Now()
		if err := tx.Exec(`INSERT INTO orders (id, buyer_id, seller_id, subtotal, platform_fee, payment_fee, total, currency, status, created_at, updated_at, payment_method, reference_type, reference_id)
			VALUES (?, ?, ?, ?, ?, 0, ?, 'USD', 'pending', ?, ?, 'wallet', 'live_auction', ?)`,
			orderID, buyerID, sellerID, amountFloat, platformFee, amountFloat, orderNow, orderNow, refID).Error; err != nil {
			// Release winner's reserved funds — order failed
			_ = wallet.ReleaseReservedFunds(tx, buyerID, item.CurrentBidCents)
			tx.Model(&locked).Update("status", ItemPaymentFailed)
			return fmt.Errorf("order_failed:%s", err.Error())
		}

		// Mark sold
		return tx.Model(&locked).Update("status", ItemSold).Error
	})

	// Broadcast + audit (outside TX — non-critical path)
	if settleErr != nil {
		slog.Error("live-auction: settlement failed",
			"item_id", item.ID, "buyer", buyerID, "error", settleErr)
		h.broadcastItemUpdate(item.SessionID, item, "item_payment_failed")
		freeze.LogAudit(h.db, "live_item_payment_failed", buyerID, item.ID,
			fmt.Sprintf("error=%s amount_cents=%d reserve_released=true", settleErr.Error(), item.CurrentBidCents))
		return
	}

	h.broadcastItemUpdate(item.SessionID, item, "item_settling")
	h.broadcastItemUpdate(item.SessionID, item, "item_sold")

	// Sprint 12+13: record commission + creator revenue split
	if IsLiveFeesEnabled() && commissionCents > 0 {
		baseRate := CommissionRates[commissionTier]

		// Sprint 13: Creator revenue split (streamer ≠ seller → streamer gets 30%)
		streamerShare, _ := ComputeCreatorShare(commissionCents, sessionStreamerID, sellerID)
		if streamerShare > 0 && sessionStreamerID != nil {
			RecordStreamerEarning(h.db, *sessionStreamerID, item.SessionID, orderID, streamerShare)
		}

		// Sprint 16: Creator Economy — full revenue split + earning + milestone
		if IsCreatorsEnabled() && sessionStreamerID != nil && *sessionStreamerID != sellerID {
			split := SplitRevenue(h.db, item.CurrentBidCents, sellerID, resolveCreatorID(h.db, *sessionStreamerID))
			if split.CreatorCommCents > 0 {
				creatorID := resolveCreatorID(h.db, *sessionStreamerID)
				if creatorID != nil {
					if err := RecordCreatorEarning(h.db, *creatorID, sellerID, item.SessionID, orderID, item.ID,
						item.CurrentBidCents, split.CreatorCommPct, split.CreatorCommCents); err != nil {
						slog.Error("creator: failed to record earning", "error", err, "order_id", orderID)
					}
					go CheckCreatorMilestones(h.db, *creatorID)
				}
			}
		}

		RecordCommission(h.db, LiveCommission{
			SessionID:          item.SessionID,
			ItemID:             item.ID,
			OrderID:            orderID,
			SellerID:           sellerID,
			BuyerID:            buyerID,
			FinalPriceCents:    item.CurrentBidCents,
			Tier:               commissionTier,
			BaseRatePercent:    baseRate,
			DynamicBonusPct:    commissionDynamicBonus,
			EffectiveRatePct:   commissionRate,
			CommissionCents:    commissionCents,
			SellerAmountCents:  int64(sellerAmount * 100),
			SurgeBonusPct:      commissionSurgeBonus,
			WhaleBonusPct:      commissionWhaleBonus,
			StreamerID:         sessionStreamerID,
			StreamerShareCents: streamerShare,
		})
	}

	// Sprint 11: toast + funnel win event
	h.BroadcastToast(item.SessionID, "🎉 Item sold!", "🎉")
	{
		uid := buyerID
		iid := item.ID
		RecordConversionEvent(h.db, item.SessionID, &iid, &uid, StageWin, item.CurrentBidCents, "auction")
	}

	// Sprint 15: reward inviter if winner came via a live invite (non-blocking)
	go RewardInviterOnWin(h.db, buyerID, item.SessionID)

	freeze.LogAudit(h.db, "live_auction_settled", buyerID, orderID,
		fmt.Sprintf("item_id=%s seller=%s total=%.2f fee=%.2f seller_amount=%.2f escrow=true",
			item.ID, sellerID, amountFloat, platformFee, sellerAmount))
	slog.Info("live-auction: item settled successfully",
		"item_id", item.ID, "order_id", orderID, "buyer", buyerID,
		"total", amountFloat, "platform_fee", platformFee, "seller_amount", sellerAmount)

	// ── Deposit settlement ──────────────────────────────────────────────────
	if IsAuctionDepositEnabled() && item.RequiresEntryDeposit {
		// Convert winner's deposit to escrow
		if err := ConvertDepositToEscrow(h.db, buyerID, item.ID); err != nil {
			slog.Warn("live-auction: failed to convert winner deposit",
				"buyer", buyerID, "item", item.ID, "error", err)
		}

		// Release all losers' deposits (async)
		go h.releaseLoserDeposits(item.ID, buyerID)
	}

	// Notify winner (async — non-blocking)
	go h.notifyWinner(buyerID, item.ID, orderID)
}

// broadcastLiveEvent publishes a structured LiveEvent to the session channel.
func (h *LiveAuctionHandler) broadcastLiveEvent(sessionID uuid.UUID, evt LiveEvent) {
	if h.rdb == nil {
		return
	}
	// Enrich with viewer count
	evt.ViewerCount = h.getViewerCount(sessionID)

	payload, _ := json.Marshal(evt)
	channel := fmt.Sprintf("auction:%s", sessionID)
	h.rdb.Publish(context.Background(), channel, string(payload))
}

// broadcastItemUpdate is the legacy broadcast (backward-compatible).
func (h *LiveAuctionHandler) broadcastItemUpdate(sessionID uuid.UUID, item *LiveItem, eventType string) {
	if h.rdb == nil {
		return
	}

	// Map legacy event names to structured LiveEvent
	var liveEvt LiveEventType
	switch eventType {
	case "bid_placed":
		liveEvt = EventNewBid
	case "item_ended":
		liveEvt = EventAuctionEnd
	case "item_activated":
		liveEvt = EventItemActivated
	case "item_settling":
		liveEvt = EventItemSettling
	case "item_sold":
		liveEvt = EventItemSold
	case "item_unsold":
		liveEvt = EventItemUnsold
	case "item_payment_failed":
		liveEvt = EventItemPaymentFailed
	case "item_sold_buy_now":
		liveEvt = EventItemSoldBuyNow
	default:
		liveEvt = LiveEventType(eventType)
	}

	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:           liveEvt,
		SessionID:       sessionID.String(),
		ItemID:          item.ID.String(),
		CurrentBidCents: item.CurrentBidCents,
		HighestBidderID: uuidToStr(item.HighestBidderID),
		BidCount:        item.BidCount,
		Status:          string(item.Status),
		EndsAt:          timeToStr(item.EndsAt),
	})
}

// broadcastOutbid sends a targeted outbid event to the previous highest bidder.
func (h *LiveAuctionHandler) broadcastOutbid(sessionID, itemID uuid.UUID, prevBidderID *uuid.UUID, newAmount int64) {
	if h.rdb == nil || prevBidderID == nil {
		return
	}
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:           EventOutbid,
		SessionID:       sessionID.String(),
		ItemID:          itemID.String(),
		CurrentBidCents: newAmount,
		OutbidUserID:    strPtr(prevBidderID.String()),
	})

	// Also publish on a per-user channel for guaranteed delivery
	userChannel := fmt.Sprintf("user:%s:outbid", prevBidderID)
	payload, _ := json.Marshal(LiveEvent{
		Event:           EventOutbid,
		SessionID:       sessionID.String(),
		ItemID:          itemID.String(),
		CurrentBidCents: newAmount,
		OutbidUserID:    strPtr(prevBidderID.String()),
	})
	h.rdb.Publish(context.Background(), userChannel, string(payload))
}

// broadcastSocialProof pushes recent bidder info for social proof display.
func (h *LiveAuctionHandler) broadcastSocialProof(sessionID, itemID uuid.UUID, bidCount int, bidders []RecentBidder) {
	if h.rdb == nil {
		return
	}
	evt := LiveEvent{
		Event:         "social_proof",
		SessionID:     sessionID.String(),
		ItemID:        itemID.String(),
		BidCount:      bidCount,
		RecentBidders: bidders,
		ViewerCount:   h.getViewerCount(sessionID),
	}
	payload, _ := json.Marshal(evt)
	channel := fmt.Sprintf("auction:%s", sessionID)
	h.rdb.Publish(context.Background(), channel, string(payload))
}

// broadcastViewerEvent publishes a viewer join/leave event.
func (h *LiveAuctionHandler) broadcastViewerEvent(sessionID uuid.UUID, evtType LiveEventType, viewerID, displayName string) {
	if h.rdb == nil {
		return
	}
	h.broadcastLiveEvent(sessionID, LiveEvent{
		Event:       evtType,
		SessionID:   sessionID.String(),
		ViewerID:    viewerID,
		DisplayName: displayName,
		ViewerCount: h.getViewerCount(sessionID),
	})
}

// fetchRecentBidders returns the last N unique bidders for social proof.
func (h *LiveAuctionHandler) fetchRecentBidders(itemID uuid.UUID, limit int) []RecentBidder {
	var bids []LiveBid
	h.db.Where("item_id = ?", itemID).
		Order("created_at DESC").
		Limit(limit).
		Find(&bids)

	seen := make(map[string]bool)
	var result []RecentBidder
	for _, b := range bids {
		uid := b.UserID.String()
		if seen[uid] {
			continue
		}
		seen[uid] = true

		// Look up display name from users table (best-effort)
		var name string
		h.db.Raw("SELECT COALESCE(name, 'Bidder') FROM users WHERE id = ?", b.UserID).Scan(&name)

		result = append(result, RecentBidder{
			UserID:      uid,
			DisplayName: name,
			AmountCents: b.BidAmountCents,
			BidAt:       b.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}

// getViewerCount returns the current viewer count for a session.
func (h *LiveAuctionHandler) getViewerCount(sessionID uuid.UUID) int {
	var sess Session
	if err := h.db.Select("viewer_count").Where("id = ?", sessionID).First(&sess).Error; err != nil {
		return 0
	}
	return sess.ViewerCount
}

// strPtr returns a pointer to the string value.
func strPtr(s string) *string { return &s }

// uuidToStr converts a *uuid.UUID to *string.
func uuidToStr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

// timeToStr converts a *time.Time to *string (RFC3339).
func timeToStr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

// ════════════════════════════════════════════════════════════════════════════
// Auction Closer Scheduler (Sprint 9.5)
// Background goroutine that catches expired items missed by timers.
// ════════════════════════════════════════════════════════════════════════════

func (h *LiveAuctionHandler) StartAuctionCloser(ctx context.Context) {
	slog.Info("live-auction: auction closer scheduler started")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("live-auction: auction closer scheduler stopped")
			return
		case <-ticker.C:
			// Close expired active items
			var expiredItems []LiveItem
			h.db.Where("status = ? AND ends_at <= ?", ItemActive, time.Now()).Find(&expiredItems)
			for _, item := range expiredItems {
				go h.endItem(item.ID)
			}

			// Fix 3: Recover stuck SETTLING items (timeout after 30s)
			var stuckItems []LiveItem
			cutoff := time.Now().Add(-settlementTimeout)
			h.db.Where("status = ? AND settling_started_at < ?", ItemSettling, cutoff).Find(&stuckItems)
			for _, item := range stuckItems {
				if item.SettleRetries >= maxSettleRetries {
					slog.Error("live-auction: max settle retries reached, marking payment_failed",
						"item_id", item.ID, "retries", item.SettleRetries)
					h.db.Model(&item).Update("status", ItemPaymentFailed)
					h.broadcastItemUpdate(item.SessionID, &item, "item_payment_failed")
					// Release winner's reserved funds — stuck in pending
					if item.HighestBidderID != nil && item.CurrentBidCents > 0 {
						if err := wallet.ReleaseReservedFunds(h.db, *item.HighestBidderID, item.CurrentBidCents); err != nil {
							slog.Error("live-auction: failed to release winner reserve on stuck settlement",
								"buyer", *item.HighestBidderID, "amount", item.CurrentBidCents, "error", err)
						}
					}
				} else {
					slog.Warn("live-auction: retrying stuck settlement",
						"item_id", item.ID, "retry", item.SettleRetries+1)
					go h.finalizeItem(&item)
				}
			}
		}
	}
}

// ── Winner Notification ──────────────────────────────────────────────────────

// NotificationService is an interface satisfied by *notifications.Service.
// Using an interface avoids circular imports.
type NotificationService interface {
	Notify(input notifications.NotifyInput)
}

var globalNotifSvc NotificationService

// SetNotificationService wires the notification service into this package.
// Called once from main.go after all routes are registered.
func SetNotificationService(svc NotificationService) {
	globalNotifSvc = svc
}

// notifyWinner sends an auction_won notification to the winning bidder.
func (h *LiveAuctionHandler) notifyWinner(winnerID uuid.UUID, itemID uuid.UUID, orderID uuid.UUID) {
	if globalNotifSvc == nil {
		slog.Info("live-auction: notification service not wired, skipping winner notification",
			"winner", winnerID, "item", itemID)
		return
	}

	go globalNotifSvc.Notify(notifications.NotifyInput{
		UserID: winnerID,
		Type:   notifications.TypeAuctionWon,
		Title:  "You won the auction!",
		Body:   fmt.Sprintf("Congratulations! You won the live auction. Order #%s has been created.", orderID.String()[:8]),
		Data: map[string]string{
			"item_id":  itemID.String(),
			"order_id": orderID.String(),
			"source":   "live_auction",
		},
	})
}
