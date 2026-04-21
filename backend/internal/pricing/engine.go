package pricing

import "math"

// PricingInput holds all inputs needed to compute a dynamic price.
type PricingInput struct {
	BaseRate          float64 // route average price per kg (from route_metrics)
	Weight            float64 // shipment weight in kg
	DemandScore       float64 // route demand [0,1] — from route_metrics
	DisputeRate       float64 // route dispute rate [0,1] — risk signal
	TravelerGeoScore  float64 // traveler's GeoScore [0,100]
}

// PricingResult is the computed price breakdown.
type PricingResult struct {
	BaseRate          float64 `json:"base_rate"`
	Weight            float64 `json:"weight_kg"`
	DemandMultiplier  float64 `json:"demand_multiplier"`
	RiskFactor        float64 `json:"risk_factor"`
	TrustDiscount     float64 `json:"trust_discount"`
	FinalPrice        float64 `json:"final_price"`
	PricePerKg        float64 `json:"price_per_kg"`
}

// defaultBaseRate is used when no route history exists.
const defaultBaseRate = 10.0

// Calculate computes the dynamic price.
//
// Formula:
//
//	finalPrice = (baseRate * weight) * demandMultiplier * riskFactor * trustDiscount
//
// Where:
//   - demandMultiplier = 1.0 + (demandScore * 0.5)     → up to +50% when demand is high
//   - riskFactor       = 1.0 + (disputeRate  * 0.5)    → up to +50% for risky routes
//   - trustDiscount    = 1.0 - (geoScore/100 * 0.10)   → up to −10% for trusted traveler
func Calculate(in PricingInput) PricingResult {
	base := in.BaseRate
	if base <= 0 {
		base = defaultBaseRate
	}
	weight := math.Max(0.1, in.Weight)

	demandMul := 1.0 + clamp01(in.DemandScore)*0.5
	riskFactor := 1.0 + clamp01(in.DisputeRate)*0.5
	trustDiscount := 1.0 - (clamp(in.TravelerGeoScore, 0, 100)/100.0)*0.10

	pricePerKg := base * demandMul * riskFactor * trustDiscount
	finalPrice := math.Round(pricePerKg*weight*100) / 100

	return PricingResult{
		BaseRate:         base,
		Weight:           weight,
		DemandMultiplier: math.Round(demandMul*1000) / 1000,
		RiskFactor:       math.Round(riskFactor*1000) / 1000,
		TrustDiscount:    math.Round(trustDiscount*1000) / 1000,
		FinalPrice:       finalPrice,
		PricePerKg:       math.Round(pricePerKg*100) / 100,
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
