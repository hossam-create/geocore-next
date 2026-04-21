package experiments

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Experiments Handler + Routes ──────────────────────────────────────────────────

type ExperimentsHandler struct {
	db *gorm.DB
}

func NewExperimentsHandler(db *gorm.DB) *ExperimentsHandler {
	return &ExperimentsHandler{db: db}
}

// ── Assign Variant (GET /experiments/assign/:experiment_id/:user_id) ──────────────

func (h *ExperimentsHandler) AssignVariant(c *gin.Context) {
	experimentID, _ := uuid.Parse(c.Param("experiment_id"))
	userID, _ := uuid.Parse(c.Param("user_id"))

	variant := AssignUserVariant(h.db, userID, experimentID)
	c.JSON(http.StatusOK, gin.H{"variant": variant})
}

// ── Track Event (POST /experiments/track) ──────────────────────────────────────────

type TrackReq struct {
	ExperimentID string  `json:"experiment_id" binding:"required"`
	UserID       string  `json:"user_id" binding:"required"`
	EventType    string  `json:"event_type" binding:"required"` // click, bid, buy, session_time
	Value        float64 `json:"value"`
}

func (h *ExperimentsHandler) TrackEvent(c *gin.Context) {
	var req TrackReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	experimentID, _ := uuid.Parse(req.ExperimentID)
	userID, _ := uuid.Parse(req.UserID)

	TrackEvent(h.db, userID, experimentID, req.EventType, req.Value)
	c.JSON(http.StatusOK, gin.H{"message": "Event tracked"})
}

// ── Bandit Select (GET /experiments/bandit/:experiment_id) ──────────────────────────

func (h *ExperimentsHandler) BanditSelect(c *gin.Context) {
	experimentID, _ := uuid.Parse(c.Param("experiment_id"))
	arm := ThompsonSample(h.db, experimentID)
	if arm == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No arms found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"arm": arm.ArmName, "arm_id": arm.ID})
}

// ── Bandit Reward (POST /experiments/bandit/reward) ────────────────────────────────

type BanditRewardReq struct {
	ExperimentID string  `json:"experiment_id" binding:"required"`
	ArmID        string  `json:"arm_id" binding:"required"`
	UserID       string  `json:"user_id" binding:"required"`
	Reward       float64 `json:"reward"` // 0 or 1
}

func (h *ExperimentsHandler) BanditReward(c *gin.Context) {
	var req BanditRewardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	experimentID, _ := uuid.Parse(req.ExperimentID)
	armID, _ := uuid.Parse(req.ArmID)
	userID, _ := uuid.Parse(req.UserID)

	RecordBanditPull(h.db, experimentID, armID, userID, req.Reward)
	c.JSON(http.StatusOK, gin.H{"message": "Reward recorded"})
}

// ── Admin: Create Experiment ────────────────────────────────────────────────────────

type CreateExpReq struct {
	Name         string `json:"name" binding:"required"`
	Variants     string `json:"variants" binding:"required"`     // JSON array
	TrafficSplit string `json:"traffic_split" binding:"required"` // JSON object
	Metric       string `json:"metric" binding:"required"`
}

func (h *ExperimentsHandler) CreateExperiment(c *gin.Context) {
	var req CreateExpReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exp := CreateExperiment(h.db, req.Name, req.Variants, req.TrafficSplit, req.Metric)
	c.JSON(http.StatusOK, exp)
}

// ── Admin: Stop Experiment ──────────────────────────────────────────────────────────

func (h *ExperimentsHandler) StopExperiment(c *gin.Context) {
	experimentID, _ := uuid.Parse(c.Param("id"))
	if err := StopExperiment(h.db, experimentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Experiment stopped"})
}

// ── Admin: Get Results ──────────────────────────────────────────────────────────────

func (h *ExperimentsHandler) GetResults(c *gin.Context) {
	experimentID, _ := uuid.Parse(c.Param("id"))
	results := GetExperimentResults(h.db, experimentID)
	c.JSON(http.StatusOK, results)
}

// ── Admin: List Experiments ──────────────────────────────────────────────────────────

func (h *ExperimentsHandler) ListExperiments(c *gin.Context) {
	exps := ListExperiments(h.db)
	c.JSON(http.StatusOK, exps)
}

// ── Admin: Bandit Metrics ────────────────────────────────────────────────────────────

func (h *ExperimentsHandler) GetBanditMetrics(c *gin.Context) {
	experimentID, _ := uuid.Parse(c.Param("id"))
	metrics := GetBanditMetrics(h.db, experimentID)
	c.JSON(http.StatusOK, metrics)
}

// ── Admin: Create Bandit Arms ────────────────────────────────────────────────────────

type CreateArmsReq struct {
	ExperimentID string   `json:"experiment_id" binding:"required"`
	ArmNames     []string `json:"arm_names" binding:"required"`
}

func (h *ExperimentsHandler) CreateArms(c *gin.Context) {
	var req CreateArmsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	experimentID, _ := uuid.Parse(req.ExperimentID)
	CreateBanditArms(h.db, experimentID, req.ArmNames)
	c.JSON(http.StatusOK, gin.H{"message": "Arms created"})
}

// ── Register Routes ────────────────────────────────────────────────────────────────

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewExperimentsHandler(db)

	e := r.Group("/experiments")
	e.Use()
	{
		e.GET("/assign/:experiment_id/:user_id", h.AssignVariant)
		e.POST("/track", h.TrackEvent)
		e.GET("/bandit/:experiment_id", h.BanditSelect)
		e.POST("/bandit/reward", h.BanditReward)
	}

	admin := r.Group("/admin/experiments")
	admin.Use()
	{
		admin.GET("/", h.ListExperiments)
		admin.POST("/", h.CreateExperiment)
		admin.POST("/:id/stop", h.StopExperiment)
		admin.GET("/:id/results", h.GetResults)
		admin.GET("/:id/bandit", h.GetBanditMetrics)
		admin.POST("/bandit/arms", h.CreateArms)
	}
}

// Ensure time used
var _ = time.Now
