package redteam

import (
	"net/http"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler owns the red-team Simulator and serves admin endpoints.
type Handler struct {
	sim *Simulator
}

func NewHandler(sim *Simulator) *Handler { return &Handler{sim: sim} }

// RunHandler — POST /admin/redteam/run
// Body: {"scenario": "spam|referral|exchange|bids"}
// Returns a ScenarioResult summarising defensive outcome.
func (h *Handler) RunHandler(c *gin.Context) {
	// Gate: OFF by default in prod. Must be explicitly enabled.
	if !config.GetFlags().EnableRedTeam {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "redteam_disabled",
			"hint":  "set ENABLE_REDTEAM=true to activate the internal attack simulators",
		})
		return
	}

	var body struct {
		Scenario string `json:"scenario" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	var result ScenarioResult
	switch body.Scenario {
	case "spam":
		result = h.sim.SimulateSpamAttack(ctx)
	case "referral":
		result = h.sim.SimulateReferralAbuse(ctx)
	case "exchange":
		result = h.sim.SimulateCircularTrade(ctx)
	case "bids":
		result = h.sim.SimulateBidFlood(ctx)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "unknown_scenario",
			"allowed": []string{"spam", "referral", "exchange", "bids"},
		})
		return
	}

	// Persist audit trail + broadcast admin_action to security log.
	h.sim.db.Create(&RedTeamRun{
		ID:           uuid.New(),
		Scenario:     result.Scenario,
		StartedAt:    result.StartedAt,
		DurationMs:   result.DurationMs,
		Attempts:     result.Attempts,
		Blocked:      result.Blocked,
		FirstBlockAt: result.FirstBlockAt,
		Passed:       result.Passed,
		InitiatedBy:  currentUserID(c),
		Result:       resultToMap(result),
		CreatedAt:    time.Now().UTC(),
	})
	security.LogEventDirect(h.sim.db, currentUserID(c), security.EventAdminAction,
		c.ClientIP(), c.Request.UserAgent(),
		map[string]any{
			"action":   "redteam_run",
			"scenario": result.Scenario,
			"passed":   result.Passed,
			"attempts": result.Attempts,
			"blocked":  result.Blocked,
		},
	)

	c.JSON(http.StatusOK, result)
}

// ListRunsHandler — GET /admin/redteam/runs
// Returns the last 50 simulation runs for a trend view.
func (h *Handler) ListRunsHandler(c *gin.Context) {
	var runs []RedTeamRun
	h.sim.db.Order("started_at DESC").Limit(50).Find(&runs)
	c.JSON(http.StatusOK, gin.H{"runs": runs, "count": len(runs)})
}

// currentUserID extracts the admin user id from the gin context, or nil.
func currentUserID(c *gin.Context) *uuid.UUID {
	s := c.GetString("user_id")
	if s == "" {
		return nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &u
}

func resultToMap(r ScenarioResult) map[string]any {
	return map[string]any{
		"scenario":          r.Scenario,
		"system_responded":  r.SystemResponded,
		"defense_triggered": r.DefenseTriggered,
		"triggered_alerts":  r.TriggeredAlerts,
		"notes":             r.Notes,
		"metrics":           r.Metrics,
	}
}
