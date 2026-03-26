package chat

import (
        "github.com/geocore-next/backend/pkg/middleware"
        "github.com/gin-gonic/gin"
        "github.com/redis/go-redis/v9"
        "gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
        hub := NewHub(rdb)
        go hub.Run()

        h := NewHandler(db, rdb)
        h.SetHub(hub)

        g := r.Group("/chat", middleware.Auth())
        {
                g.GET("/conversations", h.GetConversations)
                g.POST("/conversations", h.CreateOrGetConversation)
                g.GET("/conversations/:id/messages", h.GetMessages)
                g.POST("/conversations/:id/messages", h.SendMessage)
        }

        // WebSocket endpoint — auth is handled inside ServeWS via ?token= query param
        // because browsers cannot set Authorization headers on WebSocket connections.
        r.GET("/chat/conversations/:id/ws", func(c *gin.Context) {
                ServeWS(hub, c, db)
        })
}
