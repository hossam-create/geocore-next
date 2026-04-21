package payments

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/geocore-next/backend/internal/chargebacks"
	"github.com/geocore-next/backend/internal/subscriptions"
	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v79"
	"gorm.io/gorm"
)

// WebhookHandler returns a Gin handler for Stripe webhook events.
//
// This endpoint is intentionally registered OUTSIDE the /api/v1 group and
// the Auth() middleware — authentication is performed via Stripe signature
// verification (HMAC-SHA256) using the STRIPE_WEBHOOK_SECRET env var.
//
// Supported events:
//
//	payment_intent.succeeded        — create escrow, mark payment succeeded
//	payment_intent.payment_failed   — mark payment failed
//	payment_intent.canceled         — mark payment cancelled
//	refund.created                  — mark payment & escrow as refunded
func WebhookHandler(db *gorm.DB) gin.HandlerFunc {
	h := NewHandler(db, nil)

	return func(c *gin.Context) {
		responsePayload := gin.H{"status": "ok"}

		// ── 1. Read the raw body (required for Stripe signature check) ──────────
		// IMPORTANT: must read before any middleware that parses the body.
		payload, err := io.ReadAll(io.LimitReader(c.Request.Body, 65536))
		if err != nil {
			slog.Error("webhook: failed to read request body", "error", err.Error())
			c.Status(http.StatusBadRequest)
			return
		}

		// ── 2. Verify Stripe-Signature header ───────────────────────────────────
		sigHeader := c.GetHeader("Stripe-Signature")
		event, err := VerifyWebhookSignature(payload, sigHeader)
		if err != nil {
			slog.Warn("webhook: signature verification failed",
				"error", err.Error(),
				"ip", c.ClientIP(),
			)
			c.Status(http.StatusBadRequest)
			return
		}

		// ── 3. Idempotency: deduplicate Stripe retries ─────────────────────────
		var existing ProcessedStripeEvent
		if err := db.Where("stripe_event_id = ?", event.ID).First(&existing).Error; err == nil {
			slog.Debug("webhook: duplicate event ignored (already processed)",
				"event_id", event.ID,
				"event_type", event.Type,
				"idempotency_key", event.ID,
				"actor", "stripe_webhook",
			)
			if existing.ResponseBody != "" {
				c.Data(existing.ResponseCode, "application/json", []byte(existing.ResponseBody))
				return
			}
			c.JSON(http.StatusOK, responsePayload)
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			slog.Error("webhook: db error checking processed event", "error", err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}

		// Mark as processed BEFORE dispatching to prevent double-handling on panic/restart
		processedEvt := ProcessedStripeEvent{
			StripeEventID: event.ID,
			EventType:     string(event.Type),
			ResponseCode:  http.StatusOK,
			ResponseBody:  `{"status":"ok"}`,
			ProcessedAt:   time.Now(),
		}
		if err := db.Create(&processedEvt).Error; err != nil {
			slog.Error("webhook: failed to record processed event", "event_id", event.ID, "error", err.Error())
			c.Status(http.StatusInternalServerError)
			return
		}

		slog.Info("stripe webhook received",
			"event_id", event.ID,
			"event_type", event.Type,
			"idempotency_key", event.ID,
			"actor", "stripe_webhook",
		)

		// ── 4. Route to the appropriate handler ─────────────────────────────────
		switch event.Type {

		case "payment_intent.succeeded":
			h.handleWebhookPaymentSucceeded(event)

		case "payment_intent.payment_failed":
			h.handleWebhookPaymentFailed(event)

		case "payment_intent.canceled":
			h.handleWebhookPaymentCancelled(event)

		case "refund.created":
			h.handleWebhookRefundCreated(event)

		case "customer.subscription.updated":
			h.handleWebhookSubscriptionUpdated(event)

		case "customer.subscription.deleted":
			h.handleWebhookSubscriptionDeleted(event)

		case "charge.dispute.created":
			h.handleWebhookDisputeCreated(event)

		case "charge.dispute.updated":
			h.handleWebhookDisputeUpdated(event)

		case "charge.dispute.closed":
			h.handleWebhookDisputeClosed(event)

		default:
			slog.Debug("webhook: unhandled event type", "type", event.Type)
		}

		// Always return 200 to acknowledge receipt.
		// Stripe will retry with exponential back-off if we return a non-2xx status.
		c.JSON(http.StatusOK, responsePayload)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Event handlers
// ════════════════════════════════════════════════════════════════════════════

// handleWebhookPaymentSucceeded processes the payment_intent.succeeded event.
// It marks the local Payment record as succeeded and creates an EscrowAccount.
func (h *Handler) handleWebhookPaymentSucceeded(event *stripe.Event) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		slog.Error("webhook: failed to unmarshal payment_intent.succeeded",
			"event_id", event.ID, "error", err.Error())
		return
	}

	var payment Payment
	err := h.db.Where("stripe_payment_intent_id = ?", pi.ID).First(&payment).Error
	if err != nil {
		slog.Warn("webhook: payment not found for payment_intent.succeeded",
			"stripe_pi", pi.ID, "event_id", event.ID)
		return
	}

	if payment.Status == PaymentStatusSucceeded {
		slog.Debug("webhook: payment already succeeded (idempotent)", "payment_id", payment.ID.String())
		return
	}

	// Reuse the same success logic as the polling confirm endpoint
	if err := h.handlePaymentSuccess(nil, &payment, &pi); err != nil {
		slog.Error("webhook: handlePaymentSuccess failed",
			"payment_id", payment.ID.String(), "error", err.Error())
		return
	}

	// Publish domain event
	events.Publish(events.Event{
		Type: events.EventPaymentCompleted,
		Payload: map[string]interface{}{
			"payment_id": payment.ID,
			"user_id":    payment.UserID,
			"amount":     payment.Amount,
			"currency":   payment.Currency,
			"stripe_pi":  pi.ID,
		},
	})
	metrics.IncPaymentsProcessed()

	// Transactional outbox for Kafka delivery
	_ = kafka.WriteOutbox(h.db, kafka.TopicPayments, kafka.New(
		"payment.succeeded",
		payment.ID.String(),
		"payment",
		kafka.Actor{Type: "user", ID: payment.UserID.String()},
		map[string]interface{}{
			"payment_id": payment.ID.String(),
			"user_id":    payment.UserID.String(),
			"amount":     payment.Amount,
			"currency":   payment.Currency,
		},
		kafka.EventMeta{Source: "webhook-service"},
	))
}

// handleWebhookPaymentFailed processes the payment_intent.payment_failed event.
func (h *Handler) handleWebhookPaymentFailed(event *stripe.Event) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		slog.Error("webhook: failed to unmarshal payment_intent.payment_failed",
			"event_id", event.ID, "error", err.Error())
		return
	}

	// Extract failure reason from the last payment error
	failureReason := "Payment failed"
	if pi.LastPaymentError != nil && pi.LastPaymentError.Msg != "" {
		failureReason = pi.LastPaymentError.Msg
	}

	result := h.db.Model(&Payment{}).
		Where("stripe_payment_intent_id = ? AND status = ?", pi.ID, PaymentStatusPending).
		Updates(map[string]any{
			"status":         PaymentStatusFailed,
			"failure_reason": failureReason,
		})

	if result.Error != nil {
		slog.Error("webhook: failed to update payment to failed",
			"stripe_pi", pi.ID, "error", result.Error.Error())
		return
	}

	slog.Info("webhook: payment marked failed",
		"stripe_pi", pi.ID,
		"reason", failureReason,
	)
}

