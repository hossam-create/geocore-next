package matching_test

import (
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/matching"
)

var baseOrder = matching.OrderContext{
	Origin:      "DXB",
	Destination: "CAI",
	WeightKg:    5.0,
	MaxBudget:   100.0,
	Deadline:    time.Now().Add(7 * 24 * time.Hour),
}

func trip(origin, dest string, geoScore, pricePerKg float64, arrivalDays int) matching.TripCandidate {
	return matching.TripCandidate{
		TripID:           "t1",
		TravelerID:       "u1",
		Origin:           origin,
		Destination:      dest,
		DepartureAt:      time.Now().Add(24 * time.Hour),
		ArrivalAt:        time.Now().Add(time.Duration(arrivalDays) * 24 * time.Hour),
		AvailableKg:      20.0,
		PricePerKg:       pricePerKg,
		BasePrice:        0,
		TravelerGeoScore: geoScore,
	}
}

func TestScore_PerfectMatch(t *testing.T) {
	c := trip("DXB", "CAI", 100.0, 5.0, 3)
	r := matching.Score(baseOrder, c)
	// routeMatch=1.0, geo=1.0, priceFit~1.0 (5*5=25 << 100 budget), speed~high
	if r.Score <= 80.0 {
		t.Errorf("perfect match should score > 80, got %v", r.Score)
	}
}

func TestScore_WrongRoute(t *testing.T) {
	c := trip("LHR", "JFK", 100.0, 5.0, 3)
	r := matching.Score(baseOrder, c)
	// routeMatch=0.0, so max possible = 0.25+0.20+0.15 = 0.60 → 60
	if r.Score >= 70.0 {
		t.Errorf("wrong route should score < 70, got %v", r.Score)
	}
}

func TestScore_LowGeoScore(t *testing.T) {
	perfect := trip("DXB", "CAI", 100.0, 5.0, 3)
	lowTrust := trip("DXB", "CAI", 0.0, 5.0, 3)
	rPerfect := matching.Score(baseOrder, perfect)
	rLow := matching.Score(baseOrder, lowTrust)
	if rPerfect.Score <= rLow.Score {
		t.Errorf("high GeoScore should rank higher: perfect=%v, low=%v", rPerfect.Score, rLow.Score)
	}
}

func TestRankTrips_OrderedByScore(t *testing.T) {
	candidates := []matching.TripCandidate{
		trip("LHR", "JFK", 50.0, 20.0, 15), // bad route, expensive
		trip("DXB", "CAI", 90.0, 5.0, 3),   // good match
		trip("DXB", "CAI", 40.0, 8.0, 5),   // good route, low trust
	}
	ranked := matching.RankTrips(baseOrder, candidates)

	if len(ranked) != 3 {
		t.Fatalf("expected 3 results, got %d", len(ranked))
	}
	for i := 1; i < len(ranked); i++ {
		if ranked[i].Score > ranked[i-1].Score {
			t.Errorf("results not sorted at index %d: %v > %v", i, ranked[i].Score, ranked[i-1].Score)
		}
	}

	// Best match should be the DXB→CAI with high GeoScore
	if ranked[0].TravelerGeoScore != 90.0 {
		t.Errorf("expected DXB→CAI high-trust trip to be top match, got geo=%v", ranked[0].TravelerGeoScore)
	}
}

func TestScore_BudgetExceeded(t *testing.T) {
	// Price way above budget → priceFit near 0
	expensive := trip("DXB", "CAI", 80.0, 500.0, 2)
	r := matching.Score(baseOrder, expensive)
	if r.PriceFitScore > 0.1 {
		t.Errorf("overbudget trip should have near-zero priceFit, got %v", r.PriceFitScore)
	}
}

func TestScore_Determinism(t *testing.T) {
	c := trip("DXB", "CAI", 75.0, 10.0, 4)
	a := matching.Score(baseOrder, c)
	b := matching.Score(baseOrder, c)
	if a.Score != b.Score {
		t.Errorf("non-deterministic score: %v vs %v", a.Score, b.Score)
	}
}
