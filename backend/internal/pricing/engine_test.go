package pricing_test

import (
	"testing"

	"github.com/geocore-next/backend/internal/pricing"
)

func TestCalculate_DefaultBaseRate(t *testing.T) {
	// When base rate is 0, default (10.0) is used
	in := pricing.PricingInput{
		BaseRate:         0,
		Weight:           5.0,
		DemandScore:      0.0,
		DisputeRate:      0.0,
		TravelerGeoScore: 50.0,
	}
	r := pricing.Calculate(in)
	// baseRate=10, demandMul=1.0, riskFactor=1.0, trust=1-0.05=0.95
	// pricePerKg = 10 * 1 * 1 * 0.95 = 9.5; final = 9.5 * 5 = 47.5
	if r.FinalPrice != 47.5 {
		t.Errorf("default base rate: want 47.5, got %v", r.FinalPrice)
	}
}

func TestCalculate_HighDemand(t *testing.T) {
	in := pricing.PricingInput{
		BaseRate:         10.0,
		Weight:           1.0,
		DemandScore:      1.0,  // max demand → +50%
		DisputeRate:      0.0,
		TravelerGeoScore: 0.0, // no trust discount
	}
	r := pricing.Calculate(in)
	// pricePerKg = 10 * 1.5 * 1.0 * 1.0 = 15.0; final = 15.0
	if r.FinalPrice != 15.0 {
		t.Errorf("high demand: want 15.0, got %v", r.FinalPrice)
	}
}

func TestCalculate_HighRisk(t *testing.T) {
	in := pricing.PricingInput{
		BaseRate:         10.0,
		Weight:           1.0,
		DemandScore:      0.0,
		DisputeRate:      1.0,  // max risk → +50%
		TravelerGeoScore: 0.0,
	}
	r := pricing.Calculate(in)
	// pricePerKg = 10 * 1.0 * 1.5 * 1.0 = 15.0
	if r.FinalPrice != 15.0 {
		t.Errorf("high risk: want 15.0, got %v", r.FinalPrice)
	}
}

func TestCalculate_TrustedTraveler(t *testing.T) {
	in := pricing.PricingInput{
		BaseRate:         10.0,
		Weight:           1.0,
		DemandScore:      0.0,
		DisputeRate:      0.0,
		TravelerGeoScore: 100.0, // max trust → -10%
	}
	r := pricing.Calculate(in)
	// pricePerKg = 10 * 1.0 * 1.0 * 0.9 = 9.0
	if r.FinalPrice != 9.0 {
		t.Errorf("trusted traveler: want 9.0, got %v", r.FinalPrice)
	}
}

func TestCalculate_Determinism(t *testing.T) {
	in := pricing.PricingInput{
		BaseRate:         15.0,
		Weight:           3.5,
		DemandScore:      0.6,
		DisputeRate:      0.2,
		TravelerGeoScore: 75.0,
	}
	a := pricing.Calculate(in)
	b := pricing.Calculate(in)
	if a.FinalPrice != b.FinalPrice {
		t.Errorf("non-deterministic: %v vs %v", a.FinalPrice, b.FinalPrice)
	}
}

func TestCalculate_Components(t *testing.T) {
	in := pricing.PricingInput{
		BaseRate:         20.0,
		Weight:           2.0,
		DemandScore:      0.5,  // demandMul = 1.25
		DisputeRate:      0.4,  // riskFactor = 1.20
		TravelerGeoScore: 50.0, // trustDiscount = 0.95
	}
	r := pricing.Calculate(in)
	if r.DemandMultiplier != 1.25 {
		t.Errorf("demand multiplier: want 1.25, got %v", r.DemandMultiplier)
	}
	if r.RiskFactor != 1.2 {
		t.Errorf("risk factor: want 1.2, got %v", r.RiskFactor)
	}
	if r.TrustDiscount != 0.95 {
		t.Errorf("trust discount: want 0.95, got %v", r.TrustDiscount)
	}
}
