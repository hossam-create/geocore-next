package users

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)
	users := r.Group("/users")
	{
		users.GET("/:id/profile", h.GetProfile)
		users.Use(middleware.Auth())
		users.GET("/me", h.GetMe)
		users.PUT("/me", h.UpdateMe)
	}
}
