package aiops

import "math"

// DetectAnomaly uses simple statistical analysis to detect anomalies.
// A metric is anomalous if the latest value exceeds 2x the mean.
func DetectAnomaly(metrics []float64) bool {
	if len(metrics) < 3 {
		return false
	}
	mean := avg(metrics)
	latest := metrics[len(metrics)-1]
	return latest > mean*2
}

// DetectAnomalyZScore uses z-score for more robust anomaly detection.
// A metric is anomalous if its z-score exceeds the threshold (default 2.0).
func DetectAnomalyZScore(metrics []float64, threshold float64) bool {
	if len(metrics) < 3 {
		return false
	}
	mean := avg(metrics)
	stdDev := stdDeviation(metrics)
	if stdDev == 0 {
		return false
	}
	latest := metrics[len(metrics)-1]
	zScore := (latest - mean) / stdDev
	return zScore > threshold || zScore < -threshold
}

// DetectSpike checks if there's a sudden increase in the last N values.
func DetectSpike(metrics []float64, window int) bool {
	if len(metrics) < window*2 {
		return false
	}
	recent := avg(metrics[len(metrics)-window:])
	baseline := avg(metrics[:len(metrics)-window])
	if baseline == 0 {
		return recent > 0
	}
	return recent > baseline*2
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDeviation(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	m := avg(values)
	sum := 0.0
	for _, v := range values {
		diff := v - m
		sum += diff * diff
	}
	return math.Sqrt(sum / float64(len(values)-1))
}
