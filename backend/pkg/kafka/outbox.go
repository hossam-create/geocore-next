package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	outboxStatusPending    = "pending"
	outboxStatusProcessing = "processing"
	outboxStatusPublished  = "published"
	outboxStatusFailed     = "failed"

	maxOutboxAttempts = 5
	outboxBatchSize   = 50

	// Redis key prefix for idempotency dedup. TTL = 24h.
	processedKeyPrefix = "processed_events:"
	processedTTL       = 24 * time.Hour
)

// ── Models ────────────────────────────────────────────────────────────────────

// OutboxEvent is written to DB BEFORE Kafka publish.
// A background OutboxWorker picks it up and delivers to Kafka.
// This guarantees at-least-once delivery even if the process crashes.
type OutboxEvent struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	EventType   string `gorm:"not null;index"`
	Topic       string `gorm:"not null"`
	Payload     string `gorm:"type:text;not null"`
	Status      string `gorm:"default:'pending';index"`
	Attempts    int    `gorm:"default:0"`
	LastError   string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

// ProcessedEvent prevents duplicate processing (idempotent consumers).
// Before handling any event, consumers check this table.
type ProcessedEvent struct {
	ID            string    `gorm:"type:uuid;primaryKey"`
	EventID       string    `gorm:"not null;uniqueIndex:uidx_processed_event"`
	ConsumerGroup string    `gorm:"not null;uniqueIndex:uidx_processed_event"`
	ProcessedAt   time.Time `gorm:"not null;autoCreateTime"`
}

// ── WriteOutbox ───────────────────────────────────────────────────────────────

// WriteOutbox persists an event to the outbox table.
// Call this inside your business DB transaction for true atomicity.
// The OutboxWorker will pick it up and publish to Kafka asynchronously.
func WriteOutbox(db *gorm.DB, topic string, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	row := &OutboxEvent{
		ID:        uuid.NewString(),
		EventType: event.Type,
		Topic:     topic,
		Payload:   string(payload),
		Status:    outboxStatusPending,
	}
	return db.Create(row).Error
}

// ── Idempotency helpers (Redis-first, DB fallback) ────────────────────────────

var redisClient *redis.Client

// SetRedis sets the Redis client for idempotency dedup.
// Call this once at startup after creating the Redis client.
func SetRedis(rdb *redis.Client) {
	redisClient = rdb
}

// IsProcessed returns true if a consumer group already handled this event.
// Checks Redis first (fast, 24h TTL), falls back to DB.
// Always call this at the START of every consumer handler.
func IsProcessed(db *gorm.DB, eventID, consumerGroup string) bool {
	if redisClient != nil {
		key := fmt.Sprintf("%s%s:%s", processedKeyPrefix, consumerGroup, eventID)
		val, err := redisClient.Exists(context.Background(), key).Result()
		if err == nil && val > 0 {
			return true
		}
	}
	var count int64
	db.Model(&ProcessedEvent{}).
		Where("event_id = ? AND consumer_group = ?", eventID, consumerGroup).
		Count(&count)
	return count > 0
}

// MarkProcessed records that a consumer group handled the event.
// Writes to Redis (24h TTL) AND DB for durability.
// Call this at the END of a successful handler (inside the same tx if possible).
func MarkProcessed(db *gorm.DB, eventID, consumerGroup string) error {
	if redisClient != nil {
		key := fmt.Sprintf("%s%s:%s", processedKeyPrefix, consumerGroup, eventID)
		redisClient.Set(context.Background(), key, "1", processedTTL)
	}
	return db.Create(&ProcessedEvent{
		ID:            uuid.NewString(),
		EventID:       eventID,
		ConsumerGroup: consumerGroup,
		ProcessedAt:   time.Now(),
	}).Error
}

// ── OutboxWorker ──────────────────────────────────────────────────────────────

// OutboxWorker polls outbox_events and publishes pending rows to Kafka.
// Run exactly one worker per service instance.
type OutboxWorker struct {
	db       *gorm.DB
	producer *Producer
	interval time.Duration
}

// NewOutboxWorker creates a worker with the given poll interval.
// Recommended: 2 * time.Second for dev, 500ms for high-throughput prod.
func NewOutboxWorker(db *gorm.DB, interval time.Duration) *OutboxWorker {
	return &OutboxWorker{
		db:       db,
		producer: Global(),
		interval: interval,
	}
}

