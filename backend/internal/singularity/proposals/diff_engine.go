package proposals

import (
	"log/slog"
)

// Diff represents the difference between current and desired state.
type Diff struct {
	Target      string `json:"target"`
	Field       string `json:"field"`
	OldValue    string `json:"old_value"`
	NewValue    string `json:"new_value"`
	ImpactLevel string `json:"impact_level"` // low, medium, high
}

// DiffEngine computes the differences between current and proposed states.
type DiffEngine struct{}

// NewDiffEngine creates a new diff engine.
func NewDiffEngine() *DiffEngine {
	return &DiffEngine{}
}

// Compute generates a diff for a change proposal.
func (d *DiffEngine) Compute(proposal ChangeProposal) []Diff {
	var diffs []Diff

	impactLevel := "low"
	if proposal.RiskScore > 0.5 {
		impactLevel = "high"
	} else if proposal.RiskScore > 0.2 {
		impactLevel = "medium"
	}

	diff := Diff{
		Target:      proposal.Target,
		Field:       string(proposal.Type),
		OldValue:    proposal.CurrentState,
		NewValue:    proposal.DesiredState,
		ImpactLevel: impactLevel,
	}
	diffs = append(diffs, diff)

	slog.Debug("singularity: diff computed",
		"target", proposal.Target,
		"field", proposal.Type,
		"old", proposal.CurrentState,
		"new", proposal.DesiredState,
		"impact", impactLevel,
	)

	return diffs
}

// Summarize returns a human-readable summary of all diffs.
func (d *DiffEngine) Summarize(diffs []Diff) string {
	if len(diffs) == 0 {
		return "no changes"
	}
	summary := ""
	for _, diff := range diffs {
		summary += diff.Target + "." + diff.Field + ": " + diff.OldValue + " → " + diff.NewValue + " (" + diff.ImpactLevel + ")\n"
	}
	return summary
}
