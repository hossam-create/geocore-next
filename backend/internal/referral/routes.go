package referral

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers referral routes under /api/v1/referral
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	g := r.Group("/referral")
	g.Use(middleware.Auth())
	{
		g.GET("/code", h.GetMyCode)
		g.GET("/stats", h.GetStats)
	}
}
