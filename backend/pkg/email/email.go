package email

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"
	"strings"

	"github.com/geocore-next/backend/pkg/circuit"
)

// GenerateToken creates a cryptographically secure random hex token.
// byteLen controls entropy: 32 bytes → 64-char hex string.
func GenerateToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ════════════════════════════════════════════════════════════════════════════
// Shared SMTP helper
// ════════════════════════════════════════════════════════════════════════════

// smtpConfig holds values read once from environment variables.
type smtpConfig struct {
	Host    string
	Port    string
	User    string
	Pass    string
	From    string
	BaseURL string
}

func loadSMTP() smtpConfig {
	cfg := smtpConfig{
		Host:    os.Getenv("SMTP_HOST"),
		Port:    os.Getenv("SMTP_PORT"),
		User:    os.Getenv("SMTP_USER"),
		Pass:    os.Getenv("SMTP_PASS"),
		From:    os.Getenv("SMTP_FROM"),
		BaseURL: os.Getenv("APP_BASE_URL"),
	}
	if cfg.Port == "" {
		cfg.Port = "587"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://geocore.app"
	}
	return cfg
}

// send transmits a plain-text email (protected by circuit breaker).
// If SMTP is unconfigured it falls back to stdout logging so development works without a mail server.
func send(cfg smtpConfig, to, subject, body string) error {
	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] To: %s | Subject: %s\n%s\n", to, subject, body)
		return nil
	}

	var sb strings.Builder
	printf := func(format string, args ...any) { fmt.Fprintf(&sb, format, args...) }
	printf("From: GeoCore <%s>\r\n", cfg.From)
	printf("To: %s\r\n", to)
	printf("Subject: %s\r\n", subject)
	printf("MIME-Version: 1.0\r\n")
	printf("Content-Type: text/plain; charset=UTF-8\r\n")
	printf("\r\n")
	sb.WriteString(body)

	msg := []byte(sb.String())
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	var auth smtp.Auth
	if cfg.User != "" && cfg.Pass != "" {
		auth = smtp.PlainAuth("", cfg.User, cfg.Pass, cfg.Host)
	}

	err := circuit.EmailBreaker.Execute(func(_ context.Context) error {
		return smtp.SendMail(addr, auth, cfg.From, []string{to}, msg)
	})
	if err != nil {
		slog.Warn("email circuit breaker blocked or send failed", "to", to, "error", err)
	}
	return err
}

// ════════════════════════════════════════════════════════════════════════════
// Email: account verification
// ════════════════════════════════════════════════════════════════════════════

// SendVerificationEmail sends an account-verification link to the given address.
func SendVerificationEmail(to, token string) error {
	cfg := loadSMTP()
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", cfg.BaseURL, token)

	body := fmt.Sprintf(
		"Welcome to GeoCore!\n\n"+
			"Please verify your email address by clicking the link below:\n\n"+
			"  %s\n\n"+
			"This link expires in 24 hours.\n\n"+
			"If you did not create an account, you can safely ignore this email.\n\n"+
			"— The GeoCore Team",
		verifyURL,
	)

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Verification token for %s: %s\n", to, token)
		return nil
	}
	return send(cfg, to, "Verify your GeoCore email", body)
}

// ════════════════════════════════════════════════════════════════════════════
// Email: password reset request
// ════════════════════════════════════════════════════════════════════════════

// SendPasswordResetEmail delivers a password-reset link to the user.
// Token is embedded in the URL — the frontend must extract it and POST it
// to /api/v1/auth/reset-password.
func SendPasswordResetEmail(to, name, token string) error {
	cfg := loadSMTP()

	if name == "" {
		name = "there"
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", cfg.BaseURL, token)

	body := fmt.Sprintf(
		"Hi %s,\n\n"+
			"We received a request to reset the password for your GeoCore account.\n\n"+
			"Click the link below to set a new password:\n\n"+
			"  %s\n\n"+
			"⏰ This link expires in 1 hour.\n\n"+
			"──────────────────────────────────────────────────\n"+
			"If you did not request a password reset, you can safely ignore this\n"+
			"email. Your password will not be changed.\n\n"+
			"For security, this link can only be used once.\n\n"+
			"— The GeoCore Security Team",
		name, resetURL,
	)

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Password reset token for %s: %s\n", to, token)
		return nil
	}
	return send(cfg, to, "Reset your GeoCore password", body)
}

// ════════════════════════════════════════════════════════════════════════════
// Email: password changed confirmation
// ════════════════════════════════════════════════════════════════════════════

// SendPasswordChangedEmail sends a security alert after a successful password reset.
// This notifies the account owner of the change so they can take action if it
// was not initiated by them.
func SendPasswordChangedEmail(to, name string) error {
	cfg := loadSMTP()

	if name == "" {
		name = "there"
	}

	body := fmt.Sprintf(
		"Hi %s,\n\n"+
			"✅ Your GeoCore password has been successfully changed.\n\n"+
			"If you made this change, no further action is needed.\n\n"+
			"──────────────────────────────────────────────────\n"+
			"⚠️  If you did NOT change your password, your account may be\n"+
			"compromised. Please contact us immediately at security@geocore.app\n"+
			"or reset your password again at:\n\n"+
			"  %s/forgot-password\n\n"+
			"— The GeoCore Security Team",
		name, cfg.BaseURL,
	)

	if cfg.Host == "" || cfg.From == "" {
		fmt.Printf("[email-dev] Password changed confirmation sent to %s\n", to)
		return nil
	}
	return send(cfg, to, "Your GeoCore password was changed", body)
}
