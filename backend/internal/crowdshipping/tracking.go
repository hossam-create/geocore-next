package crowdshipping

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UpdateTrackingReq struct {
	Status        ShipmentStatus `json:"status" binding:"required"`
	Location      string         `json:"location"`
	Note          string         `json:"note"`
	ProofImageURL string         `json:"proof_image_url"`
}

// UpdateTracking allows traveler to update shipment status.
// POST /tracking/update
func (h *OfferHandler) UpdateTracking(c *gin.Context) {
	var req UpdateTrackingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	travelerID, _ := uuid.Parse(c.GetString("user_id"))
	orderIDStr := c.Query("order_id")
	if orderIDStr == "" {
		response.BadRequest(c, "order_id query param required")
		return
	}
	orderID, _ := uuid.Parse(orderIDStr)

	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var ord order.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND seller_id=?", orderID, travelerID).First(&ord).Error; err != nil {
			return fmt.Errorf("order not found")
		}
		if ord.Status != order.StatusConfirmed && ord.Status != order.StatusShipped {
			return fmt.Errorf("order not trackable: %s", ord.Status)
		}
		// Validate FSM transition
		var lastEvent TrackingEvent
		if err := tx.Where("order_id=?", orderID).Order("created_at DESC").First(&lastEvent).Error; err == nil {
			valid := false
			for _, s := range NextAllowedStatuses(lastEvent.Status) {
				if s == req.Status {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid transition: %s → %s", lastEvent.Status, req.Status)
			}
		} else if req.Status != ShipmentAccepted {
			return fmt.Errorf("first tracking event must be accepted")
		}

		te := TrackingEvent{
			OrderID: orderID, TravelerID: travelerID,
			Status: req.Status, Location: req.Location,
			Note: req.Note, ProofImageURL: req.ProofImageURL,
		}
		if err := tx.Create(&te).Error; err != nil {
			return err
		}

		// Update order status based on tracking
		orderUpdates := map[string]any{}
		switch req.Status {
		case ShipmentPurchased:
			orderUpdates["status"] = order.StatusProcessing
		case ShipmentInTransit:
			orderUpdates["status"] = order.StatusShipped
		case ShipmentDelivered:
			orderUpdates["status"] = order.StatusDelivered
		}
		if len(orderUpdates) > 0 {
			tx.Model(&ord).Updates(orderUpdates)
		}
		return nil
	})

	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}

	// Notify buyer
	var ord order.Order
	h.db.Where("id=?", orderID).First(&ord)
	h.notifyAsync(notifications.NotifyInput{
		UserID: ord.BuyerID, Type: "shipment_updated", Title: "Shipment Update",
		Body: fmt.Sprintf("Your shipment is now: %s", req.Status),
		Data: map[string]string{"order_id": orderID.String(), "status": string(req.Status)},
	})
	response.OK(c, gin.H{"message": "tracking updated"})
}

// GetTracking returns tracking events for an order.
// GET /tracking/:order_id
func (h *OfferHandler) GetTracking(c *gin.Context) {
	orderID, _ := uuid.Parse(c.Param("order_id"))
	var events []TrackingEvent
	if err := h.db.Where("order_id=?", orderID).Order("created_at ASC").Find(&events).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, events)
}

// ConfirmDelivery allows buyer to confirm delivery → triggers escrow release.
// POST /tracking/:order_id/confirm
func (h *OfferHandler) ConfirmDelivery(c *gin.Context) {
	orderID, _ := uuid.Parse(c.Param("order_id"))
	buyerID, _ := uuid.Parse(c.GetString("user_id"))

	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		// Lock order
		var ord order.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id=? AND buyer_id=?", orderID, buyerID).First(&ord).Error; err != nil {
			return fmt.Errorf("order not found")
		}
		if ord.Status != order.StatusDelivered {
			return fmt.Errorf("order not delivered yet: %s", ord.Status)
		}

		// Create confirmed tracking event
		te := TrackingEvent{
			OrderID: orderID, TravelerID: ord.SellerID,
			Status: ShipmentConfirmed, Note: "Buyer confirmed delivery",
		}
		if err := tx.Create(&te).Error; err != nil {
			return err
		}

		// Update order to completed
		tx.Model(&ord).Updates(map[string]any{
			"status": order.StatusCompleted,
		})

		// Find the linked offer and mark completed
		var offer TravelerOffer
		if err := tx.Where("order_id=?", orderID).First(&offer).Error; err == nil {
			tx.Model(&offer).Update("status", OfferCompleted)
		}

		// Release escrow: find escrow by reference and release
		// The escrow was created with refType=crowdshipping_offer and refID=offer.ID
		if err := releaseEscrowForOrder(tx, &ord); err != nil {
			slog.Error("crowdshipping: escrow release failed on confirm",
				"order_id", orderID.String(), "error", err.Error())
			// Don't fail the confirmation — escrow release can be retried
		}

		return nil
	})

	if dbErr != nil {
		response.BadRequest(c, dbErr.Error())
		return
	}

	// Notify traveler
	var ord order.Order
	h.db.Where("id=?", orderID).First(&ord)
	h.notifyAsync(notifications.NotifyInput{
		UserID: ord.SellerID, Type: "delivery_confirmed", Title: "Delivery Confirmed!",
		Body: "Buyer confirmed delivery — funds will be released to your wallet",
		Data: map[string]string{"order_id": orderID.String()},
	})
	response.OK(c, gin.H{"message": "delivery confirmed — escrow release initiated"})
}

// releaseEscrowForOrder programmatically releases escrow for a crowdshipping order.
func releaseEscrowForOrder(tx *gorm.DB, ord *order.Order) error {
	var offer TravelerOffer
	if err := tx.Where("order_id=?", ord.ID).First(&offer).Error; err != nil {
		return fmt.Errorf("offer not found: %w", err)
	}
	refID := offer.ID.String()
	refType := "crowdshipping_offer"

	result := tx.Model(&wallet.Escrow{}).
		Where("reference_id=? AND type=? AND status=?", refID, refType, wallet.StatusPending).
		Updates(map[string]any{"status": wallet.StatusCompleted, "released_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		slog.Warn("crowdshipping: no pending escrow to release", "order_id", ord.ID.String())
	}

	var sw wallet.Wallet
	if err := tx.Where("user_id=?", ord.SellerID).First(&sw).Error; err != nil {
		return fmt.Errorf("seller wallet not found: %w", err)
	}
	var sb wallet.WalletBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("wallet_id=? AND currency=?", sw.ID, offer.Currency).First(&sb).Error; err != nil {
		return fmt.Errorf("seller balance not found: %w", err)
	}

	earnings := decimal.NewFromFloat(fromCents(offer.TravelerEarningsCents))
	balBefore := sb.Balance
	sb.AvailableBalance = sb.AvailableBalance.Add(earnings)
	sb.Balance = sb.Balance.Add(earnings)
	if err := tx.Save(&sb).Error; err != nil {
		return err
	}

	now := time.Now()
	wtx := wallet.WalletTransaction{
		WalletID: sw.ID, Type: wallet.TransactionRelease,
		Currency: wallet.Currency(offer.Currency), Amount: earnings,
		BalanceBefore: balBefore, BalanceAfter: sb.Balance,
		Status: wallet.StatusCompleted, ReferenceID: &refID, ReferenceType: &refType,
		Description: fmt.Sprintf("Escrow release for crowdshipping order #%s", ord.ID.String()),
		CompletedAt: &now,
	}
	return tx.Create(&wtx).Error
}
