package gitops

import (
	"log/slog"

	"github.com/geocore-next/backend/internal/cloudos/resources"
)

// Diff computes the difference between current and desired state.
func Diff(state resources.ClusterState, desired resources.DesiredState) []Proposal {
	var proposals []Proposal

	// Version drift
	if desired.APIVersion != "" && state.API.Version != desired.APIVersion {
		proposals = append(proposals, Proposal{
			Resource: "api",
			Action:   "update_version",
			From:     state.API.Version,
			To:       desired.APIVersion,
			Risk:     0.3,
		})
	}

	// Replica drift
	if state.API.Replicas < desired.MinReplicas {
		proposals = append(proposals, Proposal{
			Resource: "api",
			Action:   "scale_up",
			From:     intToStr(state.API.Replicas),
			To:       intToStr(desired.MinReplicas),
			Risk:     0.1,
		})
	}

	if len(proposals) > 0 {
		slog.Info("cloudos: drift detected", "count", len(proposals))
	}

	return proposals
}

// Proposal is a GitOps-level diff proposal.
type Proposal struct {
	Resource string  `json:"resource"`
	Action   string  `json:"action"`
	From     string  `json:"from"`
	To       string  `json:"to"`
	Risk     float64 `json:"risk"`
}

func intToStr(n int) string {
	if n <= 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
