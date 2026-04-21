package fraud

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// Consumer processes fraud-related Kafka events and scores them in real-time.
type Consumer struct {
	engine *Engine
}

// NewConsumer creates a fraud event consumer.
func NewConsumer(engine *Engine) *Consumer {
	return &Consumer{engine: engine}
}

// HandleEvent processes a single Kafka event through the fraud engine.
func (c *Consumer) HandleEvent(ctx context.Context, msg kafka.Event) error {
	ctx, span := tracing.StartSpan(ctx, "fraud.Consumer.HandleEvent",
		attribute.String("event_type", string(msg.Type)),
		attribute.String("aggregate_id", msg.AggregateID),
	)
	defer span.End()

	req, err := extractScoreRequest(msg)
	if err != nil || req == nil {
		slog.Debug("fraud: skipping non-financial event", "event_type", msg.Type)
		return nil // not all events need fraud scoring
	}

	req.TraceID = tracing.TraceIDFromContext(ctx)

	result := c.engine.Score(ctx, *req)

	// Publish fraud.score event for downstream consumers
	if result.Decision == DecisionBlock || result.Decision == DecisionChallenge {
		events.Publish(events.Event{
			Type: events.EventFraudChecked,
			Payload: map[string]interface{}{
				"user_id":    req.UserID,
				"event_type": req.EventType,
				"decision":   string(result.Decision),
				"risk_score": result.RiskScore,
				"risk_level": result.RiskLevel,
				"signals":    result.Signals,
				"request_id": req.RequestID,
				"trace_id":   req.TraceID,
			},
		})
		slog.Warn("fraud: transaction flagged",
			"decision", string(result.Decision),
			"user_id", req.UserID,
			"risk_score", result.RiskScore,
			"event_type", req.EventType,
		)
	}

	return nil
}

// extractScoreRequest maps a Kafka event to a fraud ScoreRequest.
// Only financial events are mapped; others return nil (skipped).
func extractScoreRequest(msg kafka.Event) (*ScoreRequest, error) {
	data, err := json.Marshal(msg.Data)
	if err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	userID, _ := payload["user_id"].(string)
	amount, _ := payload["amount"].(float64)
	currency, _ := payload["currency"].(string)
	ip, _ := payload["ip"].(string)
	country, _ := payload["country"].(string)
	deviceID, _ := payload["device_id"].(string)
	requestID, _ := payload["request_id"].(string)
	if requestID == "" {
		requestID = msg.EventID
	}

	// Map event type to fraud event type
	eventType := mapEventType(msg.Type)
	if eventType == "" {
		return nil, nil // not a scorable event
	}

	return &ScoreRequest{
		UserID:    userID,
		Amount:    amount,
		Currency:  currency,
		IP:        ip,
		Country:   country,
		DeviceID:  deviceID,
		EventType: eventType,
		RequestID: requestID,
	}, nil
}

func mapEventType(eventType string) string {
	switch eventType {
	case string(events.EventOrderCreated):
		return "order.create"
	case string(events.EventWalletDeposited), string(events.EventWalletDebited):
		return "wallet.withdraw"
	case string(events.EventEscrowCreated):
		return "escrow.create"
	case string(events.EventEscrowReleased):
		return "escrow.release"
	case string(events.EventPaymentCompleted):
		return "payment.webhook"
	default:
		return ""
	}
}
