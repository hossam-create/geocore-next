package plugins

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	g := v1.Group("/plugins")
	{
		g.GET("", h.List)
		g.GET("/:slug", h.Get)
	}

	auth := g.Group("")
	auth.Use(middleware.Auth())
	{
		auth.POST("", h.Create)
		auth.PATCH("/:slug", h.Update)
		auth.POST("/:slug/publish", h.Publish)
		auth.POST("/:slug/install", h.Install)
		auth.POST("/:slug/uninstall", h.Uninstall)
		auth.GET("/installed", h.MyInstalled)
	}
}
