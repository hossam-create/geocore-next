package observability

import (
	"context"
	"fmt"
)

type correlationKey string

const (
	traceIDKey correlationKey = "trace_id"
	regionKey  correlationKey = "region"
	userIDKey  correlationKey = "user_id"
	requestKey correlationKey = "request_id"
)

// WithTraceID adds a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// WithRegion adds a region to the context.
func WithRegion(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, regionKey, region)
}

// WithUserID adds a user ID to the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestKey, requestID)
}

// Correlate returns a formatted correlation string from context values.
// Format: trace_id|region|user_id|request_id
func Correlate(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey).(string)
	region, _ := ctx.Value(regionKey).(string)
	userID, _ := ctx.Value(userIDKey).(string)
	requestID, _ := ctx.Value(requestKey).(string)
	return fmt.Sprintf("%s|%s|%s|%s", traceID, region, userID, requestID)
}

// GetTraceID extracts the trace ID from context.
func GetTraceID(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey).(string)
	return v
}

// GetCorrelationMap returns all correlation values as a map (for structured logging).
func GetCorrelationMap(ctx context.Context) map[string]string {
	return map[string]string{
		"trace_id":   strVal(ctx.Value(traceIDKey)),
		"region":     strVal(ctx.Value(regionKey)),
		"user_id":    strVal(ctx.Value(userIDKey)),
		"request_id": strVal(ctx.Value(requestKey)),
	}
}

func strVal(v any) string {
	s, _ := v.(string)
	return s
}
