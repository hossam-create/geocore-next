package settings

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts settings engine routes.
// Admin routes require Auth() + AdminWithDB() middleware.
// Public routes have no auth.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Seed defaults on startup
	h.SeedDefaults()

	// ── Public (no auth) ────────────────────────────────────────────
	r.GET("/config/public", h.GetPublicConfig)
	r.GET("/features", h.GetPublicFeatures)

	// ── Admin routes (auth + admin middleware) ──────────────────────
	adm := r.Group("/admin")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db))
	{
		// Settings
		adm.GET("/settings", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetAllSettings)
		adm.PUT("/settings/bulk", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.BulkUpdateSettings) // must be before :param
		adm.GET("/settings/:category", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetSettingsByCategory)
		adm.PUT("/settings/:key", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateSetting)

		// Feature flags
		adm.GET("/features", middleware.RequireAnyPermission(middleware.PermSettingsRead), h.GetAllFeatures)
		adm.PUT("/features/:key", middleware.RequireAnyPermission(middleware.PermSettingsWrite), h.UpdateFeature)

		// Support tickets
		adm.GET("/tickets", middleware.RequireAnyPermission(middleware.PermSupportTicketsRead), h.ListTickets)
		adm.GET("/tickets/:id", middleware.RequireAnyPermission(middleware.PermSupportTicketsRead), h.GetTicket)
		adm.POST("/tickets/:id/reply", middleware.RequireAnyPermission(middleware.PermSupportTicketsReply), h.ReplyToTicket)
		adm.PATCH("/tickets/:id", middleware.RequireAnyPermission(middleware.PermSupportTicketsWrite), h.UpdateTicketStatus)
	}
}
