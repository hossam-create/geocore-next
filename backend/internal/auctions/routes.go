package auctions

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)
	a := r.Group("/auctions")
	{
		a.GET("", h.List)
		a.GET("/:id", h.Get)
		a.GET("/:id/bids", h.GetBids)
		a.Use(middleware.Auth())
		a.POST("", h.Create)
		a.POST("/:id/bid", h.PlaceBid)
	}
}
