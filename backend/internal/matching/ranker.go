package matching

import (
	"math"
	"sort"
	"time"
)

// TripCandidate is a de-normalised trip enriched with scoring signals.
type TripCandidate struct {
	TripID       string    `json:"trip_id"`
	TravelerID   string    `json:"traveler_id"`
	TravelerName string    `json:"traveler_name"`
	Origin       string    `json:"origin"`
	Destination  string    `json:"destination"`
	DepartureAt  time.Time `json:"departure_at"`
	ArrivalAt    time.Time `json:"arrival_at"`
	AvailableKg  float64   `json:"available_kg"`
	PricePerKg   float64   `json:"price_per_kg"`
	BasePrice    float64   `json:"base_price"`

	// Scoring signals (populated by service before ranking)
	TravelerGeoScore float64 `json:"traveler_geo_score"` // 0–100
}

// MatchResult holds a ranked trip with its score breakdown.
type MatchResult struct {
	TripCandidate
	Score           float64 `json:"score"`            // 0–100
	RouteMatchScore float64 `json:"route_match_score"` // 0–1
	GeoScoreComp    float64 `json:"geo_score_component"`
	PriceFitScore   float64 `json:"price_fit_score"`
	SpeedScore      float64 `json:"speed_score"`
	EstimatedCost   float64 `json:"estimated_cost"`
}

// OrderContext carries the signals from the delivery order used for ranking.
type OrderContext struct {
	Origin      string
	Destination string
	WeightKg    float64
	MaxBudget   float64   // 0 = no limit
	Deadline    time.Time // zero = no deadline
}

// Score computes the matching score for a single trip candidate.
//
// Formula (weights sum to 1.0):
//
//	score = routeMatch   * 0.40
//	      + geoScore     * 0.25   (traveler GeoScore / 100)
//	      + priceFit     * 0.20
//	      + deliverySpeed * 0.15
//
// Returns a value in [0, 100].
func Score(order OrderContext, trip TripCandidate) MatchResult {
	// ── Route match (binary: does the trip cover the required route?) ─────────
	routeMatch := 0.0
	if sameRoute(trip.Origin, order.Origin) && sameRoute(trip.Destination, order.Destination) {
		routeMatch = 1.0
	} else if sameRoute(trip.Origin, order.Origin) || sameRoute(trip.Destination, order.Destination) {
		routeMatch = 0.5 // partial match
	}

	// ── GeoScore component ────────────────────────────────────────────────────
	geoComp := clamp(trip.TravelerGeoScore, 0, 100) / 100.0

	// ── Price fit ─────────────────────────────────────────────────────────────
	estimatedCost := trip.PricePerKg*order.WeightKg + trip.BasePrice
	priceFit := 1.0
	if order.MaxBudget > 0 && estimatedCost > 0 {
		priceFit = math.Max(0, 1.0-math.Max(0, estimatedCost-order.MaxBudget)/order.MaxBudget)
	} else if estimatedCost > 0 {
		// No budget — price fit decreases with price, normalised to $100 reference
		priceFit = math.Max(0, 1.0-estimatedCost/100.0)
	}
	priceFit = clamp01(priceFit)

	// ── Delivery speed ────────────────────────────────────────────────────────
	speedScore := 0.5 // neutral default
	if !order.Deadline.IsZero() && !trip.ArrivalAt.IsZero() {
		timeToDeadline := order.Deadline.Sub(time.Now()).Hours()
		timeToArrival := trip.ArrivalAt.Sub(time.Now()).Hours()
		if timeToDeadline > 0 && timeToArrival < timeToDeadline {
			speedScore = 1.0 - math.Min(1.0, timeToArrival/timeToDeadline)
		} else if timeToArrival <= 0 {
			speedScore = 0
		}
	} else if !trip.ArrivalAt.IsZero() {
		days := trip.ArrivalAt.Sub(time.Now()).Hours() / 24
		speedScore = math.Max(0, 1.0-days/30.0) // normalised over 30 days
	}
	speedScore = clamp01(speedScore)

	raw := routeMatch*0.40 + geoComp*0.25 + priceFit*0.20 + speedScore*0.15
	finalScore := math.Round(clamp01(raw)*10000) / 100

	return MatchResult{
		TripCandidate:   trip,
		Score:           finalScore,
		RouteMatchScore: math.Round(routeMatch*100) / 100,
		GeoScoreComp:    math.Round(geoComp*100) / 100,
		PriceFitScore:   math.Round(priceFit*100) / 100,
		SpeedScore:      math.Round(speedScore*100) / 100,
		EstimatedCost:   math.Round(estimatedCost*100) / 100,
	}
}

// RankTrips scores all candidates and returns them sorted by score descending.
func RankTrips(order OrderContext, candidates []TripCandidate) []MatchResult {
	results := make([]MatchResult, 0, len(candidates))
	for _, c := range candidates {
		results = append(results, Score(order, c))
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	return results
}

func sameRoute(a, b string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	// Case-insensitive prefix match (city codes like "DXB" or full city names)
	if len(a) > len(b) {
		a, b = b, a
	}
	for i := range a {
		if a[i]|0x20 != b[i]|0x20 {
			return false
		}
	}
	return true
}

func clamp01(v float64) float64 { return clamp(v, 0, 1) }
func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
