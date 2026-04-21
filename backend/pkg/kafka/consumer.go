package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	kafkago "github.com/segmentio/kafka-go"
)

const (
	// MaxHandlerRetries is the number of times a handler can fail on the same
	// message before we skip it and route to the DLQ. Prevents poison pills
	// from blocking the consumer group indefinitely.
	MaxHandlerRetries = 3

	// retryTrackerKey is the Redis key prefix for tracking per-message retry counts.
	retryTrackerKey = "kafka_retry:"
)

// HandlerFunc processes a decoded Kafka event.
type HandlerFunc func(ctx context.Context, event Event) error

// Consumer wraps kafka-go Reader with automatic reconnect, dedup, poison pill
// handling, and consumer lag reporting.
type Consumer struct {
	reader  *kafkago.Reader
	topic   string
	groupID string
	enabled bool
	dedup   *DedupStore
}

// NewConsumer creates a consumer for the given topic + consumer group.
// Returns a no-op consumer when KAFKA_BROKERS is empty.
func NewConsumer(topic, groupID string) *Consumer {
	brokerEnv := os.Getenv("KAFKA_BROKERS")
	if brokerEnv == "" {
		return &Consumer{enabled: false, topic: topic, groupID: groupID}
	}
	brokers := strings.Split(brokerEnv, ",")
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10 MB
		CommitInterval: 0,    // manual commit
	})
	return &Consumer{reader: r, topic: topic, groupID: groupID, enabled: true}
}

// Run blocks and calls handler for every message until ctx is cancelled.
// Messages are committed only after the handler returns nil.
// Poison pill protection: after MaxHandlerRetries failures on the same message,
// the message is committed (skipped) and routed to the DLQ topic.
func (c *Consumer) Run(ctx context.Context, handler HandlerFunc) {
	if !c.enabled {
		slog.Info("kafka: consumer is no-op (KAFKA_BROKERS not set)", "topic", c.topic)
		<-ctx.Done()
		return
	}

	slog.Info("kafka: consumer started", "topic", c.topic, "group", c.groupID)

	// Start lag reporter goroutine
	go c.reportLag(ctx)

	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("kafka: consumer draining — context cancelled, committing offsets", "topic", c.topic)
				return // shutdown — all processed messages already committed
			}
			slog.Warn("kafka: fetch error", "topic", c.topic, "error", err)
			continue
		}

		var event Event
		if err := json.Unmarshal(m.Value, &event); err != nil {
			slog.Warn("kafka: unmarshal error — routing to DLQ", "topic", c.topic, "error", err)
			c.routeToDLQ(ctx, m, event, "unmarshal_error")
			_ = c.reader.CommitMessages(ctx, m) // skip malformed messages
			continue
		}

		// Schema version validation — reject unknown versions.
		if event.Version < 1 {
			slog.Warn("kafka: invalid event version — skipping",
				"topic", c.topic, "event_id", event.EventID, "version", event.Version)
			_ = c.reader.CommitMessages(ctx, m)
			continue
		}
		if event.Version > 1 {
			slog.Warn("kafka: future event version — handler may not support all fields",
				"topic", c.topic, "event_id", event.EventID, "version", event.Version)
		}

		// ── Cross-region dedup ──────────────────────────────────────────────────
		if c.dedup != nil && event.IdempotencyKey != "" {
			isDup, err := c.dedup.CheckAndMark(ctx, event.IdempotencyKey)
			if err != nil {
				slog.Warn("kafka: dedup check error", "error", err)
			} else if isDup {
				_ = c.reader.CommitMessages(ctx, m) // skip duplicate, commit offset
				continue
			}
		}

		// ── Inject region into context for downstream ──────────────────────────
		if event.Region != "" {
			ctx = WithRegion(ctx, event.Region)
		}

		if err := handler(ctx, event); err != nil {
			// ── Poison pill protection ────────────────────────────────────────
			retryCount := c.incrementRetryCount(event.EventID)
			if retryCount >= MaxHandlerRetries {
				slog.Error("kafka: poison pill — max retries exceeded, routing to DLQ",
					"topic", c.topic, "event_type", event.Type, "event_id", event.EventID,
					"retries", retryCount, "error", err)
				c.routeToDLQ(ctx, m, event, fmt.Sprintf("handler_failed_%d_times: %s", retryCount, err.Error()))
				_ = c.reader.CommitMessages(ctx, m) // skip poison pill, commit offset
				c.clearRetryCount(event.EventID)
				continue
			}
			slog.Warn("kafka: handler error — message not committed (will re-deliver)",
				"topic", c.topic, "event_type", event.Type, "event_id", event.EventID,
				"attempt", retryCount, "error", err)
			continue // will re-deliver
		}

		// Handler succeeded — clear retry counter and commit
		c.clearRetryCount(event.EventID)
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			slog.Warn("kafka: commit error", "topic", c.topic, "error", err)
		}
	}
}

// ── Retry counter (Redis-backed, 1h TTL) ──────────────────────────────────

func (c *Consumer) incrementRetryCount(eventID string) int {
	if redisClient == nil {
		return 1 // no Redis → assume first attempt
	}
	key := retryTrackerKey + c.groupID + ":" + eventID
	val, err := redisClient.Incr(context.Background(), key).Result()
	if err != nil {
		return 1
	}
	// Set 1h TTL on first increment
	if val == 1 {
		redisClient.Expire(context.Background(), key, time.Hour)
	}
	return int(val)
}

func (c *Consumer) clearRetryCount(eventID string) {
	if redisClient == nil {
		return
	}
	key := retryTrackerKey + c.groupID + ":" + eventID
	redisClient.Del(context.Background(), key)
}

// ── DLQ routing ───────────────────────────────────────────────────────────

func (c *Consumer) routeToDLQ(ctx context.Context, m kafkago.Message, event Event, reason string) {
	dlqTopic := c.topic + ".dlq"
	dlqMsg := kafkago.Message{
		Topic: dlqTopic,
		Key:   m.Key,
		Value: m.Value,
		Headers: append(m.Headers, []kafkago.Header{
			{Key: "dlq_reason", Value: []byte(reason)},
			{Key: "dlq_original_topic", Value: []byte(c.topic)},
			{Key: "dlq_consumer_group", Value: []byte(c.groupID)},
			{Key: "dlq_routed_at", Value: []byte(time.Now().UTC().Format(time.RFC3339))},
		}...),
	}

	writer := Global().writerFor(dlqTopic, Global().brokers)
	dlqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := writer.WriteMessages(dlqCtx, dlqMsg); err != nil {
		slog.Error("kafka: DLQ publish failed — event may be lost!",
			"original_topic", c.topic, "dlq_topic", dlqTopic, "error", err)
	} else {
		slog.Info("kafka: event routed to DLQ", "original_topic", c.topic, "dlq_topic", dlqTopic, "reason", reason)
	}
	metrics.IncKafkaFailed(c.topic)
}

// ── Consumer lag reporting ─────────────────────────────────────────────────

// reportLag periodically reads consumer lag and exposes it as a Prometheus gauge.
func (c *Consumer) reportLag(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if c.reader == nil {
				continue
			}
			lag, err := c.reader.ReadLag(ctx)
			if err != nil {
				// Lag read can fail if topic/partition metadata unavailable
				continue
			}
			metrics.SetKafkaConsumerLag(c.topic, c.groupID, float64(lag))
		}
	}
}

// Close drains and closes the reader.
func (c *Consumer) Close() error {
	if !c.enabled || c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

// SetDedupStore attaches a dedup store for cross-region duplicate filtering.
func (c *Consumer) SetDedupStore(d *DedupStore) {
	c.dedup = d
}
