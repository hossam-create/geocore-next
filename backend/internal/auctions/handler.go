package auctions

import (
	"fmt"
	"strconv"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db, rdb}
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
	response.Created(c, auction)
}

func (h *Handler) PlaceBid(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	auctionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
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

	var auction Auction
	if err := h.db.First(&auction, "id = ? AND status = ?", auctionID, StatusActive).Error; err != nil {
		response.NotFound(c, "Auction")
		return
	}

	if time.Now().After(auction.EndsAt) {
		response.BadRequest(c, "Auction has ended")
		return
	}

	if auction.SellerID == userID {
		response.BadRequest(c, "Cannot bid on your own auction")
		return
	}

	// Handle different auction types
	switch auction.Type {
	case AuctionTypeDutch:
		// Dutch auction: first bid at current price wins
		currentPrice := auction.GetCurrentDutchPrice()
		if req.Amount < currentPrice {
			response.BadRequest(c, fmt.Sprintf("Bid must be at least %.2f (current Dutch price)", currentPrice))
			return
		}
		// Dutch auction ends immediately on first valid bid
		h.completeDutchAuction(&auction, userID, currentPrice)
		response.OK(c, gin.H{"message": "You won the Dutch auction!", "price": currentPrice})
		return

	case AuctionTypeReverse:
		// Reverse auction: lower bids are better
		if auction.BidCount > 0 && req.Amount >= auction.CurrentBid {
			response.BadRequest(c, fmt.Sprintf("Bid must be lower than %.2f", auction.CurrentBid))
			return
		}

	default: // Standard auction
		minBid := auction.CurrentBid
		if auction.BidCount == 0 {
			minBid = auction.StartPrice - 0.01
		}
		if req.Amount <= minBid {
			response.BadRequest(c, fmt.Sprintf("Bid must be higher than %.2f", minBid))
			return
		}
	}

	bid := Bid{
		ID:             uuid.New(),
		AuctionID:      auctionID,
		UserID:         userID,
		Amount:         req.Amount,
		IsAuto:         req.IsAuto,
		MaxAmount:      req.MaxAmount,
		IdempotencyKey: req.IdempotencyKey,
		PlacedAt:       time.Now(),
	}

	// Find previous leader to notify
	var prevBid Bid
	var prevLeaderID *uuid.UUID
	if auction.BidCount > 0 {
		if h.db.Where("auction_id = ? AND user_id != ?", auctionID, userID).
			Order("amount DESC").First(&prevBid).Error == nil {
			prevLeaderID = &prevBid.UserID
		}
	}

	// Anti-sniping: extend auction if bid in last 2 minutes
	var extended bool
	if auction.AntiSnipeEnabled && auction.ExtensionCount < MaxExtensions {
		timeRemaining := time.Until(auction.EndsAt)
		if timeRemaining <= AntiSnipeWindow {
			auction.EndsAt = auction.EndsAt.Add(AntiSnipeExtension)
			auction.ExtensionCount++
			extended = true
		}
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Create(&bid)
		updates := map[string]interface{}{
			"current_bid": req.Amount,
			"bid_count":   gorm.Expr("bid_count + 1"),
		}
		if extended {
			updates["ends_at"] = auction.EndsAt
			updates["extension_count"] = auction.ExtensionCount
		}
		tx.Model(&auction).Updates(updates)
		return nil
	})

	// Process proxy bids from other users
	go h.processProxyBids(&auction, userID, req.Amount)

	// Broadcast via Redis Pub/Sub
	h.rdb.Publish(c, fmt.Sprintf("auction:%s", auctionID),
		fmt.Sprintf(`{"bid": %.2f, "user": "%s", "extended": %v, "ends_at": "%s"}`,
			req.Amount, userID, extended, auction.EndsAt.Format(time.RFC3339)))

	// Send notifications async
	go notifyNewBid(&auction, userID, prevLeaderID, req.Amount)

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

	var auction Auction
	if err := h.db.First(&auction, "id = ? AND status = ?", auctionID, StatusActive).Error; err != nil {
		response.NotFound(c, "Auction")
		return
	}

	if auction.BuyNowPrice == nil {
		response.BadRequest(c, "Buy Now not available for this auction")
		return
	}

	if auction.SellerID == userID {
		response.BadRequest(c, "Cannot buy your own item")
		return
	}

	// Complete the auction
	now := time.Now()
	h.db.Model(&auction).Updates(map[string]interface{}{
		"status":      StatusSold,
		"winner_id":   userID,
		"current_bid": *auction.BuyNowPrice,
		"ends_at":     now,
	})

	// Broadcast auction ended
	h.rdb.Publish(c, fmt.Sprintf("auction:%s", auctionID),
		fmt.Sprintf(`{"event": "buy_now", "winner": "%s", "price": %.2f}`, userID, *auction.BuyNowPrice))

	// Notify seller
	go notifyAuctionWon(userID, auction.SellerID, auctionID.String(), *auction.BuyNowPrice, auction.Currency)

	response.OK(c, gin.H{
		"message": "Purchase successful!",
		"price":   *auction.BuyNowPrice,
	})
}

// SetProxyBid sets up automatic bidding
func (h *Handler) SetProxyBid(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	auctionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
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
	h.db.Model(auction).Updates(map[string]interface{}{
		"status":      StatusSold,
		"winner_id":   winnerID,
		"current_bid": price,
		"ends_at":     time.Now(),
	})
	go notifyAuctionWon(winnerID, auction.SellerID, auction.ID.String(), price, auction.Currency)
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
