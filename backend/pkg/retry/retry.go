package retry

import (
	"context"
	"log/slog"
	"math/rand"
	"time"
)

// Do retries fn up to attempts times with exponential backoff.
// Backoff: 100ms, 200ms, 400ms, 800ms, ...
func Do(attempts int, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		backoff := time.Duration(1<<uint(i)) * 100 * time.Millisecond
		slog.Debug("retry: attempt failed", "attempt", i+1, "error", err, "backoff", backoff)
		time.Sleep(backoff)
	}
	return err
}

// DoWithContext retries with context cancellation support.
func DoWithContext(ctx context.Context, attempts int, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = fn()
		if err == nil {
			return nil
		}
		backoff := time.Duration(1<<uint(i)) * 100 * time.Millisecond
		slog.Debug("retry: attempt failed", "attempt", i+1, "error", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
	return err
}

// DoWithMaxBackoff retries with a capped maximum backoff duration.
func DoWithMaxBackoff(attempts int, maxBackoff time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		backoff := time.Duration(1<<uint(i)) * 100 * time.Millisecond
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		slog.Debug("retry: attempt failed", "attempt", i+1, "error", err, "backoff", backoff)
		time.Sleep(backoff)
	}
	return err
}

// DoWithContextAndJitter retries with exponential backoff + random jitter.
// This is the production-safe version: jitter prevents retry storms when
// many callers retry the same failing service simultaneously.
//
// Use for external service calls (Stripe, PayMob, SMS, email):
//
//	err := retry.DoWithContextAndJitter(ctx, 3, 2*time.Second, func() error {
//	    return circuit.PaymentsBreaker.Execute(fn)
//	})
//
// Pattern: circuit breaker INSIDE retry — breaker fails fast on open state,
// retry backs off and tries again after the breaker cooldown.
func DoWithContextAndJitter(ctx context.Context, attempts int, maxBackoff time.Duration, fn func() error) error {
	var err error
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = fn()
		if err == nil {
			return nil
		}
		// Exponential backoff: 200ms, 400ms, 800ms, ...
		backoff := time.Duration(1<<uint(i)) * 200 * time.Millisecond
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		// Add ±25% jitter to prevent synchronized retries
		jitter := time.Duration(float64(backoff) * 0.25)
		backoff += time.Duration(rand.Int63n(int64(2*jitter+1))) - jitter //nolint:gosec

		slog.Debug("retry: attempt failed",
			"attempt", i+1, "error", err, "backoff", backoff)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
	return err
}
