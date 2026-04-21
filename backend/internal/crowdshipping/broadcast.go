package crowdshipping

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const broadcastCooldownHours = 24

// BroadcastRequest finds all matching travelers and batch-notifies them.
// Cooldown: 1 notification per traveler per 24h per request.
func BroadcastRequest(db *gorm.DB, notifSvc *notifications.Service, requestID uuid.UUID) {
	matches, err := FindBestTravelersForRequest(db, requestID, 50)
	if err != nil {
		slog.Error("broadcast: match failed", "error", err)
		return
	}
	if len(matches) == 0 {
		return
	}

	var dr DeliveryRequest
	if err := db.Where("id=?", requestID).First(&dr).Error; err != nil {
		return
	}

	notified := 0
	cutoff := time.Now().Add(-broadcastCooldownHours * time.Hour)
	for _, m := range matches {
		// Check cooldown: has this traveler been notified about this request recently?
		var cnt int64
		db.Model(&BroadcastLog{}).
			Where("traveler_id=? AND request_id=? AND created_at>?", m.TravelerID, requestID, cutoff).
			Count(&cnt)
		if cnt > 0 {
			continue // still in cooldown
		}

		if notifSvc != nil {
			go notifSvc.Notify(notifications.NotifyInput{
				UserID: m.TravelerID,
				Type:   "delivery_broadcast",
				Title:  "New Delivery Request in Your Area",
				Body:   "A buyer needs something delivered on your route — check it out",
				Data: map[string]string{
					"request_id": requestID.String(),
					"route":      dr.PickupCity + " → " + dr.DeliveryCity,
				},
			})
		}

		// Log the broadcast for cooldown tracking
		db.Create(&BroadcastLog{
			TravelerID: m.TravelerID,
			RequestID:  requestID,
		})
		notified++
	}
	slog.Info("broadcast: notified", "request_id", requestID, "notified", notified, "candidates", len(matches))
}

// BroadcastLog tracks notification cooldown per traveler/request pair.
type BroadcastLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TravelerID uuid.UUID `gorm:"type:uuid;not null;index" json:"traveler_id"`
	RequestID  uuid.UUID `gorm:"type:uuid;not null;index" json:"request_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (BroadcastLog) TableName() string { return "broadcast_logs" }
