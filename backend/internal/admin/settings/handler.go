package settings

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	repo *Repository
	db   *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{repo: NewRepository(db), db: db}
}

// ── Settings CRUD ───────────────────────────────────────────────────────────

// GetAllSettings returns all settings grouped by category.
func (h *Handler) GetAllSettings(c *gin.Context) {
	groups, err := h.repo.GroupedSettings()
	if err != nil {
		response.InternalError(c, err)
		return
	}
	// Mask secret values
	for i := range groups {
		for j := range groups[i].Settings {
			groups[i].Settings[j].Value = maskSecret(groups[i].Settings[j])
		}
	}
	response.OK(c, groups)
}

// GetSettingsByCategory returns settings for a single category.
func (h *Handler) GetSettingsByCategory(c *gin.Context) {
	cat := c.Param("category")
	settings, err := h.repo.GetSettingsByCategory(cat)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	for i := range settings {
		settings[i].Value = maskSecret(settings[i])
	}
	response.OK(c, settings)
}

// UpdateSetting updates a single setting by key.
func (h *Handler) UpdateSetting(c *gin.Context) {
	key := c.Param("key")
	adminID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		Value json.RawMessage `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Fetch existing to log old value
	existing, err := h.repo.GetSetting(key)
	if err != nil {
		response.NotFound(c, "setting")
		return
	}

	oldValue := existing.Value
	newValue := string(req.Value)

	if err := h.repo.UpdateSetting(key, newValue, adminID); err != nil {
		response.InternalError(c, err)
		return
	}

	h.logAudit(c, adminID, "update_setting", "setting", key, oldValue, newValue)
	response.OK(c, gin.H{"key": key, "value": newValue})
}

// BulkUpdateSettings updates multiple settings at once.
func (h *Handler) BulkUpdateSettings(c *gin.Context) {
	adminID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		Settings map[string]json.RawMessage `json:"settings" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := make(map[string]string, len(req.Settings))
	for k, v := range req.Settings {
		updates[k] = string(v)
	}

	if err := h.repo.BulkUpdateSettings(updates, adminID); err != nil {
		response.InternalError(c, err)
		return
	}

	h.logAudit(c, adminID, "bulk_update_settings", "setting", "bulk", "", fmt.Sprintf("%d settings updated", len(updates)))
	response.OK(c, gin.H{"updated": len(updates)})
}

// GetPublicConfig is the PUBLIC endpoint — no auth required.
// Returns only is_public=true settings as a flat key:value map.
func (h *Handler) GetPublicConfig(c *gin.Context) {
	settings, err := h.repo.GetPublicSettings()
	if err != nil {
		response.InternalError(c, err)
		return
	}
	result := make(map[string]interface{}, len(settings))
	for _, s := range settings {
		var parsed interface{}
		if err := json.Unmarshal([]byte(s.Value), &parsed); err != nil {
			parsed = s.Value
		}
		result[s.Key] = parsed
	}
	response.OK(c, result)
}

// ── Feature Flags ───────────────────────────────────────────────────────────

// GetAllFeatures returns all feature flags.
func (h *Handler) GetAllFeatures(c *gin.Context) {
	flags, err := h.repo.GetAllFlags()
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, flags)
}

