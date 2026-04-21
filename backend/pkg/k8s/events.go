package k8s

import (
	"log/slog"
	"time"
)

// EventType defines the type of Kubernetes event.
type EventType string

const (
	EventNormal  EventType = "Normal"
	EventWarning EventType = "Warning"
)

// Event represents a Kubernetes event recorded by the operator.
type Event struct {
	Kind     string    `json:"kind"`
	Name     string    `json:"name"`
	Namespace string   `json:"namespace"`
	Action   string    `json:"action"`
	Reason   string    `json:"reason"`
	Message  string    `json:"message"`
	Type     EventType `json:"type"`
	Time     time.Time `json:"time"`
}

// EventRecorder records Kubernetes events for operator actions.
type EventRecorder struct {
	events []Event
}

// NewEventRecorder creates an event recorder.
func NewEventRecorder() *EventRecorder {
	return &EventRecorder{events: make([]Event, 0, 100)}
}

// RecordEvent logs a Kubernetes event.
func (er *EventRecorder) RecordEvent(kind, name, namespace, action, reason, message string, eventType EventType) {
	event := Event{
		Kind:      kind,
		Name:      name,
		Namespace: namespace,
		Action:    action,
		Reason:    reason,
		Message:   message,
		Type:      eventType,
		Time:      time.Now().UTC(),
	}
	er.events = append(er.events, event)

	if len(er.events) > 100 {
		er.events = er.events[1:]
	}

	switch eventType {
	case EventNormal:
		slog.Info("k8s-event: "+message, "kind", kind, "name", name, "action", action)
	case EventWarning:
		slog.Warn("k8s-event: "+message, "kind", kind, "name", name, "action", action)
	}
}

// RecentEvents returns the last N events.
func (er *EventRecorder) RecentEvents(n int) []Event {
	if n > len(er.events) {
		n = len(er.events)
	}
	result := make([]Event, n)
	copy(result, er.events[len(er.events)-n:])
	return result
}
