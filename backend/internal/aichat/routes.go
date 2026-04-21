package aichat

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts AI chat routes under /api/v1
func RegisterRoutes(v1 *gin.RouterGroup) {
	h := NewHandler()

	ai := v1.Group("/ai")
	ai.Use(middleware.Auth())
	{
		ai.POST("/chat", h.Chat)
	}
}
