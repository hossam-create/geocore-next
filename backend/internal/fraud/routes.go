package fraud

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Internal analyze endpoint (auth required)
	auth := v1.Group("")
	auth.Use(middleware.Auth())
	auth.POST("/fraud/analyze", h.Analyze)

	// Admin fraud management
	admin := v1.Group("/admin/fraud")
	admin.Use(middleware.Auth())
	{
		admin.GET("/stats", h.Stats)
		admin.GET("/alerts", h.ListAlerts)
		admin.GET("/alerts/:id", h.GetAlert)
		admin.PATCH("/alerts/:id", h.UpdateAlert)
		admin.GET("/rules", h.ListRules)
		admin.PATCH("/rules/:id", h.UpdateRule)
		admin.GET("/risk-profiles", h.ListRiskProfiles)
	}
}
