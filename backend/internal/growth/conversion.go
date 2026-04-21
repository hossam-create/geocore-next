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
// Conversion Optimization
// Conversion signals, smart nudges, auto-discount for stale listings.
// ════════════════════════════════════════════════════════════════════════════

// ConversionSignal represents a social proof signal shown to users.
type ConversionSignal struct {
	Type      string           `json:"type"`      // travelers_interested, offers_received, price_dropped
	EntityID  uuid.UUID        `json:"entity_id"` // listing or request ID
	Count     int              `json:"count"`     // "3 travelers interested"
	OldPrice  *decimal.Decimal `json:"old_price,omitempty"`
	NewPrice  *decimal.Decimal `json:"new_price,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
}

// StaleListing tracks listings that have been active too long without offers.
type StaleListing struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID      uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex" json:"listing_id"`
	OriginalPrice  decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"original_price"`
	SuggestedPrice decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"suggested_price"`
	DiscountPct    decimal.Decimal `gorm:"type:decimal(5,2);not null" json:"discount_pct"`
	DaysStale      int             `gorm:"not null" json:"days_stale"`
	NotifiedAt     *time.Time      `json:"notified_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

func (StaleListing) TableName() string { return "stale_listings" }

// GetConversionSignals generates social proof signals for a delivery request.
func GetConversionSignals(db *gorm.DB, requestID uuid.UUID) []ConversionSignal {
	var signals []ConversionSignal

	// Count travelers who viewed/matched
	var travelerCount int64
	db.Table("traveler_offers").
		Where("delivery_request_id = ? AND is_auto_generated = ? AND deleted_at IS NULL",
			requestID, true).
		Count(&travelerCount)
	if travelerCount > 0 {
		signals = append(signals, ConversionSignal{
			Type:     "travelers_interested",
			EntityID: requestID,
			Count:    int(travelerCount),
		})
	}

	// Count real offers received
	var offerCount int64
	db.Table("traveler_offers").
		Where("delivery_request_id = ? AND is_auto_generated = ? AND deleted_at IS NULL",
			requestID, false).
		Count(&offerCount)
	if offerCount > 0 {
		signals = append(signals, ConversionSignal{
			Type:     "offers_received",
			EntityID: requestID,
			Count:    int(offerCount),
		})
	}

	return signals
}

// SendSmartNudge notifies a buyer when an offer is close to their budget.
func SendSmartNudge(db *gorm.DB, notifSvc *notifications.Service, buyerID uuid.UUID, offerPrice, budget float64, requestID uuid.UUID) {
	if notifSvc == nil {
		return
	}

	// Only nudge if offer is within 20% of budget
	diff := (budget - offerPrice) / budget
	if diff >= 0 && diff <= 0.20 {
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: buyerID,
			Type:   "smart_nudge",
			Title:  "Offer Close to Budget!",
			Body:   fmt.Sprintf("An offer of $%.2f is close to your budget of $%.2f", offerPrice, budget),
			Data:   map[string]string{"request_id": requestID.String()},
		})
	}
}

// NotifyTravelerBuyerActive notifies travelers when a buyer is actively looking.
func NotifyTravelerBuyerActive(db *gorm.DB, notifSvc *notifications.Service, buyerID uuid.UUID, route string) {
	if notifSvc == nil {
		return
	}

	// Find travelers with active trips on this route
	var travelerIDs []uuid.UUID
	db.Table("trips t").
		Joins("JOIN users u ON u.id = t.traveler_id").
		Where("t.status = ? AND t.origin_country || '→' || t.dest_country = ?", "active", route).
		Pluck("t.traveler_id", &travelerIDs)

	for _, tid := range travelerIDs {
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: tid,
			Type:   "buyer_active",
			Title:  "Active Buyer on Your Route!",
			Body:   fmt.Sprintf("A buyer is actively looking for delivery on your route: %s", route),
			Data:   map[string]string{"buyer_id": buyerID.String(), "route": route},
		})
	}
}

// DetectStaleListings finds listings active > 3 days without offers and suggests price drops.
func DetectStaleListings(db *gorm.DB, notifSvc *notifications.Service) error {
	cutoff := time.Now().AddDate(0, 0, -3)

	// Find listings with no offers in the last 3 days
	type staleRow struct {
		ID        uuid.UUID
		UserID    uuid.UUID
		Price     decimal.Decimal
		Title     string
		CreatedAt time.Time
	}

	var stale []staleRow
	db.Table("listings l").
		Select("l.id, l.user_id, l.price, l.title, l.created_at").
		Where("l.status = ? AND l.created_at <= ? AND l.deleted_at IS NULL", "active", cutoff).
		Where("NOT EXISTS (SELECT 1 FROM traveler_offers o WHERE o.delivery_request_id IN (SELECT dr.id FROM delivery_requests dr WHERE dr.item_name = l.title) AND o.deleted_at IS NULL)").
		Limit(100).
		Find(&stale)

	for _, s := range stale {
		daysStale := int(time.Since(s.CreatedAt).Hours() / 24)
		discountPct := decimal.NewFromFloat(0.10) // 10% suggested discount
		if daysStale > 7 {
			discountPct = decimal.NewFromFloat(0.20) // 20% after a week
		}

		suggestedPrice := s.Price.Mul(decimal.NewFromInt(1).Sub(discountPct))

		staleEntry := StaleListing{
			ID:             uuid.New(),
			ListingID:      s.ID,
			OriginalPrice:  s.Price,
			SuggestedPrice: suggestedPrice,
			DiscountPct:    discountPct.Mul(decimal.NewFromInt(100)),
			DaysStale:      daysStale,
		}

		// Upsert stale listing record
		var existing StaleListing
		if db.Where("listing_id = ?", s.ID).First(&existing).Error != nil {
			db.Create(&staleEntry)
		} else {
			db.Model(&existing).Updates(map[string]interface{}{
				"suggested_price": suggestedPrice,
				"discount_pct":    discountPct.Mul(decimal.NewFromInt(100)),
				"days_stale":      daysStale,
			})
		}

		// Notify seller about price suggestion
		if notifSvc != nil {
			go notifSvc.Notify(notifications.NotifyInput{
				UserID: s.UserID,
				Type:   "price_suggestion",
				Title:  "Boost Your Listing!",
				Body:   fmt.Sprintf("Your listing '%s' has been active for %d days. Consider a %s%% price drop to $%s?", s.Title, daysStale, discountPct.Mul(decimal.NewFromInt(100)).StringFixed(0), suggestedPrice.StringFixed(2)),
				Data:   map[string]string{"listing_id": s.ID.String(), "suggested_price": suggestedPrice.StringFixed(2)},
			})
		}
	}

	if len(stale) > 0 {
		slog.Info("conversion: detected stale listings", "count", len(stale))
	}
	return nil
}

// GetConversionStats returns overall conversion metrics.
func GetConversionStats(db *gorm.DB) map[string]interface{} {
	var totalRequests int64
	var requestsWithOffers int64
	var requestsCompleted int64

	db.Table("delivery_requests").Where("deleted_at IS NULL").Count(&totalRequests)
	db.Table("delivery_requests dr").
		Joins("JOIN traveler_offers o ON o.delivery_request_id = dr.id AND o.deleted_at IS NULL").
		Where("dr.deleted_at IS NULL").
		Distinct("dr.id").Count(&requestsWithOffers)
	db.Table("delivery_requests").Where("status = ? AND deleted_at IS NULL", "delivered").Count(&requestsCompleted)

	conversionRate := decimal.Zero
	if totalRequests > 0 {
		conversionRate = decimal.NewFromInt(requestsCompleted).Div(decimal.NewFromInt(totalRequests)).Mul(decimal.NewFromInt(100))
	}

	return map[string]interface{}{
		"total_requests":       totalRequests,
		"requests_with_offers": requestsWithOffers,
		"requests_completed":   requestsCompleted,
		"conversion_rate":      conversionRate.StringFixed(1) + "%",
	}
}
