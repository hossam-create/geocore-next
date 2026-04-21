package listings

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/auctions"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Auction integration ────────────────────────────────────────────────────────

// CreateAuctionForListing creates an auction from a listing with listing_type=auction.
// POST /api/v1/listings/:id/auction
func (h *Handler) CreateAuctionForListing(c *gin.Context) {
	sellerID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	var req struct {
		StartPrice    float64  `json:"start_price" binding:"required,gt=0"`
		ReservePrice  *float64 `json:"reserve_price"`
		BuyNowPrice   *float64 `json:"buy_now_price"`
		DurationHours int      `json:"duration_hours"` // default 72
		AuctionType   string   `json:"auction_type"`   // standard | dutch | reverse | sealed
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var listing Listing
	if err := h.writeDB().First(&listing, "id = ? AND user_id = ? AND status = ?",
		listingID, sellerID, "active").Error; err != nil {
		response.NotFound(c, "Listing")
		return
	}

	cfg := listing.GetTradeConfig()
	if !cfg.AuctionEnabled && listing.ListingType != ListingTypeAuction {
		response.BadRequest(c, "Auctions are not enabled for this listing")
		return
	}

	// Check if auction already exists for this listing
	var existingCount int64
	h.writeDB().Model(&auctions.Auction{}).Where("listing_id = ?", listingID).Count(&existingCount)
	if existingCount > 0 {
		response.BadRequest(c, "Auction already exists for this listing")
		return
	}

	durationHours := req.DurationHours
	if durationHours <= 0 {
		durationHours = 72
	}

	auctionType := auctions.AuctionType(req.AuctionType)
	if auctionType == "" {
		auctionType = auctions.AuctionTypeStandard
	}

	startsAt := timeNow()
	endsAt := startsAt.Add(hoursDuration(durationHours))

	auction := auctions.Auction{
		ID:               uuid.New(),
		ListingID:        listingID,
		SellerID:         sellerID,
		Type:             auctionType,
		StartPrice:       req.StartPrice,
		ReservePrice:     req.ReservePrice,
		BuyNowPrice:      req.BuyNowPrice,
		CurrentBid:       0,
		Status:           auctions.StatusActive,
		StartsAt:         startsAt,
		EndsAt:           endsAt,
		Currency:         listing.Currency,
		AntiSnipeEnabled: true,
		ProxyBidEnabled:  true,
	}

	if err := h.writeDB().Create(&auction).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	slog.Info("trading: auction created for listing",
		"listing_id", listingID,
		"auction_id", auction.ID,
		"start_price", req.StartPrice,
		"type", string(auctionType),
	)

	response.Created(c, auction)
}

// AuctionBidToOrder converts a winning auction bid to an order + escrow.
// Called when a bid meets or exceeds the buy_now_price, or when auction ends with a winner.
// POST /api/v1/listings/:id/auction/convert
func (h *Handler) AuctionBidToOrder(c *gin.Context) {
	buyerID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	var req struct {
		AuctionID string `json:"auction_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	auctionID, _ := uuid.Parse(req.AuctionID)

	var listing Listing
	var auction auctions.Auction

	dbErr := h.writeDB().Transaction(func(tx *gorm.DB) error {
		// Lock listing
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&listing, "id = ? AND status = ?", listingID, "active").Error; err != nil {
			return fmt.Errorf("listing_not_found")
		}

		// Lock auction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&auction, "id = ? AND status IN ?", auctionID,
				[]auctions.AuctionStatus{auctions.StatusSold, auctions.StatusEnded}).Error; err != nil {
			return fmt.Errorf("auction_not_found")
		}

		if auction.WinnerID == nil || *auction.WinnerID != buyerID {
			return fmt.Errorf("not_winner")
		}

		totalCents := toCents(auction.CurrentBid)
		total := fromCents(totalCents)

		// Hold escrow
		_, err := wallet.HoldFunds(tx, buyerID, listing.UserID, total, auction.Currency,
			"auction", auctionID.String())
		if err != nil {
			return fmt.Errorf("escrow_failed: %w", err)
		}

		// Mark listing as sold
		now := timeNow()
		return tx.Model(&listing).Updates(map[string]interface{}{
			"status":  "sold",
			"sold_at": now,
		}).Error
	})

	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}

	totalCents := toCents(auction.CurrentBid)
	platformCents := totalCents * 15 / 100

	// Kafka outbox
	_ = kafka.WriteOutbox(h.writeDB(), kafka.TopicOrders, kafka.New(
		"order.created",
		auctionID.String(),
		"order",
		kafka.Actor{Type: "user", ID: buyerID.String()},
		map[string]interface{}{
			"auction_id":     auctionID.String(),
			"listing_id":     listingID.String(),
			"buyer_id":       buyerID.String(),
			"seller_id":      listing.UserID.String(),
			"total_cents":    totalCents,
			"platform_cents": platformCents,
			"currency":       auction.Currency,
			"source":         "auction",
		},
		kafka.EventMeta{Source: "api-service"},
	))

	// Notification
	if globalNotifSvc != nil {
		go globalNotifSvc.Notify(notifications.NotifyInput{
			UserID: listing.UserID,
			Type:   "auction_won",
			Title:  "Auction item sold!",
			Body:   fmt.Sprintf("Your auction ended with a winning bid of %.2f %s.", auction.CurrentBid, auction.Currency),
			Data:   map[string]string{"listing_id": listingID.String(), "auction_id": auctionID.String()},
		})
	}

	slog.Info("trading: auction converted to order",
		"listing_id", listingID,
		"auction_id", auctionID,
		"winner_id", buyerID,
		"total_cents", totalCents,
	)

	response.OK(c, gin.H{
		"message":        "Auction purchase completed!",
		"total_cents":    totalCents,
		"platform_cents": platformCents,
		"currency":       auction.Currency,
	})
}

// timeNow and hoursDuration are testable helpers.
var timeNow = func() time.Time { return time.Now() }

func hoursDuration(h int) time.Duration { return time.Duration(h) * time.Hour }

// ── RISK 4: Auto-convert when bid >= BuyNowPrice ──────────────────────────────

// AutoConvertHighBid checks if a bid meets or exceeds the listing's BuyNowPrice
// and automatically converts it to an order. Called via event listener.
func AutoConvertHighBid(db *gorm.DB, listingID uuid.UUID, auctionID uuid.UUID, bidderID uuid.UUID, bidAmount float64) {
	var listing Listing
	var auction auctions.Auction

	err := db.Transaction(func(tx *gorm.DB) error {
		// Lock listing
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&listing, "id = ? AND status = ?", listingID, "active").Error; err != nil {
			return fmt.Errorf("listing_not_available")
		}
		// Lock auction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&auction, "id = ? AND status = ?", auctionID, auctions.StatusActive).Error; err != nil {
			return fmt.Errorf("auction_not_active")
		}

		// Check if bid >= BuyNowPrice
		if auction.BuyNowPrice == nil || bidAmount < *auction.BuyNowPrice {
			return fmt.Errorf("bid_below_buy_now")
		}

		// Mark auction as sold with this bidder as winner
		now := timeNow()
		auction.Status = auctions.StatusSold
		auction.WinnerID = &bidderID
		auction.CurrentBid = bidAmount
		if err := tx.Save(&auction).Error; err != nil {
			return err
		}

		// Hold escrow
		totalCents := toCents(bidAmount)
		total := fromCents(totalCents)
		_, escrowErr := wallet.HoldFunds(tx, bidderID, listing.UserID, total, auction.Currency,
			"auction", auctionID.String())
		if escrowErr != nil {
			// Escrow failed — mark auction back to active so it can be retried
			auction.Status = auctions.StatusActive
			tx.Save(&auction)
			slog.Error("trading: auto-convert escrow failed",
				"auction_id", auctionID, "error", escrowErr)
			return fmt.Errorf("escrow_failed: %w", escrowErr)
		}

		// Mark listing as sold
		return tx.Model(&listing).Updates(map[string]interface{}{
			"status":  "sold",
			"sold_at": now,
		}).Error
	})

	if err != nil {
		slog.Error("trading: auto-convert high bid failed",
			"listing_id", listingID, "auction_id", auctionID, "error", err)
		return
	}

	totalCents := toCents(bidAmount)
	platformCents := totalCents * 15 / 100

	// Kafka outbox
	_ = kafka.WriteOutbox(db, kafka.TopicOrders, kafka.New(
		"order.created",
		auctionID.String(),
		"order",
		kafka.Actor{Type: "user", ID: bidderID.String()},
		map[string]interface{}{
			"auction_id":     auctionID.String(),
			"listing_id":     listingID.String(),
			"buyer_id":       bidderID.String(),
			"seller_id":      listing.UserID.String(),
			"total_cents":    totalCents,
			"platform_cents": platformCents,
			"currency":       auction.Currency,
			"source":         "auction_auto_convert",
		},
		kafka.EventMeta{Source: "api-service"},
	))

	// Notifications
	notifyOffer(listingID, listing.UserID, "auction_won", "Auction Item Sold via Buy Now!",
		fmt.Sprintf("A bid of %.2f %s met your Buy Now price. The item has been sold!", bidAmount, auction.Currency))
	notifyOffer(listingID, bidderID, "auction_won", "You Won the Auction!",
		fmt.Sprintf("Your bid of %.2f %s met the Buy Now price. The item is yours!", bidAmount, auction.Currency))

	slog.Info("trading: auto-converted high bid to order",
		"listing_id", listingID,
		"auction_id", auctionID,
		"winner_id", bidderID,
		"total_cents", totalCents,
	)
}

// globalDB is set via SetGlobalDB and used by the event listener for auto-convert.
var globalDB *gorm.DB

// SetGlobalDB stores a reference to the write DB for use by background listeners.
func SetGlobalDB(db *gorm.DB) { globalDB = db }

// RegisterAuctionAutoConvert subscribes to auction bid events and auto-converts
// when a bid meets or exceeds the BuyNowPrice. Call this from main.go on startup.
func RegisterAuctionAutoConvert(db *gorm.DB) {
	SetGlobalDB(db)
	events.Subscribe("auction.bid.placed", func(e events.Event) {
		listingIDStr, _ := e.Payload["listing_id"].(string)
		auctionIDStr, _ := e.Payload["auction_id"].(string)
		bidderIDStr, _ := e.Payload["bidder_id"].(string)
		bidAmount, _ := e.Payload["bid_amount"].(float64)

		if listingIDStr == "" || auctionIDStr == "" || bidderIDStr == "" || bidAmount <= 0 {
			return
		}

		listingID, err1 := uuid.Parse(listingIDStr)
		auctionID, err2 := uuid.Parse(auctionIDStr)
		bidderID, err3 := uuid.Parse(bidderIDStr)
		if err1 != nil || err2 != nil || err3 != nil {
			return
		}

		// Run auto-convert in a goroutine to not block the event bus
		go AutoConvertHighBid(globalDB, listingID, auctionID, bidderID, bidAmount)
	})
	slog.Info("trading: auction auto-convert listener registered")
}
