package crowdshipping

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// PriceGapThresholdCents is the max gap between buyer reward and traveler offer
	// for auto-accept to trigger. Default: $5.00 = 500 cents
	PriceGapThresholdCents int64 = 500
	// MaxAutoClosePerUserPerDay limits auto-closes per buyer per day
	MaxAutoClosePerUserPerDay int64 = 5
	// StaleOfferThreshold is how old an offer can be before it's considered stale
	StaleOfferThreshold = 10 * time.Minute
	// CancelWindowMinutes is how long a buyer has to cancel an auto-accepted deal
	CancelWindowMinutes = 2
)

// AutoAcceptSettings represents a user's auto-accept preferences.
type AutoAcceptSettings struct {
	AutoAcceptEnabled        bool  `gorm:"not null;default:false" json:"auto_accept_enabled"`
	MaxAutoAcceptAmountCents int64 `gorm:"default:0" json:"max_auto_accept_amount_cents"` // 0 = no limit
}

var dailyAutoCloseCount = map[uuid.UUID]*int64{}
var dailyAutoCloseReset time.Time

// AutoAcceptOffer attempts to auto-accept an offer when the price gap
// between buyer reward and traveler offer is within threshold.
// Safety: buyer must have auto_accept enabled, daily limit enforced.
func AutoAcceptOffer(db *gorm.DB, notifSvc *notifications.Service, offerID uuid.UUID) error {
	if !config.GetFlags().EnableDealCloser {
		slog.Info("deal_closer: skipped — feature flag disabled")
		return nil
	}

	var result TravelerOffer
	var createdOrd *order.Order

	err := locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		var offer TravelerOffer
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id=? AND status=? AND is_auto_generated=?", offerID, OfferPending, true).
			First(&offer).Error; err != nil {
			return fmt.Errorf("offer not eligible for auto-accept")
		}

		// Stale offer check: reject offers older than 10 minutes
		if time.Since(offer.CreatedAt) > StaleOfferThreshold {
			return fmt.Errorf("offer is stale (created %v ago)", time.Since(offer.CreatedAt).Round(time.Minute))
		}

		// Get delivery request
		var dr DeliveryRequest
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id=?", offer.DeliveryRequestID).First(&dr).Error; err != nil {
			return fmt.Errorf("delivery request not found")
		}

		// Check buyer auto-accept settings
		if !getAutoAcceptEnabled(tx, dr.BuyerID) {
			return fmt.Errorf("buyer has not enabled auto-accept")
		}

		// Check max auto-accept amount
		maxAmt := getMaxAutoAcceptAmount(tx, dr.BuyerID)
		if maxAmt > 0 && offer.PriceCents > maxAmt {
			return fmt.Errorf("offer amount %d exceeds buyer max auto-accept %d", offer.PriceCents, maxAmt)
		}

		// Trust check: only auto-close if both parties are trust-approved
		buyerTrust := GetTrustScore(tx, dr.BuyerID)
		travelerTrust := GetTrustScore(tx, offer.TravelerID)
		if !IsTrustApprovedForDeal(buyerTrust) {
			return fmt.Errorf("buyer trust score too low (%.1f)", buyerTrust.OverallScore)
		}
		if !IsTrustApprovedForDeal(travelerTrust) {
			return fmt.Errorf("traveler trust score too low (%.1f)", travelerTrust.OverallScore)
		}

		// Check: is the price gap within threshold?
		rewardCents := toCents(dr.Reward)
		gap := rewardCents - offer.PriceCents
		if gap < 0 {
			gap = -gap
		}
		if gap > PriceGapThresholdCents {
			return fmt.Errorf("price gap too large: %d cents > %d threshold", gap, PriceGapThresholdCents)
		}

		// Check: another offer already accepted?
		var cnt int64
		tx.Model(&TravelerOffer{}).Where("delivery_request_id=? AND status IN ? AND id!=?",
			dr.ID, []OfferStatus{OfferAccepted, OfferFundsHeld}, offer.ID).Count(&cnt)
		if cnt > 0 {
			return fmt.Errorf("already_accepted")
		}

		// Daily limit per buyer
		if !checkDailyLimit(dr.BuyerID) {
			return fmt.Errorf("daily auto-close limit reached")
		}

		// Execute: PAYMENT_PENDING → FUNDS_HELD → ACCEPTED
		offer.Status = OfferPaymentPending
		tx.Save(&offer)

		// SPRINT 7: Verify payment confirmed before escrow — NO ESCROW WITHOUT REAL MONEY
		if err := wallet.BlockWithdrawalIfUnconfirmed(tx, dr.BuyerID, decimal.NewFromFloat(offer.Price), wallet.Currency(offer.Currency)); err != nil {
			offer.Status = OfferPaymentFailed
			offer.PaymentRetryAllowed = true
			tx.Save(&offer)
			return fmt.Errorf("auto-accept wallet sync failed: %w", err)
		}

		escrow, holdErr := wallet.HoldFunds(tx, dr.BuyerID, offer.TravelerID, offer.Price, offer.Currency, "crowdshipping_auto_accept", offer.ID.String())
		if holdErr != nil {
			offer.Status = OfferPaymentFailed
			offer.PaymentRetryAllowed = true
			tx.Save(&offer)
			return fmt.Errorf("auto-accept payment failed: %w", holdErr)
		}

		offer.Status = OfferFundsHeld
		tx.Save(&offer)

		offer.Status = OfferAccepted
		tx.Save(&offer)

		// Lock delivery request
		tx.Model(&DeliveryRequest{}).Where("id=?", dr.ID).Update("status", DeliveryLocked)

		// Create order
		ord, ordErr := createOrderFromOffer(tx, &offer, "Auto-accepted offer (smart deal closer)")
		if ordErr != nil {
			return ordErr
		}
		_ = escrow
		result = offer
		createdOrd = ord
		return nil
	})

	if err != nil {
		slog.Info("deal_closer: skipped", "offer_id", offerID, "reason", err.Error())
		return err
	}

	// Notify both parties with cancel window
	if notifSvc != nil {
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: result.BuyerID, Type: "offer_auto_accepted",
			Title: "Offer Auto-Accepted!",
			Body:  fmt.Sprintf("An offer of $%.2f was auto-accepted — cancel within %d minutes if unintended", result.Price, CancelWindowMinutes),
			Data:  map[string]string{"offer_id": result.ID.String(), "order_id": createdOrd.ID.String(), "cancel_window": fmt.Sprintf("%d", CancelWindowMinutes)},
		})
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: result.TravelerID, Type: "offer_auto_accepted",
			Title: "Your Offer Was Auto-Accepted!",
			Body:  fmt.Sprintf("Your offer of $%.2f was auto-accepted — order created", result.Price),
			Data:  map[string]string{"offer_id": result.ID.String(), "order_id": createdOrd.ID.String()},
		})
	}

	slog.Info("deal_closer: auto-accepted", "offer_id", offerID)
	return nil
}

