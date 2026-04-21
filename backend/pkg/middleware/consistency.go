package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/geocore-next/backend/pkg/rwtracker"
)

// CriticalRead marks a route as requiring read-after-write consistency.
// It sets the rwtracker key in the context so downstream handlers know
// to route reads to the primary DB regardless of the rwtracker state.
// This is essential for order, payment, and wallet read endpoints where
// stale replica data is unacceptable.
func CriticalRead() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("critical_read", true)
		c.Next()
	}
}

// ShouldUsePrimaryDB determines whether the current request should read from
// the primary DB. Returns true if:
//   - The route is marked as a critical read (orders, payments, wallet)
//   - The user has recently written (rwtracker says so)
func ShouldUsePrimaryDB(c *gin.Context, rwt *rwtracker.RecentWriteTracker) bool {
	// Critical reads always use primary
	if v, ok := c.Get("critical_read"); ok && v.(bool) {
		return true
	}
	// Recently-written users use primary
	if rwt != nil {
		if uid, exists := c.Get("user_id"); exists {
			if uidStr, ok := uid.(string); ok && rwt.ShouldReadPrimary(uidStr) {
				return true
			}
		}
	}
	return false
}
