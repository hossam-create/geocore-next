package events

import (
	"log/slog"

	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/metrics"
)

// enqueueGeoScore is a helper to enqueue an async GeoScore recompute.
func enqueueGeoScore(userID string) {
	if userID == "" {
		return
	}
	_ = jobs.EnqueueDefault(&jobs.Job{
		Type:     jobs.JobTypeGeoScoreUpdate,
		Priority: 7,
		Payload:  map[string]interface{}{"user_id": userID},
	})
}

// RegisterDefaultConsumers wires all system-level event consumers.
// This is called once at startup and adds async fanout for:
//   - analytics (PostHog tracking)
//   - notifications (push/email triggers)
//   - fraud (velocity + behavior scoring)
//   - reputation (score invalidation)
func RegisterDefaultConsumers() {
	b := Global()

	// ── Order Created ─────────────────────────────────────────────────────────
	b.Subscribe(EventOrderCreated, func(e Event) {
		metrics.IncWalletOp("order_created_event", "processed")
		slog.Info("event: order created", "payload", e.Payload, "request_id", e.RequestID)

		// Enqueue analytics job
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type:      jobs.JobTypeAnalytics,
			RequestID: e.RequestID,
			Payload: map[string]interface{}{
				"event":      "order_created",
				"user_id":    e.Payload["buyer_id"],
				"properties": e.Payload,
			},
		})

		// Trigger fraud check
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type:      jobs.JobTypeAnalytics,
			RequestID: e.RequestID,
			Payload: map[string]interface{}{
				"event":   "fraud_check",
				"user_id": e.Payload["buyer_id"],
				"context": e.Payload,
			},
		})
	})

	// ── Payment Completed ─────────────────────────────────────────────────────
	b.Subscribe(EventPaymentCompleted, func(e Event) {
		slog.Info("event: payment completed", "payload", e.Payload, "request_id", e.RequestID)
		metrics.IncWalletOp("payment_completed_event", "processed")

		// Enqueue analytics
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type:      jobs.JobTypeAnalytics,
			RequestID: e.RequestID,
			Payload: map[string]interface{}{
				"event":      "payment_completed",
				"user_id":    e.Payload["user_id"],
				"properties": e.Payload,
			},
		})

		// GeoScore update — wallet tx success
		if uid, _ := e.Payload["user_id"].(string); uid != "" {
			enqueueGeoScore(uid)
		}
	})

	// ── Escrow Released ───────────────────────────────────────────────────────
	b.Subscribe(EventEscrowReleased, func(e Event) {
		slog.Info("event: escrow released", "payload", e.Payload, "request_id", e.RequestID)
		metrics.IncWalletOp("escrow_released_event", "processed")

		// Notify seller
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type:      jobs.JobTypeEmail,
			RequestID: e.RequestID,
			Payload: map[string]interface{}{
				"to":      e.Payload["seller_email"],
				"subject": "Your escrow has been released",
				"body":    "Good news! Your funds have been released to your wallet.",
			},
		})

		// GeoScore update for both buyer and seller on escrow release
		enqueueGeoScore(e.Payload["buyer_id"].(string))
		enqueueGeoScore(e.Payload["seller_id"].(string))

		// Route intelligence update
		if origin, _ := e.Payload["origin"].(string); origin != "" {
			dest, _ := e.Payload["destination"].(string)
			_ = jobs.EnqueueDefault(&jobs.Job{
				Type:     jobs.JobTypeRouteUpdate,
				Priority: 8,
				Payload:  map[string]interface{}{"origin": origin, "destination": dest},
			})
		}
	})

	// ── User Registered ───────────────────────────────────────────────────────
	b.Subscribe(EventUserRegistered, func(e Event) {
		slog.Info("event: user registered", "user_id", e.Payload["user_id"])
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type:      jobs.JobTypeAnalytics,
			RequestID: e.RequestID,
			Payload: map[string]interface{}{
				"event":   "user_registered",
				"user_id": e.Payload["user_id"],
			},
		})
	})

	// ── Review Posted ─────────────────────────────────────────────────────────
	b.Subscribe(EventReviewPosted, func(e Event) {
		slog.Info("event: review posted", "seller_id", e.Payload["seller_id"])
		// Invalidate reputation cache by re-queuing a refresh job (no-op if
		// reputation.Get is called — it will refresh automatically)
	})

	// ── Dispute Opened ────────────────────────────────────────────────────────
	b.Subscribe(EventDisputeOpened, func(e Event) {
		slog.Info("event: dispute opened", "dispute_id", e.Payload["dispute_id"])
		// Fraud signal — bump risk score for the reported user
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type:      jobs.JobTypeAnalytics,
			RequestID: e.RequestID,
			Payload: map[string]interface{}{
				"event":   "fraud_signal_dispute",
				"user_id": e.Payload["respondent_id"],
				"context": e.Payload,
			},
		})
		// GeoScore: re-evaluate both parties when a dispute opens
		enqueueGeoScore(func() string { s, _ := e.Payload["buyer_id"].(string); return s }())
		enqueueGeoScore(func() string { s, _ := e.Payload["seller_id"].(string); return s }())
	})

	slog.Info("event bus: default consumers registered")
}
