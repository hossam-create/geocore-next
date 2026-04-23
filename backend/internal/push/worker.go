package push

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════
// Priority-based async worker with separate queues per priority level
// ════════════════════════════════════════════════════════════════════════════

const (
	highQueueSize   = 500
	mediumQueueSize = 1000
	lowQueueSize    = 2000
	highWorkers     = 4
	mediumWorkers   = 2
	lowWorkers      = 1
)

type pushWorkerSingleton struct {
	highCh  chan *PushMessage
	medCh   chan *PushMessage
	lowCh   chan *PushMessage
	service *PushService
	wg      sync.WaitGroup
	stopCh  chan struct{}
}

var pushWorker *pushWorkerSingleton

// StartWorker initialises the priority-based async push worker.
// Must be called after SetDefault.
func StartWorker() {
	if defaultService == nil {
		slog.Warn("push: StartWorker called but no service initialised")
		return
	}
	if pushWorker != nil {
		return // already started
	}

	pushWorker = &pushWorkerSingleton{
		highCh:  make(chan *PushMessage, highQueueSize),
		medCh:   make(chan *PushMessage, mediumQueueSize),
		lowCh:   make(chan *PushMessage, lowQueueSize),
		service: defaultService,
		stopCh:  make(chan struct{}),
	}

	// Start HIGH priority workers
	for i := 0; i < highWorkers; i++ {
		pushWorker.wg.Add(1)
		go pushWorker.work(pushWorker.highCh, "high", i)
	}

	// Start MEDIUM priority workers
	for i := 0; i < mediumWorkers; i++ {
		pushWorker.wg.Add(1)
		go pushWorker.work(pushWorker.medCh, "medium", i)
	}

	// Start LOW priority workers
	for i := 0; i < lowWorkers; i++ {
		pushWorker.wg.Add(1)
		go pushWorker.work(pushWorker.lowCh, "low", i)
	}

	slog.Info("push: async worker started",
		"high_workers", highWorkers,
		"medium_workers", mediumWorkers,
		"low_workers", lowWorkers,
	)
}

// StopWorker gracefully shuts down all workers.
func StopWorker() {
	if pushWorker == nil {
		return
	}
	close(pushWorker.stopCh)
	pushWorker.wg.Wait()
	pushWorker = nil
	slog.Info("push: async worker stopped")
}

// enqueue routes a PushMessage to the appropriate priority queue.
// Falls back to synchronous delivery if the queue is full.
func (w *pushWorkerSingleton) enqueue(msg *PushMessage) {
	var ch chan *PushMessage
	switch msg.Priority {
	case PriorityHigh:
		ch = w.highCh
	case PriorityLow:
		ch = w.lowCh
	default:
		ch = w.medCh
	}

	select {
	case ch <- msg:
		// enqueued successfully
	default:
		// Queue full — fall back to synchronous delivery to avoid message loss
		slog.Warn("push: queue full, falling back to sync",
			"priority", msg.Priority, "user_id", msg.UserID)
		if err := w.service.deliver(context.Background(), msg); err != nil {
			slog.Error("push: sync fallback failed", "user_id", msg.UserID, "error", err)
		}
	}
}

// work is the main loop for a single worker goroutine.
func (w *pushWorkerSingleton) work(ch chan *PushMessage, priority string, id int) {
	defer w.wg.Done()
	for {
		select {
		case msg := <-ch:
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			if err := w.service.deliver(ctx, msg); err != nil {
				slog.Error("push: worker delivery failed",
					"priority", priority, "worker_id", id,
					"user_id", msg.UserID, "error", err)
			}
			cancel()
		case <-w.stopCh:
			return
		}
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Kafka consumer for push events from notifications.events topic
// ════════════════════════════════════════════════════════════════════════════

// ProcessKafkaPushEvent handles a push event consumed from Kafka.
// This allows other services to trigger pushes by publishing to Kafka.
func ProcessKafkaPushEvent(data map[string]any) error {
	if defaultService == nil {
		return nil
	}

	userIDStr, _ := data["user_id"].(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil // invalid user ID, skip
	}

	notificationType, _ := data["notification_type"].(string)
	title, _ := data["title"].(string)
	body, _ := data["body"].(string)

	dataMap := map[string]string{}
	if rawData, ok := data["data"].(map[string]any); ok {
		for k, v := range rawData {
			dataMap[k] = fmt.Sprintf("%v", v)
		}
	}

	msg := &PushMessage{
		UserID:           userID,
		NotificationType: notificationType,
		Priority:         ResolvePriority(notificationType),
		Title:            title,
		Body:             body,
		Data:             dataMap,
	}

	return defaultService.Send(context.Background(), msg)
}
