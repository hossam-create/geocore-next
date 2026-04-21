package stress

import (
	"net/http"
	"sort"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

// Handler exposes the stress testing API.
type Handler struct {
	o *Orchestrator
}

func newHandler(o *Orchestrator) *Handler { return &Handler{o: o} }

// GET /api/v1/stress/scenarios
func (h *Handler) ListScenarios(c *gin.Context) {
	out := make([]Scenario, 0, len(BuiltinScenarios))
	for _, s := range BuiltinScenarios {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	response.OK(c, out)
}

// POST /api/v1/stress/run/:scenario_id
func (h *Handler) RunScenario(c *gin.Context) {
	id := c.Param("scenario_id")
	scenario, ok := BuiltinScenarios[id]
	if !ok {
		response.NotFound(c, "scenario")
		return
	}
	if err := h.o.StartAsync(scenario); err != nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":  err.Error(),
			"status": h.o.Status(),
		})
		return
	}
	response.OK(c, gin.H{
		"message":  "stress test started",
		"scenario": scenario.Name,
		"status":   h.o.Status(),
	})
}

// GET /api/v1/stress/status
func (h *Handler) Status(c *gin.Context) {
	response.OK(c, h.o.Status())
}

// GET /api/v1/stress/report
func (h *Handler) Report(c *gin.Context) {
	r := h.o.LastReport()
	if r == nil {
		response.NotFound(c, "report")
		return
	}
	response.OK(c, r)
}

// POST /api/v1/stress/stop
func (h *Handler) Stop(c *gin.Context) {
	if !h.o.Stop() {
		c.JSON(http.StatusConflict, gin.H{"error": "no stress test is currently running"})
		return
	}
	response.OK(c, gin.H{"message": "stress test stopped"})
}
