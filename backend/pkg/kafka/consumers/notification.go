package consumers

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/geocore-next/backend/pkg/email"
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/kafka"
	"gorm.io/gorm"
)

const groupNotificationService = "notification-service"

// startNotificationConsumer listens to orders.events, escrow.events,
// payments.events, and users.events — sends email/push notifications.
// All emails use the production EmailService pipeline with HTML templates,
// idempotency, rate limiting, retry, and async delivery.
func startNotificationConsumer(ctx context.Context, db *gorm.DB) {
	// ── Orders: notify seller on order.created ────────────────────────────────
	ordersConsumer := kafka.NewConsumer(kafka.TopicOrders, groupNotificationService)
	go ordersConsumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		if event.Type != "order.created" {
			return nil
		}
		group := groupNotificationService + "-orders"
		if kafka.IsProcessed(db, event.EventID, group) {
			return nil
		}

		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		sellerID, _ := data["seller_id"].(string)
		sellerEmail, _ := data["seller_email"].(string)
		orderID, _ := data["order_id"].(string)

		// Template-based email notification to seller
		if sellerEmail != "" {
			baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
			_ = email.Default().SendAsync(ctx, &email.Message{
				To:           sellerEmail,
				UserID:       sellerID,
				Subject:      fmt.Sprintf("New order received! #%s", orderID),
				TemplateName: "notification",
				Data: email.NotificationData(
					"there",
					"New Order Received",
					fmt.Sprintf("You have a new order #%s. Please prepare for delivery.", orderID),
					"View Orders",
					baseURL+"/orders",
				),
				IdempotencyKey: "order_created_seller:" + orderID,
			})
		}

		// Push notification to seller
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type: jobs.JobTypePushNotification,
			Payload: map[string]interface{}{
				"user_id": sellerID,
				"title":   "New Order",
				"body":    fmt.Sprintf("You received order #%s", orderID),
			},
		})

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: notification sent for order.created", "order_id", orderID)
		return nil
	})

	// ── Escrow: notify seller on escrow.released ─────────────────────────────
	escrowConsumer := kafka.NewConsumer(kafka.TopicEscrow, groupNotificationService)
	go escrowConsumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		if event.Type != "escrow.released" {
			return nil
		}
		group := groupNotificationService + "-escrow"
		if kafka.IsProcessed(db, event.EventID, group) {
			return nil
		}

		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		sellerEmail, _ := data["seller_email"].(string)
		sellerID, _ := data["seller_id"].(string)
		escrowID, _ := data["escrow_id"].(string)
		amount := floatVal(data["amount"])
		currency, _ := data["currency"].(string)

		// Template-based escrow released email
		if sellerEmail != "" {
			_ = email.SendEscrowReleasedEmail(sellerEmail, "", sellerID, escrowID, amount, currency)
		}

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: notification sent for escrow.released", "escrow_id", escrowID)
		return nil
	})

	// ── Payments: notify on payment.succeeded ─────────────────────────────────
	paymentsConsumer := kafka.NewConsumer(kafka.TopicPayments, groupNotificationService)
	go paymentsConsumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		if event.Type != "payment.succeeded" {
			return nil
		}
		group := groupNotificationService + "-payments"
		if kafka.IsProcessed(db, event.EventID, group) {
			return nil
		}

		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		userID, _ := data["user_id"].(string)
		userEmail, _ := data["user_email"].(string)
		orderID, _ := data["order_id"].(string)
		amount := floatVal(data["amount"])
		currency, _ := data["currency"].(string)

		// Template-based transaction receipt email
		if userEmail != "" && orderID != "" {
			_ = email.SendTransactionReceiptEmail(userEmail, "", userID, orderID, "Order #"+orderID, amount, currency)
		}

		// Push notification
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type: jobs.JobTypePushNotification,
			Payload: map[string]interface{}{
				"user_id": userID,
				"title":   "Payment Confirmed",
				"body":    fmt.Sprintf("Your payment for order #%s has been confirmed.", orderID),
			},
		})

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: notification sent for payment.succeeded", "order_id", orderID)
		return nil
	})

	// ── Users: welcome email on user.created ──────────────────────────────────
	usersConsumer := kafka.NewConsumer(kafka.TopicUsers, groupNotificationService)
	go usersConsumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		if event.Type != "user.created" {
			return nil
		}
		group := groupNotificationService + "-users"
		if kafka.IsProcessed(db, event.EventID, group) {
			return nil
		}

		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		userEmail, _ := data["email"].(string)
		userName, _ := data["name"].(string)
		if userEmail != "" {
			_ = email.SendWelcomeEmail(userEmail, userName)
		}

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: welcome notification sent for user.created", "email", userEmail)
		return nil
	})
}

// ── Helpers ────────────────────────────────────────────────────────────────────

func floatVal(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
