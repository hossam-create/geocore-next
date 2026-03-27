package middleware

  import (
  	"net/http"

  	"github.com/gin-gonic/gin"
  	"gorm.io/gorm"
  )

  // EmailVerified is a Gin middleware that ensures the authenticated user has
  // confirmed their email address. Must be placed after Auth() middleware so
  // that "user_id" is already present in the context.
  //
  // Usage:
  //   protected.Use(middleware.Auth(), middleware.EmailVerified(db))
  func EmailVerified(db *gorm.DB) gin.HandlerFunc {
  	return func(c *gin.Context) {
  		userID, exists := c.Get("user_id")
  		if !exists {
  			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
  				"success": false,
  				"message": "Unauthorized",
  			})
  			return
  		}

  		var emailVerified bool
  		err := db.Raw(
  			"SELECT email_verified FROM users WHERE id = ? AND deleted_at IS NULL",
  			userID,
  		).Scan(&emailVerified).Error

  		if err != nil {
  			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
  				"success": false,
  				"message": "Internal server error",
  			})
  			return
  		}

  		if !emailVerified {
  			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
  				"success": false,
  				"message": "Please verify your email address before performing this action",
  				"code":    "EMAIL_NOT_VERIFIED",
  			})
  			return
  		}

  		c.Next()
  	}
  }
  