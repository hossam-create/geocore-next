package watchlist

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	watchlist := r.Group("/watchlist")
	watchlist.Use(middleware.Auth())
	{
		watchlist.POST("/:listing_id", h.Add)
		watchlist.DELETE("/:listing_id", h.Remove)
		watchlist.GET("", h.List)
	}
}
