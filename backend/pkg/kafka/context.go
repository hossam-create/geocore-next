package kafka

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const (
	regionKey        contextKey = "region"
	idempotencyKey   contextKey = "idempotency_key"
)

// GetRegionFromContext extracts the region from context.
// Returns "primary" if not set.
func GetRegionFromContext(ctx context.Context) string {
	if r, ok := ctx.Value(regionKey).(string); ok && r != "" {
		return r
	}
	return "primary"
}

// GetIdempotencyKeyFromContext extracts the idempotency key from context.
// Generates a new UUID if not set.
func GetIdempotencyKeyFromContext(ctx context.Context) string {
	if k, ok := ctx.Value(idempotencyKey).(string); ok && k != "" {
		return k
	}
	return uuid.NewString()
}

// WithRegion returns a context with the region value set.
func WithRegion(ctx context.Context, region string) context.Context {
	return context.WithValue(ctx, regionKey, region)
}

// WithIdempotencyKey returns a context with the idempotency key set.
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKey, key)
}

// EnrichEvent populates Region and IdempotencyKey on an event from context.
func EnrichEvent(ctx context.Context, evt *Event) {
	evt.Region = GetRegionFromContext(ctx)
	if evt.IdempotencyKey == "" {
		evt.IdempotencyKey = GetIdempotencyKeyFromContext(ctx)
	}
}
