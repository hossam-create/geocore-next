package operator

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/api/v1"
	"github.com/geocore-next/backend/internal/operator/chaos"
	"github.com/geocore-next/backend/internal/operator/fraud"
	"github.com/geocore-next/backend/internal/operator/region"
	"github.com/geocore-next/backend/internal/operator/scaling"
	"github.com/geocore-next/backend/pkg/k8s"
)

// Reconciler is the heart of the operator.
// It dispatches to specialized sub-reconcilers for each domain.
type Reconciler struct {
	Fraud  *fraud.FraudReconciler
	Scale  *scaling.ScaleReconciler
	Region *region.RegionReconciler
	Chaos  *chaos.ChaosReconciler
	Events *k8s.EventRecorder
}

// NewReconciler creates a reconciler with all sub-reconcilers.
func NewReconciler(k8sClient *k8s.Client, events *k8s.EventRecorder) *Reconciler {
	return &Reconciler{
		Fraud:  fraud.NewFraudReconciler(events),
		Scale:  scaling.NewScaleReconciler(k8sClient, events),
		Region: region.NewRegionReconciler(events),
		Chaos:  chaos.NewChaosReconciler(events),
		Events: events,
	}
}

// Reconcile runs all sub-reconcilers against the current ControlPlane CR.
func (r *Reconciler) Reconcile(ctx context.Context, cr *v1.ControlPlane) {
	slog.Debug("operator: reconciling", "name", cr.Metadata.Name)

	// 1. Fraud — adapt security sensitivity
	r.Fraud.Reconcile(ctx, cr)

	// 2. Scaling — auto-scale based on load
	r.Scale.Reconcile(ctx, cr)

	// 3. Region — failover if primary is down
	r.Region.Reconcile(ctx, cr)

	// 4. Chaos — gameday automation
	r.Chaos.Reconcile(ctx, cr)

	// Update overall health status
	cr.Status.Healthy = r.isHealthy(cr)
}

func (r *Reconciler) isHealthy(cr *v1.ControlPlane) bool {
	return cr.Status.ActiveRegion != "" &&
		cr.Status.ErrorBudget >= 0 &&
		cr.Status.CurrentLoad < 90
}
