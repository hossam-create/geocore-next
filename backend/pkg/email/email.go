package email

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"
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
// Legacy SMTP config helper (kept for transactional.go BaseURL resolution)
// ════════════════════════════════════════════════════════════════════════════

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

// send is the shared delivery helper used by transactional.go and the auth
// functions below. It delegates to the global EmailService provider so the
// active backend (smtp/ses/sendgrid) is respected without changing every call site.
func send(cfg smtpConfig, to, subject, body string) error {
	msg := &Message{
		To:        to,
		Subject:   subject,
		Text:      body,
		CreatedAt: time.Now(),
	}
	return Default().Send(context.Background(), msg)
}

// ════════════════════════════════════════════════════════════════════════════
// Email: account verification
// ════════════════════════════════════════════════════════════════════════════

// SendVerificationEmail sends an account-verification link via the OTP template.
// Async — does not block the request lifecycle.
func SendVerificationEmail(to, token string) error {
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)
	return Default().SendAsync(context.Background(), &Message{
		To:           to,
		Subject:      "Verify your GeoCore email",
		TemplateName: "notification",
		Data: NotificationData(
			"there",
			"Verify Your Email Address",
			"Please verify your email address to complete your GeoCore registration. This link expires in 24 hours.",
			"Verify Email",
			verifyURL,
		),
		IdempotencyKey: "verify:" + token,
		CreatedAt:      time.Now(),
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Email: OTP verification code
// ════════════════════════════════════════════════════════════════════════════

// SendOTPEmail delivers a one-time verification code using the "otp" HTML template.
// Async — does not block the request lifecycle.
func SendOTPEmail(to, name, userID, otp string, expiresMin int) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	return Default().SendAsync(context.Background(), &Message{
		To:             to,
		ToName:         name,
		UserID:         userID,
		Subject:        "Your GeoCore verification code",
		TemplateName:   "otp",
		Data:           OTPData(name, otp, expiresMin, baseURL),
		IdempotencyKey: "otp:" + userID + ":" + otp,
		CreatedAt:      time.Now(),
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Email: password reset request
// ════════════════════════════════════════════════════════════════════════════

// SendPasswordResetEmail delivers a password-reset link using the HTML template.
// Async — does not block the request lifecycle.
func SendPasswordResetEmail(to, name, userID, token string) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", baseURL, token)
	return Default().SendAsync(context.Background(), &Message{
		To:             to,
		ToName:         name,
		UserID:         userID,
		Subject:        "Reset your GeoCore password",
		TemplateName:   "password_reset",
		Data:           PasswordResetData(name, resetURL, 1),
		IdempotencyKey: "pwreset:" + token,
		CreatedAt:      time.Now(),
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Email: password changed confirmation
// ════════════════════════════════════════════════════════════════════════════

// SendPasswordChangedEmail sends a security alert after a successful password reset.
// Async — does not block the request lifecycle.
func SendPasswordChangedEmail(to, name, userID string) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	return Default().SendAsync(context.Background(), &Message{
		To:           to,
		UserID:       userID,
		Subject:      "Your GeoCore password was changed",
		TemplateName: "notification",
		Data: NotificationData(
			name,
			"Your password was changed",
			"Your GeoCore password has been successfully changed. If you did not make this change, contact security@geocore.app immediately.",
			"Go to Account",
			baseURL+"/profile",
		),
		IdempotencyKey: "pwchanged:" + userID,
		CreatedAt:      time.Now(),
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Email: transaction receipt
// ════════════════════════════════════════════════════════════════════════════

// SendTransactionReceiptEmail delivers a payment receipt using the
// "transaction_receipt" HTML template. Called after successful payment.
// Async — does not block the request lifecycle.
func SendTransactionReceiptEmail(to, name, userID, orderID, itemTitle string, amount float64, currency string) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	orderURL := fmt.Sprintf("%s/orders/%s", baseURL, orderID)
	return Default().SendAsync(context.Background(), &Message{
		To:             to,
		ToName:         name,
		UserID:         userID,
		Subject:        fmt.Sprintf("Receipt for order #%s", orderID),
		TemplateName:   "transaction_receipt",
		Data:           TransactionReceiptData(name, orderID, itemTitle, amount, currency, orderURL),
		IdempotencyKey: "receipt:" + orderID,
		CreatedAt:      time.Now(),
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Email: escrow released
// ════════════════════════════════════════════════════════════════════════════

// SendEscrowReleasedEmail notifies a seller that their escrow has been released.
// Async — does not block the request lifecycle.
func SendEscrowReleasedEmail(to, name, userID, escrowID string, amount float64, currency string) error {
	if name == "" {
		name = "there"
	}
	baseURL := getEnvOr("APP_BASE_URL", "https://geocore.app")
	return Default().SendAsync(context.Background(), &Message{
		To:           to,
		ToName:       name,
		UserID:       userID,
		Subject:      "Your escrow has been released",
		TemplateName: "notification",
		Data: NotificationData(
			name,
			"Escrow Released",
			fmt.Sprintf("Good news! Your escrow for %.2f %s has been released to your wallet. The funds are now available for withdrawal.", amount, currency),
			"View Wallet",
			baseURL+"/wallet",
		),
		IdempotencyKey: "escrow_released:" + escrowID,
		CreatedAt:      time.Now(),
	})
}
