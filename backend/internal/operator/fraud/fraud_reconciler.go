package fraud

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/api/v1"
	"github.com/geocore-next/backend/pkg/k8s"
)

// FraudReconciler dynamically adjusts fraud detection sensitivity.
type FraudReconciler struct {
	events        *k8s.EventRecorder
	currentLevel  string // low, medium, high, aggressive
	falsePosRate  float64
	fraudRate     float64
}

// NewFraudReconciler creates a fraud reconciler.
func NewFraudReconciler(events *k8s.EventRecorder) *FraudReconciler {
	return &FraudReconciler{
		events:       events,
		currentLevel: "medium",
	}
}

// Reconcile adjusts fraud sensitivity based on current threat level.
func (f *FraudReconciler) Reconcile(ctx context.Context, cr *v1.ControlPlane) {
	// Read current fraud metrics (from Prometheus in production)
	fraudRate := f.fraudRate
	falsePositiveRate := f.falsePosRate

	// Fraud spike detected → increase sensitivity
	if fraudRate > 5.0 {
		f.increaseSensitivity(cr)
		f.events.RecordEvent("ControlPlane", cr.Metadata.Name, cr.Metadata.Namespace,
			"IncreaseFraudSensitivity", "FraudSpikeDetected",
			"Fraud rate spike detected, increasing sensitivity to aggressive",
			k8s.EventWarning)
		return
	}

	// High false positive rate → decrease sensitivity
	if falsePositiveRate > 30.0 {
		f.decreaseSensitivity(cr)
		f.events.RecordEvent("ControlPlane", cr.Metadata.Name, cr.Metadata.Namespace,
			"DecreaseFraudSensitivity", "HighFalsePositiveRate",
			"False positive rate too high, decreasing sensitivity",
			k8s.EventNormal)
		return
	}

	// Apply CR-specified sensitivity
	f.currentLevel = cr.Spec.FraudSensitivity
	cr.Status.FraudLevel = f.currentLevel
}

func (f *FraudReconciler) increaseSensitivity(cr *v1.ControlPlane) {
	switch f.currentLevel {
	case "low":
		f.currentLevel = "medium"
	case "medium":
		f.currentLevel = "high"
	case "high":
		f.currentLevel = "aggressive"
	}
	cr.Status.FraudLevel = f.currentLevel
	slog.Warn("operator: fraud sensitivity INCREASED", "level", f.currentLevel)
}

func (f *FraudReconciler) decreaseSensitivity(cr *v1.ControlPlane) {
	switch f.currentLevel {
	case "aggressive":
		f.currentLevel = "high"
	case "high":
		f.currentLevel = "medium"
	case "medium":
		f.currentLevel = "low"
	}
	cr.Status.FraudLevel = f.currentLevel
	slog.Info("operator: fraud sensitivity DECREASED", "level", f.currentLevel)
}

// UpdateMetrics updates the fraud metrics from telemetry.
func (f *FraudReconciler) UpdateMetrics(fraudRate, falsePositiveRate float64) {
	f.fraudRate = fraudRate
	f.falsePosRate = falsePositiveRate
}
