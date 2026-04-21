package subscriptions

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes registers subscription and plan routes
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Public
	r.GET("/plans", h.ListPlans)

	// Auth required
	subs := r.Group("/subscriptions")
	subs.Use(middleware.Auth())
	{
		subs.GET("/me", h.GetMySubscription)
		subs.POST("", h.CreateSubscription)
		subs.DELETE("/me", h.CancelSubscription)
	}
}
