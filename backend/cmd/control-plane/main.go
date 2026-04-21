package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geocore-next/backend/internal/controlplane/analyzer"
	"github.com/geocore-next/backend/internal/controlplane/core"
	"github.com/geocore-next/backend/internal/controlplane/executor"
	"github.com/geocore-next/backend/internal/controlplane/planner"
	"github.com/geocore-next/backend/internal/controlplane/policy"
	"github.com/geocore-next/backend/internal/controlplane/simulator"
)

func main() {
	slog.Info("controlplane: starting standalone control plane")

	// Wire all components
	metricsReader := analyzer.NewMetricsReader()
	a := analyzer.NewAnalyzer(metricsReader)
	p := planner.NewPlanner()
	s := simulator.NewSimulator()
	e := executor.NewExecutor()
	pol := policy.NewPolicyEngine()

	reconciler := core.NewReconciler(a, p, s, e, pol)

	interval := 10 * time.Second
	if v := os.Getenv("RECONCILE_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			interval = d
		}
	}

	controller := core.NewController(reconciler, interval)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("controlplane: received signal, shutting down", "signal", sig)
		cancel()
	}()

	controller.Run(ctx)
	slog.Info("controlplane: shutdown complete")
}
