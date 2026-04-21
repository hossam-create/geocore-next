package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Liveness returns 200 if the process is alive (no dependency checks).
// K8s: restart pod if this fails.
func Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Readiness returns 200 only if all critical dependencies are healthy.
// K8s: remove pod from service if this fails.
func Readiness(checks map[string]func() bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		failed := ""
		for name, fn := range checks {
			if !fn() {
				failed = name
				break
			}
		}
		if failed != "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"failed": failed,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
