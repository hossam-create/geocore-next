package planner

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/gitops/diff"
	"github.com/geocore-next/backend/internal/gitops/watcher"
)

// RolloutStrategy defines how a change should be rolled out.
type RolloutStrategy string

const (
	StrategyRolling  RolloutStrategy = "rolling"  // standard rolling update
	StrategyCanary   RolloutStrategy = "canary"   // canary with progressive traffic
	StrategyBlueGreen RolloutStrategy = "blue_green" // blue/green deployment
)

// RolloutPlan is the execution plan for a deployment.
type RolloutPlan struct {
	Service    string          `json:"service"`
	From       string          `json:"from_version"`
	To         string          `json:"to_version"`
	Env        string          `json:"env"`
	Strategy   RolloutStrategy `json:"strategy"`
	Stages     []RolloutStage  `json:"stages"`
	Risk       float64         `json:"risk"`
	CreatedAt  time.Time       `json:"created_at"`
}

// RolloutStage is a single stage in a progressive rollout.
type RolloutStage struct {
	TrafficPercent int           `json:"traffic_percent"`
	Duration       time.Duration `json:"duration"`
	Verify         string        `json:"verify"` // verification condition
}

// RolloutPlanner creates rollout plans from diffs.
type RolloutPlanner struct{}

// NewRolloutPlanner creates a rollout planner.
func NewRolloutPlanner() *RolloutPlanner {
	return &RolloutPlanner{}
}

// BuildPlan creates a rollout plan from a detected change.
func (p *RolloutPlanner) BuildPlan(change watcher.Change, d diff.StateDiff) RolloutPlan {
	strategy := p.selectStrategy(d)

	plan := RolloutPlan{
		Service:   change.Service,
		From:      d.From,
		To:        d.To,
		Env:       change.Env,
		Strategy:  strategy,
		Risk:      d.Risk,
		CreatedAt: time.Now().UTC(),
	}

	plan.Stages = p.buildStages(strategy)

	slog.Info("gitops: rollout plan created",
		"service", plan.Service,
		"from", plan.From,
		"to", plan.To,
		"strategy", string(plan.Strategy),
		"stages", len(plan.Stages),
	)

	return plan
}

func (p *RolloutPlanner) selectStrategy(d diff.StateDiff) RolloutStrategy {
	switch {
	case d.Risk > 0.5:
		return StrategyCanary // high risk = canary
	case d.Risk > 0.3:
		return StrategyBlueGreen // moderate = blue/green
	default:
		return StrategyRolling // low risk = rolling
	}
}

func (p *RolloutPlanner) buildStages(strategy RolloutStrategy) []RolloutStage {
	switch strategy {
	case StrategyCanary:
		return []RolloutStage{
			{TrafficPercent: 5, Duration: 5 * time.Minute, Verify: "error_rate < 1%"},
			{TrafficPercent: 25, Duration: 10 * time.Minute, Verify: "error_rate < 1%"},
			{TrafficPercent: 50, Duration: 15 * time.Minute, Verify: "p95 < 300ms"},
			{TrafficPercent: 100, Duration: 0, Verify: "all healthy"},
		}

	case StrategyBlueGreen:
		return []RolloutStage{
			{TrafficPercent: 0, Duration: 5 * time.Minute, Verify: "new pods ready"},
			{TrafficPercent: 100, Duration: 0, Verify: "switch traffic"},
		}

	default: // rolling
		return []RolloutStage{
			{TrafficPercent: 100, Duration: 0, Verify: "pods ready"},
		}
	}
}
