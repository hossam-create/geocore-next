package controlplane

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// KillSwitchMiddleware blocks all traffic when the kill switch is active.
func KillSwitchMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !AllowRequest() {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "service temporarily unavailable — kill switch active",
			})
			return
		}
		c.Next()
	}
}
