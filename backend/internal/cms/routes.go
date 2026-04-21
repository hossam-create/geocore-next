package cms

import (
	"net/http"
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db)
	rl := middleware.NewRateLimiter(rdb)

	// ── Admin CMS Routes (auth required) ──────────────────────────────────
	adm := r.Group("/admin/cms")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db), rl.LimitByUser(120, time.Minute, "admin:cms"))
	{
		// Hero Slides
		adm.GET("/slides", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListSlides)
		adm.POST("/slides", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.CreateSlide)
		adm.PUT("/slides/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateSlide)
		adm.DELETE("/slides/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DeleteSlide)
		adm.PUT("/slides/reorder", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.ReorderSlides)

		// Content Blocks
		adm.GET("/blocks", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListContentBlocks)
		adm.GET("/blocks/:slug", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetContentBlock)
		adm.POST("/blocks", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.CreateContentBlock)
		adm.PUT("/blocks/:slug", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateContentBlock)
		adm.DELETE("/blocks/:slug", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DeleteContentBlock)

		// Media Library
		adm.GET("/media", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListMedia)
		adm.POST("/media", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UploadMedia)
		adm.DELETE("/media/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DeleteMedia)

		// Site Settings
		adm.GET("/settings", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListSettings)
		adm.GET("/settings/:key", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetSetting)
		adm.PUT("/settings/:key", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateSetting)
		adm.PUT("/settings/bulk", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.BulkUpdateSettings)

		// Navigation Menus
		adm.GET("/nav", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.ListNavMenus)
		adm.POST("/nav", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.CreateNavItem)
		adm.PUT("/nav/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateNavItem)
		adm.DELETE("/nav/:id", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.DeleteNavItem)
		adm.PUT("/nav/reorder", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.ReorderNav)
	}

	// ── Public CMS Routes (no auth — for frontend consumption) ────────────
	pub := r.Group("/cms")
	{
		pub.GET("/slides", h.PublicSlides)
		pub.GET("/blocks/:page", h.PublicContentBlocks)
		pub.GET("/settings", h.PublicSettings)
		pub.GET("/nav/:location", h.PublicNav)
	}
}

// RegisterStaticFiles serves uploaded files under /uploads/.
func RegisterStaticFiles(r *gin.Engine) {
	r.StaticFS("/uploads", http.Dir("./uploads"))
}
