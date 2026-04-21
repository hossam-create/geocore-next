package forex

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all /forex endpoints.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	fx := r.Group("/forex")
	fx.Use(middleware.Auth())
	{
		fx.GET("/rate", h.GetRate)
		fx.POST("/convert", h.Convert)
	}
}
