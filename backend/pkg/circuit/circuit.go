package circuit

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota // normal — requests flow through
	StateOpen                  // tripped — requests fail fast
	StateHalfOpen              // probing — one request allowed to test recovery
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Breaker implements the circuit breaker pattern for external service calls.
// When failures exceed the threshold the breaker opens and all calls fail fast.
// After the reset timeout it transitions to half-open, allowing one probe call.
type Breaker struct {
	name       string
	threshold  int           // consecutive failures to trip
	timeout    time.Duration // how long to stay open before half-open
	maxTimeout time.Duration // cap for external call timeout

	mu           sync.Mutex
	state        State
	failures     int
	lastFailTime time.Time
}

// NewBreaker creates a circuit breaker with the given configuration.
//   - name: identifier for logging
//   - threshold: consecutive failures before opening (e.g. 5)
//   - timeout: duration to stay open before half-open probe (e.g. 30s)
//   - callTimeout: per-call context timeout (e.g. 3s)
func NewBreaker(name string, threshold int, timeout, callTimeout time.Duration) *Breaker {
	return &Breaker{
		name:       name,
		threshold:  threshold,
		timeout:    timeout,
		maxTimeout: callTimeout,
		state:      StateClosed,
	}
}

// Execute runs fn through the circuit breaker. If the breaker is open it
// returns an error immediately. If closed or half-open, fn is executed with
// a context timeout. On success the breaker resets; on failure it increments
// the failure count and potentially trips the breaker.
func (b *Breaker) Execute(fn func(ctx context.Context) error) error {
	b.mu.Lock()
	switch b.state {
	case StateOpen:
		if time.Since(b.lastFailTime) < b.timeout {
			b.mu.Unlock()
			return fmt.Errorf("circuit[%s]: open — fail fast", b.name)
		}
		// Transition to half-open
		b.state = StateHalfOpen
		slog.Info("circuit breaker half-open", "name", b.name)
		fallthrough
	case StateHalfOpen:
		b.mu.Unlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), b.maxTimeout)
	defer cancel()

	err := fn(ctx)

	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		b.failures++
		b.lastFailTime = time.Now()
		if b.failures >= b.threshold {
			if b.state != StateOpen {
				slog.Warn("circuit breaker tripped open",
					"name", b.name,
					"failures", b.failures,
					"threshold", b.threshold,
				)
				metrics.IncCircuitBreakerOpen(b.name)
			}
			b.state = StateOpen
		}
		metrics.IncCircuitBreakerFailure(b.name)
		return fmt.Errorf("circuit[%s]: %w", b.name, err)
	}

	// Success — reset
	b.failures = 0
	b.state = StateClosed
	return nil
}

// State returns the current breaker state (for metrics/inspection).
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}

// Pre-configured breakers for common external dependencies.
var (
	// SMSBreaker protects Twilio SMS calls.
	SMSBreaker = NewBreaker("sms", 5, 30*time.Second, 3*time.Second)
	// EmailBreaker protects SMTP email calls.
	EmailBreaker = NewBreaker("email", 5, 30*time.Second, 5*time.Second)
	// PaymentsBreaker protects Stripe/PayMob payment calls.
	PaymentsBreaker = NewBreaker("payments", 3, 60*time.Second, 3*time.Second)
	// PushBreaker protects FCM push notification calls.
	PushBreaker = NewBreaker("push", 5, 30*time.Second, 3*time.Second)
)
