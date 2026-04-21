package ads

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes sets up admin and public ad endpoints.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Public endpoints
	pub := r.Group("/ads")
	{
		pub.GET("", h.GetPublicAds)
		pub.POST("/:id/click", h.TrackClick)
	}

	// Admin-only endpoints
	admin := r.Group("/admin/ads")
	admin.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		admin.GET("", h.ListAds)
		admin.POST("", h.CreateAd)
		admin.PUT("/:id", h.UpdateAd)
		admin.DELETE("/:id", h.DeleteAd)
		admin.PATCH("/:id/toggle", h.ToggleAd)
	}
}
