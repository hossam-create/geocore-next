package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const sendgridEndpoint = "https://api.sendgrid.com/v3/mail/send"

// SendGridProvider delivers email via the SendGrid v3 REST API.
// No external SDK — uses standard net/http.
//
// ENV:
//
//	SENDGRID_API_KEY — API key with "Mail Send" permission (starts with "SG.")
//	EMAIL_FROM       — verified sender address
//	EMAIL_FROM_NAME  — sender display name
type SendGridProvider struct {
	cfg    ProviderConfig
	apiKey string
	client *http.Client
}

// NewSendGridProvider creates a SendGrid provider.
func NewSendGridProvider(cfg ProviderConfig) *SendGridProvider {
	return &SendGridProvider{
		cfg:    cfg,
		apiKey: os.Getenv("SENDGRID_API_KEY"),
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *SendGridProvider) Name() string { return "sendgrid" }

// Send delivers via the SendGrid v3 API.
// Falls back to dev-print when SENDGRID_API_KEY is absent.
func (p *SendGridProvider) Send(ctx context.Context, msg *Message) error {
	if p.apiKey == "" {
		devPrint(msg)
		return nil
	}

	body, err := json.Marshal(p.buildPayload(msg))
	if err != nil {
		return fmt.Errorf("sendgrid: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sendgridEndpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sendgrid: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		rb, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("sendgrid: status %d: %s", resp.StatusCode, string(rb))
	}
	return nil
}

// ─── SendGrid v3 payload structs ─────────────────────────────────────────────

type sgPayload struct {
	Personalizations []sgPersonalization `json:"personalizations"`
	From             sgAddress           `json:"from"`
	Subject          string              `json:"subject"`
	Content          []sgContent         `json:"content"`
}

type sgPersonalization struct {
	To []sgAddress `json:"to"`
}

type sgAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type sgContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (p *SendGridProvider) buildPayload(msg *Message) sgPayload {
	to := sgAddress{Email: msg.To, Name: msg.ToName}

	var content []sgContent
	if msg.Text != "" {
		content = append(content, sgContent{Type: "text/plain", Value: msg.Text})
	}
	if msg.HTML != "" {
		content = append(content, sgContent{Type: "text/html", Value: msg.HTML})
	}
	if len(content) == 0 {
		content = []sgContent{{Type: "text/plain", Value: msg.Subject}}
	}

	from := sgAddress{Email: p.cfg.From, Name: p.cfg.FromName}
	if from.Email == "" {
		from.Email = os.Getenv("EMAIL_FROM")
	}

	return sgPayload{
		Personalizations: []sgPersonalization{{To: []sgAddress{to}}},
		From:             from,
		Subject:          msg.Subject,
		Content:          content,
	}
}
