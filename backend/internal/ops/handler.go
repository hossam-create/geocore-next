package ops

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	rdb       *redis.Client
	scheduler *CronScheduler
	alertEng  *AlertEngine
}

func NewHandler(db *gorm.DB, rdb *redis.Client, scheduler *CronScheduler, alertEng *AlertEngine) *Handler {
	return &Handler{db: db, rdb: rdb, scheduler: scheduler, alertEng: alertEng}
}

// ═══════════════════════════════════════════════════════════
// Health / System Status
// ═══════════════════════════════════════════════════════════

func (h *Handler) GetStatus(c *gin.Context) {
	dbOK := true
	if sql, err := h.db.DB(); err != nil || sql.PingContext(c.Request.Context()) != nil {
		dbOK = false
	}
	redisOK := h.rdb.Ping(c.Request.Context()).Err() == nil

	var jobStats map[string]interface{}
	if h.scheduler != nil && h.scheduler.jobQueue != nil {
		jobStats = h.scheduler.jobQueue.GetStats()
	}

	var pendingAlerts int64
	h.db.Model(&AlertHistory{}).Where("fired_at >= ?", time.Now().Add(-24*time.Hour)).Count(&pendingAlerts)

	status := "healthy"
	if !dbOK || !redisOK {
		status = "degraded"
	}

	response.OK(c, gin.H{
		"status":         status,
		"db":             dbOK,
		"redis":          redisOK,
		"job_queue":      jobStats,
		"alerts_24h":     pendingAlerts,
		"server_time":    time.Now().UTC(),
	})
}

// ═══════════════════════════════════════════════════════════
// Cron Schedules
// ═══════════════════════════════════════════════════════════

func (h *Handler) ListCron(c *gin.Context) {
	var items []CronSchedule
	h.db.Order("name").Find(&items)
	response.OK(c, items)
}

func (h *Handler) CreateCron(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Schedule    string `json:"schedule" binding:"required"`
		Action      string `json:"action" binding:"required"`
		Payload     string `json:"payload"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := validateCronExpr(req.Schedule); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	next := nextRunAfter(req.Schedule, time.Now().UTC())
	item := CronSchedule{
		Name:        req.Name,
		Description: req.Description,
		Schedule:    req.Schedule,
		Action:      req.Action,
		Payload:     req.Payload,
		Enabled:     enabled,
		NextRunAt:   next,
	}
	if err := h.db.Create(&item).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *Handler) UpdateCron(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var item CronSchedule
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		response.NotFound(c, "cron schedule")
		return
	}
	var req struct {
		Description string `json:"description"`
		Schedule    string `json:"schedule"`
		Action      string `json:"action"`
		Payload     string `json:"payload"`
		Enabled     *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updates := map[string]interface{}{}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Schedule != "" {
		if err := validateCronExpr(req.Schedule); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		updates["schedule"] = req.Schedule
		updates["next_run_at"] = nextRunAfter(req.Schedule, time.Now().UTC())
	}
	if req.Action != "" {
		updates["action"] = req.Action
	}
	if req.Payload != "" {
		updates["payload"] = req.Payload
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	h.db.Model(&item).Updates(updates)
	response.OK(c, item)
}

func (h *Handler) DeleteCron(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	result := h.db.Delete(&CronSchedule{}, "id = ?", id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "cron schedule")
		return
	}
	response.OK(c, gin.H{"message": "deleted"})
}

// ═══════════════════════════════════════════════════════════
// Alert Rules
// ═══════════════════════════════════════════════════════════

func (h *Handler) ListAlerts(c *gin.Context) {
	var items []AlertRule
	h.db.Order("name").Find(&items)
	response.OK(c, items)
}

func (h *Handler) CreateAlert(c *gin.Context) {
	var req struct {
		Name      string  `json:"name" binding:"required"`
		Metric    string  `json:"metric" binding:"required"`
		Condition string  `json:"condition" binding:"required"`
		Threshold float64 `json:"threshold"`
		Window    string  `json:"window"`
		Enabled   *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	validConditions := map[string]bool{"gt": true, "gte": true, "lt": true, "lte": true, "eq": true}
	if !validConditions[req.Condition] {
		response.BadRequest(c, "condition must be one of: gt, gte, lt, lte, eq")
		return
	}
	window := req.Window
	if window == "" {
		window = "1h"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	item := AlertRule{
		Name:      req.Name,
		Metric:    req.Metric,
		Condition: req.Condition,
		Threshold: req.Threshold,
		Window:    window,
		Enabled:   enabled,
	}
	if err := h.db.Create(&item).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *Handler) UpdateAlert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	var item AlertRule
	if err := h.db.First(&item, "id = ?", id).Error; err != nil {
		response.NotFound(c, "alert rule")
		return
	}
	var req struct {
		Name      string   `json:"name"`
		Metric    string   `json:"metric"`
		Condition string   `json:"condition"`
		Threshold *float64 `json:"threshold"`
		Window    string   `json:"window"`
		Enabled   *bool    `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Metric != "" {
		updates["metric"] = req.Metric
	}
	if req.Condition != "" {
		updates["condition"] = req.Condition
	}
	if req.Threshold != nil {
		updates["threshold"] = *req.Threshold
	}
	if req.Window != "" {
		updates["window"] = req.Window
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}
	h.db.Model(&item).Updates(updates)
	response.OK(c, item)
}

