package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
)

// Client provides a simplified Kubernetes API client.
// In production, replace with controller-runtime client.
type Client struct {
	Namespace string
}

// NewClient creates a K8s client for the given namespace.
func NewClient(namespace string) *Client {
	return &Client{Namespace: namespace}
}

// ScaleDeployment sets the replica count for a deployment.
func (c *Client) ScaleDeployment(ctx context.Context, deployment string, replicas int) error {
	slog.Info("k8s: scaling deployment", "deployment", deployment, "replicas", replicas, "ns", c.Namespace)
	cmd := exec.CommandContext(ctx, "kubectl",
		"scale", "deployment", deployment,
		"--replicas", fmt.Sprintf("%d", replicas),
		"-n", c.Namespace,
	)
	return cmd.Run()
}

// RolloutUndo reverts a deployment to its previous revision.
func (c *Client) RolloutUndo(ctx context.Context, deployment string) error {
	slog.Error("k8s: rolling back deployment", "deployment", deployment, "ns", c.Namespace)
	cmd := exec.CommandContext(ctx, "kubectl",
		"rollout", "undo", "deployment/"+deployment,
		"-n", c.Namespace,
	)
	return cmd.Run()
}

// GetDeploymentReplicas queries the current replica count.
func (c *Client) GetDeploymentReplicas(ctx context.Context, deployment string) (int, error) {
	cmd := exec.CommandContext(ctx, "kubectl",
		"get", "deployment", deployment,
		"-o", "jsonpath={.spec.replicas}",
		"-n", c.Namespace,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	var replicas int
	fmt.Sscanf(string(output), "%d", &replicas)
	return replicas, nil
}

// ApplyCRD applies a Custom Resource Definition to the cluster.
func (c *Client) ApplyCRD(ctx context.Context, crd any) error {
	data, err := json.Marshal(crd)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	go func() {
		defer stdin.Close()
		stdin.Write(data)
	}()

	return cmd.Run()
}

// PatchStatus patches the status subresource of a CRD.
func (c *Client) PatchStatus(ctx context.Context, name string, status any) error {
	slog.Debug("k8s: patching status", "name", name)
	// In production: use controller-runtime status.Patch
	return nil
}
