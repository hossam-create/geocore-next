package region

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/api/v1"
	"github.com/geocore-next/backend/pkg/k8s"
)

// RegionReconciler handles automatic region failover.
type RegionReconciler struct {
	events     *k8s.EventRecorder
	regionHealth map[string]bool
}

// NewRegionReconciler creates a region reconciler.
func NewRegionReconciler(events *k8s.EventRecorder) *RegionReconciler {
	return &RegionReconciler{
		events:       events,
		regionHealth:  map[string]bool{"us-east-1": true, "eu-west-1": true, "ap-southeast-1": true},
	}
}

// Reconcile checks region health and triggers failover if needed.
func (r *RegionReconciler) Reconcile(ctx context.Context, cr *v1.ControlPlane) {
	primary := cr.Spec.PrimaryRegion
	failover := cr.Spec.FailoverRegion

	// Check if primary region is healthy
	if !r.regionHealth[primary] {
		slog.Error("operator: PRIMARY REGION DOWN", "region", primary)
		r.failoverTo(ctx, cr, failover)

		r.events.RecordEvent("ControlPlane", cr.Metadata.Name, cr.Metadata.Namespace,
			"RegionFailover", "PrimaryRegionDown",
			"Primary region "+primary+" is down, failing over to "+failover,
			k8s.EventWarning)
		return
	}

	// If currently failed over, check if primary is back
	if cr.Status.ActiveRegion != primary && r.regionHealth[primary] {
		slog.Info("operator: primary region recovered, failing back", "region", primary)
		cr.Status.ActiveRegion = primary

		r.events.RecordEvent("ControlPlane", cr.Metadata.Name, cr.Metadata.Namespace,
			"RegionFailback", "PrimaryRegionRecovered",
			"Primary region "+primary+" recovered, failing back",
			k8s.EventNormal)
	}
}

func (r *RegionReconciler) failoverTo(ctx context.Context, cr *v1.ControlPlane, targetRegion string) {
	slog.Error("operator: FAILOVER triggered", "from", cr.Status.ActiveRegion, "to", targetRegion)
	cr.Status.ActiveRegion = targetRegion
	// In production: update DNS, shift traffic, notify services
}

// UpdateRegionHealth sets the health status for a region.
func (r *RegionReconciler) UpdateRegionHealth(region string, healthy bool) {
	r.regionHealth[region] = healthy
}
