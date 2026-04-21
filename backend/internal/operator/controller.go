package operator

import (
	"context"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/api/v1"
	"github.com/geocore-next/backend/pkg/k8s"
)

// Controller runs the operator reconciliation loop.
type Controller struct {
	Reconciler *Reconciler
	K8sClient  *k8s.Client
	Events     *k8s.EventRecorder
	CR         *v1.ControlPlane
	Interval   time.Duration
}

// NewController creates an operator controller.
func NewController(reconciler *Reconciler, k8sClient *k8s.Client, cr *v1.ControlPlane, interval time.Duration) *Controller {
	return &Controller{
		Reconciler: reconciler,
		K8sClient:  k8sClient,
		Events:     k8s.NewEventRecorder(),
		CR:         cr,
		Interval:   interval,
	}
}

// Run starts the operator control loop. Blocks until context is cancelled.
func (c *Controller) Run(ctx context.Context) {
	slog.Info("operator: controller started", "interval", c.Interval)

	ticker := time.NewTicker(c.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("operator: controller stopped")
			return
		case <-ticker.C:
			c.Reconciler.Reconcile(ctx, c.CR)
			c.CR.Status.LastReconcile = time.Now().UTC()
		}
	}
}
