package chat

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)
	hub := NewHub(rdb)
	go hub.Run()

	g := r.Group("/chat", middleware.Auth())
	{
		g.GET("/conversations", h.GetConversations)
		g.POST("/conversations", h.CreateOrGetConversation)
		g.GET("/conversations/:id/messages", h.GetMessages)
		g.POST("/conversations/:id/messages", h.SendMessage)

		// WebSocket for live chat
		g.GET("/conversations/:id/ws", func(c *gin.Context) {
			ServeWS(hub, c, db)
		})
	}
}
