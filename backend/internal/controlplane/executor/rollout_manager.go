package executor

import (
	"log/slog"

	"github.com/geocore-next/backend/internal/controlplane/planner"
)

// RolloutManager handles gradual canary rollouts for proposals.
type RolloutManager struct {
	currentStage int
	stages       []RolloutStage
}

// RolloutStage is a single stage in a gradual rollout.
type RolloutStage struct {
	TrafficPercent int     `json:"traffic_percent"`
	ErrorThreshold float64 `json:"error_threshold"`
	Passed         bool    `json:"passed"`
}

// NewRolloutManager creates a rollout manager with standard canary stages.
func NewRolloutManager() *RolloutManager {
	return &RolloutManager{
		stages: []RolloutStage{
			{TrafficPercent: 1, ErrorThreshold: 2.0},
			{TrafficPercent: 10, ErrorThreshold: 1.5},
			{TrafficPercent: 50, ErrorThreshold: 1.0},
			{TrafficPercent: 100, ErrorThreshold: 0.5},
		},
	}
}

// Advance moves to the next rollout stage if the error rate is acceptable.
func (rm *RolloutManager) Advance(p planner.Proposal, errorRate float64) bool {
	if rm.currentStage >= len(rm.stages) {
		return true // already complete
	}

	stage := &rm.stages[rm.currentStage]
	if errorRate <= stage.ErrorThreshold {
		stage.Passed = true
		rm.currentStage++
		slog.Info("controlplane: rollout stage passed",
			"proposal", p.Type, "traffic_pct", stage.TrafficPercent)
		return true
	}

	stage.Passed = false
	slog.Warn("controlplane: rollout stage FAILED",
		"proposal", p.Type, "traffic_pct", stage.TrafficPercent,
		"error_rate", errorRate, "threshold", stage.ErrorThreshold)
	return false
}

// IsComplete returns true if all rollout stages passed.
func (rm *RolloutManager) IsComplete() bool {
	return rm.currentStage >= len(rm.stages)
}

// Reset prepares the rollout manager for a new proposal.
func (rm *RolloutManager) Reset() {
	rm.currentStage = 0
	for i := range rm.stages {
		rm.stages[i].Passed = false
	}
}