// getAutoAcceptEnabled reads the buyer's auto-accept preference.
// Defaults to false (opt-in) for safety.
func getAutoAcceptEnabled(tx *gorm.DB, buyerID uuid.UUID) bool {
	var settings AutoAcceptSettings
	if err := tx.Table("user_auto_accept_settings").
		Where("user_id=?", buyerID).First(&settings).Error; err != nil {
		return false // default: disabled
	}
	return settings.AutoAcceptEnabled
}

// getMaxAutoAcceptAmount reads the buyer's max auto-accept amount.
// Returns 0 for no limit.
func getMaxAutoAcceptAmount(tx *gorm.DB, buyerID uuid.UUID) int64 {
	var settings AutoAcceptSettings
	if err := tx.Table("user_auto_accept_settings").
		Where("user_id=?", buyerID).First(&settings).Error; err != nil {
		return 0 // default: no limit
	}
	return settings.MaxAutoAcceptAmountCents
}

func checkDailyLimit(buyerID uuid.UUID) bool {
	now := time.Now()
	if dailyAutoCloseReset.IsZero() || now.Sub(dailyAutoCloseReset) > 24*time.Hour {
		dailyAutoCloseCount = map[uuid.UUID]*int64{}
		dailyAutoCloseReset = now
	}
	ptr, ok := dailyAutoCloseCount[buyerID]
	if !ok {
		var v int64 = 0
		dailyAutoCloseCount[buyerID] = &v
		ptr = &v
	}
	if atomic.LoadInt64(ptr) >= MaxAutoClosePerUserPerDay {
		return false
	}
	atomic.AddInt64(ptr, 1)
	return true
}

// TryAutoCloseOffer checks if an auto-generated offer should be auto-accepted.
// Called after GenerateAutoOffers.
func TryAutoCloseOffer(db *gorm.DB, notifSvc *notifications.Service, requestID uuid.UUID) {
	var offers []TravelerOffer
	db.Where("delivery_request_id=? AND status=? AND is_auto_generated=?",
		requestID, OfferPending, true).Find(&offers)

	for _, o := range offers {
		// Best effort — skip on failure
		_ = AutoAcceptOffer(db, notifSvc, o.ID)
	}
}
