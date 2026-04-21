package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// SheddedRequestsTotal counts requests rejected by load shedding.
	SheddedRequestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "load_shedded_requests_total",
		Help: "Total number of requests rejected by load shedding.",
	})

	// currentSaturation is the latest saturation score (0–100) used by the
	// middleware. Updated by the background probe goroutine.
	currentSaturation atomic.Int64

	// ShedThreshold is the saturation percentage above which non-critical
	// requests are rejected. Configurable via env LOAD_SHED_THRESHOLD.
	ShedThreshold int64 = 80
)

func init() {
	prometheus.MustRegister(SheddedRequestsTotal)
}

// SaturationSignal is a function that returns a saturation score (0–100).
// Multiple signals are combined by taking the maximum.
type SaturationSignal func() float64

// LoadShed returns a Gin middleware that rejects non-critical requests when
// system saturation exceeds the threshold. Critical requests (marked with
// X-Critical: true header or financial endpoints) always pass through.
//
// Saturation is computed from multiple signals:
//   - Goroutine count (vs GOMAXPROCS * 10000)
//   - DB pool utilization (in-use / open)
//   - Request rate (vs rate limit capacity)
//
// Rejected requests receive 503 with a Retry-After header.
func LoadShed(signals ...SaturationSignal) gin.HandlerFunc {
	return func(c *gin.Context) {
		sat := computeSaturation(signals)
		currentSaturation.Store(int64(sat))

		threshold := ShedThreshold

		// Critical requests always pass through (financial ops, webhooks)
		if isCritical(c) {
			c.Header("X-Saturation", formatSat(sat))
			c.Next()
			return
		}

		if sat >= float64(threshold) {
			SheddedRequestsTotal.Inc()
			slog.Warn("load shed: rejecting request",
				"saturation", sat,
				"threshold", threshold,
				"path", c.Request.URL.Path,
				"method", c.Request.Method,
			)
			c.Header("Retry-After", "5")
			c.Header("X-Saturation", formatSat(sat))
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success":     false,
				"error":       "Service overloaded — please retry shortly",
				"retry_after": 5,
			})
			return
		}

		c.Header("X-Saturation", formatSat(sat))
		c.Next()
	}
}

// GoroutineSignal returns a SaturationSignal based on goroutine count.
// At GOMAXPROCS * 10000 goroutines → 100% saturation.
func GoroutineSignal() SaturationSignal {
	return func() float64 {
		maxProcs := float64(runtime.GOMAXPROCS(0))
		goroutines := float64(runtime.NumGoroutine())
		return min(goroutines/(maxProcs*10000)*100, 100)
	}
}

// DBPoolSignal returns a SaturationSignal based on DB pool utilization.
// Requires a function that returns (inUse, open) connection counts.
func DBPoolSignal(statsFn func() (inUse, open int)) SaturationSignal {
	return func() float64 {
		inUse, open := statsFn()
		if open == 0 {
			return 0
		}
		return float64(inUse) / float64(open) * 100
	}
}

// computeSaturation takes the maximum of all signals.
func computeSaturation(signals []SaturationSignal) float64 {
	if len(signals) == 0 {
		return 0
	}
	maxSat := 0.0
	for _, sig := range signals {
		if v := sig(); v > maxSat {
			maxSat = v
		}
	}
	return maxSat
}

// isCritical returns true for requests that must never be shed.
// Financial operations, webhooks, and health checks are always critical.
func isCritical(c *gin.Context) bool {
	if c.GetHeader("X-Critical") == "true" {
		return true
	}
	path := c.Request.URL.Path
	criticalPrefixes := []string{
		"/api/v1/wallet",
		"/api/v1/orders",
		"/api/v1/payments",
		"/api/v1/escrow",
		"/webhooks",
		"/health",
	}
	for _, prefix := range criticalPrefixes {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

func formatSat(sat float64) string {
	return fmt.Sprintf("%.0f", sat)
}

// StartSaturationProbe starts a background goroutine that periodically
// evaluates saturation signals and logs when saturation is high.
func StartSaturationProbe(signals []SaturationSignal, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			sat := computeSaturation(signals)
			currentSaturation.Store(int64(sat))
			if sat >= 70 {
				slog.Warn("high saturation detected",
					"saturation_pct", sat,
					"goroutines", runtime.NumGoroutine(),
				)
			}
		}
	}()
}

// GetSaturation returns the current saturation score (0–100).
// Useful for health endpoints and metrics.
func GetSaturation() float64 {
	return float64(currentSaturation.Load())
}
