package crowdshipping

import (
	"context"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ExpireStaleOffers marks expired offers and sends notifications.
func ExpireStaleOffers(db *gorm.DB, notifSvc *notifications.Service) {
	var offers []TravelerOffer
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status IN ? AND expires_at < ?", []OfferStatus{OfferPending, OfferCountered}, time.Now()).
		Find(&offers).Error; err != nil {
		slog.Error("crowdshipping: expire stale offers query failed", "error", err.Error())
		return
	}
	if len(offers) == 0 {
		return
	}

	offerIDs := make([]uuid.UUID, len(offers))
	for i, o := range offers {
		offerIDs[i] = o.ID
	}
	db.Model(&TravelerOffer{}).Where("id IN ?", offerIDs).Update("status", OfferExpired)

	for _, o := range offers {
		if notifSvc != nil {
			go notifSvc.Notify(notifications.NotifyInput{
				UserID: o.BuyerID, Type: "offer_expired", Title: "Offer Expired",
				Body: "An offer on your delivery request has expired",
				Data: map[string]string{"offer_id": o.ID.String()},
			})
			go notifSvc.Notify(notifications.NotifyInput{
				UserID: o.TravelerID, Type: "offer_expired", Title: "Offer Expired",
				Body: "Your offer has expired",
				Data: map[string]string{"offer_id": o.ID.String()},
			})
		}
	}
	slog.Info("crowdshipping: expired stale offers", "count", len(offers))
}

// StartOfferExpiryScheduler runs ExpireStaleOffers every 5 minutes.
func StartOfferExpiryScheduler(ctx context.Context, db *gorm.DB, notifSvc *notifications.Service) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("offer_expiry_scheduler: panic recovered", "panic", r)
			}
		}()
		for {
			select {
			case <-ticker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							slog.Error("offer_expiry: tick panic recovered", "panic", r)
						}
					}()
					ExpireStaleOffers(db, notifSvc)
				}()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}
