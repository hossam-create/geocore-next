package protection

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Authenticated — buyer protection endpoints
	auth := v1.Group("")
	auth.Use(middleware.Auth())
	{
		// Protection purchase + pricing
		auth.POST("/orders/:id/protection", h.PurchaseProtection)
		auth.GET("/orders/:id/protection", h.GetOrderProtection)
		auth.GET("/orders/:id/protection-pricing", h.GetProtectionPricing)

		// Guarantee claims
		auth.POST("/orders/:id/claim", h.FileClaim)
		auth.GET("/orders/:id/delay-status", h.CheckDelayStatus)

		// A/B variant
		auth.GET("/protection/variant", h.GetABVariant)
	}

	// Admin — protection management + metrics
	admin := v1.Group("/admin/protection")
	admin.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		admin.GET("/metrics", h.GetAdminMetrics)
		admin.GET("/claims", h.ListClaims)
		admin.POST("/claims/:id/review", h.ReviewClaim)
		admin.GET("/ab-test", h.GetABTestResults)
		admin.GET("/daily-metrics", h.GetDailyMetrics)
		admin.POST("/aggregate", h.TriggerAggregation)
		admin.POST("/scan-delays", h.TriggerDelayScan)
	}
}
