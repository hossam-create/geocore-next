package reslab

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync/atomic"
	"time"

	"github.com/geocore-next/backend/internal/stress"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Lab is the autonomous resilience experiment engine.
// It runs stress experiments, scores them, and generates improvement suggestions.
type Lab struct {
	running atomic.Bool
}

func newLab() *Lab { return &Lab{} }

// RunExperiment executes one experiment synchronously and returns the full result.
func (l *Lab) RunExperiment(ctx context.Context, id string) (*RunResult, error) {
	exp, ok := BuiltinExperiments[id]
	if !ok {
		return nil, fmt.Errorf("experiment %q not found", id)
	}
	if !l.running.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("an experiment is already running")
	}
	defer l.running.Store(false)

	slog.Info("reslab: experiment started", "id", id, "name", exp.Name)

	orch := stress.NewOrchestrator()
	report, err := orch.RunSync(ctx, exp.Scenario)
	if err != nil {
		return nil, fmt.Errorf("stress run failed: %w", err)
	}
	if report == nil {
		return nil, fmt.Errorf("no report produced")
	}

	expScore := Score(*report)
	suggestions := GenerateSuggestions(report.Metrics, report.Validation, expScore)

	baseline := globalHistory.Baseline(id)
	delta := 0.0
	if baseline != nil {
		delta = expScore.Overall - baseline.Score.Overall
	}

	result := &RunResult{
		ID:           uuid.New().String(),
		ExperimentID: id,
		RunAt:        time.Now(),
		Score:        expScore,
		Metrics:      report.Metrics,
		Validation:   report.Validation,
		Suggestions:  suggestions,
		Report:       *report,
	}
	globalHistory.Add(result)

	slog.Info("reslab: experiment complete",
		"id", id,
		"score", expScore.Overall,
		"grade", expScore.Grade,
		"delta", delta,
		"suggestions", len(suggestions),
	)
	return result, nil
}

func (l *Lab) IsRunning() bool { return l.running.Load() }

// ── HTTP handler ─────────────────────────────────────────────────────────────

type handler struct{ lab *Lab }

// GET /api/v1/reslab/experiments
func (h *handler) ListExperiments(c *gin.Context) {
	out := make([]Experiment, 0, len(BuiltinExperiments))
	for _, e := range BuiltinExperiments {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	response.OK(c, out)
}

// POST /api/v1/reslab/run/:experiment_id
func (h *handler) Run(c *gin.Context) {
	id := c.Param("experiment_id")
	if _, ok := BuiltinExperiments[id]; !ok {
		response.NotFound(c, "experiment")
		return
	}
	if h.lab.IsRunning() {
		c.JSON(409, gin.H{"error": "an experiment is already running"})
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()
		if _, err := h.lab.RunExperiment(ctx, id); err != nil {
			slog.Error("reslab: experiment failed", "id", id, "error", err)
		}
	}()
	response.OK(c, gin.H{"message": "experiment started", "experiment_id": id})
}

// GET /api/v1/reslab/status
func (h *handler) Status(c *gin.Context) {
	all := globalHistory.List()
	response.OK(c, gin.H{
		"running":    h.lab.IsRunning(),
		"total_runs": len(all),
	})
}

// GET /api/v1/reslab/runs
func (h *handler) ListRuns(c *gin.Context) {
	response.OK(c, gin.H{
		"runs":  globalHistory.List(),
		"total": len(globalHistory.List()),
	})
}

// GET /api/v1/reslab/runs/latest
func (h *handler) LatestRun(c *gin.Context) {
	all := globalHistory.List()
	if len(all) == 0 {
		response.NotFound(c, "run")
		return
	}
	response.OK(c, all[0])
}

// GET /api/v1/reslab/runs/trend/:experiment_id
func (h *handler) Trend(c *gin.Context) {
	id := c.Param("experiment_id")
	scores := globalHistory.Trend(id, 10)
	baseline := globalHistory.Baseline(id)
	response.OK(c, gin.H{
		"experiment_id": id,
		"scores":        scores,
		"baseline":      baseline,
	})
}

// RegisterRoutes mounts the Resilience Lab API under /api/v1/reslab (admin-only).
//
//	GET  /api/v1/reslab/experiments             — list available experiments
//	POST /api/v1/reslab/run/:experiment_id      — start an experiment
//	GET  /api/v1/reslab/status                  — running status
//	GET  /api/v1/reslab/runs                    — all historical runs
//	GET  /api/v1/reslab/runs/latest             — most recent run
//	GET  /api/v1/reslab/runs/trend/:id          — score trend (last 10)
func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB) {
	l := newLab()
	h := &handler{lab: l}

	g := v1.Group("/reslab",
		middleware.Auth(),
		middleware.AdminWithDB(db),
	)
	{
		g.GET("/experiments", h.ListExperiments)
		g.POST("/run/:experiment_id", h.Run)
		g.GET("/status", h.Status)
		g.GET("/runs", h.ListRuns)
		g.GET("/runs/latest", h.LatestRun)
		g.GET("/runs/trend/:experiment_id", h.Trend)
	}
}
