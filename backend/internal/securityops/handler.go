package securityops

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/geocore-next/backend/internal/controltower"
	"github.com/geocore-next/backend/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Handler groups all Sprint 23 admin endpoints for security observability.
type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
	ids *security.IDS
}

func NewHandler(db *gorm.DB, rdb *redis.Client, ids *security.IDS) *Handler {
	return &Handler{db: db, rdb: rdb, ids: ids}
}

// ─── Part 3: Real-time Activity Stream ───────────────────────────────────────

// LiveHandler — GET /admin/security/live
// Returns the last 100 events + streams new ones via SSE when ?stream=1.
func (h *Handler) LiveHandler(c *gin.Context) {
	if c.Query("stream") == "1" {
		h.streamEvents(c)
		return
	}
	var events []security.SecurityAuditLog
	h.db.Order("created_at DESC").Limit(100).Find(&events)
	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"count":       len(events),
		"captured_at": time.Now().UTC(),
	})
}

func (h *Handler) streamEvents(c *gin.Context) {
	bus := controltower.GetBus()
	if bus == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "event bus not initialised"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	sub := bus.Subscribe()
	defer bus.Unsubscribe(sub)

	clientGone := c.Request.Context().Done()
	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case evt, ok := <-sub:
			if !ok {
				return false
			}
			c.SSEvent("security", evt)
			return true
		case <-time.After(25 * time.Second):
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			return true
		}
	})
}

// ─── Part 4/11: User Risk Profile ────────────────────────────────────────────

// UserRiskHandler — GET /admin/security/users/:userId
func (h *Handler) UserRiskHandler(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	profile := security.GetRiskProfile(h.db, uid)

	// Recent events for context (last 50).
	var recent []security.SecurityAuditLog
	h.db.Where("user_id = ?", uid).Order("created_at DESC").Limit(50).Find(&recent)

	c.JSON(http.StatusOK, gin.H{
		"profile":    profile,
		"risk_level": security.LevelFromScore(profile.RiskScore),
		"recent":     recent,
	})
}

// ─── Part 6: Admin Control Actions ───────────────────────────────────────────

// FreezeHandler — POST /admin/security/freeze/:userId
// Body: {"reason": "manual admin freeze"}
func (h *Handler) FreezeHandler(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Reason == "" {
		body.Reason = "manual admin freeze"
	}
	security.FreezeUser(h.db, uid, body.Reason)
	controltower.Emit("user_frozen", controltower.SevWarning,
		"Admin froze user "+uid.String(), uid.String(), c.ClientIP())
	security.LogEventDirect(h.db, &uid, security.EventAdminAction,
		c.ClientIP(), c.Request.UserAgent(),
		map[string]any{"action": "freeze", "reason": body.Reason})
	c.JSON(http.StatusOK, gin.H{"frozen": true, "user_id": uid})
}

// UnfreezeHandler — POST /admin/security/unfreeze/:userId
func (h *Handler) UnfreezeHandler(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)
	security.UnfreezeUser(h.db, uid, body.Reason)
	controltower.Emit("user_unfrozen", controltower.SevInfo,
		"Admin unfroze user "+uid.String(), uid.String(), c.ClientIP())
	security.LogEventDirect(h.db, &uid, security.EventAdminAction,
		c.ClientIP(), c.Request.UserAgent(),
		map[string]any{"action": "unfreeze", "reason": body.Reason})
	c.JSON(http.StatusOK, gin.H{"frozen": false, "user_id": uid})
}

