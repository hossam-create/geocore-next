package requests

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers product request routes under /api/v1/requests
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	g := r.Group("/requests")
	{
		// Public — no auth required
		g.GET("", h.ListRequests)
		g.GET("/:id", h.GetRequest)

		// Auth required
		auth := g.Group("")
		auth.Use(middleware.Auth())
		{
			auth.POST("", h.CreateRequest)
			auth.PUT("/:id", h.UpdateRequest)
			auth.GET("/mine", h.MyRequests)
			auth.POST("/:id/respond", h.RespondToRequest)
			auth.DELETE("/:id", h.CancelRequest)
		}
	}
}
