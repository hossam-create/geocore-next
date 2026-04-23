package email

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/circuit"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	retryMax    = 4
	retryBase   = time.Second
	retryJitter = 500 * time.Millisecond
)

// ServiceConfig holds runtime settings for the email service.
// All values are read from environment variables at startup.
type ServiceConfig struct {
	ProviderName     string // "smtp" | "ses" | "sendgrid"  (EMAIL_PROVIDER)
	From             string // EMAIL_FROM
	FromName         string // EMAIL_FROM_NAME
	BaseURL          string // APP_BASE_URL
	RateLimitPerHour int    // max emails per user per hour (default: 10)
}

func loadServiceConfig() ServiceConfig {
	return ServiceConfig{
		ProviderName:     getEnvOr("EMAIL_PROVIDER", "smtp"),
		From:             firstNonEmpty(os.Getenv("EMAIL_FROM"), os.Getenv("SMTP_FROM")),
		FromName:         getEnvOr("EMAIL_FROM_NAME", "GeoCore"),
		BaseURL:          getEnvOr("APP_BASE_URL", "https://geocore.app"),
		RateLimitPerHour: 10,
	}
}

// EmailService is the production-ready email service.
// It wraps a Provider with:
//   - idempotency (Redis dedup)
//   - per-user rate limiting (Redis sliding window)
//   - template rendering (html/template)
//   - retry with exponential backoff + jitter
//   - circuit breaker protection
//   - OpenTelemetry tracing
//   - async dispatch via internal channel worker
type EmailService struct {
	provider  Provider
	rdb       *redis.Client
	templates *TemplateEngine
	tracer    trace.Tracer
	cfg       ServiceConfig
}

var (
	defaultSvc   *EmailService
	defaultSvcMu sync.RWMutex
)

// New creates an EmailService with the provider selected from EMAIL_PROVIDER env.
// rdb may be nil — rate limiting and idempotency are silently skipped in that case.
func New(rdb *redis.Client) *EmailService {
	cfg := loadServiceConfig()
	pcfg := ProviderConfig{
		From:     cfg.From,
		FromName: cfg.FromName,
		BaseURL:  cfg.BaseURL,
	}

	var p Provider
	switch cfg.ProviderName {
	case "sendgrid":
		p = NewSendGridProvider(pcfg)
	case "ses":
		p = NewSESProvider(pcfg)
	default:
		p = NewSMTPProvider(pcfg)
	}

	svc := &EmailService{
		provider:  p,
		rdb:       rdb,
		templates: NewTemplateEngine(),
		tracer:    otel.Tracer("geocore/email"),
		cfg:       cfg,
	}

	slog.Info("email: service initialised",
		"provider", p.Name(),
		"from", cfg.From,
		"rate_limit_per_hour", cfg.RateLimitPerHour,
	)
	return svc
}

// SetDefault registers the global singleton email service (called once from main).
// Subsequent calls are no-ops; first call wins.
func SetDefault(svc *EmailService) {
	defaultSvcMu.Lock()
	defer defaultSvcMu.Unlock()
	if defaultSvc == nil {
		defaultSvc = svc
	}
}

// Default returns the global singleton, creating a dev-mode (stdout only)
// instance if SetDefault was never called.
func Default() *EmailService {
	defaultSvcMu.RLock()
	svc := defaultSvc
	defaultSvcMu.RUnlock()

	if svc != nil {
		return svc
	}
	// Dev-mode fallback: no Redis, SMTP prints to stdout
	return &EmailService{
		provider:  NewSMTPProvider(ProviderConfig{}),
		templates: NewTemplateEngine(),
		tracer:    otel.Tracer("geocore/email"),
		cfg:       loadServiceConfig(),
	}
}

// ─── Core send path ──────────────────────────────────────────────────────────