// BlockIPHandler — POST /admin/security/block-ip
// Body: {"ip":"1.2.3.4","reason":"abuse","duration_minutes":60}
func (h *Handler) BlockIPHandler(c *gin.Context) {
	var body struct {
		IP              string `json:"ip" binding:"required"`
		Reason          string `json:"reason"`
		DurationMinutes int    `json:"duration_minutes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dur := time.Duration(body.DurationMinutes) * time.Minute
	if dur <= 0 {
		dur = 60 * time.Minute
	}
	if body.Reason == "" {
		body.Reason = "manual admin block"
	}
	h.ids.Block(c.Request.Context(), body.IP, body.Reason, dur)
	controltower.Emit("admin_block_ip", controltower.SevWarning,
		"Admin blocked IP "+body.IP, "", body.IP)
	security.LogEventDirect(h.db, nil, security.EventAdminAction,
		c.ClientIP(), c.Request.UserAgent(),
		map[string]any{"action": "block_ip", "ip": body.IP, "reason": body.Reason})
	c.JSON(http.StatusOK, gin.H{"blocked": body.IP, "duration": dur.String()})
}

// UnblockIPHandler — POST /admin/security/unblock-ip
// Body: {"ip":"1.2.3.4"}
func (h *Handler) UnblockIPHandler(c *gin.Context) {
	var body struct {
		IP string `json:"ip" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.rdb != nil {
		h.rdb.Del(c.Request.Context(), "ids:blocked:"+body.IP)
	}
	controltower.Emit("admin_unblock_ip", controltower.SevInfo,
		"Admin unblocked IP "+body.IP, "", body.IP)
	security.LogEventDirect(h.db, nil, security.EventAdminAction,
		c.ClientIP(), c.Request.UserAgent(),
		map[string]any{"action": "unblock_ip", "ip": body.IP})
	c.JSON(http.StatusOK, gin.H{"unblocked": body.IP})
}

// ─── Part 7: System Health Panel ──────────────────────────────────────────────

// HealthHandler — GET /admin/system/health
func (h *Handler) HealthHandler(c *gin.Context) {
	var activeUsers, rpsCount, errors, fraudToday, blockedUsers, rateLimitHits int64
	ctx := context.Background()

	h.db.Raw(`SELECT COUNT(DISTINCT user_id) FROM security_audit_log
	          WHERE created_at > NOW() - INTERVAL '15 minutes' AND user_id IS NOT NULL`).Scan(&activeUsers)
	h.db.Raw(`SELECT COUNT(*) FROM security_audit_log WHERE created_at > NOW() - INTERVAL '60 seconds'`).Scan(&rpsCount)
	h.db.Raw(`SELECT COUNT(*) FROM security_audit_log
	          WHERE event_type IN ('login_failed','rate_limited','fraud_flag')
	            AND created_at > NOW() - INTERVAL '5 minutes'`).Scan(&errors)
	h.db.Raw(`SELECT COUNT(*) FROM exchange_risk_flags WHERE created_at >= CURRENT_DATE`).Scan(&fraudToday)
	h.db.Raw(`SELECT COUNT(*) FROM user_security_profiles WHERE frozen = true`).Scan(&blockedUsers)
	h.db.Raw(`SELECT COUNT(*) FROM security_audit_log
	          WHERE event_type = 'rate_limited' AND created_at >= CURRENT_DATE`).Scan(&rateLimitHits)

	blockedIPs := int64(0)
	if h.rdb != nil {
		keys, _ := h.rdb.Keys(ctx, "ids:blocked:*").Result()
		blockedIPs = int64(len(keys))
	}

	rps := float64(rpsCount) / 60.0
	errorRate := 0.0
	if rpsCount > 0 {
		errorRate = float64(errors) / float64(rpsCount) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"active_users":          activeUsers,
		"requests_per_second":   rps,
		"error_rate_pct":        errorRate,
		"fraud_flags_today":     fraudToday,
		"blocked_users":         blockedUsers,
		"blocked_ips":           blockedIPs,
		"rate_limit_hits_today": rateLimitHits,
		"captured_at":           time.Now().UTC(),
	})
}

// ─── Part 8: Threat Dashboard ────────────────────────────────────────────────

// OverviewHandler — GET /admin/security/overview
func (h *Handler) OverviewHandler(c *gin.Context) {
	type topIP struct {
		IPAddress string `json:"ip"`
		Count     int64  `json:"count"`
	}
	type topRule struct {
		EventType string `json:"event_type"`
		Count     int64  `json:"count"`
	}

	var highRisk int64
	var suspicious24h int64
	var topIPs []topIP
	var topRules []topRule

	h.db.Raw(`SELECT COUNT(*) FROM user_security_profiles WHERE risk_score >= 61`).Scan(&highRisk)
	h.db.Raw(`SELECT COUNT(*) FROM security_audit_log
	          WHERE created_at > NOW() - INTERVAL '24 hours'
	            AND (risk_score >= 40
	                 OR event_type IN ('fraud_flag','suspicious_activity','rapid_actions','replay_attack','auto_block'))`).Scan(&suspicious24h)

	h.db.Raw(`SELECT ip_address, COUNT(*) AS count
	          FROM security_audit_log
	          WHERE created_at > NOW() - INTERVAL '24 hours'
	          GROUP BY ip_address
	          ORDER BY count DESC
	          LIMIT 20`).Scan(&topIPs)

	h.db.Raw(`SELECT event_type, COUNT(*) AS count
	          FROM security_audit_log
	          WHERE created_at > NOW() - INTERVAL '24 hours'
	          GROUP BY event_type
	          ORDER BY count DESC
	          LIMIT 10`).Scan(&topRules)

	// Top high-risk users.
	type topUser struct {
		UserID    uuid.UUID `json:"user_id"`
		RiskScore int       `json:"risk_score"`
		Frozen    bool      `json:"frozen"`
	}
	var topUsers []topUser
	h.db.Raw(`SELECT user_id, risk_score, frozen FROM user_security_profiles
	          WHERE risk_score >= 40
	          ORDER BY risk_score DESC LIMIT 20`).Scan(&topUsers)

	c.JSON(http.StatusOK, gin.H{
		"high_risk_users":       highRisk,
		"suspicious_events_24h": suspicious24h,
		"top_ips":               topIPs,
		"most_triggered_rules":  topRules,
		"top_risky_users":       topUsers,
		"captured_at":           time.Now().UTC(),
	})
}

// UsersHandler — GET /admin/security/users?min_score=40&limit=50
// Paginated list of risk profiles for the admin table view.
func (h *Handler) UsersHandler(c *gin.Context) {
	minScore, _ := strconv.Atoi(c.DefaultQuery("min_score", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var profiles []security.UserRiskProfile
	h.db.Where("risk_score >= ?", minScore).
		Order("risk_score DESC").
		Limit(limit).
		Find(&profiles)

	// Tag each with risk level for UI colour coding.
	out := make([]map[string]any, 0, len(profiles))
	for _, p := range profiles {
		out = append(out, map[string]any{
			"profile":    p,
			"risk_level": security.LevelFromScore(p.RiskScore),
		})
	}
	c.JSON(http.StatusOK, gin.H{"users": out, "count": len(out)})
}
