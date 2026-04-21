package controltower

import (
	"io"
	"net/http"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Handler holds dependencies for all control tower endpoints.
type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
	bus *EventBus
}

func NewHandler(db *gorm.DB, rdb *redis.Client, bus *EventBus) *Handler {
	return &Handler{db: db, rdb: rdb, bus: bus}
}

// ─── Part 1: System Metrics ───────────────────────────────────────────────────

// MetricsHandler — GET /admin/system/metrics
func (h *Handler) MetricsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, GetSystemMetrics(h.db, h.rdb))
}

// BlockedIPsHandler — GET /admin/system/metrics/blocked
func (h *Handler) BlockedIPsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"blocked_ips": BlockedIPList(h.rdb),
		"captured_at": time.Now().UTC(),
	})
}

// ─── Part 2: Fraud Radar ──────────────────────────────────────────────────────

// FraudHandler — GET /admin/system/fraud
func (h *Handler) FraudHandler(c *gin.Context) {
	c.JSON(http.StatusOK, GetFraudRadar(h.db))
}

// ─── Part 3: Revenue ─────────────────────────────────────────────────────────

// RevenueHandler — GET /admin/system/revenue
func (h *Handler) RevenueHandler(c *gin.Context) {
	c.JSON(http.StatusOK, GetRevenueDashboard(h.db))
}

// ─── Part 4: Liquidity ───────────────────────────────────────────────────────

// LiquidityHandler — GET /admin/system/liquidity
func (h *Handler) LiquidityHandler(c *gin.Context) {
	c.JSON(http.StatusOK, GetLiquidityDashboard(h.db))
}

// ─── Part 5: Growth ──────────────────────────────────────────────────────────

// GrowthHandler — GET /admin/system/growth
func (h *Handler) GrowthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, GetGrowthDashboard(h.db))
}

// ─── Part 6: Event Stream (SSE) ───────────────────────────────────────────────

// EventStreamHandler — GET /admin/system/events
// Streams SystemEvents as Server-Sent Events (text/event-stream).
// Client example: const evtSrc = new EventSource('/api/v1/admin/system/events', {withCredentials:true})
func (h *Handler) EventStreamHandler(c *gin.Context) {
	if h.bus == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "event bus not initialised"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx buffering

	sub := h.bus.Subscribe()
	defer h.bus.Unsubscribe(sub)

	clientGone := c.Request.Context().Done()
	c.Stream(func(w io.Writer) bool {
		select {
		case <-clientGone:
			return false
		case evt, ok := <-sub:
			if !ok {
				return false
			}
			c.SSEvent("message", evt)
			return true
		case <-time.After(25 * time.Second):
			// Heartbeat to keep the connection alive through proxies.
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			return true
		}
	})
}

// ─── Part 7: Emergency Control ───────────────────────────────────────────────

// EmergencyHandler — POST /admin/system/emergency
// Body: {"enabled": true}
// Toggles ENABLE_EMERGENCY_MODE at runtime and broadcasts the change.
func (h *Handler) EmergencyHandler(c *gin.Context) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.SetFlag("emergency_mode", body.Enabled)

	// Broadcast the change to event stream.
	msg := "Emergency mode DEACTIVATED — writes restored"
	sev := SevInfo
	if body.Enabled {
		msg = "Emergency mode ACTIVATED — all write ops blocked"
		sev = SevCritical
	}
	Emit("emergency_mode", sev, msg, c.GetString("user_id"), c.ClientIP())

	// Audit log.
	security.LogEventDirect(h.db, nil, security.EventAdminAction,
		c.ClientIP(), c.Request.UserAgent(),
		map[string]any{"action": "emergency_mode_toggle", "enabled": body.Enabled},
	)

	c.JSON(http.StatusOK, gin.H{
		"emergency_mode": body.Enabled,
		"message":        msg,
	})
}

// EmergencyStatusHandler — GET /admin/system/emergency
func (h *Handler) EmergencyStatusHandler(c *gin.Context) {
	active := config.GetFlags().EnableEmergencyMode
	c.JSON(http.StatusOK, gin.H{
		"emergency_mode": active,
		"captured_at":    time.Now().UTC(),
	})
}
