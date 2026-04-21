package controlplane

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/internal/singularity/proposals"
)

// Planner converts detected inefficiencies into optimization proposals.
type Planner struct{}

// NewPlanner creates a new optimization planner.
func NewPlanner() *Planner {
	return &Planner{}
}

// Plan generates change proposals from detected inefficiencies.
func (p *Planner) Plan(ctx context.Context, inefficiencies []Inefficiency) []proposals.ChangeProposal {
	var proposals_ []proposals.ChangeProposal

	for _, in := range inefficiencies {
		proposal := p.planForInefficiency(in)
		if proposal != nil {
			proposals_ = append(proposals_, *proposal)
		}
	}

	if len(proposals_) > 0 {
		slog.Info("singularity: proposals generated", "count", len(proposals_))
	}

	return proposals_
}

func (p *Planner) planForInefficiency(in Inefficiency) *proposals.ChangeProposal {
	switch in.Type {
	case "latency":
		return &proposals.ChangeProposal{
			Type:         proposals.ProposalScaleUp,
			Target:       in.Target,
			Action:       "scale_up",
			CurrentState: "current_replicas",
			DesiredState: "current_replicas+2",
			ExpectedGain: in.Severity * 40, // up to 40% latency improvement
			RiskScore:    0.1,               // low risk
			RollbackPlan: "scale back to original replica count",
			Reason:       in.Description,
		}

	case "overprovisioned":
		return &proposals.ChangeProposal{
			Type:         proposals.ProposalScaleDown,
			Target:       in.Target,
			Action:       "scale_down",
			CurrentState: "current_replicas",
			DesiredState: "current_replicas-1",
			ExpectedGain: in.Severity * 20, // up to 20% cost savings
			RiskScore:    0.2,               // moderate risk
			RollbackPlan: "scale back to original replica count",
			Reason:       in.Description,
		}

	case "kafka_lag":
		return &proposals.ChangeProposal{
			Type:         proposals.ProposalKafkaRebalance,
			Target:       in.Target,
			Action:       "increase_consumer_group",
			CurrentState: "current_consumers",
			DesiredState: "current_consumers+2",
			ExpectedGain: in.Severity * 60, // up to 60% lag reduction
			RiskScore:    0.3,               // moderate risk (rebalance)
			RollbackPlan: "reduce consumer group to original size",
			Reason:       in.Description,
		}

	case "error_rate":
		return &proposals.ChangeProposal{
			Type:         proposals.ProposalRollback,
			Target:       in.Target,
			Action:       "rollback",
			CurrentState: "current_deployment",
			DesiredState: "previous_deployment",
			ExpectedGain: in.Severity * 80, // up to 80% error reduction
			RiskScore:    0.05,              // low risk (rollback is safe)
			RollbackPlan: "re-deploy current version",
			Reason:       in.Description,
		}

	default:
		return nil
	}
}