// Send delivers an email synchronously.
// The full pipeline runs: idempotency → rate-limit → render → retry-send → mark-sent.
// Blocks until delivery succeeds or all retries are exhausted.
func (s *EmailService) Send(ctx context.Context, msg *Message) error {
	ctx, span := s.tracer.Start(ctx, "email.Send",
		trace.WithAttributes(
			attribute.String("email.to", msg.To),
			attribute.String("email.template", msg.TemplateName),
			attribute.String("email.provider", s.provider.Name()),
		),
	)
	defer span.End()

	// 1. Idempotency guard — skip if already delivered
	if msg.IdempotencyKey != "" && s.rdb != nil {
		if sent, _ := isAlreadySent(ctx, s.rdb, msg.IdempotencyKey); sent {
			slog.Info("email: duplicate suppressed",
				"key", msg.IdempotencyKey, "to", msg.To)
			return nil
		}
	}

	// 2. Per-user rate limit
	if msg.UserID != "" && s.rdb != nil {
		if limited, _ := isRateLimited(ctx, s.rdb, msg.UserID, s.cfg.RateLimitPerHour); limited {
			slog.Warn("email: rate limited", "user_id", msg.UserID, "to", msg.To)
			return fmt.Errorf("email: rate limit exceeded for user %s", msg.UserID)
		}
	}

	// 3. Render template (lazy — only when HTML not pre-built)
	if msg.TemplateName != "" && msg.HTML == "" && s.templates != nil {
		html, text, err := s.templates.Render(msg.TemplateName, msg.Data)
		if err != nil {
			slog.Error("email: template render failed",
				"template", msg.TemplateName, "error", err)
			return fmt.Errorf("email: render %q: %w", msg.TemplateName, err)
		}
		msg.HTML = html
		msg.Text = text
	}

	// 4. Send with retry + exponential backoff
	if err := s.sendWithRetry(ctx, msg); err != nil {
		span.RecordError(err)
		return err
	}

	// 5. Mark idempotency key as delivered
	if msg.IdempotencyKey != "" && s.rdb != nil {
		_ = markSent(ctx, s.rdb, msg.IdempotencyKey)
	}

	// 6. Increment rate-limit counter
	if msg.UserID != "" && s.rdb != nil {
		_ = incrementRateLimit(ctx, s.rdb, msg.UserID)
	}

	return nil
}

// sendWithRetry executes provider.Send with exponential backoff + jitter.
// Delays: 1s, 2s, 4s (+ 0–500ms jitter each round).
func (s *EmailService) sendWithRetry(ctx context.Context, msg *Message) error {
	var lastErr error

	for attempt := 1; attempt <= retryMax; attempt++ {
		err := circuit.EmailBreaker.Execute(func(bCtx context.Context) error {
			return s.provider.Send(bCtx, msg)
		})
		if err == nil {
			slog.Info("email: delivered",
				"to", msg.To,
				"template", msg.TemplateName,
				"provider", s.provider.Name(),
				"attempt", attempt,
			)
			return nil
		}

		lastErr = err
		if attempt < retryMax {
			delay := retryBase * time.Duration(1<<uint(attempt-1))
			jitter := time.Duration(rand.Int64N(int64(retryJitter)))
			slog.Warn("email: retrying",
				"attempt", attempt,
				"delay_ms", (delay+jitter).Milliseconds(),
				"error", err,
			)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay + jitter):
			}
		}
	}

	slog.Error("email: all retries exhausted",
		"to", msg.To,
		"attempts", retryMax,
		"last_error", lastErr,
	)
	return fmt.Errorf("email: %d attempts failed: %w", retryMax, lastErr)
}

// ─── Async helpers ────────────────────────────────────────────────────────────

// SendAsync enqueues the email for non-blocking background delivery.
// The calling request lifecycle is not blocked.
// Falls back to synchronous send if the internal queue is full.
func (s *EmailService) SendAsync(ctx context.Context, msg *Message) error {
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	return EnqueueEmail(ctx, msg)
}

// SendTemplate is a convenience helper that builds a Message and enqueues it.
//
//	email.Default().SendTemplate(ctx, "user@example.com", "Alice", "Your OTP", "otp", data)
func (s *EmailService) SendTemplate(
	ctx context.Context,
	to, toName, subject, templateName string,
	data map[string]any,
) error {
	return s.SendAsync(ctx, &Message{
		To:           to,
		ToName:       toName,
		Subject:      subject,
		TemplateName: templateName,
		Data:         data,
		CreatedAt:    time.Now(),
	})
}

// ProcessQueuedMessage is called by the background worker to actually send.
// It deserialises a raw job payload back into a Message and calls Send.
func (s *EmailService) ProcessQueuedMessage(ctx context.Context, msg *Message) error {
	return s.Send(ctx, msg)
}

// ─── Utilities ────────────────────────────────────────────────────────────────

func getEnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
