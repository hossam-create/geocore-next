package controlplane

import (
	"context"
	"log/slog"
	"time"
)

// Controller runs the Cloud OS reconciliation loop.
// This is the top-level orchestrator — the "kernel" of the Cloud OS.
type Controller struct {
	Reconciler *Reconciler
	Interval   time.Duration
}

// NewController creates a Cloud OS controller.
func NewController(reconciler *Reconciler, interval time.Duration) *Controller {
	return &Controller{
		Reconciler: reconciler,
		Interval:   interval,
	}
}

// Run starts the Cloud OS control loop. Blocks until context is cancelled.
func (c *Controller) Run(ctx context.Context) {
	slog.Info("cloudos: controller started", "interval", c.Interval)

	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("cloudos: controller stopped")
			return
		case <-ticker.C:
			c.Reconciler.Reconcile(ctx)
		}
	}
}
