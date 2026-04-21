package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware records request count and duration by route.
func PrometheusMiddleware() gin.HandlerFunc {
	metrics.Init()

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		metrics.ObserveHTTPRequest(
			c.Request.Method,
			route,
			c.Writer.Status(),
			time.Since(start),
		)
	}
}

// MetricsAuth protects /metrics endpoint with Bearer METRICS_TOKEN.
// If token is empty, endpoint is left unprotected (dev-friendly).
func MetricsAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(authHeader, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		provided := strings.TrimPrefix(authHeader, prefix)
		if provided != token {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid metrics token"})
			return
		}

		c.Next()
	}
}
