package warroom

import "time"

// SystemState represents the war room's current view of system health.
type SystemState string

const (
	StateNormal         SystemState = "NORMAL"
	StateDegraded       SystemState = "DEGRADED"
	StateStressTesting  SystemState = "STRESS_TESTING"
	StateIncidentActive SystemState = "INCIDENT_ACTIVE"
	StateRecovery       SystemState = "RECOVERY_MODE"
	StateLockdown       SystemState = "LOCKDOWN"
)

// StateTransition records a single state change event.
type StateTransition struct {
	From   SystemState `json:"from"`
	To     SystemState `json:"to"`
	Reason string      `json:"reason"`
	At     time.Time   `json:"at"`
}

// severity returns a numeric weight for comparison (higher = more severe).
func (s SystemState) severity() int {
	switch s {
	case StateNormal:
		return 0
	case StateRecovery:
		return 1
	case StateDegraded:
		return 2
	case StateStressTesting:
		return 2
	case StateIncidentActive:
		return 3
	case StateLockdown:
		return 4
	default:
		return 0
	}
}
