package kafka

import (
	"time"

	"github.com/google/uuid"
)

// Actor identifies who triggered the event.
type Actor struct {
	Type string `json:"type"` // "user", "system", "admin", "cron"
	ID   string `json:"id"`
}

// EventMeta carries cross-cutting observability fields.
type EventMeta struct {
	TraceID string `json:"trace_id,omitempty"`
	Source  string `json:"source,omitempty"` // e.g. "api-service", "worker-service"
}

// Event is the GLOBAL Kafka event contract. Every topic carries this envelope.
//
//	{
//	  "event_id":       "uuid",
//	  "event_type":     "order.created",
//	  "aggregate_id":   "order_id",
//	  "aggregate_type": "order",
//	  "version":        1,
//	  "timestamp":      "2026-04-13T00:00:00Z",
//	  "actor":          { "type": "user", "id": "user_id" },
//	  "data":           {...},
//	  "metadata":       { "trace_id": "...", "source": "api-service" }
//	}
type Event struct {
	EventID        string      `json:"event_id"`
	Type           string      `json:"event_type"`
	AggregateID    string      `json:"aggregate_id"`
	AggregateType  string      `json:"aggregate_type"`
	Version        int         `json:"version"`
	Timestamp      time.Time   `json:"timestamp"`
	Actor          Actor       `json:"actor"`
	Data           interface{} `json:"data"`
	Metadata       EventMeta   `json:"metadata"`
	Region         string      `json:"region,omitempty"`          // emitting region (e.g. us-east-1)
	IdempotencyKey string      `json:"idempotency_key,omitempty"` // cross-region dedup key
}

// New builds a version-1 Event with an auto-generated ID and UTC timestamp.
// source should be the emitting service, e.g. "api-service", "payment-service".
func New(eventType, aggregateID, aggregateType string, actor Actor, data interface{}, meta EventMeta) Event {
	return Event{
		EventID:       uuid.NewString(),
		Type:          eventType,
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Version:       1,
		Timestamp:     time.Now().UTC(),
		Actor:         actor,
		Data:          data,
		Metadata:      meta,
	}
}
