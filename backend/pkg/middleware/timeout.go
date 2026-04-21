package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Timeout sets a hard deadline on the request context. All downstream calls
// (DB, Redis, HTTP, Kafka) that respect context will fail-fast when the
// deadline expires — no goroutines, no race conditions.
//
// Use per-route to give expensive endpoints more time:
//
//	r.GET("/export", middleware.Timeout(30*time.Second), h.Export)
//	r.GET("/listings", middleware.Timeout(5*time.Second), h.List)
//
// After c.Next(), if the context deadline was exceeded and no response has
// been written yet, a 504 is sent. Otherwise the handler's own error
// response is preserved.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// Post-handler: if deadline exceeded and nobody wrote a response yet,
		// send 504 + log. If the handler already responded, do nothing.
		if ctx.Err() == context.DeadlineExceeded && !c.Writer.Written() {
			slog.Warn("request timeout",
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
				"timeout", d.String(),
			)
			c.AbortWithStatusJSON(504, gin.H{
				"success": false,
				"error":   "Request timed out",
			})
		}
	}
}
