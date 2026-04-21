package p2p

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	g := v1.Group("/p2p")
	{
		g.GET("/requests", h.List)
		g.GET("/requests/:id", h.Get)
	}

	auth := g.Group("")
	auth.Use(middleware.Auth())
	{
		auth.POST("/requests", h.Create)
		auth.POST("/requests/:id/accept", h.Accept)
		auth.POST("/requests/:id/complete", h.Complete)
		auth.POST("/requests/:id/cancel", h.Cancel)
		auth.GET("/requests/:id/messages", h.ListMessages)
		auth.POST("/requests/:id/messages", h.SendMessage)
	}
}
