package crowdshipping

import (
	"fmt"
	"net/http"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (h *OfferHandler) RetryPayment(c *gin.Context) {
	offerID, _ := uuid.Parse(c.Param("id"))
	buyerID, _ := uuid.Parse(c.GetString("user_id"))
	var result TravelerOffer
	var createdOrd *order.Order

	dbErr := locking.RetryOnDeadlock(h.db, func(tx *gorm.DB) error {
		var offer TravelerOffer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id=? AND buyer_id=? AND status=? AND payment_retry_allowed=?", offerID, buyerID, OfferPaymentFailed, true).
			First(&offer).Error; err != nil {
			return fmt.Errorf("offer not retryable")
		}
		offer.Status = OfferPaymentPending
		tx.Save(&offer)

		escrow, holdErr := wallet.HoldFunds(tx, buyerID, offer.TravelerID, offer.Price, offer.Currency, "crowdshipping_offer_retry", offer.ID.String())
		if holdErr != nil {
			offer.Status = OfferPaymentFailed
			offer.PaymentRetryAllowed = true
			tx.Save(&offer)
			return fmt.Errorf("payment_failed")
		}
		// HoldFunds succeeded → FUNDS_HELD → ACCEPTED
		offer.Status = OfferFundsHeld
		tx.Save(&offer)
		offer.Status = OfferAccepted
		tx.Save(&offer)

		// Lock delivery request — prevents race conditions
		tx.Model(&DeliveryRequest{}).Where("id=?", offer.DeliveryRequestID).
			Update("status", DeliveryLocked)

		ord, ordErr := createOrderFromOffer(tx, &offer, "Order from payment retry")
		if ordErr != nil {
			return ordErr
		}
		_ = escrow
		result = offer
		createdOrd = ord
		return nil
	})

	if dbErr != nil {
		switch dbErr.Error() {
		case "payment_failed":
			c.JSON(http.StatusPaymentRequired, gin.H{"success": false, "error": "payment_failed", "message": "Escrow hold failed again"})
		case "offer not retryable":
			response.BadRequest(c, "offer not retryable")
		default:
			response.InternalError(c, dbErr)
		}
		return
	}
	h.notifyAsync(notifications.NotifyInput{UserID: result.TravelerID, Type: "offer_accepted", Title: "Offer Accepted!", Body: fmt.Sprintf("Payment retry succeeded — offer of $%.2f accepted", result.Price), Data: map[string]string{"offer_id": result.ID.String(), "order_id": createdOrd.ID.String()}})
	metrics.IncOrdersCreated()
	response.OK(c, gin.H{"offer": result, "order": createdOrd})
}
