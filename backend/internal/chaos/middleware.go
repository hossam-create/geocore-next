package chaos

import (
	"time"

	"github.com/gin-gonic/gin"
)

// ChaosMiddleware injects controlled failures based on Redis-backed rates.
// Reads chaos:<key> from Redis to determine injection percentage.
// Only active in non-production environments.
func ChaosMiddleware(engine *ChaosEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		if engine == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		// API latency injection
		if engine.ShouldInject(ctx, "api_latency") {
			InjectAPILatency(300)
		}

		// API error injection (returns 500)
		if engine.ShouldInject(ctx, "api_error") {
			c.AbortWithStatusJSON(500, gin.H{
				"error":  "chaos: injected server error",
				"chaos":  true,
			})
			return
		}

		// Request timeout simulation
		if engine.ShouldInject(ctx, "api_timeout") {
			time.Sleep(30 * time.Second) // will hit client timeout
		}

		c.Next()
	}
}
