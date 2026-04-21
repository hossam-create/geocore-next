package listings

import (
	"fmt"
	"log/slog"
	"time"

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

// ── Buy Now handler ────────────────────────────────────────────────────────────

// BuyNow executes an instant purchase on a listing.
// POST /api/v1/listings/:id/buy-now
func (h *Handler) BuyNow(c *gin.Context) {
	buyerID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	var listing Listing
	var totalCents int64
	var total float64

	dbErr := h.writeDB().Transaction(func(tx *gorm.DB) error {
		// Lock listing row to prevent concurrent purchases
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&listing, "id = ? AND status = ?", listingID, "active").Error; err != nil {
			return fmt.Errorf("listing_not_found")
		}

		if listing.UserID == buyerID {
			return fmt.Errorf("own_item")
		}

		// ── RISK 2 FIX: explicit status guard ──
		if listing.Status != "active" {
			return fmt.Errorf("already_sold")
		}

		cfg := listing.GetTradeConfig()
		if !cfg.BuyNowEnabled && listing.ListingType != ListingTypeBuyNow && listing.ListingType != ListingTypeHybrid {
			return fmt.Errorf("buy_now_not_enabled")
		}

		// Determine price — use PriceCents if available, else Price
		if listing.PriceCents > 0 {
			totalCents = listing.PriceCents
		} else if listing.Price != nil {
			totalCents = toCents(*listing.Price)
		} else {
			return fmt.Errorf("no_price")
		}
		total = fromCents(totalCents)

		// Hold escrow funds
		_, err := wallet.HoldFunds(tx, buyerID, listing.UserID, total, listing.Currency,
			"buy_now", listingID.String())
		if err != nil {
			return fmt.Errorf("escrow_failed: %w", err)
		}

		// Mark listing as sold
		now := time.Now()
		return tx.Model(&listing).Updates(map[string]interface{}{
			"status":  "sold",
			"sold_at": now,
		}).Error
	})

	if dbErr != nil {
		switch dbErr.Error() {
		case "listing_not_found":
			response.NotFound(c, "Listing")
		case "own_item":
			response.BadRequest(c, "Cannot buy your own listing")
		case "already_sold":
			response.BadRequest(c, "Listing is no longer available (already sold)")
		case "buy_now_not_enabled":
			response.BadRequest(c, "Buy Now is not enabled for this listing")
		case "no_price":
			response.BadRequest(c, "Listing has no price set")
		default:
			response.BadRequest(c, dbErr.Error())
		}
		return
	}

	platformCents := totalCents * 15 / 100
	travelerCents := totalCents - platformCents

	// Publish domain event
	events.Publish(events.Event{
		Type: events.EventListingCreated,
		Payload: map[string]interface{}{
			"event_type":     "order.created",
			"listing_id":     listingID.String(),
			"buyer_id":       buyerID.String(),
			"seller_id":      listing.UserID.String(),
			"total_cents":    totalCents,
			"platform_cents": platformCents,
			"traveler_cents": travelerCents,
			"currency":       listing.Currency,
			"source":         "buy_now",
		},
	})

	// Kafka outbox for order creation
	_ = kafka.WriteOutbox(h.writeDB(), kafka.TopicOrders, kafka.New(
		"order.created",
		listingID.String(),
		"order",
		kafka.Actor{Type: "user", ID: buyerID.String()},
		map[string]interface{}{
			"listing_id":     listingID.String(),
			"buyer_id":       buyerID.String(),
			"seller_id":      listing.UserID.String(),
			"total_cents":    totalCents,
			"platform_cents": platformCents,
			"currency":       listing.Currency,
			"source":         "buy_now",
		},
		kafka.EventMeta{Source: "api-service"},
	))

	// Notification
	if globalNotifSvc != nil {
		go globalNotifSvc.Notify(notifications.NotifyInput{
			UserID: listing.UserID,
			Type:   "buy_now",
			Title:  "Item Sold!",
			Body:   fmt.Sprintf("Your listing was purchased via Buy Now for %.2f %s.", total, listing.Currency),
			Data:   map[string]string{"listing_id": listingID.String()},
		})
	}

	slog.Info("trading: buy now completed",
		"listing_id", listingID,
		"buyer_id", buyerID,
		"total_cents", totalCents,
		"platform_cents", platformCents,
		"traveler_cents", travelerCents,
	)

	response.OK(c, gin.H{
		"message":        "Purchase successful!",
		"total_cents":    totalCents,
		"platform_cents": platformCents,
		"traveler_cents": travelerCents,
		"currency":       listing.Currency,
	})
}
