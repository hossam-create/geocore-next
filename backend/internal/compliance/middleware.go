package compliance

import "github.com/gin-gonic/gin"

// DefaultDisclaimer is the non-custodial notice attached to every response.
// Keep it short enough to live in an HTTP header (<8KB hard limit but
// most gateways cap at ~1KB per header).
const DefaultDisclaimer = "Platform does not hold customer funds. All transactions are peer-to-peer. Users are solely responsible for settlement, tax and KYC obligations applicable in their jurisdiction."

// DisclaimerMiddleware attaches a standard non-custodial disclaimer header
// to every API response. Cheap (single header write) and non-invasive —
// does not modify JSON bodies, so it never breaks serialisation.
func DisclaimerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Platform-Notice", DefaultDisclaimer)
		c.Next()
	}
}

// DisclaimerHandler — GET /meta/disclaimer
// Public endpoint for apps/clients that prefer a canonical body over header.
func DisclaimerHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"notice": DefaultDisclaimer,
		"links": gin.H{
			"terms":   "/legal/terms",
			"privacy": "/legal/privacy",
		},
		"non_custodial": true,
	})
}
