package controlplane

import (
	"log/slog"
	"sync/atomic"
)

var globalKillSwitch atomic.Bool

// EnableKillSwitch stops all incoming API traffic.
func EnableKillSwitch() {
	globalKillSwitch.Store(true)
	slog.Error("controlplane: KILL SWITCH ENABLED — all traffic blocked")
}

// DisableKillSwitch restores normal traffic flow.
func DisableKillSwitch() {
	globalKillSwitch.Store(false)
	slog.Info("controlplane: kill switch disabled — traffic restored")
}

// AllowRequest returns false when kill switch is active.
func AllowRequest() bool {
	return !globalKillSwitch.Load()
}

// IsKillSwitchActive returns the current kill switch state.
func IsKillSwitchActive() bool {
	return globalKillSwitch.Load()
}
