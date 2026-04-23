package integration

import (
	"fmt"
	"sync"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/google/uuid"
)

// ════════════════════════════════════════════════════════════════════════════════
// Mock Providers — Stripe, FCM, Email
// ════════════════════════════════════════════════════════════════════════════════

// CapturedNotification is a recorded notification for test assertions.
type CapturedNotification struct {
	UserID uuid.UUID
	Type   string
	Title  string
	Body   string
	Data   map[string]string
}

// CaptureNotifier intercepts notification calls for assertion.
// Implements the notifications.ServiceInterface pattern used by auction/chat/etc.
type CaptureNotifier struct {
	mu   sync.Mutex
	caps []CapturedNotification
}

func NewCaptureNotifier() *CaptureNotifier {
	return &CaptureNotifier{}
}

func (c *CaptureNotifier) Notify(input notifications.NotifyInput) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.caps = append(c.caps, CapturedNotification{
		UserID: input.UserID,
		Type:   input.Type,
		Title:  input.Title,
		Body:   input.Body,
		Data:   input.Data,
	})
}

func (c *CaptureNotifier) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.caps = nil
}

func (c *CaptureNotifier) FindBy(userID uuid.UUID, notifType string) []CapturedNotification {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []CapturedNotification
	for _, n := range c.caps {
		if n.UserID == userID && n.Type == notifType {
			result = append(result, n)
		}
	}
	return result
}

func (c *CaptureNotifier) AllFor(userID uuid.UUID) []CapturedNotification {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result []CapturedNotification
	for _, n := range c.caps {
		if n.UserID == userID {
			result = append(result, n)
		}
	}
	return result
}

func (c *CaptureNotifier) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.caps)
}

func (c *CaptureNotifier) All() []CapturedNotification {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]CapturedNotification, len(c.caps))
	copy(result, c.caps)
	return result
}

// ── Mock Stripe ────────────────────────────────────────────────────────────────

// MockStripe records payment intent creations and webhook events.
type MockStripe struct {
	mu            sync.Mutex
	Intents       []StripeIntent
	WebhookEvents []WebhookEvent
	Refunds       []RefundRequest
	NextIntentID  string
	ShouldFail    bool
}

type StripeIntent struct {
	ID       string
	Amount   int64
	Currency string
	UserID   uuid.UUID
	Metadata map[string]string
}

type WebhookEvent struct {
	Type     string
	IntentID string
	Payload  map[string]interface{}
}

type RefundRequest struct {
	IntentID string
	Amount   int64
	Reason   string
}

func NewMockStripe() *MockStripe {
	return &MockStripe{
		NextIntentID: "pi_mock_1234567890",
	}
}

func (m *MockStripe) CreatePaymentIntent(userID uuid.UUID, amount int64, currency string, metadata map[string]string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ShouldFail {
		return "", fmt.Errorf("stripe: simulated failure")
	}
	id := m.NextIntentID
	m.Intents = append(m.Intents, StripeIntent{
		ID: id, Amount: amount, Currency: currency, UserID: userID, Metadata: metadata,
	})
	return id, nil
}

func (m *MockStripe) SimulateWebhook(eventType, intentID string, payload map[string]interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.WebhookEvents = append(m.WebhookEvents, WebhookEvent{
		Type: eventType, IntentID: intentID, Payload: payload,
	})
}

func (m *MockStripe) RecordRefund(intentID string, amount int64, reason string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Refunds = append(m.Refunds, RefundRequest{IntentID: intentID, Amount: amount, Reason: reason})
}

func (m *MockStripe) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Intents = nil
	m.WebhookEvents = nil
	m.Refunds = nil
	m.ShouldFail = false
}

// ── Mock FCM ───────────────────────────────────────────────────────────────────

// MockFCM records push notification sends.
type MockFCM struct {
	mu         sync.Mutex
	Messages   []FCMMessage
	ShouldFail bool
}

type FCMMessage struct {
	Token   string
	Title   string
	Body    string
	Data    map[string]string
	Success bool
}

func NewMockFCM() *MockFCM {
	return &MockFCM{}
}

func (m *MockFCM) Send(token, title, body string, data map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ShouldFail {
		m.Messages = append(m.Messages, FCMMessage{Token: token, Title: title, Body: body, Data: data, Success: false})
		return fmt.Errorf("fcm: simulated failure")
	}
	m.Messages = append(m.Messages, FCMMessage{Token: token, Title: title, Body: body, Data: data, Success: true})
	return nil
}

func (m *MockFCM) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Messages = nil
	m.ShouldFail = false
}

func (m *MockFCM) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Messages)
}

// ── Mock Email ─────────────────────────────────────────────────────────────────

// MockEmail records email sends without actually delivering.
type MockEmail struct {
	mu         sync.Mutex
	Emails     []SentEmail
	ShouldFail bool
}

type SentEmail struct {
	To      string
	Subject string
	Body    string
	Type    string // verification, welcome, reset, etc.
}

func NewMockEmail() *MockEmail {
	return &MockEmail{}
}

func (m *MockEmail) Send(to, subject, body, emailType string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ShouldFail {
		return fmt.Errorf("email: simulated failure")
	}
	m.Emails = append(m.Emails, SentEmail{To: to, Subject: subject, Body: body, Type: emailType})
	return nil
}

func (m *MockEmail) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Emails = nil
	m.ShouldFail = false
}

func (m *MockEmail) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Emails)
}

func (m *MockEmail) FindByType(emailType string) []SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []SentEmail
	for _, e := range m.Emails {
		if e.Type == emailType {
			result = append(result, e)
		}
	}
	return result
}

func (m *MockEmail) FindByRecipient(to string) []SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []SentEmail
	for _, e := range m.Emails {
		if e.To == to {
			result = append(result, e)
		}
	}
	return result
}
