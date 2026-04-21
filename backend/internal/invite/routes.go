package invite

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts invite endpoints under the given router group.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)

	inv := rg.Group("/invite")
	inv.Use(middleware.Auth())
	{
		inv.POST("", h.CreateInviteHandler)
		inv.GET("", h.GetMyInvitesHandler)
		inv.GET("/rewards", h.GetMyRewardsHandler)
	}

	admin := rg.Group("/admin/invites")
	admin.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		admin.GET("", h.AdminInviteAnalyticsHandler)
	}
}
