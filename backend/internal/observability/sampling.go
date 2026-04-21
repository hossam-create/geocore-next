package observability

import (
	"hash/fnv"
)

// ShouldSample determines if a trace should be sampled based on traceID and rate.
// Uses deterministic hashing so the same traceID always gets the same decision.
func ShouldSample(traceID string, rate float64) bool {
	if rate >= 100.0 {
		return true
	}
	if rate <= 0.0 {
		return false
	}

	h := fnv.New32a()
	h.Write([]byte(traceID))
	hashVal := h.Sum32()

	return float64(hashVal%100) < rate
}

// AdaptiveRate returns a sampling rate adjusted by error rate.
// Higher error rates → higher sampling (more visibility when things break).
func AdaptiveRate(baseRate, errorRate float64) float64 {
	if errorRate > 5.0 {
		return 100.0 // sample everything when errors are high
	}
	if errorRate > 1.0 {
		return min(baseRate*3, 100.0)
	}
	return baseRate
}
