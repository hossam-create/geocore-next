package chaos

import (
	"context"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/api/v1"
	"github.com/geocore-next/backend/pkg/k8s"
)

// ChaosReconciler manages gameday chaos injection on schedule.
type ChaosReconciler struct {
	events     *k8s.EventRecorder
	gamedayActive bool
	lastGameday   time.Time
}

// NewChaosReconciler creates a chaos reconciler.
func NewChaosReconciler(events *k8s.EventRecorder) *ChaosReconciler {
	return &ChaosReconciler{
		events: events,
	}
}

// Reconcile checks if chaos should be injected (gameday windows).
func (c *ChaosReconciler) Reconcile(ctx context.Context, cr *v1.ControlPlane) {
	if !cr.Spec.ChaosEnabled {
		return // chaos is disabled in spec
	}

	if c.isGameDayWindow() {
		if !c.gamedayActive {
			c.startGameDay(ctx, cr)
		}
	} else {
		if c.gamedayActive {
			c.stopGameDay(ctx, cr)
		}
	}
}

func (c *ChaosReconciler) isGameDayWindow() bool {
	// GameDays run weekly (same day, 2am-4am UTC)
	now := time.Now().UTC()
	// Simple check: if it's been 7+ days since last gameday
	if c.lastGameday.IsZero() || now.Sub(c.lastGameday) >= 7*24*time.Hour {
		hour := now.Hour()
		return hour >= 2 && hour < 4
	}
	return false
}

func (c *ChaosReconciler) startGameDay(ctx context.Context, cr *v1.ControlPlane) {
	c.gamedayActive = true
	c.lastGameday = time.Now().UTC()

	slog.Info("operator: GAMEDAY STARTED — injecting chaos")

	c.injectLatency("api", 200)
	c.injectKafkaLag(5000)
	c.injectRedisFailure(5)

	c.events.RecordEvent("ControlPlane", cr.Metadata.Name, cr.Metadata.Namespace,
		"GameDayStart", "ChaosInjection",
		"GameDay started: injecting latency, Kafka lag, Redis failures",
		k8s.EventWarning)
}

func (c *ChaosReconciler) stopGameDay(ctx context.Context, cr *v1.ControlPlane) {
	c.gamedayActive = false
	slog.Info("operator: GAMEDAY ENDED — chaos removed")

	c.events.RecordEvent("ControlPlane", cr.Metadata.Name, cr.Metadata.Namespace,
		"GameDayEnd", "ChaosRemoved",
		"GameDay ended: all chaos injections removed",
		k8s.EventNormal)
}

func (c *ChaosReconciler) injectLatency(service string, ms int) {
	slog.Info("operator: chaos — injecting latency", "service", service, "ms", ms)
	// In production: apply chaos mesh / litmus fault
}

func (c *ChaosReconciler) injectKafkaLag(messages int) {
	slog.Info("operator: chaos — injecting Kafka lag", "messages", messages)
}

func (c *ChaosReconciler) injectRedisFailure(pct int) {
	slog.Info("operator: chaos — injecting Redis failure", "percent", pct)
}
