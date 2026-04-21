package autonomy

import (
	"log/slog"
	"time"
)

var (
	lastRollbackTime time.Time
	rollbackCooldown = 5 * time.Minute
	lastScaleTime    time.Time
	scaleCooldown    = 2 * time.Minute
)

// AllowAction checks if a decision action is safe to execute.
// Prevents over-reaction, infinite scaling, and rollback storms.
func AllowAction(d AutonomyDecision) bool {
	switch d.Action {
	case "rollback":
		// Always allow rollback, but with cooldown to prevent storms
		if time.Since(lastRollbackTime) < rollbackCooldown {
			slog.Warn("autonomy: rollback blocked by cooldown",
				"cooldown_remaining", rollbackCooldown-time.Since(lastRollbackTime),
			)
			return false
		}
		return true

	case "scale_up":
		// Don't scale beyond max replicas
		if CurrentReplicas() >= 10 {
			slog.Warn("autonomy: scale_up blocked — at max replicas", "replicas", CurrentReplicas())
			return false
		}
		// Cooldown to prevent rapid scaling
		if time.Since(lastScaleTime) < scaleCooldown {
			slog.Warn("autonomy: scale_up blocked by cooldown")
			return false
		}
		return true

	case "scale_down":
		// Don't scale below minimum
		if CurrentReplicas() <= 2 {
			slog.Warn("autonomy: scale_down blocked — at min replicas", "replicas", CurrentReplicas())
			return false
		}
		// Don't scale down if error rate is elevated
		if snap := CollectMetrics(); snap.ErrorRate > 1.0 {
			slog.Warn("autonomy: scale_down blocked — error rate elevated", "error_rate", snap.ErrorRate)
			return false
		}
		if time.Since(lastScaleTime) < scaleCooldown {
			return false
		}
		return true

	case "throttle":
		// Don't throttle if system is healthy
		if snap := CollectMetrics(); snap.ErrorRate < 1.0 && snap.P95Latency < 500 {
			slog.Warn("autonomy: throttle blocked — system healthy")
			return false
		}
		return true

	case "noop":
		return true

	default:
		return false
	}
}

// RecordAction updates cooldown timers after an action is executed.
func RecordAction(d AutonomyDecision) {
	switch d.Action {
	case "rollback":
		lastRollbackTime = time.Now()
	case "scale_up", "scale_down":
		lastScaleTime = time.Now()
	}
}

// SetRollbackCooldown configures the minimum time between rollbacks.
func SetRollbackCooldown(d time.Duration) {
	rollbackCooldown = d
}

// SetScaleCooldown configures the minimum time between scaling actions.
func SetScaleCooldown(d time.Duration) {
	scaleCooldown = d
}
