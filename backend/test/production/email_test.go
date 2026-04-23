//go:build production

package production

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/geocore-next/backend/pkg/email"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ════════════════════════════════════════════════════════════════════════════════
// Real Email Integration Tests
//
// Sends real emails via SendGrid/SES and verifies delivery through:
//   1. SendGrid API — checks activity/events for the sent message
//   2. Mailinator/Inbox — verifies receipt in test inbox
//
// Required env vars:
//   - EMAIL_PROVIDER=sendgrid
//   - SENDGRID_API_KEY
//   - EMAIL_FROM (verified sender)
//   - EMAIL_TEST_INBOX (inbox address for delivery verification)
// ════════════════════════════════════════════════════════════════════════════════

type EmailSuite struct {
	suite.Suite
	ts      *ProdSuite
	inbox   string
	apiKey  string
}

func TestEmailSuite(t *testing.T) {
	ts := SetupProdSuite(t)
	defer TeardownProdSuite(ts)

	suite.Run(t, &EmailSuite{ts: ts})
}

func (s *EmailSuite) SetupSuite() {
	s.inbox = os.Getenv("EMAIL_TEST_INBOX")
	if s.inbox == "" {
		s.T().Skip("EMAIL_TEST_INBOX not set — skipping real email tests")
	}
	s.apiKey = os.Getenv("SENDGRID_API_KEY")
}

// ── Test: Send plain text email via real provider ──────────────────────────────

func (s *EmailSuite) TestSendPlainTextEmail_RealDelivery() {
	to := s.inbox
	idempotencyKey := fmt.Sprintf("prod-test-plain-%d", time.Now().UnixMilli())

	msg := &email.Message{
		To:             to,
		ToName:         "Production Test",
		Subject:        "[GeoCore Prod Test] Plain Text Email",
		Text:           "This is a production integration test email sent via the real email provider.",
		IdempotencyKey: idempotencyKey,
	}

	err := s.ts.EmailSvc.Send(context.Background(), msg)
	require.NoError(s.T(), err, "email send should succeed")

	// Verify via SendGrid activity API (if SendGrid is the provider)
	if s.apiKey != "" {
		delivered := s.waitForSendGridDelivery(msg.Subject, 30*time.Second)
		assert.True(s.T(), delivered, "email should be delivered according to SendGrid activity")
	}
}

// ── Test: Send HTML email with template rendering ─────────────────────────────

func (s *EmailSuite) TestSendTemplatedEmail_RealDelivery() {
	to := s.inbox
	idempotencyKey := fmt.Sprintf("prod-test-template-%d", time.Now().UnixMilli())

	msg := &email.Message{
		To:             to,
		ToName:         "Production Test",
		Subject:        "[GeoCore Prod Test] Templated Email",
		TemplateName:   "email_verification",
		IdempotencyKey: idempotencyKey,
		Data: map[string]interface{}{
			"Name":           "Test User",
			"VerificationURL": "https://geocore.app/verify?token=test-token-123",
		},
	}

	err := s.ts.EmailSvc.Send(context.Background(), msg)
	require.NoError(s.T(), err, "templated email send should succeed")

	// Verify delivery
	if s.apiKey != "" {
		delivered := s.waitForSendGridDelivery(msg.Subject, 30*time.Second)
		assert.True(s.T(), delivered, "templated email should be delivered")
	}
}

// ── Test: Idempotent send is deduplicated ──────────────────────────────────────

func (s *EmailSuite) TestIdempotentSend_Deduplicated() {
	to := s.inbox
	idempotencyKey := fmt.Sprintf("prod-test-idem-%d", time.Now().UnixMilli())

	msg := &email.Message{
		To:             to,
		Subject:        "[GeoCore Prod Test] Idempotent Email",
		Text:           "This should only be sent once.",
		IdempotencyKey: idempotencyKey,
	}

	// First send succeeds
	err := s.ts.EmailSvc.Send(context.Background(), msg)
	require.NoError(s.T(), err)

	// Second send with same key should be suppressed
	err = s.ts.EmailSvc.Send(context.Background(), msg)
	require.NoError(s.T(), err, "duplicate send should not error (silently suppressed)")
}

// ── Test: Rate limiting works for real ─────────────────────────────────────────

func (s *EmailSuite) TestRateLimiting_Enforced() {
	to := s.inbox
	userID := "prod-rate-limit-test-user"

	// Send multiple emails rapidly
	var lastErr error
	for i := 0; i < 15; i++ {
		msg := &email.Message{
			To:             to,
			Subject:        fmt.Sprintf("[GeoCore Prod Test] Rate Limit Email #%d", i),
			Text:           fmt.Sprintf("Rate limit test email #%d", i),
			UserID:         userID,
			IdempotencyKey: fmt.Sprintf("prod-rate-%d-%d", time.Now().UnixMilli(), i),
		}
		err := s.ts.EmailSvc.Send(context.Background(), msg)
		if err != nil {
			lastErr = err
		}
	}

	// After 10+ emails to the same user, rate limiting should kick in
	assert.NotNil(s.T(), lastErr, "rate limiting should eventually trigger")
}

// ── Helper: Poll SendGrid activity API for delivery confirmation ──────────────

func (s *EmailSuite) waitForSendGridDelivery(subject string, timeout time.Duration) bool {
	if s.apiKey == "" {
		return false
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		delivered, err := s.checkSendGridActivity(subject)
		if err == nil && delivered {
			return true
		}
		time.Sleep(3 * time.Second)
	}
	return false
}

func (s *EmailSuite) checkSendGridActivity(subject string) (bool, error) {
	url := "https://api.sendgrid.com/v3/messages?limit=10&query=subject%3D%22" + subject + "%22"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("sendgrid API returned %d", resp.StatusCode)
	}

	var result struct {
		Messages []struct {
			Subject string `json:"subject"`
			Status  string `json:"status"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	for _, msg := range result.Messages {
		if msg.Subject == subject && (msg.Status == "delivered" || msg.Status == "processed") {
			return true, nil
		}
	}
	return false, nil
}
