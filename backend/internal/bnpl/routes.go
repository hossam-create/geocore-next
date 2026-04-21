package bnpl

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all /bnpl endpoints under /api/v1
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	// Public — list available providers
	v1.GET("/bnpl/providers", h.Providers)

	// Authenticated — create BNPL checkout session
	bnpl := v1.Group("/bnpl")
	bnpl.Use(middleware.Auth())
	{
		bnpl.POST("/create", h.Create)
	}

	// Webhook callbacks (public, called by Tamara/Tabby servers)
	v1.POST("/bnpl/tamara/webhook", h.TamaraWebhook)
	v1.POST("/bnpl/tabby/webhook", h.TabbyWebhook)
}
