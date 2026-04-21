package security

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FrozenUserMiddleware blocks any authenticated request belonging to a user
// whose UserRiskProfile.Frozen == true. Mount this AFTER middleware.Auth()
// so that "user_id" has already been set on the Gin context.
//
// Read-only requests (GET/HEAD) are allowed so frozen users can still view
// their own state — align with emergency-mode semantics.
func FrozenUserMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}
		userID := c.GetString("user_id")
		if userID == "" {
			c.Next()
			return
		}
		uid, err := uuid.Parse(userID)
		if err != nil {
			c.Next()
			return
		}
		if IsUserFrozen(db, uid) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "account_frozen",
				"message": "Your account is temporarily frozen pending review. Please contact support.",
			})
			return
		}
		c.Next()
	}
}
