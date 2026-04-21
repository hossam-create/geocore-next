package crowdshipping

import (
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LiquidityStats struct {
	RequestID           string  `json:"request_id"`
	OffersPerRequest    int     `json:"offers_per_request"`
	TimeToFirstOfferMin float64 `json:"time_to_first_offer_min"`
	AutoOfferAcceptRate float64 `json:"auto_offer_accept_rate"`
	MatchSuccessRate    float64 `json:"match_success_rate"`
	AutoOffersCount     int     `json:"auto_offers_count"`
	ManualOffersCount   int     `json:"manual_offers_count"`
}

// GetLiquidityStats computes liquidity metrics for a delivery request.
func GetLiquidityStats(db *gorm.DB, requestID uuid.UUID) LiquidityStats {
	var s LiquidityStats
	s.RequestID = requestID.String()

	// Total offers
	var totalOffers int64
	db.Model(&TravelerOffer{}).Where("delivery_request_id=?", requestID).Count(&totalOffers)
	s.OffersPerRequest = int(totalOffers)

	// Auto vs manual
	var autoCount int64
	db.Model(&TravelerOffer{}).Where("delivery_request_id=? AND is_auto_generated=?", requestID, true).Count(&autoCount)
	s.AutoOffersCount = int(autoCount)
	s.ManualOffersCount = int(totalOffers) - int(autoCount)

	// Time to first offer
	var dr DeliveryRequest
	if err := db.Where("id=?", requestID).First(&dr).Error; err == nil {
		var firstOffer TravelerOffer
		if err := db.Where("delivery_request_id=?", requestID).
			Order("created_at ASC").First(&firstOffer).Error; err == nil {
			s.TimeToFirstOfferMin = firstOffer.CreatedAt.Sub(dr.CreatedAt).Minutes()
		}
	}

	// Auto-offer accept rate
	if autoCount > 0 {
		var acceptedAuto int64
		db.Model(&TravelerOffer{}).
			Where("delivery_request_id=? AND is_auto_generated=? AND status IN ?",
				requestID, true, []OfferStatus{OfferAccepted, OfferFundsHeld, OfferCompleted}).
			Count(&acceptedAuto)
		s.AutoOfferAcceptRate = float64(acceptedAuto) / float64(autoCount) * 100
	}

	// Match success rate: accepted offers / total matched travelers
	var matches []TravelerMatch
	matches, _ = FindBestTravelersForRequest(db, requestID, 50)
	if len(matches) > 0 {
		var acceptedTotal int64
		db.Model(&TravelerOffer{}).
			Where("delivery_request_id=? AND status IN ?",
				requestID, []OfferStatus{OfferAccepted, OfferFundsHeld, OfferCompleted}).
			Count(&acceptedTotal)
		s.MatchSuccessRate = float64(acceptedTotal) / float64(len(matches)) * 100
	}

	return s
}

// RecordTimeToFirstOffer records the metric for monitoring.
func RecordTimeToFirstOffer(db *gorm.DB, requestID uuid.UUID) {
	stats := GetLiquidityStats(db, requestID)
	slog.Info("liquidity: time_to_first_offer",
		"request_id", requestID,
		"minutes", stats.TimeToFirstOfferMin,
		"offers", stats.OffersPerRequest,
		"auto", stats.AutoOffersCount,
	)
}
