package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	degradedMode bool
	degradedMu   sync.RWMutex

	degradedGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "geocore_degraded_mode",
		Help: "1 when system is in degraded mode (serving stale cache), 0 when normal.",
	})
)

func init() {
	prometheus.MustRegister(degradedGauge)
}

// SetDegraded enables or disables degraded mode globally.
// When true, read endpoints serve stale cache (if available) instead of
// hitting the DB. This is toggled by the health/remediation system when
// the DB becomes slow or unreachable.
func SetDegraded(on bool) {
	degradedMu.Lock()
	defer degradedMu.Unlock()
	if degradedMode != on {
		degradedMode = on
		if on {
			slog.Warn("degraded mode ENABLED — serving stale cache for reads")
			degradedGauge.Set(1)
		} else {
			slog.Info("degraded mode DISABLED — normal DB reads restored")
			degradedGauge.Set(0)
		}
	}
}

// IsDegraded returns whether the system is in degraded mode.
func IsDegraded() bool {
	degradedMu.RLock()
	defer degradedMu.RUnlock()
	return degradedMode
}

// DegradedRead is a Gin middleware for public read endpoints (listings, search,
// categories). When the system is in degraded mode, it sets the
// "X-GeoCore-Degraded: true" response header and the gin context key
// "degraded" = true so handlers can decide to skip DB and serve cache only.
//
// If no cached data is available, the handler still falls through to DB.
// The handler is responsible for checking c.GetBool("degraded").
func DegradedRead() gin.HandlerFunc {
	return func(c *gin.Context) {
		if IsDegraded() {
			c.Set("degraded", true)
			c.Header("X-GeoCore-Degraded", "true")
			c.Next()
			return
		}
		c.Next()
	}
}

// AutoDegraded monitors DB latency and automatically enables degraded mode
// when latency exceeds the threshold for sustained periods.
// Call once at startup with a reasonable check interval (e.g. 10s).
//
// This is a background goroutine that pings the DB and toggles degraded mode.
// It complements the health check endpoint which is pull-based.
func AutoDegraded(pingFn func() time.Duration, threshold time.Duration, interval time.Duration) {
	go func() {
		consecutiveSlow := 0
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			latency := pingFn()
			if latency > threshold {
				consecutiveSlow++
				if consecutiveSlow >= 3 && !IsDegraded() {
					slog.Warn("auto-degraded: DB slow for 3 consecutive checks",
						"latency_ms", latency.Milliseconds(),
						"threshold_ms", threshold.Milliseconds(),
					)
					SetDegraded(true)
				}
			} else {
				if consecutiveSlow > 0 {
					consecutiveSlow--
				}
				if consecutiveSlow == 0 && IsDegraded() {
					slog.Info("auto-degraded: DB recovered, restoring normal mode")
					SetDegraded(false)
				}
			}
		}
	}()
}

// DegradedResponse sends a 503 Service Unavailable with a retry-after header
// when the system is in degraded mode and no cached data is available.
// Handlers should call this as a fallback when c.GetBool("degraded") is true
// and cache is empty.
func DegradedResponse(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"success":     false,
		"error":       "Service temporarily degraded — please retry shortly",
		"retry_after": 10,
	})
}
