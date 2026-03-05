package listings

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)

	r.GET("/categories", h.GetCategories)

	listings := r.Group("/listings")
	{
		listings.GET("", h.List)
		listings.GET("/:id", h.Get)
		listings.Use(middleware.Auth())
		listings.POST("", h.Create)
		listings.PUT("/:id", h.Update)
		listings.DELETE("/:id", h.Delete)
		listings.POST("/:id/favorite", h.ToggleFavorite)
		listings.GET("/me", h.GetMyListings)
	}
}
