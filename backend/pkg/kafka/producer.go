package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/chaos"
	kafkago "github.com/segmentio/kafka-go"
)

const maxPublishAttempts = 5

// Producer is a thread-safe Kafka writer pool.
// When KAFKA_BROKERS is empty it silently becomes a no-op so existing flows
// are never disrupted during the gradual rollout.
type Producer struct {
	writers  map[string]*kafkago.Writer
	mu       sync.RWMutex
	brokers  []string
	failover []string // secondary region brokers
	enabled  bool
}

var (
	global   *Producer
	globalMu sync.Mutex
)

// Init creates the singleton producer.  Safe to call multiple times.
func Init() *Producer {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global != nil {
		return global
	}
	global = newProducer()
	return global
}

// Global returns the singleton (may be a no-op if Kafka is not configured).
func Global() *Producer {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		global = newProducer()
	}
	return global
}

func newProducer() *Producer {
	brokerEnv := os.Getenv("KAFKA_BROKERS")
	if brokerEnv == "" {
		slog.Info("kafka: KAFKA_BROKERS not set — running in no-op mode")
		return &Producer{enabled: false, writers: make(map[string]*kafkago.Writer)}
	}
	brokers := strings.Split(brokerEnv, ",")

	// Optional failover brokers (secondary region)
	var failover []string
	if fb := os.Getenv("KAFKA_FAILOVER_BROKERS"); fb != "" {
		failover = strings.Split(fb, ",")
		slog.Info("kafka: failover brokers configured", "brokers", failover)
	}

	slog.Info("kafka: producer initialised", "brokers", brokers)
	return &Producer{
		brokers:  brokers,
		failover: failover,
		writers:  make(map[string]*kafkago.Writer),
		enabled:  true,
	}
}

// Publish marshals event, enriches with region/idempotency from context,
// and writes to topic with exponential-backoff retry + failover.
// Returns nil immediately if Kafka is not configured (no-op mode).
func (p *Producer) Publish(ctx context.Context, topic string, event Event) error {
	if !p.enabled {
		return nil
	}

	// Chaos hook: simulate Kafka down
	if chaos.IsKafkaDown() {
		return errors.New("kafka: forced down (chaos)")
	}

	// Enrich event with region + idempotency from context
	EnrichEvent(ctx, &event)

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafkago.Message{
		Key:   []byte(event.AggregateID), // partition key = aggregate_id for ordering
		Value: payload,
		Headers: []kafkago.Header{
			{Key: "region", Value: []byte(event.Region)},
			{Key: "idempotency_key", Value: []byte(event.IdempotencyKey)},
		},
	}

	// ── Try primary brokers ──────────────────────────────────────────────────
	w := p.writerFor(topic, p.brokers)
	var lastErr error
	for attempt := 0; attempt < maxPublishAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
		err = w.WriteMessages(ctx, msg)
		if err == nil {
			return nil
		}
		lastErr = err
		slog.Warn("kafka: publish attempt failed",
			"topic", topic, "attempt", attempt+1, "error", err)
	}

	// ── Failover to secondary brokers ────────────────────────────────────────
	if len(p.failover) > 0 {
		slog.Warn("kafka: primary failed — failing over to secondary", "topic", topic, "primary_err", lastErr)
		fw := p.writerFor(topic, p.failover)
		if err := fw.WriteMessages(ctx, msg); err != nil {
			slog.Error("kafka: failover also failed", "topic", topic, "error", err)
			return err
		}
		slog.Info("kafka: failover publish succeeded", "topic", topic)
		return nil
	}

	return lastErr
}

// PublishAsync fires-and-forgets in a goroutine. Never blocks the caller.
// Prefer WriteOutbox + OutboxWorker for guaranteed delivery.
func (p *Producer) PublishAsync(topic string, event Event) {
	if !p.enabled {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := p.Publish(ctx, topic, event); err != nil {
			slog.Warn("kafka: async publish failed", "topic", topic, "error", err)
		}
	}()
}

// Close flushes and closes all writers.
func (p *Producer) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for topic, w := range p.writers {
		if err := w.Close(); err != nil {
			slog.Warn("kafka: writer close error", "topic", topic, "error", err)
		}
	}
}

func (p *Producer) writerFor(topic string, brokers []string) *kafkago.Writer {
	key := topic
	// Separate cache key for failover brokers
	if len(brokers) > 0 && len(p.failover) > 0 && brokers[0] == p.failover[0] {
		key = topic + ":failover"
	}

	p.mu.RLock()
	if w, ok := p.writers[key]; ok {
		p.mu.RUnlock()
		return w
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	if w, ok := p.writers[key]; ok {
		return w
	}
	w := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafkago.LeastBytes{},
		Async:        false,
		RequiredAcks: kafkago.RequireOne,
		MaxAttempts:  1, // retries handled in Publish()
		BatchTimeout: 10 * time.Millisecond,
	}
	p.writers[key] = w
	return w
}

// Publish is a package-level convenience wrapper.
func Publish(ctx context.Context, topic string, event Event) error {
	return Global().Publish(ctx, topic, event)
}

// PublishAsync is a package-level convenience wrapper.
func PublishAsync(topic string, event Event) {
	Global().PublishAsync(topic, event)
}
