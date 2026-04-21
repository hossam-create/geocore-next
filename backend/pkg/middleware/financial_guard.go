package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FinancialGuard checks the admin setting `payments.financial_mode_paused`.
// When paused, ALL money-moving endpoints return 503 Service Unavailable.
// Adapted from OLD project's `SYSTEM_FINANCIAL_MODE = PAUSED` check.
//
// The setting is cached for 10 seconds to avoid a DB hit per request while
// still responding quickly when an admin flips the kill switch.
func FinancialGuard(db *gorm.DB) gin.HandlerFunc {
	var (
		mu       sync.RWMutex
		cachedAt time.Time
		isPaused bool
		cacheTTL = 10 * time.Second
	)

	check := func() bool {
		mu.RLock()
		if time.Since(cachedAt) < cacheTTL {
			v := isPaused
			mu.RUnlock()
			return v
		}
		mu.RUnlock()

		// Refresh from DB
		var setting struct{ Value string }
		err := db.Table("admin_settings").
			Select("value").
			Where("key = ?", "payments.financial_mode_paused").
			Scan(&setting).Error

		mu.Lock()
		defer mu.Unlock()
		cachedAt = time.Now()
		if err != nil {
			isPaused = false // fail-open: if setting doesn't exist, allow traffic
			return false
		}
		isPaused = setting.Value == "true"
		return isPaused
	}

	return func(c *gin.Context) {
		if check() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "All financial operations are temporarily paused. Please try again later.",
			})
			return
		}
		c.Next()
	}
}
