package waitlist

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all waitlist endpoints on the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)

	// Public — no auth required (pre-signup flow)
	wl := rg.Group("/waitlist")
	{
		wl.POST("/join", h.JoinHandler)
		wl.GET("/status", h.StatusHandler)
		wl.GET("/stats", h.StatsHandler)
	}

	// Admin — requires super_admin / admin role
	adm := rg.Group("/admin/waitlist")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adm.POST("/release", h.AdminReleaseHandler)
		adm.POST("/recalc", h.AdminRecalcHandler)
		adm.POST("/flag", h.AdminFlagHandler)
		adm.GET("/analytics", h.AdminAnalyticsHandler)
		adm.POST("/limit", h.AdminSetLimitHandler)
	}
}
