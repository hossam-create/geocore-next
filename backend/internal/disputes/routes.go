package disputes

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	d := r.Group("/disputes")
	d.Use(middleware.Auth())
	{
		d.POST("", h.CreateDispute)
		d.GET("", h.ListDisputes)
		d.GET("/:id", h.GetDispute)
		d.POST("/:id/messages", h.AddMessage)
		d.POST("/:id/evidence", h.AddEvidence)
		d.POST("/:id/escalate", h.EscalateDispute)
		d.POST("/:id/close", h.CloseDispute)
		d.GET("/:id/activity", h.GetActivity)
	}

	// Admin routes
	admin := r.Group("/admin/disputes")
	admin.Use(middleware.Auth(), middleware.AdminOnly())
	{
		admin.GET("", h.AdminListDisputes)
		admin.POST("/:id/assign", h.AdminAssignDispute)
		admin.POST("/:id/resolve", h.ResolveDispute)
	}
}
