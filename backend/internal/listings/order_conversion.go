package listings

import (
	"fmt"
	"log/slog"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ── Order conversion handler ────────────────────────────────────────────────────

// ConvertOfferToOrder is the explicit endpoint to convert an accepted negotiation
// into an order. This is needed because auto-accepted offers may need a separate
// trigger (e.g. buyer confirms), or the seller may accept manually.
// POST /api/v1/listings/:id/negotiation/:thread_id/convert
func (h *Handler) ConvertOfferToOrder(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}
	threadID, err := uuid.Parse(c.Param("thread_id"))
	if err != nil {
		response.BadRequest(c, "Invalid thread ID")
		return
	}

	var thread NegotiationThread
	var listing Listing

	dbErr := h.writeDB().Transaction(func(tx *gorm.DB) error {
		// Lock both thread and listing
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&thread, "id = ? AND (buyer_id = ? OR seller_id = ?) AND status = ?",
				threadID, userID, userID, NegotiationAccepted).Error; err != nil {
			return fmt.Errorf("thread_not_found_or_not_accepted")
		}

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&listing, "id = ? AND status = ?", listingID, "active").Error; err != nil {
			return fmt.Errorf("listing_not_found_or_sold")
		}

		return ConvertAcceptedOfferToOrder(tx, &thread, &listing)
	})

	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}

	slog.Info("trading: offer converted to order via endpoint",
		"listing_id", listingID,
		"thread_id", threadID,
		"agreed_cents", thread.AgreedCents,
	)

	response.OK(c, gin.H{
		"message":      "Offer converted to order successfully",
		"thread_id":    threadID,
		"agreed_cents": thread.AgreedCents,
		"currency":     thread.Currency,
	})
}

// GetTradeInfo returns the trading configuration and status for a listing.
// GET /api/v1/listings/:id/trade
func (h *Handler) GetTradeInfo(c *gin.Context) {
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	var listing Listing
	if err := h.readDB().First(&listing, "id = ? AND status = ?", listingID, "active").Error; err != nil {
		response.NotFound(c, "Listing")
		return
	}

	cfg := listing.GetTradeConfig()

	// Count open negotiations
	var openNegotiations int64
	h.readDB().Model(&NegotiationThread{}).
		Where("listing_id = ? AND status IN ?", listingID,
			[]NegotiationStatus{NegotiationOpen, NegotiationCountered}).
		Count(&openNegotiations)

	// Check if auction exists
	var auctionCount int64
	h.readDB().Table("auctions").Where("listing_id = ?", listingID).Count(&auctionCount)

	response.OK(c, gin.H{
		"listing_type":      listing.ListingType,
		"trade_config":      cfg,
		"price_cents":       listing.PriceCents,
		"open_negotiations": openNegotiations,
		"has_auction":       auctionCount > 0,
	})
}
