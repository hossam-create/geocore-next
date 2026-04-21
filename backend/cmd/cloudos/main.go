package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geocore-next/backend/internal/cloudos/controlplane"
	"github.com/geocore-next/backend/internal/cloudos/gitops"
	"github.com/geocore-next/backend/internal/cloudos/intelligence"
	"github.com/geocore-next/backend/internal/cloudos/policy"
	"github.com/geocore-next/backend/internal/cloudos/resources"
	"github.com/geocore-next/backend/internal/cloudos/runtime"
	"github.com/geocore-next/backend/pkg/telemetry"
)

func main() {
	slog.Info("cloudos: starting Cloud Operating System")

	// Initialize all subsystems
	resourceManager := resources.NewManager()
	intelHub := intelligence.NewHub()
	policyEngine := policy.NewEngine()
	gitWatcher := gitops.NewWatcher(
		envOr("GIT_REPO_URL", "https://github.com/geocore-next/infra.git"),
		envOr("GIT_BRANCH", "main"),
	)
	executor := runtime.NewExecutor()
	telemetryCollector := telemetry.NewCollector()

	// Create reconciler
	reconciler := controlplane.NewReconciler(
		resourceManager,
		intelHub,
		policyEngine,
		gitWatcher,
		executor,
		telemetryCollector,
	)

	// Create controller
	interval := 10 * time.Second
	if v := os.Getenv("RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	controller := controlplane.NewController(reconciler, interval)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("cloudos: received signal, shutting down", "signal", sig)
		cancel()
	}()

	controller.Run(ctx)
	slog.Info("cloudos: shutdown complete")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
