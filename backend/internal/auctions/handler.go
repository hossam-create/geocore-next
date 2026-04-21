package auctions

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/moderation"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Handler struct {
	db           *gorm.DB
	rdb          *redis.Client
	dutchManager *DutchAuctionManager
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db: db, rdb: rdb}
}

// SetDutchManager sets the DutchAuctionManager on the handler.
func (h *Handler) SetDutchManager(m *DutchAuctionManager) {
	h.dutchManager = m
}

func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage := 20
	status := c.DefaultQuery("status", "active")
	auctionType := c.Query("type") // standard, dutch, reverse, sealed

	var auctions []Auction
	var total int64
	q := h.db.Model(&Auction{})

	if auctionType != "" {
		q = q.Where("type = ?", auctionType)
	}

	switch status {
	case "ended":
		q = q.Where("status = ?", StatusEnded)
	case "upcoming":
		q = q.Where("status = ? AND starts_at > ?", StatusScheduled, time.Now())
	case "ending_soon":
		soon := time.Now().Add(time.Hour)
		q = q.Where("status = ? AND ends_at > ? AND ends_at <= ?", StatusActive, time.Now(), soon)
	default:
		q = q.Where("status = ? AND ends_at > ?", StatusActive, time.Now())
	}
	q.Count(&total)
	q.Preload("Bids").Offset((page - 1) * perPage).Limit(perPage).
		Order("ends_at ASC").Find(&auctions)
	response.OKMeta(c, auctions, gin.H{"total": total, "page": page})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}
	var auction Auction
	if err := h.db.Preload("Bids", func(db *gorm.DB) *gorm.DB {
		return db.Order("amount DESC").Limit(20)
	}).First(&auction, "id = ?", id).Error; err != nil {
		response.NotFound(c, "Auction")
		return
	}

	// For Dutch auctions, include current price
	if auction.Type == AuctionTypeDutch {
		currentPrice := auction.GetCurrentDutchPrice()
		response.OK(c, gin.H{
			"auction":       auction,
			"current_price": currentPrice,
		})
		return
	}

	response.OK(c, auction)
}

