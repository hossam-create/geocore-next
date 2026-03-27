package admin

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all /admin endpoints.
// All routes require: Auth() + AdminOnly() (role must be "admin" or "super_admin")
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, _ *redis.Client) {
	h := NewHandler(db)

	// Auto-migrate the integration_configs table
	db.AutoMigrate(&IntegrationConfig{})

	adm := r.Group("/admin")
	adm.Use(middleware.Auth(), middleware.AdminWithDB(db))
	{
		// Dashboard
		adm.GET("/stats", h.GetStats)

		// Users
		adm.GET("/users", h.ListUsers)
		adm.GET("/users/:id", h.GetUser)
		adm.PUT("/users/:id", h.UpdateUser)
		adm.DELETE("/users/:id", h.DeleteUser)
		adm.POST("/users/:id/ban", h.BanUser)
		adm.POST("/users/:id/unban", h.UnbanUser)

		// Listings moderation
		adm.GET("/listings", h.ListListings)
		adm.PUT("/listings/:id/approve", h.ApproveListing)
		adm.PUT("/listings/:id/reject", h.RejectListing)
		adm.DELETE("/listings/:id", h.DeleteListing)

		// Revenue & transactions
		adm.GET("/revenue", h.GetRevenue)
		adm.GET("/transactions", h.GetTransactions)

		// Audit logs
		adm.GET("/logs", h.GetAuditLogs)

		// Integrations (external API keys stored securely in DB)
		adm.GET("/integrations", h.GetIntegrations)
		adm.POST("/integrations", h.SaveIntegrations)
	}
}
