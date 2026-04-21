package planner

import (
	"github.com/geocore-next/backend/internal/controlplane/analyzer"
)

// ProposalType defines the category of optimization proposal.
type ProposalType string

const (
	ScaleUp        ProposalType = "SCALE_UP"
	ScaleDown      ProposalType = "SCALE_DOWN"
	Rollback       ProposalType = "ROLLBACK"
	KafkaRebalance ProposalType = "KAFKA_REBALANCE"
	Throttle       ProposalType = "THROTTLE"
	CachePromote   ProposalType = "CACHE_PROMOTE"
)

// Proposal represents a proposed optimization action.
type Proposal struct {
	Type         ProposalType `json:"type"`
	Target       string       `json:"target"`
	Action       string       `json:"action"`
	RiskScore    float64      `json:"risk_score"`    // 0-1
	ExpectedGain float64      `json:"expected_gain"` // % improvement
	Reason       string       `json:"reason"`
}

// Planner generates optimization proposals from metrics and anomalies.
type Planner struct{}

// NewPlanner creates a new proposal planner.
func NewPlanner() *Planner {
	return &Planner{}
}

// Generate creates proposals based on current metrics and detected anomalies.
func (p *Planner) Generate(m analyzer.Metrics, anomalies []analyzer.Anomaly) []Proposal {
	var proposals []Proposal

	for _, a := range anomalies {
		switch a.Type {
		case "high_latency":
			proposals = append(proposals, Proposal{
				Type:         ScaleUp,
				Target:       a.Target,
				Action:       "increase replicas by 2",
				RiskScore:    0.1,
				ExpectedGain: a.Severity * 40,
				Reason:       a.Description,
			})
		case "high_error":
			proposals = append(proposals, Proposal{
				Type:         Rollback,
				Target:       a.Target,
				Action:       "rollback to previous deployment",
				RiskScore:    0.05,
				ExpectedGain: a.Severity * 80,
				Reason:       a.Description,
			})
		case "overprovisioned":
			proposals = append(proposals, Proposal{
				Type:         ScaleDown,
				Target:       a.Target,
				Action:       "reduce replicas by 1",
				RiskScore:    0.2,
				ExpectedGain: a.Severity * 20,
				Reason:       a.Description,
			})
		case "kafka_lag":
			proposals = append(proposals, Proposal{
				Type:         KafkaRebalance,
				Target:       a.Target,
				Action:       "increase consumer group by 2",
				RiskScore:    0.3,
				ExpectedGain: a.Severity * 60,
				Reason:       a.Description,
			})
		case "cost_spike":
			proposals = append(proposals, Proposal{
				Type:         ScaleDown,
				Target:       a.Target,
				Action:       "optimize instance allocation",
				RiskScore:    0.25,
				ExpectedGain: a.Severity * 30,
				Reason:       a.Description,
			})
		}
	}

	return proposals
}
