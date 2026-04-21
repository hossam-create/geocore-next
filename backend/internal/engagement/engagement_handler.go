package engagement

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Engagement Handler ────────────────────────────────────────────────────────────

type EngagementHandler struct {
	db *gorm.DB
}

func NewEngagementHandler(db *gorm.DB) *EngagementHandler {
	return &EngagementHandler{db: db}
}

// ── Session Momentum ────────────────────────────────────────────────────────────────

// POST /engagement/momentum/action
type MomentumActionReq struct {
	SessionID string `json:"session_id" binding:"required"`
	UserID    string `json:"user_id" binding:"required"`
	Action    string `json:"action" binding:"required"` // view, click, bid, save, purchase, back
}

func (h *EngagementHandler) RecordAction(c *gin.Context) {
	var req MomentumActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)

	// Ensure session exists
	var m SessionMomentum
	if err := h.db.Where("session_id = ?", req.SessionID).First(&m).Error; err != nil {
		m = SessionMomentum{
			UserID:    userID,
			SessionID: req.SessionID,
		}
		h.db.Create(&m)
	}

	if err := RecordMomentumAction(h.db, req.SessionID, req.Action); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Recompute momentum
	momentum, _ := UpdateMomentum(h.db, userID, req.SessionID)
	config := loadEngagementConfig(h.db)
	rec := GetFeedRecommendation(momentum, config)

	c.JSON(http.StatusOK, gin.H{
		"momentum":           momentum,
		"feed_recommendation": rec,
	})
}

// GET /engagement/momentum/:session_id
func (h *EngagementHandler) GetMomentum(c *gin.Context) {
	sessionID := c.Param("session_id")
	momentum, _ := GetMomentum(h.db, sessionID)
	config := loadEngagementConfig(h.db)
	rec := GetFeedRecommendation(momentum, config)

	c.JSON(http.StatusOK, gin.H{
		"momentum":            momentum,
		"feed_recommendation": rec,
	})
}

// ── Notification AI ──────────────────────────────────────────────────────────────────

// POST /engagement/notification/decide
type DecideReq struct {
	UserID     string  `json:"user_id" binding:"required"`
	EventType  string  `json:"event_type" binding:"required"`
	ValueScore float64 `json:"value_score"`
}

func (h *EngagementHandler) Decide(c *gin.Context) {
	var req DecideReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	if req.ValueScore == 0 {
		req.ValueScore = 0.5
	}

	decision := Decide(h.db, NotifyEvent{
		UserID:     userID,
		EventType:  req.EventType,
		ValueScore: req.ValueScore,
	})

	c.JSON(http.StatusOK, decision)
}

// POST /engagement/notification/send
type SendReq struct {
	UserID    string  `json:"user_id" binding:"required"`
	EventType string  `json:"event_type" binding:"required"`
	Payload   string  `json:"payload"`
	ValueScore float64 `json:"value_score"`
}

func (h *EngagementHandler) Send(c *gin.Context) {
	var req SendReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	if req.ValueScore == 0 {
		req.ValueScore = 0.5
	}

	event := NotifyEvent{
		UserID:     userID,
		EventType:  req.EventType,
		Payload:    req.Payload,
		ValueScore: req.ValueScore,
	}

	decision := Decide(h.db, event)
	if decision.ShouldSend {
		SendNotification(h.db, event, decision)
	}

	c.JSON(http.StatusOK, gin.H{
		"decision": decision,
		"sent":     decision.ShouldSend,
	})
}

// POST /engagement/notification/outcome
type OutcomeReq struct {
	NotificationID string `json:"notification_id" binding:"required"`
	Opened         bool   `json:"opened"`
	Acted          bool   `json:"acted"`
	OptedOut       bool   `json:"opted_out"`
}

func (h *EngagementHandler) RecordOutcome(c *gin.Context) {
	var req OutcomeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notifID, _ := uuid.Parse(req.NotificationID)
	if err := RecordNotificationOutcome(h.db, notifID, req.Opened, req.Acted, req.OptedOut); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Outcome recorded"})
}

// ── Re-engagement ────────────────────────────────────────────────────────────────────

// GET /engagement/segment/:user_id
func (h *EngagementHandler) GetSegment(c *gin.Context) {
	userID, _ := uuid.Parse(c.Param("user_id"))
	segment := SegmentUser(h.db, userID)
	profile := loadOrCreateProfile(h.db, userID)

	c.JSON(http.StatusOK, gin.H{
		"segment":          segment,
		"last_active":      profile.LastActiveAt,
		"notifications_today": profile.NotificationsToday,
		"open_rate":        profile.OpenRate,
	})
}

// POST /engagement/reengage/:user_id
func (h *EngagementHandler) PlanReEngagement(c *gin.Context) {
	userID, _ := uuid.Parse(c.Param("user_id"))
	touches := PlanReEngagement(h.db, userID)

	c.JSON(http.StatusOK, gin.H{
		"planned_touches": touches,
	})
}

// ── Timing ────────────────────────────────────────────────────────────────────────────

// POST /engagement/activity
type ActivityReq struct {
	UserID string `json:"user_id" binding:"required"`
}

func (h *EngagementHandler) RecordActivity(c *gin.Context) {
	var req ActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)

	// Update profile last active
	profile := loadOrCreateProfile(h.db, userID)
	now := time.Now()
	profile.LastActiveAt = &now
	h.db.Save(profile)

	// Record activity hour
	RecordActivity(h.db, userID)

	c.JSON(http.StatusOK, gin.H{"message": "Activity recorded"})
}

