package scaling

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/api/v1"
	"github.com/geocore-next/backend/pkg/k8s"
)

// ScaleReconciler auto-scales deployments based on latency, CPU, and load.
type ScaleReconciler struct {
	k8sClient *k8s.Client
	events    *k8s.EventRecorder
	p95Ms     float64
	cpuPct    float64
	rps       float64
}

// NewScaleReconciler creates a scaling reconciler.
func NewScaleReconciler(k8sClient *k8s.Client, events *k8s.EventRecorder) *ScaleReconciler {
	return &ScaleReconciler{
		k8sClient: k8sClient,
		events:    events,
	}
}

// Reconcile checks metrics and scales deployments accordingly.
func (s *ScaleReconciler) Reconcile(ctx context.Context, cr *v1.ControlPlane) {
	targetP95 := float64(cr.Spec.TargetLatencyP95)
	minReplicas := cr.Spec.MinReplicas
	maxReplicas := cr.Spec.MaxReplicas

	// Scale up: latency breach or high load
	if s.p95Ms > targetP95 {
		desired := cr.Status.Replicas + 2
		if desired > maxReplicas {
			desired = maxReplicas
		}
		if desired > cr.Status.Replicas {
			s.scaleUp(ctx, cr, desired)
		}
		return
	}

	// Scale up: high CPU
	if s.cpuPct > 70 {
		desired := cr.Status.Replicas + 1
		if desired > maxReplicas {
			desired = maxReplicas
		}
		if desired > cr.Status.Replicas {
			s.scaleUp(ctx, cr, desired)
		}
		return
	}

	// Scale down: low CPU + low RPS
	if s.cpuPct < 30 && s.rps < 50 {
		desired := cr.Status.Replicas - 1
		if desired < minReplicas {
			desired = minReplicas
		}
		if desired < cr.Status.Replicas {
			s.scaleDown(ctx, cr, desired)
		}
	}
}

func (s *ScaleReconciler) scaleUp(ctx context.Context, cr *v1.ControlPlane, desired int) {
	slog.Info("operator: scaling UP", "deployment", "api", "replicas", desired)
	if s.k8sClient != nil {
		_ = s.k8sClient.ScaleDeployment(ctx, "api", desired)
	}
	cr.Status.Replicas = desired
	s.events.RecordEvent("ControlPlane", cr.Metadata.Name, cr.Metadata.Namespace,
		"ScaleUp", "LatencyBreach",
		"Scaling up due to latency or load breach",
		k8s.EventNormal)
}

func (s *ScaleReconciler) scaleDown(ctx context.Context, cr *v1.ControlPlane, desired int) {
	slog.Info("operator: scaling DOWN", "deployment", "api", "replicas", desired)
	if s.k8sClient != nil {
		_ = s.k8sClient.ScaleDeployment(ctx, "api", desired)
	}
	cr.Status.Replicas = desired
	s.events.RecordEvent("ControlPlane", cr.Metadata.Name, cr.Metadata.Namespace,
		"ScaleDown", "Underutilized",
		"Scaling down due to low utilization",
		k8s.EventNormal)
}

// UpdateMetrics updates the scaling metrics from telemetry.
func (s *ScaleReconciler) UpdateMetrics(p95Ms, cpuPct, rps float64) {
	s.p95Ms = p95Ms
	s.cpuPct = cpuPct
	s.rps = rps
}
