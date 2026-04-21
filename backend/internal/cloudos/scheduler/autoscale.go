package scheduler

import (
	"log/slog"
)

// AutoscaleScheduler makes scaling decisions based on resource metrics.
type AutoscaleScheduler struct {
	MinReplicas int
	MaxReplicas int
	TargetCPU   float64 // target CPU utilization %
}

// NewAutoscaleScheduler creates an autoscale scheduler with defaults.
func NewAutoscaleScheduler() *AutoscaleScheduler {
	return &AutoscaleScheduler{
		MinReplicas: 2,
		MaxReplicas: 10,
		TargetCPU:   70,
	}
}

// ComputeDesiredReplicas calculates the optimal replica count.
func (a *AutoscaleScheduler) ComputeDesiredReplicas(currentReplicas int, cpuPct, rps float64) int {
	desired := currentReplicas

	// Scale up: CPU > target or high RPS per pod
	if cpuPct > a.TargetCPU {
		desired = currentReplicas + 2
		slog.Info("cloudos: autoscale UP", "from", currentReplicas, "to", desired, "cpu", cpuPct)
	} else if rps/float64(currentReplicas) > 50 {
		desired = currentReplicas + 1
		slog.Info("cloudos: autoscale UP (RPS)", "from", currentReplicas, "to", desired, "rps_per_pod", rps/float64(currentReplicas))
	}

	// Scale down: CPU < 30% and low RPS
	if cpuPct < 30 && rps < 50 && currentReplicas > a.MinReplicas {
		desired = currentReplicas - 1
		slog.Info("cloudos: autoscale DOWN", "from", currentReplicas, "to", desired, "cpu", cpuPct)
	}

	// Enforce bounds
	if desired < a.MinReplicas {
		desired = a.MinReplicas
	}
	if desired > a.MaxReplicas {
		desired = a.MaxReplicas
	}

	return desired
}