// GET /engagement/best-time/:user_id
func (h *EngagementHandler) GetBestTime(c *gin.Context) {
	userID, _ := uuid.Parse(c.Param("user_id"))
	hour, score := GetBestSendTime(h.db, userID)

	c.JSON(http.StatusOK, gin.H{
		"best_hour": hour,
		"score":     score,
	})
}

// ── User Preferences ────────────────────────────────────────────────────────────────

// PUT /engagement/preferences/:user_id
type PreferencesReq struct {
	OptOutPush    *bool  `json:"opt_out_push"`
	OptOutEmail   *bool  `json:"opt_out_email"`
	OptOutAll     *bool  `json:"opt_out_all"`
	QuietStart    *int   `json:"quiet_hours_start"`
	QuietEnd      *int   `json:"quiet_hours_end"`
	Channels      *string `json:"preferred_channels"`
}

func (h *EngagementHandler) UpdatePreferences(c *gin.Context) {
	userID, _ := uuid.Parse(c.Param("user_id"))
	var req PreferencesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile := loadOrCreateProfile(h.db, userID)
	if req.OptOutPush != nil {
		profile.OptOutPush = *req.OptOutPush
	}
	if req.OptOutEmail != nil {
		profile.OptOutEmail = *req.OptOutEmail
	}
	if req.OptOutAll != nil {
		profile.OptOutAll = *req.OptOutAll
	}
	if req.QuietStart != nil {
		profile.QuietHoursStart = *req.QuietStart
	}
	if req.QuietEnd != nil {
		profile.QuietHoursEnd = *req.QuietEnd
	}
	if req.Channels != nil {
		profile.PreferredChannels = *req.Channels
	}
	h.db.Save(&profile)

	c.JSON(http.StatusOK, gin.H{"message": "Preferences updated"})
}

// ── Admin ────────────────────────────────────────────────────────────────────────────

func (h *EngagementHandler) GetDashboard(c *gin.Context) {
	dashboard := GetEngagementDashboard(h.db)
	c.JSON(http.StatusOK, dashboard)
}

func (h *EngagementHandler) ActivateKillSwitch(c *gin.Context) {
	h.db.Model(&EngagementConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{"kill_switch_active": true, "updated_at": time.Now()})
	c.JSON(http.StatusOK, gin.H{"message": "Engagement kill switch activated"})
}

func (h *EngagementHandler) DeactivateKillSwitch(c *gin.Context) {
	h.db.Model(&EngagementConfig{}).Where("is_active = ?", true).
		Updates(map[string]interface{}{"kill_switch_active": false, "updated_at": time.Now()})
	c.JSON(http.StatusOK, gin.H{"message": "Engagement kill switch deactivated"})
}

func (h *EngagementHandler) SegmentAll(c *gin.Context) {
	counts := SegmentAllUsers(h.db)
	c.JSON(http.StatusOK, gin.H{"segments": counts})
}

func (h *EngagementHandler) ProcessTouches(c *gin.Context) {
	sent := ProcessPlannedTouches(h.db)
	c.JSON(http.StatusOK, gin.H{"sent": sent})
}

// GetEngagementDashboard builds the admin dashboard.
func GetEngagementDashboard(db *gorm.DB) *EngagementDashboard {
	var totalSent int64
	db.Model(&NotificationEvent{}).Where("sent_at IS NOT NULL").Count(&totalSent)

	var totalOpened, totalActed, totalOpted int64
	db.Model(&NotificationEvent{}).Where("opened = ?", true).Count(&totalOpened)
	db.Model(&NotificationEvent{}).Where("acted = ?", true).Count(&totalActed)
	db.Model(&NotificationEvent{}).Where("opted_out = ?", true).Count(&totalOpted)

	openRate := 0.0
	actRate := 0.0
	optOutRate := 0.0
	if totalSent > 0 {
		openRate = float64(totalOpened) / float64(totalSent)
		actRate = float64(totalActed) / float64(totalSent)
		optOutRate = float64(totalOpted) / float64(totalSent)
	}

	// Users by segment
	segments := map[string]int64{}
	for _, s := range []UserSegmentType{SegmentActive, SegmentWarm, SegmentCold, SegmentChurnRisk} {
		var count int64
		db.Model(&UserEngagementProfile{}).Where("segment = ?", s).Count(&count)
		segments[string(s)] = count
	}

	// Avg momentum
	var avgMomentum struct{ Avg float64 }
	db.Model(&SessionMomentum{}).Select("COALESCE(AVG(momentum_score), 0) as avg").Scan(&avgMomentum)

	// Top event types
	var eventTypes []EventTypeStats
	db.Model(&NotificationEvent{}).
		Select("event_type, COUNT(*) as count, "+
			"CAST(SUM(CASE WHEN opened THEN 1 ELSE 0 END) AS FLOAT) / NULLIF(COUNT(*), 0) as open_rate, "+
			"CAST(SUM(CASE WHEN acted THEN 1 ELSE 0 END) AS FLOAT) / NULLIF(COUNT(*), 0) as act_rate").
		Group("event_type").Order("count DESC").Limit(5).Scan(&eventTypes)

	config := loadEngagementConfig(db)

	return &EngagementDashboard{
		TotalNotificationsSent: totalSent,
		OpenRate:               openRate,
		ActRate:                actRate,
		OptOutRate:             optOutRate,
		UsersBySegment:         segments,
		AvgMomentumScore:       avgMomentum.Avg,
		KillSwitchActive:       config.KillSwitchActive,
		TopEventTypes:          eventTypes,
	}
}
