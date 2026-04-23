package push

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Handler exposes push notification HTTP endpoints.
type Handler struct {
	svc *PushService
	db  *gorm.DB
}

func NewHandler(svc *PushService, db *gorm.DB) *Handler {
	return &Handler{svc: svc, db: db}
}

// ════════════════════════════════════════════════════════════════════════════
// Device registration
// ════════════════════════════════════════════════════════════════════════════

// RegisterDevice upserts a device token for the authenticated user.
// POST /api/v1/push/devices
func (h *Handler) RegisterDevice(c *gin.Context) {
	var req struct {
		DeviceToken string `json:"device_token" binding:"required"`
		Platform    string `json:"platform"    binding:"required"`
		AppVersion  string `json:"app_version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	dev, err := h.svc.RegisterDevice(c.Request.Context(), userID, req.DeviceToken, req.Platform, req.AppVersion)
	if err != nil {
		slog.Error("push: register device failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to register device"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         dev.ID,
		"platform":   dev.Platform,
		"is_active":  dev.IsActive,
		"last_seen":  dev.LastSeenAt,
	})
}

// ListDevices returns all active devices for the authenticated user.
// GET /api/v1/push/devices
func (h *Handler) ListDevices(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var devices []UserDevice
	h.db.Where("user_id = ? AND is_active = true", userID).Find(&devices)
	c.JSON(http.StatusOK, devices)
}

// DeleteDevice soft-deletes a device token.
// DELETE /api/v1/push/devices/:id
func (h *Handler) DeleteDevice(c *gin.Context) {
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device ID"})
		return
	}

	if err := h.svc.UnregisterDevice(c.Request.Context(), deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete device"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "device unregistered"})
}

// ════════════════════════════════════════════════════════════════════════════
// Push delivery
// ════════════════════════════════════════════════════════════════════════════

// SendPush sends a push notification to the authenticated user.
// POST /api/v1/push/send
func (h *Handler) SendPush(c *gin.Context) {
	var req struct {
		NotificationType string            `json:"notification_type" binding:"required"`
		Title            string            `json:"title"            binding:"required"`
		Body             string            `json:"body"`
		Data             map[string]string `json:"data"`
		Priority         string            `json:"priority"` // optional override
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	msg := &PushMessage{
		UserID:           userID,
		NotificationType: req.NotificationType,
		Priority:         req.Priority,
		Title:            req.Title,
		Body:             req.Body,
		Data:             req.Data,
	}

	if err := h.svc.Send(c.Request.Context(), msg); err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "queued"})
}

// ════════════════════════════════════════════════════════════════════════════
// Webhook endpoints (called by FCM / SendGrid for delivery status)
// ════════════════════════════════════════════════════════════════════════════

// HandleFCMWebhook processes FCM delivery status callbacks.
// POST /webhooks/push/fcm
func (h *Handler) HandleFCMWebhook(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// FCM doesn't have native webhooks like SendGrid, but if using
	// FCM with a callback proxy or Pub/Sub integration, this handles it.
	slog.Info("push: FCM webhook received", "payload", payload)
	c.JSON(http.StatusOK, gin.H{"status": "processed"})
}

// HandleEmailWebhook processes SendGrid/SES delivery status callbacks.
// POST /webhooks/email
func (h *Handler) HandleEmailWebhook(c *gin.Context) {
	body, _ := c.GetRawData()

	var events []map[string]any
	if err := json.Unmarshal(body, &events); err != nil {
		// Try single event
		var single map[string]any
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
			return
		}
		events = append(events, single)
	}

	for _, evt := range events {
		eventType, _ := evt["event"].(string)
		email, _ := evt["email"].(string)
		messageID, _ := evt["sg_message_id"].(string) // SendGrid
		if messageID == "" {
			messageID, _ = evt["messageId"].(string) // SES
		}
		reason, _ := evt["reason"].(string)

		slog.Info("email: webhook event",
			"event", eventType,
			"email", email,
			"message_id", messageID,
			"reason", reason,
		)

		// Update email delivery status in the email system
		h.processEmailEvent(eventType, email, messageID, reason)
	}

	c.JSON(http.StatusOK, gin.H{"processed": len(events)})
}

// processEmailEvent updates email delivery logs based on webhook events.
func (h *Handler) processEmailEvent(eventType, email, messageID, reason string) {
	if h.db == nil {
		return
	}

	switch eventType {
	case "delivered":
		// Update push_logs or email tracking table
		slog.Info("email: delivered", "email", email, "message_id", messageID)

	case "bounce", "dropped":
		slog.Warn("email: bounced/dropped", "email", email, "reason", reason)
		// Could update user trust score here

	case "spamreport", "spam":
		slog.Warn("email: spam complaint", "email", email)
		// Mark user as spam reporter — could affect trust/reputation

	case "deferred":
		slog.Info("email: deferred", "email", email, "reason", reason)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Admin endpoints
// ════════════════════════════════════════════════════════════════════════════

// GetPushStats returns push delivery statistics.
// GET /admin/push/stats
func (h *Handler) GetPushStats(c *gin.Context) {
	var stats struct {
		TotalSent      int64 `json:"total_sent"`
		TotalFailed    int64 `json:"total_failed"`
		TotalDelivered int64 `json:"total_delivered"`
		TotalBounced   int64 `json:"total_bounced"`
		ActiveDevices  int64 `json:"active_devices"`
	}
	h.db.Model(&PushLog{}).Where("status = ?", PushStatusSent).Count(&stats.TotalSent)
	h.db.Model(&PushLog{}).Where("status = ?", PushStatusFailed).Count(&stats.TotalFailed)
	h.db.Model(&PushLog{}).Where("status = ?", PushStatusDelivered).Count(&stats.TotalDelivered)
	h.db.Model(&PushLog{}).Where("status = ?", PushStatusBounced).Count(&stats.TotalBounced)
	h.db.Model(&UserDevice{}).Where("is_active = true").Count(&stats.ActiveDevices)

	c.JSON(http.StatusOK, stats)
}

// CleanupDevices triggers stale device cleanup.
// POST /admin/push/cleanup
func (h *Handler) CleanupDevices(c *gin.Context) {
	affected, err := h.svc.CleanupStaleDevices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices_cleaned": affected})
}

// GetRecentPushLogs returns recent push delivery logs for admin debugging.
// GET /admin/push/logs
func (h *Handler) GetRecentPushLogs(c *gin.Context) {
	since := time.Now().Add(-24 * time.Hour)
	if s := c.Query("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}

	status := c.Query("status")
	priority := c.Query("priority")

	q := h.db.Model(&PushLog{}).Where("created_at > ?", since)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if priority != "" {
		q = q.Where("priority = ?", priority)
	}

	var logs []PushLog
	q.Order("created_at DESC").Limit(200).Find(&logs)
	c.JSON(http.StatusOK, logs)
}

// ensure unused import guard
var _ = strings.Contains
