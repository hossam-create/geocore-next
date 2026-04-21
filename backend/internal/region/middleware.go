package region

import (
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/gin-gonic/gin"
)

// RegionMiddleware injects the resolved region into the Gin context.
// Sets c.Set("region", regionName) for downstream handlers.
func RegionMiddleware(router *Router) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		region := router.Route(c.Request.Context(), userID)
		c.Set("region", region.Name)
		c.Header("X-Region", region.Name)
		c.Next()
	}
}

// InjectKafkaContext pushes region + idempotency key into the request context
// so that downstream Kafka publishes automatically carry these fields.
func InjectKafkaContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Inject region from Gin context (set by RegionMiddleware)
		if r, exists := c.Get("region"); exists {
			if region, ok := r.(string); ok && region != "" {
				ctx = kafka.WithRegion(ctx, region)
			}
		}

		// Inject idempotency key from header (client-provided or auto-generated)
		idemKey := c.GetHeader("X-Idempotency-Key")
		if idemKey != "" {
			ctx = kafka.WithIdempotencyKey(ctx, idemKey)
		}

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
