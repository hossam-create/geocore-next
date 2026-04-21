// Package logging provides structured logging wrappers with trace context.
package logging

import (
	"context"
	"log/slog"

	"github.com/geocore-next/backend/pkg/tracing"
)

// Logger wraps slog with automatic trace_id and request_id injection.
type Logger struct {
	service string
}

// New creates a structured logger for the given service.
func New(service string) *Logger {
	return &Logger{service: service}
}

// Info logs with trace context.
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	args = l.injectContext(ctx, args)
	slog.Info(msg, args...)
}

// Warn logs with trace context.
func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	args = l.injectContext(ctx, args)
	slog.Warn(msg, args...)
}

// Error logs with trace context.
func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	args = l.injectContext(ctx, args)
	slog.Error(msg, args...)
}

// Debug logs with trace context.
func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	args = l.injectContext(ctx, args)
	slog.Debug(msg, args...)
}

func (l *Logger) injectContext(ctx context.Context, args []any) []any {
	out := make([]any, 0, len(args)+6)
	out = append(out, "service", l.service)
	if traceID := tracing.TraceIDFromContext(ctx); traceID != "" {
		out = append(out, "trace_id", traceID)
	}
	out = append(out, args...)
	return out
}
