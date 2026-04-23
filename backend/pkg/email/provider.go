package email

import (
	"context"
	"time"
)

// Provider abstracts the email delivery mechanism (SMTP, SES, SendGrid, etc.)
// All implementations must be safe for concurrent use.
type Provider interface {
	Send(ctx context.Context, msg *Message) error
	Name() string
}

// Message is the canonical email payload passed through the system.
// Templates are rendered lazily before the provider.Send() call.
type Message struct {
	// Recipient
	To     string `json:"to"`
	ToName string `json:"to_name,omitempty"`

	// Content — at least one of HTML/Text must be set, OR TemplateName must be set
	Subject string `json:"subject"`
	HTML    string `json:"html,omitempty"`
	Text    string `json:"text,omitempty"`

	// Template rendering — if HTML is empty the template engine populates it
	TemplateName string         `json:"template_name,omitempty"`
	Data         map[string]any `json:"data,omitempty"`

	// Idempotency — prevents duplicate delivery across retries/restarts.
	// Format: "{event_type}:{entity_id}:{timestamp_minute}" (caller's choice)
	IdempotencyKey string `json:"idempotency_key,omitempty"`

	// UserID is used for per-user rate limiting (optional — skip if empty)
	UserID string `json:"user_id,omitempty"`

	// Internal bookkeeping
	CreatedAt time.Time `json:"created_at"`
}

// ProviderConfig holds settings shared across all provider implementations.
type ProviderConfig struct {
	From     string // sender email address
	FromName string // sender display name
	BaseURL  string // app base URL for links in templates
}
