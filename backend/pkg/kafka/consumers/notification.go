package consumers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/kafka"
	"gorm.io/gorm"
)

const groupNotificationService = "notification-service"

// startNotificationConsumer listens to orders.events, escrow.events,
// payments.events, and users.events — sends email/push notifications.
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
		orderID, _ := data["order_id"].(string)

		_ = jobs.EnqueueDefault(&jobs.Job{
			Type: jobs.JobTypeEmail,
			Payload: map[string]interface{}{
				"to":      data["seller_email"],
				"subject": "New order received! #" + orderID,
				"body":    fmt.Sprintf("You have a new order #%s. Please prepare for delivery.", orderID),
			},
		})
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

		_ = jobs.EnqueueDefault(&jobs.Job{
			Type: jobs.JobTypeEmail,
			Payload: map[string]interface{}{
				"to":      data["seller_email"],
				"subject": "Your escrow has been released",
				"body":    "Good news! Your funds have been released to your wallet.",
			},
		})

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: notification sent for escrow.released", "escrow_id", data["escrow_id"])
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
		orderID, _ := data["order_id"].(string)

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

		email, _ := data["email"].(string)
		if email != "" {
			_ = jobs.EnqueueDefault(&jobs.Job{
				Type: jobs.JobTypeEmail,
				Payload: map[string]interface{}{
					"to":      email,
					"subject": "Welcome to GEOCore!",
					"body":    "Your account has been created successfully. Start exploring now!",
				},
			})
		}

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: welcome notification sent for user.created", "email", email)
		return nil
	})
}
