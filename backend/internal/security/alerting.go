package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// AlertSeverity classifies alert urgency.
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// AlertEvent is the payload sent to all channels.
type AlertEvent struct {
	Event     string            `json:"event"`
	Severity  AlertSeverity     `json:"severity"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Env       string            `json:"env"`
}

// AlertService dispatches alerts to Slack webhook and/or SMTP email.
type AlertService struct {
	WebhookURL string // Slack / Discord / custom webhook
	SMTPHost   string
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string
	EmailFrom  string
	EmailTo    []string
	Env        string
	hc         *http.Client
}

// AlertServiceFromEnv builds an AlertService from environment variables:
//
//	ALERT_WEBHOOK_URL, SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS,
//	ALERT_EMAIL_FROM, ALERT_EMAIL_TO (comma-separated), APP_ENV
func AlertServiceFromEnv() *AlertService {
	emailTo := []string{}
	if raw := os.Getenv("ALERT_EMAIL_TO"); raw != "" {
		for _, e := range strings.Split(raw, ",") {
			if t := strings.TrimSpace(e); t != "" {
				emailTo = append(emailTo, t)
			}
		}
	}
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "production"
	}
	return &AlertService{
		WebhookURL: os.Getenv("ALERT_WEBHOOK_URL"),
		SMTPHost:   os.Getenv("SMTP_HOST"),
		SMTPPort:   os.Getenv("SMTP_PORT"),
		SMTPUser:   os.Getenv("SMTP_USER"),
		SMTPPass:   os.Getenv("SMTP_PASS"),
		EmailFrom:  os.Getenv("ALERT_EMAIL_FROM"),
		EmailTo:    emailTo,
		Env:        env,
		hc:         &http.Client{Timeout: 10 * time.Second},
	}
}

// Send dispatches an alert to all configured channels.
func (a *AlertService) Send(event string, severity AlertSeverity, message string, details map[string]string) {
	payload := AlertEvent{
		Event:     event,
		Severity:  severity,
		Message:   message,
		Details:   details,
		Timestamp: time.Now().UTC(),
		Env:       a.Env,
	}
	slog.Warn("alert", "event", event, "severity", severity, "message", message)
	if a.WebhookURL != "" {
		go a.sendWebhook(payload)
	}
	if a.SMTPHost != "" && len(a.EmailTo) > 0 {
		go a.sendEmail(payload)
	}
}

// ─── Convenience helpers ─────────────────────────────────────────────────────

func (a *AlertService) BackupFailed(reason string) {
	a.Send("backup_failure", SeverityCritical, "Automated backup failed", map[string]string{"reason": reason})
}

func (a *AlertService) RestoreFailed(reason string) {
	a.Send("restore_failure", SeverityCritical, "Backup validation/restore failed", map[string]string{"reason": reason})
}

func (a *AlertService) IntrusionDetected(ip, reason string) {
	a.Send("intrusion_detected", SeverityWarning, "Suspicious activity detected",
		map[string]string{"ip": ip, "reason": reason})
}

func (a *AlertService) EmergencyModeChanged(enabled bool) {
	msg := "Emergency mode ACTIVATED — write operations blocked"
	if !enabled {
		msg = "Emergency mode DEACTIVATED — write operations restored"
	}
	a.Send("emergency_mode", SeverityCritical, msg, nil)
}

// ToAlertFunc returns a simple backup.AlertFunc compatible callback.
func (a *AlertService) ToAlertFunc() func(event, message string) {
	return func(event, message string) {
		sev := SeverityWarning
		if strings.Contains(event, "failure") {
			sev = SeverityCritical
		}
		a.Send(event, sev, message, nil)
	}
}

// ─── Webhook ─────────────────────────────────────────────────────────────────

func (a *AlertService) sendWebhook(evt AlertEvent) {
	// Slack-compatible payload: {"text": "..."}
	text := fmt.Sprintf("[%s] *%s* — %s\n> %s",
		strings.ToUpper(string(evt.Severity)), evt.Event, evt.Env, evt.Message)
	if len(evt.Details) > 0 {
		parts := []string{}
		for k, v := range evt.Details {
			parts = append(parts, k+": "+v)
		}
		text += "\n" + strings.Join(parts, " | ")
	}
	body, _ := json.Marshal(map[string]string{"text": text})
	resp, err := a.hc.Post(a.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		slog.Error("alert: webhook failed", "err", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		slog.Warn("alert: webhook non-2xx", "status", resp.StatusCode)
	}
}

// ─── SMTP email ───────────────────────────────────────────────────────────────

func (a *AlertService) sendEmail(evt AlertEvent) {
	subject := fmt.Sprintf("[%s][%s] %s", strings.ToUpper(string(evt.Severity)), a.Env, evt.Message)
	body := fmt.Sprintf("Event: %s\nTime: %s\nMessage: %s\n",
		evt.Event, evt.Timestamp.Format(time.RFC3339), evt.Message)
	for k, v := range evt.Details {
		body += fmt.Sprintf("%s: %s\n", k, v)
	}
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		a.EmailFrom,
		strings.Join(a.EmailTo, ", "),
		subject,
		body,
	)
	addr := a.SMTPHost + ":" + a.SMTPPort
	var auth smtp.Auth
	if a.SMTPUser != "" {
		auth = smtp.PlainAuth("", a.SMTPUser, a.SMTPPass, a.SMTPHost)
	}
	if err := smtp.SendMail(addr, auth, a.EmailFrom, a.EmailTo, []byte(msg)); err != nil {
		slog.Error("alert: email failed", "err", err)
	}
}
