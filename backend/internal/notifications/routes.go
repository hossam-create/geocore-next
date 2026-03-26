package notifications

  import (
  	"github.com/geocore-next/backend/pkg/middleware"
  	"github.com/gin-gonic/gin"
  	"github.com/redis/go-redis/v9"
  	"gorm.io/gorm"
  )

  // RegisterRoutes mounts all /notifications endpoints and initialises the service.
  // Returns the Hub and Service so main.go can wire up the WebSocket route and
  // call Service.Notify() from other packages.
  func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, _ *redis.Client) (*Hub, *Service) {
  	hub := NewHub()
  	go hub.Run()

  	fcm := NewFCMClientFromEnv()
  	svc := NewService(db, hub, fcm)

  	// Start background escrow auto-release worker
  	StartAutoReleaseWorker(db, svc)

  	h := NewHandler(db, hub, svc)

  	n := r.Group("/notifications")
  	n.Use(middleware.Auth())
  	{
  		n.GET("",                        h.List)
  		n.GET("/unread-count",            h.UnreadCount)
  		n.PUT("/mark-all-read",           h.MarkAllRead)
  		n.PUT("/:id/read",               h.MarkRead)
  		n.DELETE("/:id",                 h.Delete)
  		n.POST("/register-push-token",   h.RegisterPushToken)
  		n.DELETE("/push-tokens/:id",     h.DeletePushToken)
  		n.GET("/preferences",            h.GetPreferences)
  		n.PUT("/preferences",            h.UpdatePreferences)
  	}

  	return hub, svc
  }
  