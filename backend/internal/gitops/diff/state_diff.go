package diff

import (
	"log/slog"

	"github.com/geocore-next/backend/internal/gitops/watcher"
)

// StateDiff represents the difference between current and desired state.
type StateDiff struct {
	Service    string  `json:"service"`
	From       string  `json:"from_version"`
	To         string  `json:"to_version"`
	Env        string  `json:"env"`
	Risk       float64 `json:"risk"` // 0-1
	Drift      bool    `json:"drift"` // true if live state != desired state
	Manifests  []string `json:"manifests"` // changed manifest paths
}

// DiffEngine computes the diff between desired (Git) and live (K8s) state.
type DiffEngine struct {
	liveState map[string]string // service → current version
}

// NewDiffEngine creates a diff engine.
func NewDiffEngine() *DiffEngine {
	return &DiffEngine{
		liveState: make(map[string]string),
	}
}

// Compute generates a diff for a detected change.
func (d *DiffEngine) Compute(change watcher.Change) StateDiff {
	currentVersion := d.liveState[change.Service]
	if currentVersion == "" {
		currentVersion = "unknown"
	}

	risk := d.assessRisk(currentVersion, change.Version)
	drift := currentVersion != change.Version

	diff := StateDiff{
		Service:   change.Service,
		From:      currentVersion,
		To:        change.Version,
		Env:       change.Env,
		Risk:      risk,
		Drift:     drift,
	}

	slog.Info("gitops: diff computed",
		"service", diff.Service,
		"from", diff.From,
		"to", diff.To,
		"risk", diff.Risk,
		"drift", diff.Drift,
	)

	return diff
}

// UpdateLiveState records the current live state after a successful deploy.
func (d *DiffEngine) UpdateLiveState(service, version string) {
	d.liveState[service] = version
}

// HasDrift returns true if any service has drifted from desired state.
func (d *DiffEngine) HasDrift(diffs []StateDiff) bool {
	for _, d := range diffs {
		if d.Drift {
			return true
		}
	}
	return false
}

func (d *DiffEngine) assessRisk(from, to string) float64 {
	// Same version = no risk
	if from == to {
		return 0
	}
	// Major version jump = high risk
	// Minor = moderate, patch = low
	// Simplified: any change starts at 0.2
	risk := 0.2
	if from == "unknown" {
		risk = 0.4 // first deploy is riskier
	}
	return risk
}