// Start launches the worker in a background goroutine.
// It stops cleanly when ctx is cancelled.
func (w *OutboxWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

func (w *OutboxWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	slog.Info("kafka: outbox worker started", "interval", w.interval)
	for {
		select {
		case <-ctx.Done():
			slog.Info("kafka: outbox worker stopped")
			return
		case <-ticker.C:
			w.recoverStuck()
			w.processBatch(ctx)
			w.updatePendingGauge()
		}
	}
}

// recoverStuck resets rows stuck in "processing" for > 5 min (crash recovery).
func (w *OutboxWorker) recoverStuck() {
	w.db.Model(&OutboxEvent{}).
		Where("status = ? AND updated_at < ?", outboxStatusProcessing, time.Now().Add(-5*time.Minute)).
		Update("status", outboxStatusPending)
}

// updatePendingGauge reports the current outbox backlog to Prometheus.
func (w *OutboxWorker) updatePendingGauge() {
	var count int64
	w.db.Model(&OutboxEvent{}).Where("status = ?", outboxStatusPending).Count(&count)
	metrics.SetKafkaOutboxPending(float64(count))
}

func (w *OutboxWorker) processBatch(ctx context.Context) {
	var rows []OutboxEvent
	if err := w.db.WithContext(ctx).
		Where("status = ? AND attempts < ?", outboxStatusPending, maxOutboxAttempts).
		Order("created_at ASC").
		Limit(outboxBatchSize).
		Find(&rows).Error; err != nil {
		slog.Warn("kafka: outbox query failed", "error", err)
		return
	}

	for _, row := range rows {
		// Atomic claim — if another worker already grabbed it, RowsAffected = 0.
		result := w.db.Model(&OutboxEvent{}).
			Where("id = ? AND status = ?", row.ID, outboxStatusPending).
			Update("status", outboxStatusProcessing)
		if result.RowsAffected == 0 {
			continue
		}
		w.publishRow(ctx, row)
	}
}

func (w *OutboxWorker) publishRow(ctx context.Context, row OutboxEvent) {
	var event Event
	if err := json.Unmarshal([]byte(row.Payload), &event); err != nil {
		w.db.Model(&OutboxEvent{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
			"status":     outboxStatusFailed,
			"last_error": err.Error(),
			"attempts":   row.Attempts + 1,
		})
		slog.Error("kafka: outbox unmarshal failed", "id", row.ID, "error", err)
		return
	}

	pubCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := w.producer.Publish(pubCtx, row.Topic, event)
	now := time.Now()

	if err != nil {
		newStatus := outboxStatusPending
		if row.Attempts+1 >= maxOutboxAttempts {
			newStatus = outboxStatusFailed
			slog.Error("kafka: outbox max attempts reached — routing to DLQ",
				"id", row.ID, "topic", row.Topic, "event_type", row.EventType)
			// Route to DLQ topic so failed events are not lost
			dlqTopic := row.Topic + ".dlq"
			dlqCtx, dlqCancel := context.WithTimeout(ctx, 5*time.Second)
			if dlqErr := w.producer.Publish(dlqCtx, dlqTopic, event); dlqErr != nil {
				slog.Error("kafka: DLQ publish also failed — event may be lost!",
					"original_topic", row.Topic, "dlq_topic", dlqTopic, "error", dlqErr)
			} else {
				slog.Info("kafka: event routed to DLQ", "original_topic", row.Topic, "dlq_topic", dlqTopic)
			}
			dlqCancel()
		}
		w.db.Model(&OutboxEvent{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
			"status":     newStatus,
			"attempts":   row.Attempts + 1,
			"last_error": err.Error(),
		})
		metrics.IncKafkaFailed(row.Topic)
		return
	}

	w.db.Model(&OutboxEvent{}).Where("id = ?", row.ID).Updates(map[string]interface{}{
		"status":       outboxStatusPublished,
		"attempts":     row.Attempts + 1,
		"last_error":   "",
		"published_at": &now,
	})
	metrics.IncKafkaPublished(row.Topic)
	slog.Debug("kafka: outbox published", "event_type", row.EventType, "topic", row.Topic)
}
