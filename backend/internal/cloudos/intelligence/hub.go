package intelligence

import (
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/cloudos/resources"
)

// Hub is the intelligence hub that combines all AI subsystems.
type Hub struct {
	Cost    *CostAI
	Fraud   *FraudAI
	Traffic *TrafficAI
}

// NewHub creates an intelligence hub with all AI subsystems.
func NewHub() *Hub {
	return &Hub{
		Cost:    NewCostAI(),
		Fraud:   NewFraudAI(),
		Traffic: NewTrafficAI(),
	}
}

// Analyze runs all AI subsystems against current state and diff,
// then returns enhanced proposals.
func (h *Hub) Analyze(state resources.ClusterState, diff []resources.Proposal) []resources.Proposal {
	var enhanced []resources.Proposal

	// Run cost AI
	costSuggestions := h.Cost.Analyze(state.API.CPUUtilization, state.API.Replicas)
	for _, s := range costSuggestions {
		enhanced = append(enhanced, resources.Proposal{
			Resource:     "infrastructure",
			Action:       s.Action,
			Target:       "cost-optimization",
			RiskScore:    0.2,
			ExpectedGain: s.SavingsPct,
			Reason:       s.Reason,
		})
	}

	// Run fraud AI
	fraudSuggestions := h.Fraud.Analyze(state.Fraud.FraudRate, state.Fraud.FalsePositiveRate)
	for _, s := range fraudSuggestions {
		enhanced = append(enhanced, resources.Proposal{
			Resource:     "fraud",
			Action:       s.Action,
			Target:       "fraud-engine",
			RiskScore:    0.15,
			ExpectedGain: 40,
			Reason:       s.Reason,
		})
	}

	// Run traffic AI
	now := time.Now().UTC()
	trafficSuggestions := h.Traffic.Analyze(state.API.RPS, state.API.CPUUtilization, now.Hour(), int(now.Weekday()))
	for _, s := range trafficSuggestions {
		enhanced = append(enhanced, resources.Proposal{
			Resource:     "api",
			Action:       s.Action,
			Target:       s.Target,
			RiskScore:    0.1,
			ExpectedGain: 30,
			Reason:       s.Reason,
		})
	}

	// Include original diff proposals
	enhanced = append(enhanced, diff...)

	if len(enhanced) > 0 {
		slog.Info("cloudos: intelligence hub proposals", "count", len(enhanced))
	}

	return enhanced
}
