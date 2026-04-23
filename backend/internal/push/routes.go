package push

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all push notification endpoints.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client, fcm FirebaseSender, hub WSBridge) *PushService {
	svc := NewPushService(db, rdb, fcm, hub)
	SetDefault(svc)

	h := NewHandler(svc, db)

	// ── Authenticated endpoints ───────────────────────────────────────────────
	push := r.Group("/push")
	{
		push.POST("/devices",    h.RegisterDevice)
		push.GET("/devices",     h.ListDevices)
		push.DELETE("/devices/:id", h.DeleteDevice)
		push.POST("/send",       h.SendPush)
	}

	// ── Webhook endpoints (no auth — called by providers) ────────────────────
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/push/fcm",  h.HandleFCMWebhook)
		webhooks.POST("/email",     h.HandleEmailWebhook)
	}

	// ── Admin endpoints ──────────────────────────────────────────────────────
	admin := r.Group("/admin/push")
	{
		admin.GET("/stats",   h.GetPushStats)
		admin.POST("/cleanup", h.CleanupDevices)
		admin.GET("/logs",    h.GetRecentPushLogs)
	}

	return svc
}
