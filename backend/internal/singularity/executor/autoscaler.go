package executor

import (
	"log/slog"
	"os/exec"
)

// Autoscaler adjusts Kubernetes deployment replica counts.
type Autoscaler struct {
	namespace string
}

// NewAutoscaler creates an autoscaler for the given namespace.
func NewAutoscaler(namespace string) *Autoscaler {
	return &Autoscaler{namespace: namespace}
}

// ScaleDeployment sets the replica count for a deployment.
func (a *Autoscaler) ScaleDeployment(deployment string, replicas int) error {
	slog.Info("singularity: scaling deployment",
		"deployment", deployment,
		"replicas", replicas,
		"namespace", a.namespace,
	)

	// Production: kubectl scale deployment <name> --replicas=N -n <namespace>
	cmd := exec.Command("kubectl",
		"scale", "deployment", deployment,
		"--replicas", intToStr(replicas),
		"-n", a.namespace,
	)
	return cmd.Run()
}

// GetCurrentReplicas queries the current replica count.
func (a *Autoscaler) GetCurrentReplicas(deployment string) int {
	// Production: kubectl get deployment <name> -o jsonpath='{.spec.replicas}'
	// For now, return default
	return 4
}

// RolloutUndo reverts a deployment to its previous version.
func (a *Autoscaler) RolloutUndo(deployment string) error {
	slog.Error("singularity: rolling back deployment", "deployment", deployment)

	cmd := exec.Command("kubectl",
		"rollout", "undo", "deployment/"+deployment,
		"-n", a.namespace,
	)
	return cmd.Run()
}

// RolloutStatus checks the current rollout status.
func (a *Autoscaler) RolloutStatus(deployment string) string {
	cmd := exec.Command("kubectl",
		"rollout", "status", "deployment/"+deployment,
		"-n", a.namespace,
	)
	output, _ := cmd.CombinedOutput()
	return string(output)
}

func intToStr(n int) string {
	if n <= 0 {
		return "1"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
