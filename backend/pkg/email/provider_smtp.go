package email

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// SMTPProvider delivers email via STARTTLS SMTP.
// Compatible with any SMTP relay: generic MTA, Mailgun, Postmark, AWS SES SMTP, etc.
//
// ENV:
//
//	SMTP_HOST  — mail server hostname (e.g. smtp.mailgun.org)
//	SMTP_PORT  — port, defaults to 587
//	SMTP_USER  — SMTP username
//	SMTP_PASS  — SMTP password
//	SMTP_FROM  — sender address (fallback when EMAIL_FROM is unset)
type SMTPProvider struct {
	cfg  ProviderConfig
	host string
	port string
	user string
	pass string
}

// NewSMTPProvider creates an SMTP provider, reading credentials from env.
func NewSMTPProvider(cfg ProviderConfig) *SMTPProvider {
	return &SMTPProvider{
		cfg:  cfg,
		host: os.Getenv("SMTP_HOST"),
		port: getEnvOr("SMTP_PORT", "587"),
		user: os.Getenv("SMTP_USER"),
		pass: os.Getenv("SMTP_PASS"),
	}
}

func (p *SMTPProvider) Name() string { return "smtp" }

// Send delivers a single message. If SMTP is not configured (dev mode) it
// prints the message to stdout so local development works without a mail server.
func (p *SMTPProvider) Send(ctx context.Context, msg *Message) error {
	from := firstNonEmpty(p.cfg.From, os.Getenv("SMTP_FROM"))

	if p.host == "" || from == "" {
		devPrint(msg)
		return nil
	}

	raw := buildMIME(from, p.cfg.FromName, msg)
	addr := fmt.Sprintf("%s:%s", p.host, p.port)

	var auth smtp.Auth
	if p.user != "" && p.pass != "" {
		auth = smtp.PlainAuth("", p.user, p.pass, p.host)
	}

	if err := smtp.SendMail(addr, auth, from, []string{msg.To}, raw); err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	return nil
}

// ─── MIME builder shared by SMTP + SES providers ─────────────────────────────

// buildMIME constructs a multipart/alternative MIME message.
// Falls back to text/plain if HTML is empty.
func buildMIME(from, fromName string, msg *Message) []byte {
	var sb strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&sb, f, a...) }

	displayFrom := from
	if fromName != "" {
		displayFrom = fmt.Sprintf("%s <%s>", fromName, from)
	}

	w("From: %s\r\n", displayFrom)
	w("To: %s\r\n", msg.To)
	w("Subject: %s\r\n", msg.Subject)
	w("MIME-Version: 1.0\r\n")

	if msg.HTML != "" {
		boundary := "gc_bound_" + msg.CreatedAt.Format("20060102150405")
		w("Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)

		// Plain-text part
		w("--%s\r\n", boundary)
		w("Content-Type: text/plain; charset=UTF-8\r\n")
		w("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if msg.Text != "" {
			sb.WriteString(msg.Text)
		} else {
			sb.WriteString(msg.Subject) // minimal fallback
		}

		// HTML part
		w("\r\n--%s\r\n", boundary)
		w("Content-Type: text/html; charset=UTF-8\r\n")
		w("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		sb.WriteString(msg.HTML)
		w("\r\n--%s--\r\n", boundary)
	} else {
		w("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		sb.WriteString(msg.Text)
	}

	return []byte(sb.String())
}

// devPrint logs to stdout in development mode.
func devPrint(msg *Message) {
	fmt.Printf("[email-dev] To:%s | Subject:%s | Template:%s\n",
		msg.To, msg.Subject, msg.TemplateName)
}
