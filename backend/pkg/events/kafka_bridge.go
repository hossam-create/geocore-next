package events

import (
	"log/slog"

	"github.com/geocore-next/backend/pkg/kafka"
	"gorm.io/gorm"
)

// topicMap maps domain event types to aggregate-based Kafka topics.
// Multiple event types flow into the same topic; event_type distinguishes them.
var topicMap = map[EventType]string{
	EventOrderCreated:      kafka.TopicOrders,
	EventOrderCancelled:    kafka.TopicOrders,
	EventOrderShipped:      kafka.TopicOrders,
	EventOrderDelivered:    kafka.TopicOrders,
	EventPaymentCompleted:  kafka.TopicPayments,
	EventEscrowCreated:     kafka.TopicEscrow,
	EventEscrowReleased:    kafka.TopicEscrow,
	EventWalletDeposited:   kafka.TopicWallet,
	EventWalletDebited:     kafka.TopicWallet,
	EventUserRegistered:    kafka.TopicUsers,
	EventReviewPosted:      kafka.TopicOrders,
	EventReferralCompleted: kafka.TopicOrders,
	EventListingCreated:    kafka.TopicModeration,
	EventDisputeOpened:     kafka.TopicModeration,
	EventFraudChecked:      kafka.TopicFraud,
	EventModerationBlocked: kafka.TopicModeration,
	EventShippingCreated:   kafka.TopicShipping,
}

// aggregateMap maps domain event types to their aggregate type for the Event envelope.
var aggregateMap = map[EventType]string{
	EventOrderCreated:      "order",
	EventOrderCancelled:    "order",
	EventOrderShipped:      "order",
	EventOrderDelivered:    "order",
	EventPaymentCompleted:  "payment",
	EventEscrowCreated:     "escrow",
	EventEscrowReleased:    "escrow",
	EventWalletDeposited:   "wallet",
	EventWalletDebited:     "wallet",
	EventUserRegistered:    "user",
	EventReviewPosted:      "order",
	EventReferralCompleted: "order",
	EventListingCreated:    "listing",
	EventDisputeOpened:     "dispute",
	EventFraudChecked:      "fraud",
	EventModerationBlocked: "moderation",
	EventShippingCreated:   "shipping",
}

// RegisterKafkaBridge subscribes to the global event bus and writes every
// domain event to the outbox_events table (transactional outbox pattern).
// The OutboxWorker then delivers them to Kafka asynchronously.
// Completely no-op when KAFKA_BROKERS is not set.
func RegisterKafkaBridge(db *gorm.DB) {
	b := Global()

	for eventType, topic := range topicMap {
		et := eventType
		t := topic
		aggType := aggregateMap[et]
		b.Subscribe(et, func(e Event) {
			userID, _ := e.Payload["user_id"].(string)
			aggregateID, _ := e.Payload["order_id"].(string)
			if aggregateID == "" {
				aggregateID, _ = e.Payload["payment_id"].(string)
			}
			if aggregateID == "" {
				aggregateID, _ = e.Payload["escrow_id"].(string)
			}
			if aggregateID == "" {
				aggregateID, _ = e.Payload["user_id"].(string)
			}
			if aggregateID == "" {
				aggregateID = e.RequestID // fallback
			}

			kafkaEvent := kafka.New(
				string(e.Type),
				aggregateID,
				aggType,
				kafka.Actor{Type: "user", ID: userID},
				e.Payload,
				kafka.EventMeta{
					TraceID: e.RequestID,
					Source:  "api-service",
				},
			)

			if err := kafka.WriteOutbox(db, t, kafkaEvent); err != nil {
				slog.Warn("kafka: outbox write failed",
					"event_type", et, "topic", t, "error", err)
				return
			}
			slog.Debug("kafka: event queued in outbox", "event_type", et, "topic", t)
		})
	}
	slog.Info("kafka: outbox bridge registered for all domain events")
}
