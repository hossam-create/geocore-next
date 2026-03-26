package stores

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	s := r.Group("/stores")
	{
		s.GET("", h.List)
		s.GET("/:slug", h.GetBySlug)

		authed := s.Group("")
		authed.Use(middleware.Auth())
		{
			authed.GET("/me", h.GetMyStore)
			authed.POST("", h.Create)
			authed.PUT("/me", h.Update)
		}
	}
}
