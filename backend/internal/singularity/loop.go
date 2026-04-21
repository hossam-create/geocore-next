package singularity

import (
	"context"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/autonomy"
	"github.com/geocore-next/backend/internal/singularity/controlplane"
	"github.com/geocore-next/backend/internal/singularity/executor"
	"github.com/geocore-next/backend/internal/singularity/intelligence"
	"github.com/geocore-next/backend/internal/singularity/proposals"
	"github.com/geocore-next/backend/internal/singularity/safety"
	"github.com/geocore-next/backend/internal/singularity/simulation"
)

// Singularity is the self-optimizing control plane.
// It observes, analyzes, plans, simulates, validates, and executes changes safely.
type Singularity struct {
	analyzer    *controlplane.Analyzer
	planner     *controlplane.Planner
	evaluator   *controlplane.Evaluator
	shadow      *simulation.ShadowRunner
	rollout     *executor.RolloutEngine
	autoscaler  *executor.Autoscaler
	diffEngine  *proposals.DiffEngine

	// Intelligence models
	workloadModel *intelligence.WorkloadModel
	latencyModel  *intelligence.LatencyModel
	costModel     *intelligence.CostModel

	// Safety gates
	sloGuard    *safety.SLOGuard
	budgetGuard *safety.BudgetGuard
	riskEngine  *safety.RiskEngine

	// State
	interval time.Duration
	running  bool
	history  []proposals.ChangeProposal
}

// Config holds singularity configuration.
type Config struct {
	Interval       time.Duration
	Namespace      string
	BudgetCap      float64
	MaxRiskScore   float64
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Interval:     2 * time.Minute,
		Namespace:    "default",
		BudgetCap:    5000.0,
		MaxRiskScore: 0.6,
	}
}

// New creates the singularity control plane.
func New(cfg Config) *Singularity {
	costModel := intelligence.NewCostModel()
	costModel.MonthlyBudgetCap = cfg.BudgetCap
	latencyModel := intelligence.NewLatencyModel()
	workloadModel := intelligence.NewWorkloadModel()

	sloGuard := safety.NewSLOGuard()
	budgetGuard := safety.NewBudgetGuard(costModel)
	riskEngine := safety.NewRiskEngine()

	return &Singularity{
		analyzer:      controlplane.NewAnalyzer(),
		planner:       controlplane.NewPlanner(),
		evaluator:     controlplane.NewEvaluator(sloGuard, budgetGuard, riskEngine),
		shadow:        simulation.NewShadowRunner(latencyModel, costModel),
		rollout:       executor.NewRolloutEngine(),
		autoscaler:    executor.NewAutoscaler(cfg.Namespace),
		diffEngine:    proposals.NewDiffEngine(),
		workloadModel: workloadModel,
		latencyModel:  latencyModel,
		costModel:     costModel,
		sloGuard:      sloGuard,
		budgetGuard:   budgetGuard,
		riskEngine:    riskEngine,
		interval:      cfg.Interval,
	}
}

// Start begins the self-optimization loop.
func (s *Singularity) Start(ctx context.Context) {
	if s.running {
		return
	}
	s.running = true
	slog.Info("singularity: control plane started", "interval", s.interval)

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("singularity: control plane stopped")
				s.running = false
				return
			case <-ticker.C:
				s.tick(ctx)
			}
		}
	}()
}

func (s *Singularity) tick(ctx context.Context) {
	// 1. Observe — collect telemetry
	metrics := autonomy.CollectMetrics()

	// 2. Analyze — detect inefficiencies
	inefficiencies := s.analyzer.Analyze(ctx, metrics)
	if len(inefficiencies) == 0 {
		return // system is healthy, no changes needed
	}

	// 3. Plan — generate proposals
	proposals_ := s.planner.Plan(ctx, inefficiencies)
	if len(proposals_) == 0 {
		return
	}

	// 4. Simulate — shadow test each proposal
	for i := range proposals_ {
		p := &proposals_[i]
		if !s.shadow.Simulate(ctx, p) {
			p.Reject("simulation_failed")
			slog.Warn("singularity: proposal rejected by simulation", "type", p.Type)
			continue
		}

		// 5. Evaluate — run through safety gates
		if !s.evaluator.Evaluate(ctx, p) {
			slog.Warn("singularity: proposal rejected by safety gates", "type", p.Type)
			continue
		}

		// 6. Execute — gradual canary rollout
		if err := s.rollout.Execute(ctx, p); err != nil {
			slog.Error("singularity: rollout failed", "type", p.Type, "error", err)
			continue
		}

		// 7. Record — learn from outcome
		s.history = append(s.history, *p)
		slog.Info("singularity: optimization applied",
			"type", p.Type,
			"target", p.Target,
			"status", p.Status,
		)
	}

	// 8. Update — adjust risk engine based on current state
	s.riskEngine.UpdateStressFactor(metrics.ErrorRate, metrics.P95Latency)
}

// IsRunning returns whether the singularity loop is active.
func (s *Singularity) IsRunning() bool {
	return s.running
}

// History returns all applied proposals.
func (s *Singularity) History() []proposals.ChangeProposal {
	return s.history
}

// Status returns the current singularity status.
func (s *Singularity) Status() map[string]any {
	return map[string]any{
		"running":           s.running,
		"interval":          s.interval.String(),
		"proposals_applied": len(s.history),
		"budget_utilization": s.costModel.BudgetUtilization(),
		"risk_stress":       s.riskEngine,
	}
}
