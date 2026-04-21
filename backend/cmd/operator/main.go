package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geocore-next/backend/api/v1"
	"github.com/geocore-next/backend/internal/operator"
	"github.com/geocore-next/backend/pkg/k8s"
)

func main() {
	slog.Info("operator: starting Geocore Kubernetes Operator")

	// Load CR defaults (in production: watch from K8s API)
	cr := v1.DefaultControlPlane()

	// Override from environment
	if ns := os.Getenv("OPERATOR_NAMESPACE"); ns != "" {
		cr.Metadata.Namespace = ns
	}
	if v := os.Getenv("PRIMARY_REGION"); v != "" {
		cr.Spec.PrimaryRegion = v
	}
	if v := os.Getenv("FAILOVER_REGION"); v != "" {
		cr.Spec.FailoverRegion = v
	}

	// Create K8s client
	k8sClient := k8s.NewClient(cr.Metadata.Namespace)
	events := k8s.NewEventRecorder()

	// Create reconciler with all sub-reconcilers
	reconciler := operator.NewReconciler(k8sClient, events)

	// Create controller
	interval := 5 * time.Second
	if v := os.Getenv("RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	controller := operator.NewController(reconciler, k8sClient, cr, interval)

	// Apply CRD to cluster
	if err := k8sClient.ApplyCRD(context.Background(), cr); err != nil {
		slog.Warn("operator: could not apply CRD (K8s unavailable?)", "error", err)
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("operator: received signal, shutting down", "signal", sig)
		cancel()
	}()

	controller.Run(ctx)
	slog.Info("operator: shutdown complete")
}
