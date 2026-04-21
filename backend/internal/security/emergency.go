package security

import (
	"net/http"
	"strings"

	"github.com/geocore-next/backend/internal/config"
	"github.com/gin-gonic/gin"
)

// blockedPrefixes lists path prefixes that are shut down in emergency mode.
// Read-only GET requests always pass through.
var blockedPrefixes = []string{
	"/api/v1/wallet/withdraw",
	"/api/v1/wallet/transfer",
	"/api/v1/exchange",
	"/api/v1/livestream/bids",
	"/api/v1/auctions",
	"/api/v1/orders",
	"/api/v1/payments",
}

// EmergencyMode returns a Gin middleware that, when ENABLE_EMERGENCY_MODE is
// true, blocks all mutating operations on sensitive paths while allowing
// read-only (GET/HEAD) access everywhere.
func EmergencyMode() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.GetFlags().EnableEmergencyMode {
			c.Next()
			return
		}
		// Always allow read-only methods.
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}
		// Block mutating requests on sensitive prefixes.
		path := c.Request.URL.Path
		for _, prefix := range blockedPrefixes {
			if strings.HasPrefix(path, prefix) {
				c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
					"error":   "emergency_mode",
					"message": "The platform is in emergency read-only mode. Write operations are temporarily disabled.",
				})
				return
			}
		}
		c.Next()
	}
}
