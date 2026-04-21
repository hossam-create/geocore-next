package cart

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)

	cart := r.Group("/cart")
	cart.Use(middleware.Auth())
	{
		cart.POST("/items", h.AddItem)
		cart.GET("", h.GetCart)
		cart.DELETE("/items/:listing_id", h.RemoveItem)
		cart.DELETE("", h.ClearCart)
	}
}