func (h *Handler) DeleteAlert(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}
	result := h.db.Delete(&AlertRule{}, "id = ?", id)
	if result.RowsAffected == 0 {
		response.NotFound(c, "alert rule")
		return
	}
	response.OK(c, gin.H{"message": "deleted"})
}

func (h *Handler) GetAlertHistory(c *gin.Context) {
	var items []AlertHistory
	h.db.Order("fired_at DESC").Limit(100).Find(&items)
	response.OK(c, items)
}

func (h *Handler) GetAlertMetrics(c *gin.Context) {
	response.OK(c, gin.H{"metrics": AvailableMetrics()})
}

// ═══════════════════════════════════════════════════════════
// Runtime Config
// ═══════════════════════════════════════════════════════════

func (h *Handler) ListConfig(c *gin.Context) {
	items, err := ConfigGetAll(h.db)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, items)
}

func (h *Handler) SetConfig(c *gin.Context) {
	var req struct {
		Key      string `json:"key" binding:"required"`
		Value    string `json:"value" binding:"required"`
		IsSecret bool   `json:"is_secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	updatedBy := c.GetString("user_id")
	if err := ConfigSet(h.db, h.rdb, req.Key, req.Value, updatedBy, req.IsSecret); err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"message": "config updated", "key": req.Key})
}

func (h *Handler) DeleteConfig(c *gin.Context) {
	key := c.Param("key")
	result := h.db.Delete(&OpsConfig{}, "key = ?", key)
	if result.RowsAffected == 0 {
		response.NotFound(c, "config key")
		return
	}
	if h.rdb != nil {
		h.rdb.Del(c.Request.Context(), configCachePrefix+key)
	}
	response.OK(c, gin.H{"message": "config deleted"})
}

func (h *Handler) BulkSetConfig(c *gin.Context) {
	var req []struct {
		Key      string `json:"key"`
		Value    string `json:"value"`
		IsSecret bool   `json:"is_secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// try single object
		response.BadRequest(c, err.Error())
		return
	}
	updatedBy := c.GetString("user_id")
	for _, item := range req {
		if item.Key == "" {
			continue
		}
		_ = ConfigSet(h.db, h.rdb, item.Key, item.Value, updatedBy, item.IsSecret)
	}
	response.OK(c, gin.H{"message": "config updated", "count": len(req)})
}

// ═══════════════════════════════════════════════════════════
// Job Queue Stats
// ═══════════════════════════════════════════════════════════

func (h *Handler) GetJobStats(c *gin.Context) {
	if h.scheduler == nil || h.scheduler.jobQueue == nil {
		response.OK(c, gin.H{"stats": nil})
		return
	}
	stats := h.scheduler.jobQueue.GetStats()
	response.OK(c, gin.H{"stats": stats})
}

func (h *Handler) RetryFailedJobs(c *gin.Context) {
	if h.scheduler == nil || h.scheduler.jobQueue == nil {
		response.BadRequest(c, "job queue not available")
		return
	}
	count := h.scheduler.jobQueue.RetryFailed()
	response.OK(c, gin.H{"retried": count})
}

func (h *Handler) GetFailedJobs(c *gin.Context) {
	raw, err := h.rdb.LRange(c.Request.Context(), "jobs:failed", 0, 49).Result()
	if err != nil {
		response.InternalError(c, err)
		return
	}
	jobs := make([]interface{}, 0, len(raw))
	for _, r := range raw {
		var j interface{}
		if err := json.Unmarshal([]byte(r), &j); err == nil {
			jobs = append(jobs, j)
		}
	}
	response.OK(c, gin.H{"failed_jobs": jobs, "count": len(jobs)})
}
