package listings

import (
	"fmt"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UrgencySignals struct {
	ListingID           uuid.UUID `json:"listing_id"`
	ViewsToday          int64     `json:"views_today"`
	UniqueViewsToday    int64     `json:"unique_views_today"`
	ActiveOffers        int64     `json:"active_offers"`
	TravelersInterested int64     `json:"travelers_interested"`
	ExpiresIn           string    `json:"expires_in,omitempty"`
	IsUrgent            bool      `json:"is_urgent"`
	IsVerifiedSignal    bool      `json:"is_verified_signal"`
}

// GetUrgencySignals computes real-time urgency signals for a listing.
func GetUrgencySignals(db *gorm.DB, listingID uuid.UUID) UrgencySignals {
	var s UrgencySignals
	s.ListingID = listingID

	// Views today (total)
	db.Model(&ListingView{}).
		Where("listing_id=? AND viewed_at>?", listingID, time.Now().Truncate(24*time.Hour)).
		Count(&s.ViewsToday)

	// Unique views today (distinct users/IPs — anti-manipulation)
	db.Model(&ListingView{}).
		Where("listing_id=? AND viewed_at>?", listingID, time.Now().Truncate(24*time.Hour)).
		Distinct("COALESCE(viewer_id::text, ip_hash)").
		Count(&s.UniqueViewsToday)

	// Active offers (only VALID: pending/countered, not rejected/expired)
	db.Table("negotiation_threads").
		Where("listing_id=? AND status IN ?", listingID, []string{"pending", "countered"}).
		Count(&s.ActiveOffers)

	// Travelers interested (only valid crowdshipping offers)
	db.Table("traveler_offers").
		Where("status IN ?", []string{"pending", "payment_pending", "funds_held", "accepted"}).
		Count(&s.TravelersInterested)

	// Expiry
	var listing Listing
	if err := db.Where("id=?", listingID).First(&listing).Error; err == nil {
		if listing.ExpiresAt != nil {
			remaining := time.Until(*listing.ExpiresAt)
			if remaining > 0 {
				if remaining < time.Hour {
					s.ExpiresIn = "less than 1h"
				} else if remaining < 24*time.Hour {
					s.ExpiresIn = formatDuration(remaining)
				} else {
					s.ExpiresIn = formatDuration(remaining)
				}
			}
		}
	}

	// Is urgent: unique views > 5 OR valid offers > 0 OR expires within 6h
	s.IsUrgent = s.UniqueViewsToday > 5 || s.ActiveOffers > 0
	if listing.ExpiresAt != nil {
		s.IsUrgent = s.IsUrgent || time.Until(*listing.ExpiresAt) < 6*time.Hour
	}

	// Verified signal: unique views > 0 AND no signs of manipulation
	// (i.e. unique views >= 50% of total views)
	s.IsVerifiedSignal = s.ViewsToday > 0 && s.UniqueViewsToday >= s.ViewsToday/2

	return s
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	if h >= 24 {
		days := h / 24
		return plural(days, "day")
	}
	return plural(h, "h")
}

func plural(n int, unit string) string {
	s := fmt.Sprintf("%d%s", n, unit)
	if n != 1 {
		s += "s"
	}
	return s
}

// ── HTTP Handler ──────────────────────────────────────────────────────────────

func (h *Handler) GetUrgency(c *gin.Context) {
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid listing id")
		return
	}
	signals := GetUrgencySignals(h.dbRead, listingID)
	response.OK(c, signals)
}
