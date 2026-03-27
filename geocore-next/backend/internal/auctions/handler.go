package auctions

import (
        "context"
        "fmt"
        "strconv"
        "time"

        "github.com/geocore-next/backend/internal/fraud"
        "github.com/geocore-next/backend/pkg/response"
        "github.com/geocore-next/backend/pkg/util"
        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "github.com/redis/go-redis/v9"
        "gorm.io/gorm"
        "gorm.io/gorm/clause"
)

const (
        bidIncrement      = 10.0
        antiSnipWindow    = 5 * time.Minute
        antiSnipExtension = 5 * time.Minute
        maxExtensions     = 3
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
        perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
        if page < 1 {
                page = 1
        }
        if perPage < 1 || perPage > 100 {
                perPage = 20
        }
        status := c.DefaultQuery("status", "active")

        var auctions []Auction
        var total int64
        q := h.db.Model(&Auction{})
        switch status {
        case "ended":
                q = q.Where("status = ?", "ended")
        case "upcoming":
                q = q.Where("status = ? AND starts_at > ?", "active", time.Now())
        case "ending_soon":
                soon := time.Now().Add(time.Hour)
                q = q.Where("status = ? AND ends_at > ? AND ends_at <= ?", "active", time.Now(), soon)
        default: // "active" or any unrecognised value → live auctions
                q = q.Where("status = ? AND ends_at > ?", "active", time.Now())
        }
        q.Count(&total)
        q.Preload("Bids").Offset((page-1)*perPage).Limit(perPage).
                Order("ends_at ASC").Find(&auctions)
        pages := int64(1)
        if perPage > 0 {
                pages = (total + int64(perPage) - 1) / int64(perPage)
        }
        response.OKMeta(c, auctions, gin.H{"total": total, "page": page, "per_page": perPage, "pages": pages})
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
                Currency:     util.DefaultStr(req.Currency, "USD"),
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

        var bid Bid
        var prevLeaderID *uuid.UUID
        var finalAuction Auction

        txErr := h.db.Transaction(func(tx *gorm.DB) error {
                // SELECT FOR UPDATE serialises concurrent bids on the same auction row
                var auction Auction
                if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
                        First(&auction, "id = ? AND status = ?", auctionID, "active").Error; err != nil {
                        return fmt.Errorf("auction not found")
                }

                if time.Now().After(auction.EndsAt) {
                        return fmt.Errorf("auction has ended")
                }

                if auction.SellerID == userID {
                        return fmt.Errorf("cannot bid on your own auction")
                }

                minBid := auction.CurrentBid
                if auction.BidCount == 0 {
                        minBid = auction.StartPrice - 0.01
                }
                if req.Amount <= minBid {
                        return fmt.Errorf("bid must be higher than %.2f", minBid)
                }

                // Find previous leader to notify them they were outbid
                if auction.BidCount > 0 {
                        var prevBid Bid
                        if tx.Where("auction_id = ? AND user_id != ?", auctionID, userID).
                                Order("amount DESC").First(&prevBid).Error == nil {
                                prevLeaderID = &prevBid.UserID
                        }
                }

                bid = Bid{
                        ID:        uuid.New(),
                        AuctionID: auctionID,
                        UserID:    userID,
                        Amount:    req.Amount,
                        IsAuto:    req.IsAuto,
                        MaxAmount: req.MaxAmount,
                        PlacedAt:  time.Now(),
                }

                updates := map[string]interface{}{
                        "current_bid": req.Amount,
                        "bid_count":   gorm.Expr("bid_count + 1"),
                }

                // Anti-sniping: extend ends_at if bid lands within the last 5 minutes
                timeRemaining := auction.EndsAt.Sub(time.Now())
                if timeRemaining < antiSnipWindow && auction.ExtensionCount < maxExtensions {
                        auction.EndsAt = auction.EndsAt.Add(antiSnipExtension)
                        auction.ExtensionCount++
                        updates["ends_at"] = auction.EndsAt
                        updates["extension_count"] = auction.ExtensionCount
                }

                if err := tx.Create(&bid).Error; err != nil {
                        return err
                }
                if err := tx.Model(&auction).Updates(updates).Error; err != nil {
                        return err
                }

                finalAuction = auction
                go notifyNewBid(&auction, userID, prevLeaderID, req.Amount)
                return nil
        })

        if txErr != nil {
                msg := txErr.Error()
                switch msg {
                case "auction not found":
                        response.NotFound(c, "Auction")
                default:
                        response.BadRequest(c, msg)
                }
                return
        }

        // Broadcast via Redis Pub/Sub
        h.rdb.Publish(c, fmt.Sprintf("auction:%s", auctionID),
                fmt.Sprintf(`{"bid": %.2f, "user": "%s", "ends_at": "%s"}`,
                        req.Amount, userID, finalAuction.EndsAt.UTC().Format(time.RFC3339)))

        // Auto-bid proxy: trigger counter-bids from other bidders who set a max_amount
        go h.runAutoBidProxy(auctionID, userID, req.Amount)

        // Evaluate fraud risk asynchronously — does not block the response.
        go fraud.New(h.db, h.rdb).Evaluate(context.Background(), userID)

        response.Created(c, bid)
}

