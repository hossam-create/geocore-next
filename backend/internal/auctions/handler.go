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
	var auctions []Auction
	var total int64
	q := h.db.Model(&Auction{}).Where("status = ? AND ends_at > ?", "active", time.Now())
	q.Count(&total)
	q.Preload("Bids").Offset((page-1)*perPage).Limit(perPage).
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
	response.OK(c, auction)
}

func (h *Handler) Create(c *gin.Context) {
	sellerID, _ := uuid.Parse(c.MustGet("user_id").(string))
	var req struct {
		ListingID    string   `json:"listing_id" binding:"required"`
		StartPrice   float64  `json:"start_price" binding:"required,min=0"`
		ReservePrice *float64 `json:"reserve_price"`
		BuyNowPrice  *float64 `json:"buy_now_price"`
		Currency     string   `json:"currency"`
		DurationHrs  int      `json:"duration_hours" binding:"required,min=1,max=720"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	listingID, _ := uuid.Parse(req.ListingID)
	now := time.Now()
	auction := Auction{
		ID:           uuid.New(),
		ListingID:    listingID,
		SellerID:     sellerID,
		StartPrice:   req.StartPrice,
		ReservePrice: req.ReservePrice,
		BuyNowPrice:  req.BuyNowPrice,
		CurrentBid:   0,
		Currency:     defaultStr(req.Currency, "USD"),
		Status:       "active",
		StartsAt:     now,
		EndsAt:       now.Add(time.Duration(req.DurationHrs) * time.Hour),
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
		Amount    float64  `json:"amount" binding:"required"`
		IsAuto    bool     `json:"is_auto"`
		MaxAmount *float64 `json:"max_amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var auction Auction
	if err := h.db.First(&auction, "id = ? AND status = ?", auctionID, "active").Error; err != nil {
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

	minBid := auction.CurrentBid
	if auction.BidCount == 0 {
		minBid = auction.StartPrice - 0.01
	}
	if req.Amount <= minBid {
		response.BadRequest(c, fmt.Sprintf("Bid must be higher than %.2f", minBid))
		return
	}

	bid := Bid{
		ID:        uuid.New(),
		AuctionID: auctionID,
		UserID:    userID,
		Amount:    req.Amount,
		IsAuto:    req.IsAuto,
		MaxAmount: req.MaxAmount,
		PlacedAt:  time.Now(),
	}

	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Create(&bid)
		tx.Model(&auction).Updates(map[string]interface{}{
			"current_bid": req.Amount,
			"bid_count":   gorm.Expr("bid_count + 1"),
		})
		return nil
	})

	// Broadcast via Redis Pub/Sub
	h.rdb.Publish(c, fmt.Sprintf("auction:%s", auctionID), fmt.Sprintf(`{"bid": %.2f, "user": "%s"}`, req.Amount, userID))

	response.Created(c, bid)
}

func (h *Handler) GetBids(c *gin.Context) {
	id, _ := uuid.Parse(c.Param("id"))
	var bids []Bid
	h.db.Where("auction_id = ?", id).Order("amount DESC").Limit(50).Find(&bids)
	response.OK(c, bids)
}

func defaultStr(s, d string) string {
	if s == "" { return d }
	return s
}
