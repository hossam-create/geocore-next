// Package kafka provides a thin, optional Kafka producer/consumer layer.
// All calls are no-ops when KAFKA_BROKERS env is not set, enabling gradual
// rollout without breaking existing in-process event bus.
package kafka

// ── Aggregate-based topics ────────────────────────────────────────────────────
// Each domain aggregate owns ONE topic.  event_type distinguishes the action.
// Partition key = aggregate_id guarantees ordering per entity.

const (
	TopicOrders        = "orders.events"
	TopicWallet        = "wallet.events"
	TopicEscrow        = "escrow.events"
	TopicPayments      = "payments.events"
	TopicUsers         = "users.events"
	TopicFraud         = "fraud.events"
	TopicModeration    = "moderation.events"
	TopicShipping      = "shipping.events"
	TopicNotifications = "notifications.events"
)

// ── Dead-letter queues ────────────────────────────────────────────────────────
const (
	TopicOrdersDLQ        = "orders.events.dlq"
	TopicWalletDLQ        = "wallet.events.dlq"
	TopicEscrowDLQ        = "escrow.events.dlq"
	TopicPaymentsDLQ      = "payments.events.dlq"
	TopicUsersDLQ         = "users.events.dlq"
	TopicFraudDLQ         = "fraud.events.dlq"
	TopicModerationDLQ    = "moderation.events.dlq"
	TopicShippingDLQ      = "shipping.events.dlq"
	TopicNotificationsDLQ = "notifications.events.dlq"
)

// DLQFor returns the DLQ topic for a given primary topic.
func DLQFor(topic string) string {
	switch topic {
	case TopicOrders:
		return TopicOrdersDLQ
	case TopicWallet:
		return TopicWalletDLQ
	case TopicEscrow:
		return TopicEscrowDLQ
	case TopicPayments:
		return TopicPaymentsDLQ
	case TopicUsers:
		return TopicUsersDLQ
	case TopicFraud:
		return TopicFraudDLQ
	case TopicModeration:
		return TopicModerationDLQ
	case TopicShipping:
		return TopicShippingDLQ
	case TopicNotifications:
		return TopicNotificationsDLQ
	default:
		return topic + ".dlq"
	}
}

// AllTopics is the canonical list passed to kafka-init for pre-creation.
var AllTopics = []string{
	TopicOrders,
	TopicWallet,
	TopicEscrow,
	TopicPayments,
	TopicUsers,
	TopicFraud,
	TopicModeration,
	TopicShipping,
	TopicNotifications,
	TopicOrdersDLQ,
	TopicWalletDLQ,
	TopicEscrowDLQ,
	TopicPaymentsDLQ,
	TopicUsersDLQ,
	TopicFraudDLQ,
	TopicModerationDLQ,
	TopicShippingDLQ,
	TopicNotificationsDLQ,
}
