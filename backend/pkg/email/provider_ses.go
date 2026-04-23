package email

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
)

// SESProvider delivers email via the AWS SES SMTP relay endpoint.
//
// Uses standard STARTTLS SMTP — no AWS SDK required.
// Generate SMTP credentials in the AWS SES console:
//
//	IAM → Users → Create user → Attach policy: AmazonSESFullAccess
//	SES → SMTP Settings → Create SMTP credentials
//
// ENV:
//
//	AWS_REGION        — SES region, e.g. "us-east-1" (defaults to us-east-1)
//	AWS_SES_SMTP_USER — SMTP username from SES credential generation
//	AWS_SES_SMTP_PASS — SMTP password from SES credential generation
//	EMAIL_FROM        — verified sender address in SES
type SESProvider struct {
	cfg  ProviderConfig
	host string
	user string
	pass string
}

// NewSESProvider creates an SES provider using the SMTP relay.
func NewSESProvider(cfg ProviderConfig) *SESProvider {
	region := getEnvOr("AWS_REGION", "us-east-1")
	return &SESProvider{
		cfg:  cfg,
		host: fmt.Sprintf("email-smtp.%s.amazonaws.com", region),
		user: os.Getenv("AWS_SES_SMTP_USER"),
		pass: os.Getenv("AWS_SES_SMTP_PASS"),
	}
}

func (p *SESProvider) Name() string { return "ses" }

// Send delivers via AWS SES SMTP. Falls back to dev-print when credentials
// are absent (safe for local dev without SES access).
func (p *SESProvider) Send(ctx context.Context, msg *Message) error {
	from := p.cfg.From
	if from == "" {
		from = os.Getenv("EMAIL_FROM")
	}

	if p.user == "" || p.pass == "" || from == "" {
		devPrint(msg)
		return nil
	}

	raw := buildMIME(from, p.cfg.FromName, msg)
	addr := fmt.Sprintf("%s:587", p.host)
	auth := smtp.PlainAuth("", p.user, p.pass, p.host)

	if err := smtp.SendMail(addr, auth, from, []string{msg.To}, raw); err != nil {
		return fmt.Errorf("ses: %w", err)
	}
	return nil
}
