package growth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ── Growth Handler + Routes ────────────────────────────────────────────────────────

type GrowthHandler struct {
	db       *gorm.DB
	rdb      *redis.Client
	stateSvc *UserStateService
}

func NewGrowthHandler(db *gorm.DB, rdb *redis.Client) *GrowthHandler {
	return &GrowthHandler{
		db:       db,
		rdb:      rdb,
		stateSvc: NewUserStateService(db, rdb),
	}
}

// ── Record Action (POST /growth/action) ────────────────────────────────────────────

type ActionReq struct {
	UserID    string `json:"user_id" binding:"required"`
	Action    string `json:"action" binding:"required"` // view, click, bid, purchase, save, outbid, win, lose
	ItemID    string `json:"item_id"`
	SessionID string `json:"session_id"`
	Metadata  string `json:"metadata"`
}

func (h *GrowthHandler) RecordAction(c *gin.Context) {
	var req ActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	itemID, _ := uuid.Parse(req.ItemID)

	state, err := h.stateSvc.RecordAction(userID, req.Action, itemID, req.SessionID, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Also update dopamine if it's a dopamine-relevant action
	dopamineActions := map[string]string{
		"win":      "win",
		"lose":     "multi_loss",
		"outbid":   "outbid",
		"purchase": "purchase",
		"save":     "save",
	}
	if eventType, ok := dopamineActions[req.Action]; ok {
		UpdateDopamine(h.db, h.stateSvc, userID, eventType, itemID)
	}

	c.JSON(http.StatusOK, gin.H{
		"state":          state,
		"dopamine_score": state.DopamineScore,
		"engagement":     state.EngagementScore,
		"segment":        state.Segment,
	})
}

// ── Get User State (GET /growth/state/:user_id) ────────────────────────────────────

func (h *GrowthHandler) GetUserState(c *gin.Context) {
	userID, _ := uuid.Parse(c.Param("user_id"))
	state, err := h.stateSvc.GetUserState(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User state not found"})
		return
	}

	dopamineAction := GetDopamineAction(h.db, h.stateSvc, userID)
	churnPlan := AssessChurnRisk(h.db, h.stateSvc, userID)

	c.JSON(http.StatusOK, gin.H{
		"state":           state,
		"dopamine_action": dopamineAction,
		"churn_risk":      churnPlan.DropOffRisk,
		"reengagement":    churnPlan.Actions,
	})
}

// ── Decide Next Action (GET /growth/decide/:user_id) ──────────────────────────────

func (h *GrowthHandler) DecideNextAction(c *gin.Context) {
	userID, _ := uuid.Parse(c.Param("user_id"))
	action := DecideNextBestAction(h.db, h.stateSvc, userID)
	c.JSON(http.StatusOK, action)
}

// ── Dopamine Update (POST /growth/dopamine) ────────────────────────────────────────

type DopamineReq struct {
	UserID    string `json:"user_id" binding:"required"`
	EventType string `json:"event_type" binding:"required"` // win, near_win, hot_item, badge, inactivity, multi_loss, outbid
	ItemID    string `json:"item_id"`
}

func (h *GrowthHandler) UpdateDopamine(c *gin.Context) {
	var req DopamineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := uuid.Parse(req.UserID)
	itemID, _ := uuid.Parse(req.ItemID)

	action, err := UpdateDopamine(h.db, h.stateSvc, userID, req.EventType, itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, action)
}

// ── Re-engagement (GET /growth/reengage/:user_id) ──────────────────────────────────

func (h *GrowthHandler) ReEngage(c *gin.Context) {
	userID, _ := uuid.Parse(c.Param("user_id"))
	plan := AssessChurnRisk(h.db, h.stateSvc, userID)
	c.JSON(http.StatusOK, plan)
}

// ── Record Decision Outcome (POST /growth/decision-outcome) ────────────────────────

type DecisionOutcomeReq struct {
	DecisionID string `json:"decision_id" binding:"required"`
	Outcome    string `json:"outcome" binding:"required"` // success, fail
}

func (h *GrowthHandler) RecordDecisionOutcome(c *gin.Context) {
	var req DecisionOutcomeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	decisionID, _ := uuid.Parse(req.DecisionID)
	RecordDecisionOutcome(h.db, decisionID, req.Outcome)
	c.JSON(http.StatusOK, gin.H{"message": "Outcome recorded"})
}

// ── Admin: Dashboard ────────────────────────────────────────────────────────────────

type GrowthDashboard struct {
	DAU                 int64                `json:"dau"`
	MAU                 int64                `json:"mau"`
	AvgSessionTime      float64              `json:"avg_session_time"`
	AvgEngagement       float64              `json:"avg_engagement"`
	AvgDopamine         float64              `json:"avg_dopamine"`
	UsersBySegment      map[string]int64     `json:"users_by_segment"`
	DopamineMetrics     *DopamineMetrics     `json:"dopamine_metrics"`
	DecisionMetrics     *DecisionMetrics     `json:"decision_metrics"`
	ReEngagementMetrics *ReEngagementMetrics `json:"reengagement_metrics"`
}

func (h *GrowthHandler) GetDashboard(c *gin.Context) {
	// DAU: users active in last 24h
	var dau int64
	h.db.Model(&UserState{}).Where("last_active_at > ?", time.Now().Add(-24*time.Hour)).Count(&dau)

	// MAU: users active in last 30 days
	var mau int64
	h.db.Model(&UserState{}).Where("last_active_at > ?", time.Now().Add(-30*24*time.Hour)).Count(&mau)

	// Avg engagement & dopamine
	var avgEng, avgDop, avgSession struct{ Avg float64 }
	h.db.Model(&UserState{}).Select("COALESCE(AVG(engagement_score), 0) as avg").Scan(&avgEng)
	h.db.Model(&UserState{}).Select("COALESCE(AVG(dopamine_score), 0) as avg").Scan(&avgDop)
	h.db.Model(&UserState{}).Select("COALESCE(AVG(session_duration_sec), 0) as avg").Scan(&avgSession)

	// Users by segment
	segments := map[string]int64{}
	for _, s := range []string{"active", "warm", "cold", "churn", "vip"} {
		var count int64
		h.db.Model(&UserState{}).Where("segment = ?", s).Count(&count)
		segments[s] = count
	}

	dashboard := &GrowthDashboard{
		DAU:                 dau,
		MAU:                 mau,
		AvgSessionTime:      avgSession.Avg,
		AvgEngagement:       avgEng.Avg,
		AvgDopamine:         avgDop.Avg,
		UsersBySegment:      segments,
		DopamineMetrics:     GetDopamineMetrics(h.db),
		DecisionMetrics:     GetDecisionMetrics(h.db),
		ReEngagementMetrics: GetReEngagementMetrics(h.db),
	}

	c.JSON(http.StatusOK, dashboard)
}

// ── Register Routes ────────────────────────────────────────────────────────────────

func RegisterGrowthBrainRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewGrowthHandler(db, rdb)

	g := r.Group("/growth")
	g.Use()
	{
		g.POST("/action", h.RecordAction)
		g.GET("/state/:user_id", h.GetUserState)
		g.GET("/decide/:user_id", h.DecideNextAction)
		g.POST("/dopamine", h.UpdateDopamine)
		g.GET("/reengage/:user_id", h.ReEngage)
		g.POST("/decision-outcome", h.RecordDecisionOutcome)
	}

	admin := r.Group("/admin/growth")
	admin.Use()
	{
		admin.GET("/metrics", h.GetDashboard)
	}
}