// runAutoBidProxy finds other bidders with a max_amount that still exceeds the
// current bid and places the smallest winning counter-bid on their behalf.
// It serializes via SELECT FOR UPDATE to prevent concurrent race conditions.
func (h *Handler) runAutoBidProxy(auctionID uuid.UUID, lastBidderID uuid.UUID, currentBid float64) {
        for {
                var placed bool

                err := h.db.Transaction(func(tx *gorm.DB) error {
                        // Lock the auction row
                        var auction Auction
                        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
                                First(&auction, "id = ? AND status = ?", auctionID, "active").Error; err != nil {
                                return err
                        }

                        if time.Now().After(auction.EndsAt) {
                                return nil
                        }

                        // Find the auto-bidder with the highest max_amount that exceeds the current bid,
                        // excluding the user who just placed the triggering bid.
                        // On tied max_amount, the earliest qualifying auto-bid wins (first-bidder-wins rule).
                        var autoBid Bid
                        if err := tx.Where(
                                "auction_id = ? AND user_id != ? AND max_amount > ? AND is_auto = true",
                                auctionID, lastBidderID, auction.CurrentBid,
                        ).Order("max_amount DESC, placed_at ASC").First(&autoBid).Error; err != nil {
                                // No qualifying auto-bidder found
                                return nil
                        }

                        counterAmount := auction.CurrentBid + bidIncrement
                        if counterAmount > *autoBid.MaxAmount {
                                counterAmount = *autoBid.MaxAmount
                        }
                        if counterAmount <= auction.CurrentBid {
                                return nil
                        }

                        newBid := Bid{
                                ID:        uuid.New(),
                                AuctionID: auctionID,
                                UserID:    autoBid.UserID,
                                Amount:    counterAmount,
                                IsAuto:    true,
                                MaxAmount: autoBid.MaxAmount,
                                PlacedAt:  time.Now(),
                        }

                        updates := map[string]interface{}{
                                "current_bid": counterAmount,
                                "bid_count":   gorm.Expr("bid_count + 1"),
                        }

                        // Anti-sniping check for auto-bid as well
                        timeRemaining := auction.EndsAt.Sub(time.Now())
                        if timeRemaining < antiSnipWindow && auction.ExtensionCount < maxExtensions {
                                auction.EndsAt = auction.EndsAt.Add(antiSnipExtension)
                                auction.ExtensionCount++
                                updates["ends_at"] = auction.EndsAt
                                updates["extension_count"] = auction.ExtensionCount
                        }

                        if err := tx.Create(&newBid).Error; err != nil {
                                return err
                        }
                        if err := tx.Model(&auction).Updates(updates).Error; err != nil {
                                return err
                        }

                        // Notify the user who was just outbid by the auto-bid
                        go notifyNewBid(&auction, autoBid.UserID, &lastBidderID, counterAmount)

                        // Broadcast new auto-bid via Redis
                        h.rdb.Publish(context.Background(), fmt.Sprintf("auction:%s", auctionID),
                                fmt.Sprintf(`{"bid": %.2f, "user": "%s", "is_auto": true, "ends_at": "%s"}`,
                                        counterAmount, autoBid.UserID, auction.EndsAt.UTC().Format(time.RFC3339)))

                        placed = true
                        lastBidderID = autoBid.UserID
                        currentBid = counterAmount
                        return nil
                })

                if err != nil || !placed {
                        break
                }
        }
}

