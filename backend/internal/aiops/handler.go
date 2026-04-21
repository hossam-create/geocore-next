package aiops

import (
	"fmt"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// Handler exposes the AIOps API.
type Handler struct {
	engine *Engine
}

func NewHandler(e *Engine) *Handler { return &Handler{engine: e} }

// GET /api/v1/aiops/health
func (h *Handler) Health(c *gin.Context) {
	response.OK(c, h.engine.Status())
}

// GET /api/v1/aiops/incidents
func (h *Handler) ListIncidents(c *gin.Context) {
	all := registry.List()

	// Optional filters
	if status := c.Query("status"); status != "" {
		filtered := all[:0]
		for _, inc := range all {
			if string(inc.Status) == status {
				filtered = append(filtered, inc)
			}
		}
		all = filtered
	}
	if sev := c.Query("severity"); sev != "" {
		filtered := all[:0]
		for _, inc := range all {
			if string(inc.Severity) == sev {
				filtered = append(filtered, inc)
			}
		}
		all = filtered
	}

	response.OK(c, gin.H{
		"incidents": all,
		"total":     len(all),
		"open":      registry.OpenCount(),
	})
}

// GET /api/v1/aiops/incidents/:id
func (h *Handler) GetIncident(c *gin.Context) {
	inc := registry.Get(c.Param("id"))
	if inc == nil {
		response.NotFound(c, "incident")
		return
	}
	response.OK(c, inc)
}

// POST /api/v1/aiops/incidents/:id/resolve
func (h *Handler) ResolveIncident(c *gin.Context) {
	if !registry.UpdateStatus(c.Param("id"), StatusResolved) {
		response.NotFound(c, "incident")
		return
	}
	response.OK(c, gin.H{"message": "incident resolved"})
}

// POST /api/v1/aiops/incidents/:id/ignore
func (h *Handler) IgnoreIncident(c *gin.Context) {
	if !registry.UpdateStatus(c.Param("id"), StatusIgnored) {
		response.NotFound(c, "incident")
		return
	}
	response.OK(c, gin.H{"message": "incident ignored"})
}

// POST /api/v1/aiops/analyze
// On-demand incident analysis — bypasses the 30s scan cycle.
func (h *Handler) Analyze(c *gin.Context) {
	var body struct {
		Service  string  `json:"service" binding:"required"`
		Metric   string  `json:"metric" binding:"required"`
		Value    float64 `json:"value" binding:"required"`
		Baseline float64 `json:"baseline"`
		Severity string  `json:"severity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	sev := Severity(body.Severity)
	if sev == "" {
		sev = SeverityP1
	}

	inc := &Incident{
		ID:          newIncidentID(),
		Severity:    sev,
		Service:     body.Service,
		Metric:      body.Metric,
		Value:       body.Value,
		Baseline:    body.Baseline,
		Title:       fmt.Sprintf("%s anomaly on %s", body.Metric, body.Service),
		Description: fmt.Sprintf("Metric %s = %.4f (baseline: %.4f)", body.Metric, body.Value, body.Baseline),
		Status:      StatusOpen,
		DetectedAt:  time.Now(),
	}

	ctx := c.Request.Context()
	sysCtx := h.engine.builder.Build(ctx, inc)
	inc.RCA = h.engine.rca.Analyze(ctx, sysCtx)
	inc.Runbook = h.engine.runbook.Generate(ctx, inc, inc.RCA)

	registry.Add(inc)

	_ = h.engine.notifier.Send(ctx, inc)
	_ = h.engine.notifier.SendPagerDuty(ctx, inc)

	response.OK(c, inc)
}