func (h *Handler) Create(c *gin.Context) {
	sellerID, _ := uuid.Parse(c.MustGet("user_id").(string))
	var req struct {
		ListingID         string      `json:"listing_id" binding:"required"`
		Type              AuctionType `json:"type"`
		StartPrice        float64     `json:"start_price" binding:"required,min=0"`
		ReservePrice      *float64    `json:"reserve_price"`
		BuyNowPrice       *float64    `json:"buy_now_price"`
		Currency          string      `json:"currency"`
		DurationHrs       int         `json:"duration_hours" binding:"required,min=1,max=720"`
		AntiSnipeEnabled  *bool       `json:"anti_snipe_enabled"`
		DutchStartPrice   *float64    `json:"dutch_start_price"`
		DutchEndPrice     *float64    `json:"dutch_end_price"`
		DutchDropInterval *int        `json:"dutch_drop_interval"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	listingID, _ := uuid.Parse(req.ListingID)
	now := time.Now()
	var listingText struct {
		Title       string
		Description string
	}
	h.db.Table("listings").Select("title, description").Where("id = ?", listingID).First(&listingText)
	if blocked, reason := moderation.CheckContent(listingText.Title, listingText.Description); blocked {
		response.BadRequest(c, reason)
		return
	}

	auctionType := AuctionTypeStandard
	if req.Type != "" {
		auctionType = req.Type
	}

	antiSnipe := true
	if req.AntiSnipeEnabled != nil {
		antiSnipe = *req.AntiSnipeEnabled
	}

	auction := Auction{
		ID:               uuid.New(),
		ListingID:        listingID,
		SellerID:         sellerID,
		Type:             auctionType,
		StartPrice:       req.StartPrice,
		ReservePrice:     req.ReservePrice,
		BuyNowPrice:      req.BuyNowPrice,
		CurrentBid:       0,
		Currency:         defaultStr(req.Currency, "USD"),
		Status:           StatusActive,
		StartsAt:         now,
		EndsAt:           now.Add(time.Duration(req.DurationHrs) * time.Hour),
		AntiSnipeEnabled: antiSnipe,
		ProxyBidEnabled:  true,
	}

	// Dutch auction specific fields
	if auctionType == AuctionTypeDutch {
		if req.DutchStartPrice == nil || req.DutchEndPrice == nil {
			response.BadRequest(c, "Dutch auction requires dutch_start_price and dutch_end_price")
			return
		}
		auction.DutchStartPrice = req.DutchStartPrice
		auction.DutchEndPrice = req.DutchEndPrice
		auction.DutchDropInterval = req.DutchDropInterval
	}

	if err := h.db.Create(&auction).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Sprint 8.5: Audit log for auction_started
	freeze.LogAudit(h.db, "auction_started", sellerID, auction.ID, fmt.Sprintf("type=%s start_price=%.2f auction_id=%s", auctionType, req.StartPrice, auction.ID))

	// Start dutch ticker if applicable
	if auctionType == AuctionTypeDutch && h.dutchManager != nil {
		h.dutchManager.StartTicker(auction.ID)
	}

	response.Created(c, auction)
}

func (h *Handler) Update(c *gin.Context) {
	sellerID, err := uuid.Parse(c.MustGet("user_id").(string))
	if err != nil {
		response.Unauthorized(c)
		return
	}
	auctionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var auction Auction
	if err := h.db.First(&auction, "id = ? AND seller_id = ?", auctionID, sellerID).Error; err != nil {
		response.NotFound(c, "Auction")
		return
	}

	var req struct {
		ReservePrice      *float64 `json:"reserve_price"`
		BuyNowPrice       *float64 `json:"buy_now_price"`
		Currency          string   `json:"currency"`
		AntiSnipeEnabled  *bool    `json:"anti_snipe_enabled"`
		DutchDropInterval *int     `json:"dutch_drop_interval"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var listingText struct {
		Title       string
		Description string
	}
	if err := h.db.Table("listings").Select("title, description").Where("id = ?", auction.ListingID).First(&listingText).Error; err == nil {
		if blocked, reason := moderation.CheckContent(listingText.Title, listingText.Description); blocked {
			response.BadRequest(c, reason)
			return
		}
	}

	updates := map[string]any{}
	if req.ReservePrice != nil {
		updates["reserve_price"] = req.ReservePrice
	}
	if req.BuyNowPrice != nil {
		updates["buy_now_price"] = req.BuyNowPrice
	}
	if req.Currency != "" {
		updates["currency"] = req.Currency
	}
	if req.AntiSnipeEnabled != nil {
		updates["anti_snipe_enabled"] = *req.AntiSnipeEnabled
	}
	if req.DutchDropInterval != nil {
		updates["dutch_drop_interval"] = *req.DutchDropInterval
	}

	if len(updates) == 0 {
		response.OK(c, auction)
		return
	}
	if err := h.db.Model(&auction).Updates(updates).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, auction)
}

