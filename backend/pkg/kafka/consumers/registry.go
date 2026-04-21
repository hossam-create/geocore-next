// Package consumers provides domain-specific Kafka consumer groups.
// Each consumer is a no-op when KAFKA_BROKERS is not set.
// They are designed for the future microservice split — the monolith
// continues using the in-process event bus; these consumers activate
// only when KAFKA_BROKERS is configured.
package consumers

import (
	"context"
	"log/slog"

	"gorm.io/gorm"
)

// StartAll launches every Kafka consumer group in background goroutines.
// All consumers gracefully stop when ctx is cancelled.
// No-op when KAFKA_BROKERS is not set.
func StartAll(ctx context.Context, db *gorm.DB) {
	startWalletConsumer(ctx, db)
	startEscrowConsumer(ctx, db)
	startNotificationConsumer(ctx, db)
	startFraudConsumer(ctx, db)
	slog.Info("kafka: all consumer groups started")
}
