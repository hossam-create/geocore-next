package events

import (
	"log/slog"
	"sync"
)

// EventType identifies what happened in the system.
type EventType string

const (
	EventOrderCreated      EventType = "order.created"
	EventOrderCancelled    EventType = "order.cancelled"
	EventOrderShipped      EventType = "order.shipped"
	EventOrderDelivered    EventType = "order.delivered"
	EventPaymentCompleted  EventType = "payment.completed"
	EventEscrowCreated     EventType = "escrow.created"
	EventEscrowReleased    EventType = "escrow.released"
	EventWalletDeposited   EventType = "wallet.deposited"
	EventWalletDebited     EventType = "wallet.debited"
	EventUserRegistered    EventType = "user.registered"
	EventReviewPosted      EventType = "review.posted"
	EventReferralCompleted EventType = "referral.completed"
	EventListingCreated    EventType = "listing.created"
	EventDisputeOpened     EventType = "dispute.opened"
	EventFraudChecked      EventType = "fraud.checked"
	EventModerationBlocked EventType = "moderation.blocked"
	EventShippingCreated   EventType = "shipping.created"
)

// Event is a domain event published to the bus.
type Event struct {
	Type      EventType              `json:"type"`
	Payload   map[string]interface{} `json:"payload"`
	RequestID string                 `json:"request_id,omitempty"`
}

// Handler is a function that processes a published event.
type Handler func(event Event)

// Bus is a lightweight synchronous/async in-process event bus.
// Handlers are called in separate goroutines so they never block the publisher.
type Bus struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
}

// New creates a new event bus.
func New() *Bus {
	return &Bus{handlers: make(map[EventType][]Handler)}
}

// Subscribe registers a handler for the given event type.
func (b *Bus) Subscribe(eventType EventType, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], h)
}

// Publish dispatches the event to all registered handlers asynchronously.
func (b *Bus) Publish(event Event) {
	b.mu.RLock()
	handlers := b.handlers[event.Type]
	b.mu.RUnlock()

	for _, h := range handlers {
		h := h // capture
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("event bus: handler panic",
						"event_type", event.Type,
						"panic", r,
					)
				}
			}()
			h(event)
		}()
	}
}

// ════════════════════════════════════════════════════════════════════════════
// Global singleton
// ════════════════════════════════════════════════════════════════════════════

var (
	global   *Bus
	globalMu sync.Mutex
)

// Global returns the singleton event bus, creating it if needed.
func Global() *Bus {
	globalMu.Lock()
	defer globalMu.Unlock()
	if global == nil {
		global = New()
	}
	return global
}

// Publish publishes to the global bus (convenience wrapper).
func Publish(event Event) {
	Global().Publish(event)
}

// Subscribe registers a handler on the global bus (convenience wrapper).
func Subscribe(eventType EventType, h Handler) {
	Global().Subscribe(eventType, h)
}
