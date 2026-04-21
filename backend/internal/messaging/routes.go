package messaging

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Messaging Handler + Routes ────────────────────────────────────────────────────

type MessagingHandler struct {
	db        *gorm.DB
	rdb       *redis.Client
	dispatcher *Dispatcher
}

func NewMessagingHandler(db *gorm.DB, rdb *redis.Client) *MessagingHandler {
	return &MessagingHandler{
		db:         db,
		rdb:        rdb,
		dispatcher: NewDispatcher(db, rdb),
	}
}

// ── Send Message (POST /messaging/send) ──────────────────────────────────────────

type SendReq struct {
	UserID   string                 `json:"user_id" binding:"required"`
	Type     string                  `json:"type" binding:"required"` // nudge, reminder, win, loss, promo
	Channel  string                  `json:"channel"`               // push, email, in_app
	Metadata map[string]interface{}  `json:"metadata"`
}

func (h *MessagingHandler) Send(c *gin.Context) {
	var req SendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	channel := req.Channel
	if channel == "" {
		channel = ChannelPush
	}

	result := h.dispatcher.Dispatch(userID, req.Type, channel, req.Metadata)
	c.JSON(http.StatusOK, result)
}

// ── Trigger (POST /messaging/trigger) ─────────────────────────────────────────────

type TriggerReq struct {
	UserID   string `json:"user_id" binding:"required"`
	Trigger  string `json:"trigger" binding:"required"` // inactive, outbid, ending, win, loss
	ItemName string `json:"item_name"`
	TimeLeft string `json:"time_left"`
	Price    string `json:"price"`
}

func (h *MessagingHandler) Trigger(c *gin.Context) {
	var req TriggerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)

	var result *DispatchResult
	switch req.Trigger {
	case "inactive":
		result = h.dispatcher.TriggerInactive(userID)
	case "outbid":
		result = h.dispatcher.TriggerOutbid(userID, req.ItemName)
	case "ending":
		result = h.dispatcher.TriggerItemEnding(userID, req.ItemName, req.TimeLeft)
	case "win":
		result = h.dispatcher.TriggerWin(userID, req.ItemName, req.Price)
	case "loss":
		result = h.dispatcher.TriggerLoss(userID, req.ItemName)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown trigger type"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ── Record Opened (POST /messaging/opened) ────────────────────────────────────────

type OpenedReq struct {
	MessageID string `json:"message_id" binding:"required"`
}

func (h *MessagingHandler) RecordOpened(c *gin.Context) {
	var req OpenedReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msgID, _ := uuid.Parse(req.MessageID)
	now := time.Now()
	h.db.Model(&Message{}).Where("id = ?", msgID).Updates(map[string]interface{}{
		"opened_at": now,
		"status":    "delivered",
	})

	c.JSON(http.StatusOK, gin.H{"message": "Opened recorded"})
}

// ── Update Preferences (PUT /messaging/preferences/:user_id) ──────────────────────

type PrefsReq struct {
	OptOutPush      *bool  `json:"opt_out_push"`
	OptOutEmail     *bool  `json:"opt_out_email"`
	OptOutAll       *bool  `json:"opt_out_all"`
	QuietHoursStart *int   `json:"quiet_hours_start"`
	QuietHoursEnd   *int   `json:"quiet_hours_end"`
	MaxPerHour      *int   `json:"max_per_hour"`
}

func (h *MessagingHandler) UpdatePreferences(c *gin.Context) {
	userID, _ := uuid.Parse(c.Param("user_id"))
	var req PrefsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var prefs UserMessagingPrefs
	if err := h.db.Where("user_id = ?", userID).First(&prefs).Error; err != nil {
		prefs = UserMessagingPrefs{UserID: userID, MaxPerHour: DefaultMaxPerHour}
		h.db.Create(&prefs)
	}

	if req.OptOutPush != nil {
		prefs.OptOutPush = *req.OptOutPush
	}
	if req.OptOutEmail != nil {
		prefs.OptOutEmail = *req.OptOutEmail
	}
	if req.OptOutAll != nil {
		prefs.OptOutAll = *req.OptOutAll
	}
	if req.QuietHoursStart != nil {
		prefs.QuietHoursStart = *req.QuietHoursStart
	}
	if req.QuietHoursEnd != nil {
		prefs.QuietHoursEnd = *req.QuietHoursEnd
	}
	if req.MaxPerHour != nil {
		prefs.MaxPerHour = *req.MaxPerHour
	}
	h.db.Save(&prefs)

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated"})
}

// ── Admin: Metrics ──────────────────────────────────────────────────────────────────

func (h *MessagingHandler) GetMetrics(c *gin.Context) {
	metrics := GetMessagingMetrics(h.db)
	c.JSON(http.StatusOK, metrics)
}

// ── Register Routes ────────────────────────────────────────────────────────────────

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewMessagingHandler(db, rdb)

	m := r.Group("/messaging")
	m.Use()
	{
		m.POST("/send", h.Send)
		m.POST("/trigger", h.Trigger)
		m.POST("/opened", h.RecordOpened)
		m.PUT("/preferences/:user_id", h.UpdatePreferences)
	}

	admin := r.Group("/admin/messaging")
	admin.Use()
	{
		admin.GET("/metrics", h.GetMetrics)
	}
}
