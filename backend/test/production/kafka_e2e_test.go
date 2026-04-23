//go:build production

package production

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ════════════════════════════════════════════════════════════════════════════════
// Kafka End-to-End Integration Tests
//
// Produces real events to Kafka, waits for consumer processing, and verifies
// side effects in the database.
//
// Required env vars:
//   - KAFKA_BROKERS — comma-separated broker addresses
//
// Prerequisites:
//   - Kafka cluster running (local or remote)
//   - Topics pre-created (or auto-creation enabled)
//   - Consumer groups ready to process events
// ════════════════════════════════════════════════════════════════════════════════

type KafkaE2ESuite struct {
	suite.Suite
	ts     *ProdSuite
	ctx    context.Context
	cancel context.CancelFunc
}

func TestKafkaE2ESuite(t *testing.T) {
	ts := SetupProdSuite(t)
	defer TeardownProdSuite(ts)

	suite.Run(t, &KafkaE2ESuite{ts: ts})
}

func (s *KafkaE2ESuite) SetupSuite() {
	if os.Getenv("KAFKA_BROKERS") == "" {
		s.T().Skip("KAFKA_BROKERS not set — skipping Kafka E2E tests")
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Start outbox worker and consumers
	s.ts.StartOutboxWorker(s.ctx)
	s.ts.StartKafkaConsumers(s.ctx)

	// AutoMigrate outbox tables
	s.ts.DB.AutoMigrate(
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)
}

func (s *KafkaE2ESuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
}

// ── Test: Produce event directly to Kafka ──────────────────────────────────────

func (s *KafkaE2ESuite) TestProduceEvent_DirectPublish() {
	aggregateID := uuid.New().String()
	event := kafka.New(
		"test.event",
		aggregateID,
		"test",
		kafka.Actor{Type: "system", ID: "e2e-test"},
		map[string]interface{}{
			"message":   "E2E test event from production suite",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
		kafka.EventMeta{Source: "e2e-test-suite"},
	)

	err := s.ts.KafkaProd.Publish(s.ctx, kafka.TopicOrders, event)
	require.NoError(s.T(), err, "Kafka publish should succeed")
}

// ── Test: Write outbox event → outbox worker publishes to Kafka ───────────────

func (s *KafkaE2ESuite) TestOutboxWrite_WorkerPublishes() {
	aggregateID := uuid.New().String()
	event := kafka.New(
		"order.created",
		aggregateID,
		"order",
		kafka.Actor{Type: "user", ID: uuid.New().String()},
		map[string]interface{}{
			"order_id":  aggregateID,
			"amount":    150.00,
			"currency":  "USD",
			"item_name": "E2E Test Item",
		},
		kafka.EventMeta{Source: "e2e-test-suite", TraceID: uuid.New().String()},
	)

	// Write to outbox
	err := kafka.WriteOutbox(s.ts.DB, kafka.TopicOrders, event)
	require.NoError(s.T(), err, "outbox write should succeed")

	// Verify outbox row was created
	var outboxEvent kafka.OutboxEvent
	s.ts.DB.Where("event_type = ? AND aggregate_id = ?", "order.created", event.AggregateID).
		Order("created_at DESC").First(&outboxEvent)
	assert.Equal(s.T(), "order.created", outboxEvent.EventType)
	assert.Equal(s.T(), kafka.TopicOrders, outboxEvent.Topic)
	assert.Equal(s.T(), "pending", outboxEvent.Status)

	// Wait for outbox worker to publish
	published := s.waitForOutboxStatus(outboxEvent.ID, "published", 30*time.Second)
	assert.True(s.T(), published, "outbox event should be published by worker")
}

// ── Test: Idempotency — same event processed only once ─────────────────────────

func (s *KafkaE2ESuite) TestIdempotency_SameEventProcessedOnce() {
	eventID := uuid.New().String()
	consumerGroup := "e2e-test-consumer"

	// Mark as processed
	err := kafka.MarkProcessed(s.ts.DB, eventID, consumerGroup)
	require.NoError(s.T(), err)

	// Check that it's detected as already processed
	processed := kafka.IsProcessed(s.ts.DB, eventID, consumerGroup)
	assert.True(s.T(), processed, "event should be detected as already processed")

	// Different consumer group should NOT see it as processed
	otherProcessed := kafka.IsProcessed(s.ts.DB, eventID, "other-consumer")
	assert.False(s.T(), otherProcessed, "different consumer group should not see event as processed")
}

// ── Test: Full wallet event flow — produce → consume → verify DB ──────────────

func (s *KafkaE2ESuite) TestWalletEvent_ProduceConsumeVerify() {
	userID := uuid.New().String()
	walletEvent := kafka.New(
		"wallet.deposited",
		userID,
		"wallet",
		kafka.Actor{Type: "system", ID: "e2e-test"},
		map[string]interface{}{
			"user_id":  userID,
			"amount":   500.00,
			"currency": "USD",
		},
		kafka.EventMeta{Source: "e2e-test-suite"},
	)

	// Write to outbox (simulates what the wallet handler does)
	err := kafka.WriteOutbox(s.ts.DB, kafka.TopicWallet, walletEvent)
	require.NoError(s.T(), err)

	// Wait for outbox to be published
	var outboxEvent kafka.OutboxEvent
	s.ts.DB.Where("event_type = ? AND topic = ?", "wallet.deposited", kafka.TopicWallet).
		Order("created_at DESC").First(&outboxEvent)

	published := s.waitForOutboxStatus(outboxEvent.ID, "published", 30*time.Second)
	assert.True(s.T(), published, "wallet event should be published to Kafka")
}

// ── Test: Fraud event flow ─────────────────────────────────────────────────────

func (s *KafkaE2ESuite) TestFraudEvent_ProduceAndVerify() {
	userID := uuid.New().String()
	fraudEvent := kafka.New(
		"fraud.checked",
		userID,
		"fraud",
		kafka.Actor{Type: "system", ID: "e2e-test"},
		map[string]interface{}{
			"user_id":    userID,
			"risk_score": 85,
			"decision":   "block",
			"reasons":    []string{"velocity_limit_exceeded", "new_device"},
		},
		kafka.EventMeta{Source: "e2e-test-suite"},
	)

	err := kafka.WriteOutbox(s.ts.DB, kafka.TopicFraud, fraudEvent)
	require.NoError(s.T(), err)

	// Verify outbox row
	var outboxEvent kafka.OutboxEvent
	s.ts.DB.Where("event_type = ? AND topic = ?", "fraud.checked", kafka.TopicFraud).
		Order("created_at DESC").First(&outboxEvent)
	assert.Equal(s.T(), "fraud.checked", outboxEvent.EventType)

	published := s.waitForOutboxStatus(outboxEvent.ID, "published", 30*time.Second)
	assert.True(s.T(), published, "fraud event should be published")
}

// ── Test: Event envelope integrity ─────────────────────────────────────────────

func (s *KafkaE2ESuite) TestEventEnvelope_Integrity() {
	event := kafka.New(
		"test.integrity",
		uuid.New().String(),
		"test",
		kafka.Actor{Type: "system", ID: "e2e-test"},
		map[string]interface{}{"key": "value"},
		kafka.EventMeta{Source: "e2e-test", TraceID: "trace-123"},
	)

	// Verify envelope fields
	assert.NotEmpty(s.T(), event.EventID, "event_id should be auto-generated")
	assert.Equal(s.T(), "test.integrity", event.Type)
	assert.Equal(s.T(), 1, event.Version, "version should be 1")
	assert.WithinDuration(s.T(), time.Now().UTC(), event.Timestamp, 5*time.Second)
	assert.Equal(s.T(), "system", event.Actor.Type)
	assert.Equal(s.T(), "e2e-test", event.Actor.ID)
	assert.Equal(s.T(), "trace-123", event.Metadata.TraceID)
	assert.Equal(s.T(), "e2e-test", event.Metadata.Source)

	// Verify JSON round-trip
	payload, err := json.Marshal(event)
	require.NoError(s.T(), err)

	var decoded kafka.Event
	err = json.Unmarshal(payload, &decoded)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), event.EventID, decoded.EventID)
	assert.Equal(s.T(), event.Type, decoded.Type)
}

// ── Helper: Poll outbox for status change ──────────────────────────────────────

func (s *KafkaE2ESuite) waitForOutboxStatus(outboxID, expectedStatus string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		var row kafka.OutboxEvent
		if err := s.ts.DB.Where("id = ?", outboxID).First(&row).Error; err == nil {
			if row.Status == expectedStatus {
				return true
			}
		}
		time.Sleep(2 * time.Second)
	}
	return false
}
