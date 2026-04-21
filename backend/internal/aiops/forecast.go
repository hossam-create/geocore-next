package aiops

// PredictTraffic uses simple linear extrapolation to forecast traffic.
// history is a series of request-per-second values.
func PredictTraffic(history []int) int {
	if len(history) == 0 {
		return 0
	}
	if len(history) == 1 {
		return history[0]
	}

	// Simple linear regression: y = mx + b
	n := len(history)
	var sumX, sumY, sumXY, sumX2 float64

	for i, y := range history {
		x := float64(i)
		sumX += x
		sumY += float64(y)
		sumXY += x * float64(y)
		sumX2 += x * x
	}

	denom := float64(n)*sumX2 - sumX*sumX
	if denom == 0 {
		return history[n-1]
	}

	m := (float64(n)*sumXY - sumX*sumY) / denom
	b := (sumY - m*sumX) / float64(n)

	nextX := float64(n)
	predicted := m*nextX + b

	// Clamp to non-negative
	if predicted < 0 {
		return 0
	}
	return int(predicted)
}

// PredictTrafficSimple uses a simpler moving-average forecast.
func PredictTrafficSimple(history []int, window int) int {
	if len(history) == 0 {
		return 0
	}
	start := len(history) - window
	if start < 0 {
		start = 0
	}
	sum := 0
	for _, v := range history[start:] {
		sum += v
	}
	avg := sum / len(history[start:])
	// Project with slight growth factor (1.1x)
	return int(float64(avg) * 1.1)
}
