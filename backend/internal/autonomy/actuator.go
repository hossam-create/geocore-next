package autonomy

import (
	"log/slog"
	"os/exec"

	"github.com/geocore-next/backend/pkg/remediation"
)

// Actuator executes decisions made by the decision engine.
// In production, these call kubectl / Kubernetes API.
// In development, they log the action for verification.

var currentReplicas = 4
var globalThrottle = false

// Execute carries out the decision action.
func Execute(d AutonomyDecision) {
	slog.Info("autonomy: executing decision", "action", d.Action, "reason", d.Reason)

	switch d.Action {
	case "scale_up":
		actScaleUp()
	case "scale_down":
		actScaleDown()
	case "rollback":
		actRollback(d.Reason)
	case "throttle":
		actThrottle()
	case "noop":
		// healthy — no action needed
	default:
		slog.Warn("autonomy: unknown action", "action", d.Action)
	}
}

func actScaleUp() {
	currentReplicas += 2
	if currentReplicas > 10 {
		currentReplicas = 10
	}
	slog.Info("autonomy: scaling up", "target_replicas", currentReplicas)

	// Production: kubectl scale deployment api --replicas=N
	_ = exec.Command("kubectl", "scale", "deployment", "api",
		"--replicas", intToStr(currentReplicas)).Run()
}

func actScaleDown() {
	currentReplicas -= 1
	if currentReplicas < 2 {
		currentReplicas = 2
	}
	slog.Info("autonomy: scaling down", "target_replicas", currentReplicas)

	_ = exec.Command("kubectl", "scale", "deployment", "api",
		"--replicas", intToStr(currentReplicas)).Run()
}

func actRollback(reason string) {
	slog.Error("autonomy: ROLLBACK triggered", "reason", reason)
	remediation.SignalRollback(reason)

	// Production: kubectl rollout undo deployment/api
	_ = exec.Command("kubectl", "rollout", "undo", "deployment/api").Run()
}

func actThrottle() {
	globalThrottle = true
	slog.Warn("autonomy: global throttle ENABLED (50%)")

	// In production, update rate limiter config dynamically
	// rateLimit.EnableGlobalThrottle(50)
}

// DisableThrottle removes the global throttle.
func DisableThrottle() {
	globalThrottle = true
	globalThrottle = false
	slog.Info("autonomy: global throttle DISABLED")
}

// IsThrottled returns whether global throttle is active.
func IsThrottled() bool {
	return globalThrottle
}

// CurrentReplicas returns the tracked replica count.
func CurrentReplicas() int {
	return currentReplicas
}

// SetCurrentReplicas sets the tracked replica count (for testing).
func SetCurrentReplicas(n int) {
	currentReplicas = n
}

func intToStr(n int) string {
	if n < 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if s == "" {
		return "0"
	}
	return s
}
