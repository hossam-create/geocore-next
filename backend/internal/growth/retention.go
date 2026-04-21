package growth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Retention Engine
// Price drop alerts, new offer notifications, weekly digest, re-engagement.
// ════════════════════════════════════════════════════════════════════════════

// RetentionEvent tracks a retention action sent to a user.
type RetentionEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Type      string    `gorm:"size:30;not null;index" json:"type"` // price_drop, new_offer, match_found, digest, reengage
	EntityID  *uuid.UUID `gorm:"type:uuid" json:"entity_id,omitempty"`
	SentAt    time.Time `json:"sent_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (RetentionEvent) TableName() string { return "retention_events" }

// Notification throttle: max N retention notifications per user per day
const maxRetentionNotificationsPerDay = 3

// canSendRetentionNotification checks if a user hasn't exceeded the daily notification limit.
func canSendRetentionNotification(db *gorm.DB, userID uuid.UUID) bool {
	var count int64
	since := time.Now().AddDate(0, 0, -1)
	db.Model(&RetentionEvent{}).Where("user_id = ? AND sent_at >= ?", userID, since).Count(&count)
	return count < maxRetentionNotificationsPerDay
}

// recordRetentionEvent logs a retention notification.
func recordRetentionEvent(db *gorm.DB, userID uuid.UUID, notifType string, entityID *uuid.UUID) {
	event := RetentionEvent{
		ID:     uuid.New(),
		UserID: userID,
		Type:   notifType,
		EntityID: entityID,
		SentAt: time.Now(),
	}
	db.Create(&event)
}

// NotifyPriceDrop sends a price drop alert to users watching a listing.
func NotifyPriceDrop(db *gorm.DB, notifSvc *notifications.Service, listingID uuid.UUID, oldPrice, newPrice decimal.Decimal) {
	if notifSvc == nil {
		return
	}

	// Find users who favorited this listing
	var userIDs []uuid.UUID
	db.Table("favorites").Where("listing_id = ?", listingID).
		Pluck("user_id", &userIDs)

	for _, uid := range userIDs {
		if !canSendRetentionNotification(db, uid) {
			continue
		}

		go notifSvc.Notify(notifications.NotifyInput{
			UserID: uid,
			Type:   "price_drop",
			Title:  "Price Dropped!",
			Body:   fmt.Sprintf("A listing you saved dropped from $%s to $%s", oldPrice.StringFixed(2), newPrice.StringFixed(2)),
			Data:   map[string]string{"listing_id": listingID.String()},
		})
		recordRetentionEvent(db, uid, "price_drop", &listingID)
	}
}

// NotifyNewOffer alerts a buyer when a new offer arrives on their request.
func NotifyNewOffer(db *gorm.DB, notifSvc *notifications.Service, buyerID, requestID uuid.UUID, offerPrice float64) {
	if notifSvc == nil {
		return
	}
	if !canSendRetentionNotification(db, buyerID) {
		return
	}

	go notifSvc.Notify(notifications.NotifyInput{
		UserID: buyerID,
		Type:   "new_offer",
		Title:  "New Offer Received!",
		Body:   fmt.Sprintf("You received a new offer of $%.2f", offerPrice),
		Data:   map[string]string{"request_id": requestID.String()},
	})
	recordRetentionEvent(db, buyerID, "new_offer", &requestID)
}

// NotifyMatchFound alerts a user when a match is found for their request/trip.
func NotifyMatchFound(db *gorm.DB, notifSvc *notifications.Service, userID uuid.UUID, matchType string, entityID uuid.UUID) {
	if notifSvc == nil {
		return
	}
	if !canSendRetentionNotification(db, userID) {
		return
	}

	go notifSvc.Notify(notifications.NotifyInput{
		UserID: userID,
		Type:   "match_found",
		Title:  "Match Found!",
		Body:   fmt.Sprintf("We found a %s match for you!", matchType),
		Data:   map[string]string{"entity_id": entityID.String(), "match_type": matchType},
	})
	recordRetentionEvent(db, userID, "match_found", &entityID)
}

// WeeklyDigestData holds curated deals for a user's weekly digest.
type WeeklyDigestData struct {
	UserID       uuid.UUID     `json:"user_id"`
	BestDeals    []DigestDeal  `json:"best_deals"`
	Recommended  []DigestDeal  `json:"recommended"`
	ActiveTrips  int           `json:"active_trips"`
}

// DigestDeal represents a deal in the weekly digest.
type DigestDeal struct {
	ID    uuid.UUID       `json:"id"`
	Title string          `json:"title"`
	Price decimal.Decimal `json:"price"`
	Route string          `json:"route"`
}

// SendWeeklyDigest sends a weekly digest to active users.
func SendWeeklyDigest(db *gorm.DB, notifSvc *notifications.Service) error {
	if notifSvc == nil {
		return nil
	}

	// Find users active in the last 30 days
	var activeUsers []uuid.UUID
	db.Table("users").
		Where("is_active = ? AND updated_at >= ?", true, time.Now().AddDate(0, 0, -30)).
		Limit(500). // batch limit
		Pluck("id", &activeUsers)

	// Get best deals (recent listings with price drops)
	var bestDeals []DigestDeal
	db.Table("listings").
		Select("id, title, price, pickup_country || '→' || delivery_country as route").
		Where("status = ? AND deleted_at IS NULL AND created_at >= ?", "active", time.Now().AddDate(0, 0, -7)).
		Order("price ASC").Limit(5).
		Find(&bestDeals)

	sentCount := 0
	for _, uid := range activeUsers {
		if !canSendRetentionNotification(db, uid) {
			continue
		}

		dealText := ""
		for i, d := range bestDeals {
			if i > 0 {
				dealText += ", "
			}
			dealText += fmt.Sprintf("%s ($%s)", d.Title, d.Price.StringFixed(2))
		}

		go notifSvc.Notify(notifications.NotifyInput{
			UserID: uid,
			Type:   "weekly_digest",
			Title:  "Weekly Best Deals",
			Body:   fmt.Sprintf("This week's top deals: %s", dealText),
			Data:   map[string]string{"type": "weekly_digest"},
		})
		recordRetentionEvent(db, uid, "digest", nil)
		sentCount++
	}

	if sentCount > 0 {
		slog.Info("retention: weekly digest sent", "count", sentCount)
	}
	return nil
}

// ReEngageInactiveUsers sends targeted notifications to inactive users.
func ReEngageInactiveUsers(db *gorm.DB, notifSvc *notifications.Service) error {
	if notifSvc == nil {
		return nil
	}

	// Find users inactive for 7+ days
	var inactiveUsers []uuid.UUID
	db.Table("users").
		Where("is_active = ? AND updated_at <= ?", true, time.Now().AddDate(0, 0, -7)).
		Limit(200).
		Pluck("id", &inactiveUsers)

	// Get recent popular listings
	var recentListings []DigestDeal
	db.Table("listings").
		Select("id, title, price, pickup_country || '→' || delivery_country as route").
		Where("status = ? AND deleted_at IS NULL AND created_at >= ?", "active", time.Now().AddDate(0, 0, -3)).
		Order("view_count DESC").Limit(3).
		Find(&recentListings)

	sentCount := 0
	for _, uid := range inactiveUsers {
		if !canSendRetentionNotification(db, uid) {
			continue
		}

		go notifSvc.Notify(notifications.NotifyInput{
			UserID: uid,
			Type:   "reengage",
			Title:  "We Miss You!",
			Body:   "New listings are waiting for you. Check out what's new on GeoCore!",
			Data:   map[string]string{"type": "reengage"},
		})
		recordRetentionEvent(db, uid, "reengage", nil)
		sentCount++
	}

	if sentCount > 0 {
		slog.Info("retention: re-engagement sent", "count", sentCount)
	}
	return nil
}
