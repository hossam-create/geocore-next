package consumers

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/kafka"
	"gorm.io/gorm"
)

const groupFraudService = "fraud-service"

// startFraudConsumer listens to orders.events and moderation.events.
// On order.created → enqueue fraud check job.
// On moderation.blocked → enqueue fraud signal job.
func startFraudConsumer(ctx context.Context, db *gorm.DB) {
	// ── Orders: fraud check on order.created ─────────────────────────────────
	ordersConsumer := kafka.NewConsumer(kafka.TopicOrders, groupFraudService)
	go ordersConsumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		if event.Type != "order.created" {
			return nil
		}

		group := groupFraudService + "-orders"
		if kafka.IsProcessed(db, event.EventID, group) {
			return nil
		}

		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		buyerID, _ := data["buyer_id"].(string)
		orderID, _ := data["order_id"].(string)

		_ = jobs.EnqueueDefault(&jobs.Job{
			Type: jobs.JobTypeAnalytics,
			Payload: map[string]interface{}{
				"event":   "fraud_check",
				"user_id": buyerID,
				"context": map[string]interface{}{
					"order_id": orderID,
					"amount":   data["amount"],
					"source":   "kafka-consumer",
				},
			},
		})

		// Also track behavior
		_ = jobs.EnqueueDefault(&jobs.Job{
			Type: jobs.JobTypeBehaviorTrack,
			Payload: map[string]interface{}{
				"user_id": buyerID,
				"event":   "order_created",
				"context": data,
			},
		})

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: fraud check enqueued for order.created", "order_id", orderID)
		return nil
	})

	// ── Moderation: fraud signal on moderation.blocked ────────────────────────
	moderationConsumer := kafka.NewConsumer(kafka.TopicModeration, groupFraudService)
	go moderationConsumer.Run(ctx, func(ctx context.Context, event kafka.Event) error {
		if event.Type != "moderation.blocked" && event.Type != "dispute.opened" {
			return nil
		}

		group := groupFraudService + "-moderation"
		if kafka.IsProcessed(db, event.EventID, group) {
			return nil
		}

		data, ok := event.Data.(map[string]interface{})
		if !ok {
			return nil
		}

		_ = jobs.EnqueueDefault(&jobs.Job{
			Type: jobs.JobTypeAnalytics,
			Payload: map[string]interface{}{
				"event":   "fraud_signal_" + event.Type,
				"user_id": data["respondent_id"],
				"context": data,
			},
		})

		if err := kafka.MarkProcessed(db, event.EventID, group); err != nil {
			slog.Warn("kafka: mark processed failed", "error", err)
		}
		slog.Info("kafka: fraud signal enqueued", "event_type", event.Type)
		return nil
	})
}
