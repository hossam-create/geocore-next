package payments

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)
	pay := r.Group("/payments", middleware.Auth())
	{
		pay.GET("/key", h.GetPublishableKey)
		pay.POST("/intent", h.CreatePaymentIntent)
	}
}