func (h *Handler) GetBids(c *gin.Context) {
        id, _ := uuid.Parse(c.Param("id"))
        page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
        perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
        if page < 1 {
                page = 1
        }
        if perPage < 1 || perPage > 100 {
                perPage = 50
        }
        var total int64
        h.db.Model(&Bid{}).Where("auction_id = ?", id).Count(&total)
        var bids []Bid
        h.db.Where("auction_id = ?", id).Order("amount DESC").
                Offset((page-1)*perPage).Limit(perPage).Find(&bids)
        pages := int64(1)
        if perPage > 0 {
                pages = (total + int64(perPage) - 1) / int64(perPage)
        }
        response.OKMeta(c, bids, gin.H{"total": total, "page": page, "per_page": perPage, "pages": pages})
}

// Search handles complex multi-filter auction queries.
// Supported filters: status, category_id, min_price, max_price,
// min_bid_count, ends_within_hours, sort_by (ends_asc|price_asc|price_desc|bids_desc|newest).
func (h *Handler) Search(c *gin.Context) {
        page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
        perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
        if page < 1 {
                page = 1
        }
        if perPage < 1 || perPage > 100 {
                perPage = 20
        }

        q := h.db.Model(&Auction{}).Preload("Bids")

        // ── Status filter ────────────────────────────────────────────────────
        status := c.DefaultQuery("status", "active")
        switch status {
        case "ended":
                q = q.Where("status = ?", "ended")
        case "upcoming":
                q = q.Where("status = ? AND starts_at > ?", "active", time.Now())
        case "ending_soon":
                soon := time.Now().Add(time.Hour)
                q = q.Where("status = ? AND ends_at > ? AND ends_at <= ?", "active", time.Now(), soon)
        case "all":
                // no status filter
        default:
                q = q.Where("status = ? AND ends_at > ?", "active", time.Now())
        }

        // ── Category filter ──────────────────────────────────────────────────
        if catID := c.Query("category_id"); catID != "" {
                // Join listings to filter by category
                q = q.Joins("JOIN listings ON listings.id = auctions.listing_id").
                        Where("listings.category_id = ?", catID)
        }

        // ── Price range filters ──────────────────────────────────────────────
        if minPrice := c.Query("min_price"); minPrice != "" {
                q = q.Where("auctions.current_bid >= ? OR (auctions.bid_count = 0 AND auctions.start_price >= ?)", minPrice, minPrice)
        }
        if maxPrice := c.Query("max_price"); maxPrice != "" {
                q = q.Where("auctions.current_bid <= ? OR (auctions.bid_count = 0 AND auctions.start_price <= ?)", maxPrice, maxPrice)
        }

        // ── Bid count filter ─────────────────────────────────────────────────
        if minBids := c.Query("min_bid_count"); minBids != "" {
                q = q.Where("auctions.bid_count >= ?", minBids)
        }

        // ── End time proximity filter ────────────────────────────────────────
        if endsWithin := c.Query("ends_within_hours"); endsWithin != "" {
                hours, err := strconv.Atoi(endsWithin)
                if err == nil && hours > 0 {
                        deadline := time.Now().Add(time.Duration(hours) * time.Hour)
                        q = q.Where("auctions.ends_at <= ?", deadline)
                }
        }

        // ── Seller filter ────────────────────────────────────────────────────
        if sellerID := c.Query("seller_id"); sellerID != "" {
                q = q.Where("auctions.seller_id = ?", sellerID)
        }

        // ── Count total before pagination ────────────────────────────────────
        var total int64
        q.Count(&total)

        // ── Sorting ──────────────────────────────────────────────────────────
        sortBy := c.DefaultQuery("sort_by", "ends_asc")
        switch sortBy {
        case "price_asc":
                q = q.Order("auctions.current_bid ASC, auctions.start_price ASC")
        case "price_desc":
                q = q.Order("auctions.current_bid DESC, auctions.start_price DESC")
        case "bids_desc":
                q = q.Order("auctions.bid_count DESC")
        case "newest":
                q = q.Order("auctions.created_at DESC")
        default: // "ends_asc"
                q = q.Order("auctions.ends_at ASC")
        }

        // ── Paginate + fetch ─────────────────────────────────────────────────
        var auctions []Auction
        q.Offset((page - 1) * perPage).Limit(perPage).Find(&auctions)

        pages := (total + int64(perPage) - 1) / int64(perPage)
        response.OK(c, gin.H{
                "results": auctions,
                "total":   total,
                "page":    page,
                "per_page": perPage,
                "pages":   pages,
        })
}
