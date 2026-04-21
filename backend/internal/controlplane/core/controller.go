package core

import (
	"context"
	"log/slog"
	"time"
)

// Controller runs the reconciliation loop at a fixed interval.
// Kubernetes-style: observe → diff → act → repeat.
type Controller struct {
	Reconciler *Reconciler
	Interval   time.Duration
}

// NewController creates a controller with the given reconciler and interval.
func NewController(reconciler *Reconciler, interval time.Duration) *Controller {
	return &Controller{
		Reconciler: reconciler,
		Interval:   interval,
	}
}

// Run starts the reconciliation loop. Blocks until context is cancelled.
func (c *Controller) Run(ctx context.Context) {
	slog.Info("controlplane: controller started", "interval", c.Interval)

	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("controlplane: controller stopped")
			return
		case <-ticker.C:
			c.Reconciler.Reconcile(ctx)
		}
	}
}
