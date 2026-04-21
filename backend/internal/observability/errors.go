package observability

import (
	"log/slog"
	"runtime/debug"
)

// ════════════════════════════════════════════════════════════════════════════
// Error Tracking + Logging
// CaptureError: Stub for Sentry integration.
// SafeGo: Panic-safe goroutine wrapper for background jobs.
// FinancialLog: Structured logging for financial operations.
// ════════════════════════════════════════════════════════════════════════════

// CaptureError logs an error with structured context.
// Stub for Sentry integration — replace body with sentry.CaptureException when ready.
func CaptureError(err error, context map[string]interface{}) {
	if err == nil {
		return
	}
	args := make([]interface{}, 0, len(context)*2+2)
	args = append(args, "error", err.Error())
	for k, v := range context {
		args = append(args, k, v)
	}
	slog.Error("observability: captured error", args...)
}

// CapturePanic logs a panic with stack trace.
func CapturePanic(r interface{}, context map[string]interface{}) {
	args := make([]interface{}, 0, len(context)*2+4)
	args = append(args, "panic", r)
	args = append(args, "stack", string(debug.Stack()))
	for k, v := range context {
		args = append(args, k, v)
	}
	slog.Error("observability: panic captured", args...)
}

// FinancialLog logs a financial operation with required context fields.
func FinancialLog(action string, userID string, requestID string, amountCents int64, result string) {
	slog.Info("financial",
		"action", action,
		"user_id", userID,
		"request_id", requestID,
		"amount_cents", amountCents,
		"result", result,
	)
}

// SafeGo runs a function in a goroutine with panic recovery + error tracking.
func SafeGo(fn func(), label string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				CapturePanic(r, map[string]interface{}{"label": label})
			}
		}()
		fn()
	}()
}
