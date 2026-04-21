package addons

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db)
	rl := middleware.NewRateLimiter(rdb)

	adm := r.Group("/admin/addons")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), rl.LimitByUser(60, time.Minute, "admin:addons"))
	{
		adm.GET("", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListAddons)
		adm.GET("/stats", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetMarketplaceStats)
		adm.GET("/:id", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetAddon)
		adm.POST("/:id/install", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.InstallAddon)
		adm.POST("/:id/uninstall", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UninstallAddon)
		adm.POST("/:id/enable", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.EnableAddon)
		adm.POST("/:id/disable", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DisableAddon)
		adm.PUT("/:id/config", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateAddonConfig)
		adm.GET("/:id/reviews", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListAddonReviews)
		adm.POST("/:id/reviews", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.AddAddonReview)
	}
}
