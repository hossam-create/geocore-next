package kubernetes

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
)

// Client provides Kubernetes operations for the GitOps executor.
type Client struct {
	Namespace string
}

// NewClient creates a Kubernetes client for the given namespace.
func NewClient(namespace string) *Client {
	return &Client{Namespace: namespace}
}

// SetImage updates a deployment's container image.
func (c *Client) SetImage(ctx context.Context, deployment, container, image string) error {
	slog.Info("k8s: updating deployment image",
		"deployment", deployment, "container", container, "image", image)
	cmd := exec.CommandContext(ctx, "kubectl",
		"set", "image", "deployment/"+deployment,
		fmt.Sprintf("%s=%s", container, image),
		"-n", c.Namespace,
	)
	return cmd.Run()
}

// RolloutUndo reverts a deployment to its previous revision.
func (c *Client) RolloutUndo(ctx context.Context, deployment string) error {
	slog.Error("k8s: rolling back deployment", "deployment", deployment)
	cmd := exec.CommandContext(ctx, "kubectl",
		"rollout", "undo", "deployment/"+deployment,
		"-n", c.Namespace,
	)
	return cmd.Run()
}

// RolloutStatus checks the rollout status of a deployment.
func (c *Client) RolloutStatus(ctx context.Context, deployment string) (string, error) {
	cmd := exec.CommandContext(ctx, "kubectl",
		"rollout", "status", "deployment/"+deployment,
		"-n", c.Namespace,
	)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Scale sets the replica count for a deployment.
func (c *Client) Scale(ctx context.Context, deployment string, replicas int) error {
	slog.Info("k8s: scaling deployment",
		"deployment", deployment, "replicas", replicas)
	cmd := exec.CommandContext(ctx, "kubectl",
		"scale", "deployment", deployment,
		"--replicas", fmt.Sprintf("%d", replicas),
		"-n", c.Namespace,
	)
	return cmd.Run()
}

// ApplyManifest applies a manifest from a file path.
func (c *Client) ApplyManifest(ctx context.Context, path string) error {
	slog.Info("k8s: applying manifest", "path", path)
	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", path, "-n", c.Namespace)
	return cmd.Run()
}
