package controltower

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisChannel    = "system:events"
	eventBufSize    = 256
	maxSSESubscribers = 64
)

// SeverityLevel classifies event urgency.
type SeverityLevel string

const (
	SevInfo     SeverityLevel = "info"
	SevWarning  SeverityLevel = "warning"
	SevCritical SeverityLevel = "critical"
)

// SystemEvent is a single feed entry.
type SystemEvent struct {
	Type      string        `json:"type"`
	Message   string        `json:"message"`
	UserID    string        `json:"user_id,omitempty"`
	IP        string        `json:"ip,omitempty"`
	Severity  SeverityLevel `json:"severity"`
	Timestamp time.Time     `json:"timestamp"`
}

// EventBus broadcasts SystemEvents to SSE subscribers and Redis pub/sub.
type EventBus struct {
	mu   sync.RWMutex
	subs map[chan SystemEvent]struct{}
	rdb  *redis.Client
}

var globalBus *EventBus
var busOnce sync.Once

// InitEventBus initialises the singleton event bus. Call once at startup.
func InitEventBus(rdb *redis.Client) *EventBus {
	busOnce.Do(func() {
		globalBus = &EventBus{
			subs: make(map[chan SystemEvent]struct{}),
			rdb:  rdb,
		}
		if rdb != nil {
			go globalBus.listenRedis()
		}
	})
	return globalBus
}

// GetBus returns the singleton bus (nil-safe).
func GetBus() *EventBus { return globalBus }

// Publish sends an event to all local subscribers and Redis.
func (b *EventBus) Publish(evt SystemEvent) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	// Local fans-out.
	b.mu.RLock()
	for ch := range b.subs {
		select {
		case ch <- evt:
		default: // drop if subscriber is slow
		}
	}
	b.mu.RUnlock()

	// Redis broadcast for multi-instance support.
	if b.rdb != nil {
		data, _ := json.Marshal(evt)
		b.rdb.Publish(context.Background(), redisChannel, data)
	}
}

// Subscribe returns a channel that receives events. Call Unsubscribe when done.
func (b *EventBus) Subscribe() chan SystemEvent {
	ch := make(chan SystemEvent, eventBufSize)
	b.mu.Lock()
	if len(b.subs) < maxSSESubscribers {
		b.subs[ch] = struct{}{}
	}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel and closes it.
func (b *EventBus) Unsubscribe(ch chan SystemEvent) {
	b.mu.Lock()
	delete(b.subs, ch)
	b.mu.Unlock()
	close(ch)
}

// listenRedis forwards events from Redis pub/sub to local subscribers.
// This ensures events published by other API instances reach this one.
func (b *EventBus) listenRedis() {
	sub := b.rdb.Subscribe(context.Background(), redisChannel)
	defer sub.Close()
	ch := sub.Channel()
	for msg := range ch {
		var evt SystemEvent
		if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
			slog.Warn("controltower: bad event from Redis", "err", err)
			continue
		}
		b.mu.RLock()
		for localCh := range b.subs {
			select {
			case localCh <- evt:
			default:
			}
		}
		b.mu.RUnlock()
	}
}

// ─── Package-level publish helpers used across the system ────────────────────

// Emit publishes a system event through the global bus (no-op if bus not init).
func Emit(evtType string, sev SeverityLevel, message, userID, ip string) {
	if globalBus == nil {
		return
	}
	globalBus.Publish(SystemEvent{
		Type:      evtType,
		Message:   message,
		UserID:    userID,
		IP:        ip,
		Severity:  sev,
		Timestamp: time.Now().UTC(),
	})
}

// FormatSSE serialises an event as a Server-Sent Event payload.
func FormatSSE(evt SystemEvent) string {
	data, _ := json.Marshal(evt)
	return fmt.Sprintf("data: %s\n\n", data)
}
