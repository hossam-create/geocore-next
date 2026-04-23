package email

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/kafka"
)

const (
	asyncQueueSize   = 2000 // buffered channel capacity
	asyncWorkerCount = 4    // parallel sender goroutines
	asyncSendTimeout = 30 * time.Second
)

var (
	asyncCh   chan *Message
	asyncOnce sync.Once
)

// StartWorker initialises and starts the background email worker goroutines.
// Safe to call multiple times — subsequent calls are no-ops.
// Call this once from main() after SetDefault().
func StartWorker() {
	asyncOnce.Do(func() {
		asyncCh = make(chan *Message, asyncQueueSize)
		for i := 0; i < asyncWorkerCount; i++ {
			go runWorker(i)
		}
		slog.Info("email: async worker started", "workers", asyncWorkerCount, "queue_size", asyncQueueSize)
	})
}

func runWorker(id int) {
	for msg := range asyncCh {
		ctx, cancel := context.WithTimeout(context.Background(), asyncSendTimeout)
		if err := Default().Send(ctx, msg); err != nil {
			slog.Error("email: worker delivery failed",
				"worker_id", id,
				"to", msg.To,
				"template", msg.TemplateName,
				"error", err,
			)
		}
		cancel()
	}
}

// EnqueueEmail pushes a Message onto the internal async channel.
// Non-blocking — if the channel is full it falls back to synchronous delivery
// so no email is silently dropped.
//
// Also publishes a lightweight Kafka event for audit / replay (fire-and-forget).
func EnqueueEmail(ctx context.Context, msg *Message) error {
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// ── Kafka audit signal (no-op when Kafka is not configured) ───────────────
	go func() {
		evt := kafka.New(
			"email.queued",
			msg.UserID,
			"email",
			kafka.Actor{Type: "system", ID: "email-service"},
			map[string]any{
				"to":       msg.To,
				"template": msg.TemplateName,
				"subject":  msg.Subject,
			},
			kafka.EventMeta{Source: "email-service"},
		)
		kafka.PublishAsync(kafka.TopicNotifications, evt)
	}()

	// ── Async channel dispatch ─────────────────────────────────────────────────
	if asyncCh == nil {
		// Worker not started — fall through to sync
		slog.Warn("email: worker not started, sending sync", "to", msg.To)
		return Default().Send(ctx, msg)
	}

	select {
	case asyncCh <- msg:
		slog.Debug("email: enqueued", "to", msg.To, "template", msg.TemplateName)
		return nil
	default:
		// Queue full — send synchronously to avoid message loss
		slog.Warn("email: queue full, sending sync", "to", msg.To)
		return Default().Send(ctx, msg)
	}
}

// ProcessJobPayload decodes a jobs.Job.Payload and delivers the email.
// Called by pkg/jobs/handlers.go — keeps the jobs package decoupled from
// the internals of the email service.
func ProcessJobPayload(ctx context.Context, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("email: encode payload: %w", err)
	}
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("email: decode message: %w", err)
	}
	return Default().Send(ctx, &msg)
}
