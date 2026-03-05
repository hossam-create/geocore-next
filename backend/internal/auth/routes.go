package auth

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)
	a := r.Group("/auth")
	{
		a.POST("/register", h.Register)
		a.POST("/login", h.Login)
		a.GET("/me", middleware.Auth(), h.Me)
	}
}