func (h *Handler) PlaceBid(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	auctionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	// Sprint 8.5: Block frozen users from bidding
	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	var req struct {
		Amount         float64  `json:"amount" binding:"required"`
		IsAuto         bool     `json:"is_auto"`
		MaxAmount      *float64 `json:"max_amount"`
		IdempotencyKey *string  `json:"idempotency_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Fast pre-check: reject if seller is bidding (no lock needed — self-reference is stable)
	var sellerCheck struct{ SellerID uuid.UUID }
	if h.db.Table("auctions").Select("seller_id").Where("id = ?", auctionID).Scan(&sellerCheck).Error == nil {
		if sellerCheck.SellerID == userID {
			response.BadRequest(c, "Cannot bid on your own auction")
			return
		}
	}

	// ── Idempotency check (F6): return cached bid if same key already submitted ──
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		var existing Bid
		if h.db.Where("auction_id = ? AND user_id = ? AND idempotency_key = ?",
			auctionID, userID, *req.IdempotencyKey).First(&existing).Error == nil {
			response.Created(c, gin.H{"bid": existing, "idempotent": true})
			return
		}
	}

	var bid Bid
	var auction Auction
	var extended bool
	var prevLeaderID *uuid.UUID

	// ── ALL validation + mutation inside one transaction with FOR UPDATE ─────────
	// FOR UPDATE on the auction row serialises concurrent bids — prevents TOCTOU
	// where two goroutines both read the same current_bid and both pass the check.
	dbErr := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&auction, "id = ? AND status = ?", auctionID, StatusActive).Error; err != nil {
			return fmt.Errorf("auction_not_found")
		}
		if time.Now().After(auction.EndsAt) {
			return fmt.Errorf("auction_ended")
		}

		// Validate amount per auction type — all inside the lock
		switch auction.Type {
		case AuctionTypeDutch:
			currentPrice := auction.GetCurrentDutchPrice()
			if req.Amount < currentPrice {
				return fmt.Errorf("bid_too_low:%.2f", currentPrice)
			}
		case AuctionTypeReverse:
			if auction.BidCount > 0 && req.Amount >= auction.CurrentBid {
				return fmt.Errorf("bid_not_lower:%.2f", auction.CurrentBid)
			}
		default:
			minBid := auction.CurrentBid
			if auction.BidCount == 0 {
				minBid = auction.StartPrice - 0.01
			}
			if req.Amount <= minBid {
				return fmt.Errorf("bid_too_low:%.2f", minBid)
			}
		}

		// Find previous leader while still inside the lock
		var prevBid Bid
		if auction.BidCount > 0 {
			if tx.Where("auction_id = ? AND user_id != ?", auctionID, userID).
				Order("amount DESC").First(&prevBid).Error == nil {
				prevLeaderID = &prevBid.UserID
			}
		}

		// Anti-sniping: extend if bid placed in last AntiSnipeWindow
		if auction.AntiSnipeEnabled && auction.ExtensionCount < MaxExtensions {
			if time.Until(auction.EndsAt) <= AntiSnipeWindow {
				auction.EndsAt = auction.EndsAt.Add(AntiSnipeExtension)
				auction.ExtensionCount++
				extended = true
			}
		}

		bid = Bid{
			ID: uuid.New(), AuctionID: auctionID, UserID: userID,
			Amount: req.Amount, IsAuto: req.IsAuto, MaxAmount: req.MaxAmount,
			IdempotencyKey: req.IdempotencyKey, PlacedAt: time.Now(),
		}
		if err := tx.Create(&bid).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{
			"current_bid": req.Amount,
			"bid_count":   gorm.Expr("bid_count + 1"),
		}
		if extended {
			updates["ends_at"] = auction.EndsAt
			updates["extension_count"] = auction.ExtensionCount
		}
		return tx.Model(&auction).Updates(updates).Error
	})

	if dbErr != nil {
		switch dbErr.Error() {
		case "auction_not_found":
			response.NotFound(c, "Auction")
		case "auction_ended":
			response.BadRequest(c, "Auction has ended")
		default:
			// bid_too_low or bid_not_lower carry the threshold in the sentinel
			response.BadRequest(c, dbErr.Error())
		}
		return
	}

	// ── Post-commit: proxy bids + pub/sub + notifications (non-critical) ───────
	go h.processProxyBids(&auction, userID, req.Amount)
	h.rdb.Publish(c, fmt.Sprintf("auction:%s", auctionID),
		fmt.Sprintf(`{"bid": %.2f, "user": "%s", "extended": %v, "ends_at": "%s"}`,
			req.Amount, userID, extended, auction.EndsAt.Format(time.RFC3339)))
	go notifyNewBid(&auction, userID, prevLeaderID, req.Amount)

	// Sprint 8.5: Audit log for bid_placed
	freeze.LogAudit(h.db, "bid_placed", userID, auctionID, fmt.Sprintf("amount=%.2f auction_id=%s", req.Amount, auctionID))

	result := gin.H{"bid": bid}
	if extended {
		result["extended"] = true
		result["new_end_time"] = auction.EndsAt
	}
	response.Created(c, result)
}

// BuyNow allows instant purchase at buy_now_price
func (h *Handler) BuyNow(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	auctionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	// Sprint 8.5: Block frozen users from buying
	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	var auction Auction
	var buyNowPrice float64

	// ── FOR UPDATE serialises concurrent BuyNow calls on the same auction ──
	// Without the lock two goroutines both see StatusActive and both mark SOLD.
	dbErr := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&auction, "id = ? AND status = ?", auctionID, StatusActive).Error; err != nil {
			return fmt.Errorf("not_found")
		}
		if auction.BuyNowPrice == nil {
			return fmt.Errorf("no_buy_now")
		}
		if auction.SellerID == userID {
			return fmt.Errorf("own_item")
		}
		buyNowPrice = *auction.BuyNowPrice
		return tx.Model(&auction).Updates(map[string]interface{}{
			"status":      StatusSold,
			"winner_id":   userID,
			"current_bid": buyNowPrice,
			"ends_at":     time.Now(),
		}).Error
	})

	if dbErr != nil {
		switch dbErr.Error() {
		case "not_found":
			response.NotFound(c, "Auction")
		case "no_buy_now":
			response.BadRequest(c, "Buy Now not available for this auction")
		case "own_item":
			response.BadRequest(c, "Cannot buy your own item")
		default:
			response.InternalError(c, dbErr)
		}
		return
	}

	h.rdb.Publish(c, fmt.Sprintf("auction:%s", auctionID),
		fmt.Sprintf(`{"event": "buy_now", "winner": "%s", "price": %.2f}`, userID, buyNowPrice))
	go notifyAuctionWon(userID, auction.SellerID, auctionID.String(), buyNowPrice, auction.Currency)

	// Sprint 8.5: Audit log for auction_won (BuyNow)
	freeze.LogAudit(h.db, "auction_won", userID, auctionID, fmt.Sprintf("method=buy_now price=%.2f auction_id=%s", buyNowPrice, auctionID))

	response.OK(c, gin.H{"message": "Purchase successful!", "price": buyNowPrice})
}

// SetProxyBid sets up automatic bidding
func (h *Handler) SetProxyBid(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	auctionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	// Sprint 8.5: Block frozen users from proxy bidding
	if freeze.IsUserFrozen(h.db, userID) {
		response.Forbidden(c)
		return
	}

	var req struct {
		MaxAmount float64 `json:"max_amount" binding:"required,min=1"`
		Increment float64 `json:"increment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var auction Auction
	if err := h.db.First(&auction, "id = ? AND status = ?", auctionID, StatusActive).Error; err != nil {
		response.NotFound(c, "Auction")
		return
	}

	if !auction.ProxyBidEnabled {
		response.BadRequest(c, "Proxy bidding not enabled for this auction")
		return
	}

	increment := req.Increment
	if increment <= 0 {
		increment = 1.0
	}

	// Upsert proxy bid
	proxyBid := ProxyBid{
		AuctionID: auctionID,
		UserID:    userID,
		MaxAmount: req.MaxAmount,
		Increment: increment,
		IsActive:  true,
	}

	h.db.Where("auction_id = ? AND user_id = ?", auctionID, userID).
		Assign(proxyBid).FirstOrCreate(&proxyBid)

	// Immediately place a bid if needed
	go h.processProxyBids(&auction, uuid.Nil, auction.CurrentBid)

	response.OK(c, proxyBid)
}

func (h *Handler) GetBids(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	var bids []Bid
	h.db.Where("auction_id = ?", id).Order("amount DESC").Limit(50).Find(&bids)
	response.OK(c, bids)
}

// completeDutchAuction ends a Dutch auction with the winner
func (h *Handler) completeDutchAuction(auction *Auction, winnerID uuid.UUID, price float64) {
	// Stop the dutch ticker first
	if h.dutchManager != nil {
		h.dutchManager.StopTicker(auction.ID)
	}

	h.db.Model(auction).Updates(map[string]interface{}{
		"status":      StatusSold,
		"winner_id":   winnerID,
		"current_bid": price,
		"ends_at":     time.Now(),
	})

	// Broadcast sold event via Redis
	h.rdb.Publish(context.Background(), fmt.Sprintf("auction:%s", auction.ID),
		fmt.Sprintf(`{"type":"dutch_sold","auction_id":"%s","winner":"%s","price":%.2f}`,
			auction.ID, winnerID, price))

	go notifyAuctionWon(winnerID, auction.SellerID, auction.ID.String(), price, auction.Currency)

	// Sprint 8.5: Audit log for auction_won (Dutch)
	freeze.LogAudit(h.db, "auction_won", winnerID, auction.ID, fmt.Sprintf("method=dutch price=%.2f auction_id=%s", price, auction.ID))
}

// processProxyBids handles automatic bidding
func (h *Handler) processProxyBids(auction *Auction, excludeUserID uuid.UUID, currentBid float64) {
	var proxyBids []ProxyBid
	h.db.Where("auction_id = ? AND is_active = ? AND user_id != ? AND max_amount > ?",
		auction.ID, true, excludeUserID, currentBid).
		Order("max_amount DESC").Find(&proxyBids)

	if len(proxyBids) == 0 {
		return
	}

	// Find the highest proxy bidder
	topProxy := proxyBids[0]
	newBid := currentBid + topProxy.Increment

	if newBid > topProxy.MaxAmount {
		newBid = topProxy.MaxAmount
	}

	// Place automatic bid
	bid := Bid{
		ID:        uuid.New(),
		AuctionID: auction.ID,
		UserID:    topProxy.UserID,
		Amount:    newBid,
		IsAuto:    true,
		PlacedAt:  time.Now(),
	}
	h.db.Create(&bid)
	h.db.Model(auction).Updates(map[string]interface{}{
		"current_bid": newBid,
		"bid_count":   gorm.Expr("bid_count + 1"),
	})

	// Deactivate if max reached
	if newBid >= topProxy.MaxAmount {
		h.db.Model(&topProxy).Update("is_active", false)
	}
}

func defaultStr(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