// UpdateFeature toggles or updates a feature flag.
func (h *Handler) UpdateFeature(c *gin.Context) {
	key := c.Param("key")
	adminID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		Enabled       *bool    `json:"enabled"`
		RolloutPct    *int     `json:"rollout_pct"`
		AllowedGroups []string `json:"allowed_groups"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	existing, err := h.repo.GetFlag(key)
	if err != nil {
		response.NotFound(c, "feature flag")
		return
	}

	enabled := existing.Enabled
	rollout := existing.RolloutPct
	groups := existing.AllowedGroups

	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.RolloutPct != nil {
		rollout = *req.RolloutPct
	}
	if req.AllowedGroups != nil {
		groups = req.AllowedGroups
	}

	if err := h.repo.UpdateFlag(key, enabled, rollout, groups); err != nil {
		response.InternalError(c, err)
		return
	}

	h.logAudit(c, adminID, "update_feature", "feature_flag", key,
		fmt.Sprintf("enabled=%v,rollout=%d", existing.Enabled, existing.RolloutPct),
		fmt.Sprintf("enabled=%v,rollout=%d", enabled, rollout))

	response.OK(c, gin.H{"key": key, "enabled": enabled, "rollout_pct": rollout})
}

// GetPublicFeatures returns flags relevant to the current user (public, optionally authed).
func (h *Handler) GetPublicFeatures(c *gin.Context) {
	userID := c.GetString("user_id")
	userGroup := c.GetString("user_role")
	flags, err := h.repo.GetPublicFlags(userID, userGroup)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, flags)
}

// ── Support Tickets (Admin) ─────────────────────────────────────────────────

func (h *Handler) ListTickets(c *gin.Context) {
	page, perPage := paginate(c)
	q := h.db.Model(&SupportTicket{})

	if status := c.Query("status"); status != "" {
		q = q.Where("status = ?", status)
	}

	var total int64
	q.Count(&total)

	var tickets []SupportTicket
	q.Select("support_tickets.*, u.name as user_name").
		Joins("LEFT JOIN users u ON u.id = support_tickets.user_id").
		Offset((page - 1) * perPage).Limit(perPage).
		Order("support_tickets.created_at DESC").
		Scan(&tickets)

	response.OKMeta(c, tickets, response.Meta{
		Total: total, Page: page, PerPage: perPage,
		Pages: (total + int64(perPage) - 1) / int64(perPage),
	})
}

func (h *Handler) GetTicket(c *gin.Context) {
	id := c.Param("id")
	var ticket SupportTicket
	if err := h.db.Preload("Messages").First(&ticket, "id = ?", id).Error; err != nil {
		response.NotFound(c, "ticket")
		return
	}
	response.OK(c, ticket)
}

func (h *Handler) ReplyToTicket(c *gin.Context) {
	ticketID := c.Param("id")
	adminID, _ := uuid.Parse(c.GetString("user_id"))

	var req struct {
		Body string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	msg := TicketMessage{
		TicketID: uuid.MustParse(ticketID),
		SenderID: adminID,
		Body:     req.Body,
		IsAdmin:  true,
	}
	if err := h.db.Create(&msg).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Update ticket status to in_progress
	h.db.Model(&SupportTicket{}).Where("id = ?", ticketID).
		Updates(map[string]interface{}{"status": "in_progress", "assigned_to": adminID, "updated_at": time.Now()})

	response.Created(c, msg)
}

func (h *Handler) UpdateTicketStatus(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Status string `json:"status" binding:"required,oneof=open in_progress waiting resolved closed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := map[string]interface{}{"status": req.Status, "updated_at": time.Now()}
	if req.Status == "closed" || req.Status == "resolved" {
		now := time.Now()
		updates["closed_at"] = &now
	}

	if err := h.db.Model(&SupportTicket{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"status": req.Status})
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// maskSecret hides secret settings values on read.
// Uses the is_secret column (primary) plus heuristic key matching (fallback).
func maskSecret(s AdminSetting) string {
	isSecretKey := false
	if s.IsSecret || s.Type == "secret" {
		isSecretKey = true
	} else {
		secretSuffixes := []string{"_sk", "_secret", "_api_key", "_password", "_pass", "_token", "_private_key"}
		for _, suffix := range secretSuffixes {
			if strings.Contains(s.Key, suffix) {
				isSecretKey = true
				break
			}
		}
	}
	if !isSecretKey {
		return s.Value
	}
	// Strip JSON quotes for length check
	raw := strings.Trim(s.Value, `"`)
	if raw == "" {
		return `"••••••••"`
	}
	if len(raw) <= 4 {
		return `"••••••••"`
	}
	return fmt.Sprintf(`"••••••••%s"`, raw[len(raw)-4:])
}

func (h *Handler) logAudit(c *gin.Context, adminID uuid.UUID, action, targetType, targetID, oldVal, newVal string) {
	oldJSON, _ := json.Marshal(oldVal)
	newJSON, _ := json.Marshal(newVal)
	detailsJSON, _ := json.Marshal(gin.H{"old_value": oldVal, "new_value": newVal})
	// Canonical audit table (migration 047 + 048 columns).
	h.db.Exec(`INSERT INTO admin_audit_log (admin_id, action, target_type, target_id, details, old_value, new_value, ip_address, user_agent, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		adminID, action, targetType, targetID, string(detailsJSON),
		string(oldJSON), string(newJSON), c.ClientIP(), c.GetHeader("User-Agent"), time.Now())
}

func paginate(c *gin.Context) (page, perPage int) {
	page, perPage = 1, 20
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	fmt.Sscan(c.DefaultQuery("per_page", "20"), &perPage)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	return
}

// SeedDefaults seeds all default settings and feature flags.
func (h *Handler) SeedDefaults() {
	h.repo.SeedDefaults()
	slog.Info("Admin settings and feature flags seeded")
}