// handleWebhookPaymentCancelled processes the payment_intent.canceled event.
func (h *Handler) handleWebhookPaymentCancelled(event *stripe.Event) {
	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		slog.Error("webhook: failed to unmarshal payment_intent.canceled",
			"event_id", event.ID, "error", err.Error())
		return
	}

	result := h.db.Model(&Payment{}).
		Where("stripe_payment_intent_id = ? AND status = ?", pi.ID, PaymentStatusPending).
		Update("status", PaymentStatusCancelled)

	if result.Error != nil {
		slog.Error("webhook: failed to update payment to cancelled",
			"stripe_pi", pi.ID, "error", result.Error.Error())
		return
	}

	slog.Info("webhook: payment cancelled", "stripe_pi", pi.ID)
}

// handleWebhookRefundCreated processes the refund.created event.
// This is an idempotent guard — refunds triggered by our API are already
// recorded. This handler catches refunds initiated directly in the Stripe
// dashboard by an admin.
func (h *Handler) handleWebhookRefundCreated(event *stripe.Event) {
	type refundEvent struct {
		ID            string `json:"id"`
		PaymentIntent string `json:"payment_intent"`
		Status        string `json:"status"`
	}
	var ref refundEvent
	if err := json.Unmarshal(event.Data.Raw, &ref); err != nil {
		slog.Error("webhook: failed to unmarshal refund.created",
			"event_id", event.ID, "error", err.Error())
		return
	}

	if ref.Status != "succeeded" {
		slog.Debug("webhook: refund not yet succeeded, skipping", "refund_id", ref.ID)
		return
	}

	var payment Payment
	err := h.db.Preload("Escrow").
		Where("stripe_payment_intent_id = ?", ref.PaymentIntent).
		First(&payment).Error
	if err != nil {
		slog.Warn("webhook: payment not found for refund.created",
			"pi", ref.PaymentIntent, "refund_id", ref.ID)
		return
	}

	// Idempotent check
	if payment.Status == PaymentStatusRefunded {
		slog.Debug("webhook: payment already refunded (idempotent)", "payment_id", payment.ID.String())
		return
	}

	now := time.Now()
	h.db.Model(&payment).Updates(map[string]any{
		"status":      PaymentStatusRefunded,
		"refunded_at": now,
	})
	if payment.Escrow != nil {
		h.db.Model(payment.Escrow).Update("status", EscrowStatusRefunded)
	}

	slog.Info("webhook: refund recorded from Stripe dashboard",
		"payment_id", payment.ID.String(),
		"refund_id", ref.ID,
	)
}

