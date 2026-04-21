package settlement

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts settlement routes for authenticated users.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	s := r.Group("/settlements")
	s.Use(middleware.Auth())
	{
		s.POST("", h.CreateSettlement)
		s.GET("", h.ListSettlements)
	}
}

// RegisterAdminRoutes mounts payout admin routes.
func RegisterAdminRoutes(adm *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	payouts := adm.Group("/payouts")
	{
		payouts.GET("", h.ListPayouts)
		payouts.GET("/:id", h.GetPayout)
		payouts.POST("", h.CreatePayout)
		payouts.POST("/:id/approve", h.ApprovePayout)
		payouts.POST("/:id/process", h.ProcessPayout)
	}
}
