package analytics

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	a := r.Group("/analytics")
	a.Use(middleware.Auth())
	{
		seller := a.Group("/seller")
		{
			seller.GET("/summary", h.SellerSummary)
			seller.GET("/revenue", h.SellerRevenue)
			seller.GET("/listings", h.SellerListings)
		}
		a.GET("/storefront", h.StorefrontAnalytics)
		a.GET("/platform", h.PlatformMetrics) // Admin-only for founder dashboard

		// Sprint 8: Funnel analytics
		a.GET("/funnels", h.GetFunnelDropoffsHandler)
		a.GET("/funnels/progress", h.GetUserFunnelProgressHandler)
	}

	// Sprint 8: Admin metrics dashboard
	adminMetrics := r.Group("/admin/metrics")
	adminMetrics.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adminMetrics.GET("", h.GetMetricsHandler)
		adminMetrics.GET("/timeseries", h.GetMetricsTimeSeriesHandler)
	}

	// Layer 1: Admin analytics (Traffic Watch + Overview)
	adminAnalytics := r.Group("/admin/analytics")
	adminAnalytics.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adminAnalytics.GET("/traffic", h.TrafficWatch)
		adminAnalytics.GET("/overview", h.AdminOverview)
	}
}