// handleWebhookSubscriptionUpdated processes customer.subscription.updated events.
func (h *Handler) handleWebhookSubscriptionUpdated(event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		slog.Error("webhook: failed to unmarshal customer.subscription.updated",
			"event_id", event.ID, "error", err.Error())
		return
	}
	subscriptions.HandleStripeSubscriptionUpdated(h.db, &sub)
}

// handleWebhookSubscriptionDeleted processes customer.subscription.deleted events.
func (h *Handler) handleWebhookSubscriptionDeleted(event *stripe.Event) {
	var sub stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		slog.Error("webhook: failed to unmarshal customer.subscription.deleted",
			"event_id", event.ID, "error", err.Error())
		return
	}
	subscriptions.HandleStripeSubscriptionDeleted(h.db, &sub)
}

// handleWebhookDisputeCreated processes charge.dispute.created events.
func (h *Handler) handleWebhookDisputeCreated(event *stripe.Event) {
	var d stripe.Dispute
	if err := json.Unmarshal(event.Data.Raw, &d); err != nil {
		slog.Error("webhook: failed to unmarshal charge.dispute.created",
			"event_id", event.ID, "error", err.Error())
		return
	}
	piID := ""
	if d.PaymentIntent != nil {
		piID = d.PaymentIntent.ID
	}
	var evidenceDueBy *time.Time
	if d.EvidenceDetails.DueBy > 0 {
		t := time.Unix(d.EvidenceDetails.DueBy, 0)
		evidenceDueBy = &t
	}
	chargebacks.HandleDisputeCreated(h.db, d.ID, piID, string(d.Reason), string(d.Status), float64(d.Amount)/100.0, string(d.Currency), evidenceDueBy)
}

// handleWebhookDisputeUpdated processes charge.dispute.updated events.
func (h *Handler) handleWebhookDisputeUpdated(event *stripe.Event) {
	var d stripe.Dispute
	if err := json.Unmarshal(event.Data.Raw, &d); err != nil {
		slog.Error("webhook: failed to unmarshal charge.dispute.updated",
			"event_id", event.ID, "error", err.Error())
		return
	}
	var evidenceDueBy *time.Time
	if d.EvidenceDetails.DueBy > 0 {
		t := time.Unix(d.EvidenceDetails.DueBy, 0)
		evidenceDueBy = &t
	}
	chargebacks.HandleDisputeUpdated(h.db, d.ID, string(d.Status), evidenceDueBy)
}

// handleWebhookDisputeClosed processes charge.dispute.closed events.
func (h *Handler) handleWebhookDisputeClosed(event *stripe.Event) {
	var d stripe.Dispute
	if err := json.Unmarshal(event.Data.Raw, &d); err != nil {
		slog.Error("webhook: failed to unmarshal charge.dispute.closed",
			"event_id", event.ID, "error", err.Error())
		return
	}
	chargebacks.HandleDisputeClosed(h.db, d.ID, string(d.Status))
}
