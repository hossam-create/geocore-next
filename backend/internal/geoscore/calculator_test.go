package geoscore_test

import (
	"testing"

	"github.com/geocore-next/backend/internal/geoscore"
)

func TestCalculate_MaxScore(t *testing.T) {
	in := geoscore.Input{
		SuccessRate:   1.0,
		DisputeRate:   0.0,
		KYCScore:      1.0,
		DeliveryScore: 1.0,
		FraudScore:    0.0,
	}
	got := geoscore.Calculate(in)
	if got != 100.0 {
		t.Errorf("max score: want 100.0, got %v", got)
	}
}

func TestCalculate_MinScore(t *testing.T) {
	in := geoscore.Input{
		SuccessRate:   0.0,
		DisputeRate:   1.0,
		KYCScore:      0.0,
		DeliveryScore: 0.0,
		FraudScore:    1.0,
	}
	got := geoscore.Calculate(in)
	if got != 0.0 {
		t.Errorf("min score: want 0.0, got %v", got)
	}
}

func TestCalculate_NeutralUser(t *testing.T) {
	// New user: no orders, no kyc, no disputes
	in := geoscore.Input{
		SuccessRate:   0.0,
		DisputeRate:   0.0,
		KYCScore:      0.0,
		DeliveryScore: 0.5,
		FraudScore:    0.0,
	}
	// 0*0.35 + 1*0.25 + 0*0.15 + 0.5*0.15 + 1*0.10 = 0.425 → 42.5
	got := geoscore.Calculate(in)
	if got != 42.5 {
		t.Errorf("neutral user: want 42.5, got %v", got)
	}
}

func TestCalculate_ClampsBeyondBounds(t *testing.T) {
	in := geoscore.Input{
		SuccessRate:   1.5, // over 1
		DisputeRate:   -0.1, // under 0
		KYCScore:      2.0,
		DeliveryScore: 1.0,
		FraudScore:    -1.0,
	}
	got := geoscore.Calculate(in)
	if got > 100.0 || got < 0.0 {
		t.Errorf("clamping: score %v out of [0,100] range", got)
	}
}

func TestCalculate_PartialProfile(t *testing.T) {
	// Verified KYC, moderate success, some disputes
	in := geoscore.Input{
		SuccessRate:   0.8,
		DisputeRate:   0.1,
		KYCScore:      0.5,
		DeliveryScore: 0.9,
		FraudScore:    0.0,
	}
	// 0.8*0.35 + 0.9*0.25 + 0.5*0.15 + 0.9*0.15 + 1.0*0.10
	// = 0.28 + 0.225 + 0.075 + 0.135 + 0.10 = 0.815 → 81.5
	got := geoscore.Calculate(in)
	if got != 81.5 {
		t.Errorf("partial profile: want 81.5, got %v", got)
	}
}

func TestCalculate_Determinism(t *testing.T) {
	in := geoscore.Input{
		SuccessRate:   0.6,
		DisputeRate:   0.05,
		KYCScore:      1.0,
		DeliveryScore: 0.75,
		FraudScore:    0.02,
	}
	a := geoscore.Calculate(in)
	b := geoscore.Calculate(in)
	if a != b {
		t.Errorf("non-deterministic: got %v then %v", a, b)
	}
}
