package crowdshipping

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/geocore-next/backend/internal/freeze"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (h *OfferHandler) AcceptOffer(c *gin.Context) {
	offerID, _ := uuid.Parse(c.Param("id"))
	buyerID, _ := uuid.Parse(c.GetString("user_id"))

	// Sprint 8.5: Block frozen users from accepting offers
	if freeze.IsUserFrozen(h.db, buyerID) {
		response.Forbidden(c)
		return
	}

	var result TravelerOffer
	var createdOrd *order.Order
	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var offer TravelerOffer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND buyer_id=?", offerID, buyerID).First(&offer).Error; err != nil {
			return fmt.Errorf("offer not found")
		}
		if !offer.CanTransitionTo(OfferPaymentPending) {
			if offer.Status == OfferAccepted || offer.Status == OfferFundsHeld {
				return fmt.Errorf("already_accepted")
			}
			return fmt.Errorf("not acceptable: %s", offer.Status)
		}
		var cnt int64
		tx.Model(&TravelerOffer{}).Where("delivery_request_id=? AND status IN ? AND id!=?",
			offer.DeliveryRequestID, []OfferStatus{OfferAccepted, OfferFundsHeld}, offer.ID).Count(&cnt)
		if cnt > 0 {
			return fmt.Errorf("already_accepted")
		}
		offer.Status = OfferPaymentPending
		tx.Save(&offer)

		// SPRINT 7: Verify payment confirmed before escrow — NO ESCROW WITHOUT REAL MONEY
		if err := wallet.BlockWithdrawalIfUnconfirmed(tx, buyerID, decimal.NewFromFloat(offer.Price), wallet.Currency(offer.Currency)); err != nil {
			offer.Status = OfferPaymentFailed
			offer.PaymentRetryAllowed = true
			tx.Save(&offer)
			slog.Error("crowdshipping: wallet sync verification failed", "offer_id", offer.ID.String(), "error", err.Error())
			return fmt.Errorf("payment_not_confirmed")
		}

		escrow, holdErr := wallet.HoldFunds(tx, buyerID, offer.TravelerID, offer.Price, offer.Currency, "crowdshipping_offer", offer.ID.String())
		if holdErr != nil {
			offer.Status = OfferPaymentFailed
			offer.PaymentRetryAllowed = true
			tx.Save(&offer)
			slog.Error("crowdshipping: escrow hold failed", "offer_id", offer.ID.String(), "error", holdErr.Error())
			return fmt.Errorf("payment_failed")
		}
		// HoldFunds succeeded → FUNDS_HELD (separate from acceptance)
		offer.Status = OfferFundsHeld
		tx.Save(&offer)
		// FUNDS_HELD → ACCEPTED + create order
		offer.Status = OfferAccepted
		tx.Save(&offer)

		// Lock delivery request — prevents any race condition on new offers
		tx.Model(&DeliveryRequest{}).Where("id=?", offer.DeliveryRequestID).
			Update("status", DeliveryLocked)

		ord, ordErr := createOrderFromOffer(tx, &offer, "Order from accepted offer")
		if ordErr != nil {
			return ordErr
		}
		_ = escrow
		result = offer
		createdOrd = ord
		return nil
	})
	if dbErr != nil {
		handleAcceptError(h, c, dbErr, buyerID, offerID)
		return
	}
	h.notifyAsync(notifications.NotifyInput{UserID: result.TravelerID, Type: "offer_accepted", Title: "Offer Accepted!", Body: fmt.Sprintf("Your offer of $%.2f has been accepted", result.Price), Data: map[string]string{"offer_id": result.ID.String(), "order_id": createdOrd.ID.String()}})
	metrics.IncOrdersCreated()
	metrics.IncBusinessEvent("offer_accepted")
	response.OK(c, gin.H{"offer": result, "order": createdOrd})
}

func handleAcceptError(h *OfferHandler, c *gin.Context, dbErr error, buyerID, offerID uuid.UUID) {
	switch {
	case dbErr.Error() == "already_accepted":
		c.JSON(http.StatusConflict, gin.H{"success": false, "error": "already_accepted", "message": "Offer already accepted"})
	case dbErr.Error() == "payment_not_confirmed":
		h.notifyAsync(notifications.NotifyInput{UserID: buyerID, Type: "offer_payment_not_confirmed", Title: "Payment Not Confirmed", Body: "Wallet balance not confirmed. Retry available.", Data: map[string]string{"offer_id": offerID.String()}})
		c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "error": "payment_not_confirmed", "message": "Wallet balance not confirmed — retry available"})
	case dbErr.Error() == "payment_failed":
		h.notifyAsync(notifications.NotifyInput{UserID: buyerID, Type: "offer_payment_failed", Title: "Payment Failed", Body: "Escrow hold failed. Retry available.", Data: map[string]string{"offer_id": offerID.String()}})
		c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "error": "payment_failed", "message": "Escrow hold failed — retry available"})
	default:
		response.BadRequest(c, dbErr.Error())
	}
}
