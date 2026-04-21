package deals

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	deals := r.Group("/deals")
	{
		// Public routes
		deals.GET("", h.GetDeals)
		deals.GET("/:id", h.GetDeal)

		// Protected routes
		deals.Use(middleware.Auth())
		{
			deals.POST("", h.CreateDeal)
			deals.GET("/my", h.GetMyDeals)
			deals.DELETE("/:id", h.CancelDeal)
		}
	}
}
