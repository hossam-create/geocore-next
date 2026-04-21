package crowdshipping

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PriceSuggestion struct {
	MinCents          int64 `json:"min_cents"`
	RecommendedCents  int64 `json:"recommended_cents"`
	MaxCents          int64 `json:"max_cents"`
}

// SuggestOfferPrice uses corridor pricing + historical offers + item value
// to suggest min/recommended/max offer prices for a delivery request.
func SuggestOfferPrice(db *gorm.DB, requestID uuid.UUID) (PriceSuggestion, error) {
	var dr DeliveryRequest
	if err := db.Where("id=?", requestID).First(&dr).Error; err != nil {
		return PriceSuggestion{}, fmt.Errorf("request not found: %w", err)
	}

	// 1. Corridor-based baseline
	baseline := CalculateDeliveryPrice(PricingParams{
		WeightKg:    floatVal(dr.ItemWeight, 1.0),
		DistanceKm:  100, // default estimate; corridor config handles real rates
		Urgency:     UrgencyStandard,
		ItemType:    ItemTypeOther,
		ItemValue:   dr.ItemPrice,
		Origin:      dr.PickupCountry,
		Destination: dr.DeliveryCountry,
	})

	// 2. Historical offer average for this corridor
	var histAvg float64
	db.Model(&TravelerOffer{}).
		Where("delivery_request_id IN (SELECT id FROM delivery_requests WHERE pickup_country=? AND delivery_country=?)",
			dr.PickupCountry, dr.DeliveryCountry).
		Where("status IN ?", []OfferStatus{OfferAccepted, OfferFundsHeld, OfferCompleted}).
		Select("COALESCE(AVG(price), 0)").
		Row().Scan(&histAvg)

	// 3. Item value cap (buyer won't pay more than ~30% of item value for delivery)
	itemCapCents := toCents(dr.ItemPrice * 0.30)

	// Recommended = max of corridor baseline and historical average
	recCents := baseline.TotalCents
	if histAvg > 0 {
		histCents := toCents(histAvg)
		// Weighted: 60% corridor, 40% historical
		recCents = (recCents*60 + histCents*40) / 100
	}

	// Clamp recommended to item cap
	if recCents > itemCapCents && itemCapCents > 0 {
		recCents = itemCapCents
	}

	// Min = 70% of recommended (traveler-friendly floor)
	minCents := recCents * 70 / 100
	// Max = buyer reward or 130% of recommended (whichever is lower)
	maxCents := recCents * 130 / 100
	rewardCents := toCents(dr.Reward)
	if rewardCents > 0 && rewardCents < maxCents {
		maxCents = rewardCents
	}

	return PriceSuggestion{
		MinCents:         minCents,
		RecommendedCents: recCents,
		MaxCents:         maxCents,
	}, nil
}

func floatVal(p *float64, def float64) float64 {
	if p == nil || *p <= 0 {
		return def
	}
	return *p
}
