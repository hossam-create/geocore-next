package crowdshipping

import (
	"log/slog"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TriggerLiquidityEngine is called after a new DeliveryRequest is created.
// It fires auto-match, auto-offer, and broadcast in goroutines.
func TriggerLiquidityEngine(db *gorm.DB, notifSvc *notifications.Service, requestID uuid.UUID) {
	slog.Info("liquidity: triggering engine", "request_id", requestID)

	go AutoNotifyTopTravelers(db, notifSvc, requestID)
	go func() {
		if err := GenerateAutoOffers(db, notifSvc, requestID); err != nil {
			slog.Error("liquidity: auto-offer failed", "error", err)
		}
		// Try auto-close on generated offers
		TryAutoCloseOffer(db, notifSvc, requestID)
	}()
	go BroadcastRequest(db, notifSvc, requestID)
}
