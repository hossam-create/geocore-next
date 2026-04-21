package controltower

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all Control Tower endpoints under /admin/system.
// All routes require admin authentication.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	bus := InitEventBus(rdb)
	h := NewHandler(db, rdb, bus)

	adm := rg.Group("/admin/system")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		// Part 1 — Live system metrics
		adm.GET("/metrics", h.MetricsHandler)
		adm.GET("/metrics/blocked", h.BlockedIPsHandler)

		// Part 2 — Fraud radar
		adm.GET("/fraud", h.FraudHandler)

		// Part 3 — Revenue dashboard
		adm.GET("/revenue", h.RevenueHandler)

		// Part 4 — Liquidity dashboard
		adm.GET("/liquidity", h.LiquidityHandler)

		// Part 5 — Growth engine
		adm.GET("/growth", h.GrowthHandler)

		// Part 6 — Real-time event stream (SSE)
		adm.GET("/events", h.EventStreamHandler)

		// Part 7 — Emergency kill switch
		adm.GET("/emergency", h.EmergencyStatusHandler)
		adm.POST("/emergency", h.EmergencyHandler)
	}
}
