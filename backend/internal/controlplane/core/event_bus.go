package core

import (
	"sync"
)

// Event represents a control plane event (proposal, decision, action, etc.).
type Event struct {
	Type    string
	Payload any
}

// EventBus provides pub/sub for control plane events.
type EventBus struct {
	mu        sync.RWMutex
	subscribers map[string][]chan Event
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan Event),
	}
}

// Subscribe registers a channel to receive events of the given type.
func (b *EventBus) Subscribe(eventType string) chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event, 100)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}

// Publish sends an event to all subscribers of the given type.
func (b *EventBus) Publish(eventType string, payload any) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	event := Event{Type: eventType, Payload: payload}
	for _, ch := range b.subscribers[eventType] {
		select {
		case ch <- event:
		default:
			// drop if subscriber is slow
		}
	}
}
