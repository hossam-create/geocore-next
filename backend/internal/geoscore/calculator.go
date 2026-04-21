package geoscore

import "math"

// Input holds the normalized signals (all values in [0.0, 1.0]) used to
// compute a user's GeoScore.
type Input struct {
	SuccessRate   float64 // fraction of orders completed successfully
	DisputeRate   float64 // fraction of orders ending in dispute
	KYCScore      float64 // 0=none, 0.5=basic, 1.0=full
	DeliveryScore float64 // fraction of deliveries completed on time
	FraudScore    float64 // 0=clean, 1.0=max risk (inverted in formula)
}

// Calculate computes the GeoScore from normalised signals.
//
// Formula (weights sum to 1.0):
//
//	score = SuccessRate   * 0.35
//	      + (1-DisputeRate) * 0.25
//	      + KYCScore      * 0.15
//	      + DeliveryScore * 0.15
//	      + (1-FraudScore) * 0.10
//
// Returns a value in [0.0, 100.0] (scaled for readability).
func Calculate(in Input) float64 {
	// Clamp all inputs to [0,1]
	sr := clamp01(in.SuccessRate)
	dr := clamp01(in.DisputeRate)
	ks := clamp01(in.KYCScore)
	ds := clamp01(in.DeliveryScore)
	fs := clamp01(in.FraudScore)

	raw := sr*0.35 +
		(1-dr)*0.25 +
		ks*0.15 +
		ds*0.15 +
		(1-fs)*0.10

	// Scale to 0–100 and round to 2dp
	return math.Round(raw*10000) / 100
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
