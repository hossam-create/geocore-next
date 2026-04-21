package crowdshipping

import (
	"math"
	"strings"
	"time"
)

// MatchResult holds a trip and its computed match score for a delivery request.
type MatchResult struct {
	Trip              Trip    `json:"trip"`
	MatchScore        float64 `json:"match_score"`
	TravelerScore     float64 `json:"traveler_reputation_score"` // 0–100
	EstimatedCost     float64 `json:"estimated_cost"`
	EstimatedDelivery string  `json:"estimated_delivery"`
	CanDeliver        bool    `json:"can_deliver"`
}

// CalculateMatchScore computes weighted score:
// score = weight_fit*0.35 + price_score*0.25 + time_match*0.25 + reputation*0.15
// Each component is normalized to [0,1].
// travelerReputation is 0–100 from pkg/reputation.
func CalculateMatchScore(req *DeliveryRequest, trip *Trip, travelerReputation ...float64) float64 {
	weightFit := 0.0
	if req.ItemWeight == nil || *req.ItemWeight <= 0 {
		weightFit = 1
	} else if trip.AvailableWeight > 0 {
		weightFit = math.Min(1, trip.AvailableWeight/(*req.ItemWeight))
	}

	priceScore := 1.0
	if req.ItemWeight != nil && *req.ItemWeight > 0 {
		estimated := trip.PricePerKg**req.ItemWeight + trip.BasePrice
		if req.Reward > 0 {
			priceScore = 1 - math.Min(1, estimated/req.Reward)
		}
	}

	timeMatch := 0.0
	if req.Deadline == nil {
		timeMatch = 1
	} else {
		totalWindow := time.Until(*req.Deadline).Hours()
		if totalWindow <= 0 {
			timeMatch = 0
		} else {
			untilDeparture := time.Until(trip.DepartureDate).Hours()
			timeMatch = 1 - math.Min(1, math.Max(0, untilDeparture/totalWindow))
		}
	}

	// Reputation component (optional, defaults to 50 = neutral)
	reputationScore := 50.0
	if len(travelerReputation) > 0 {
		reputationScore = math.Max(0, math.Min(100, travelerReputation[0]))
	}
	reputationComp := reputationScore / 100.0

	score := (weightFit * 0.35) + (priceScore * 0.25) + (timeMatch * 0.25) + (reputationComp * 0.15)

	if strings.EqualFold(trip.OriginCity, req.PickupCity) && strings.EqualFold(trip.DestCity, req.DeliveryCity) {
		score += 0.1
	}

	return math.Max(0, math.Min(1, score)) * 100
}

// Haversine returns the great-circle distance in km between two lat/lon points.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func toRad(deg float64) float64 {
	return deg * (math.Pi / 180)
}
