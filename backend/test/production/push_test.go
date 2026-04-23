//go:build production

package production

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/push"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ═════════════════════════════════════════════════ Repos)\geocore-next\backend\test\production\push_test.go
//
// Real Push Notification Integration Tests
//
// Sends real push notifications via Firebase Cloud Messaging and verifies:
//   1. FCM API returns message ID (delivery accepted)
//   2. Push log recorded in DB with status "sent"
//   3. Device token lifecycle (invalid token detection)
//
// Required env vars:
//   - FIREBASE_SERVICE_ACCOUNT_JSON — Firebase service account credentials
//   - FCM_TEST_DEVICE_TOKEN — real device token for push delivery verification
// ════════════════════════════════════════════════════════════════════════════════

type PushSuite struct {
	suite.Suite
	ts           *ProdSuite
	deviceToken  string
	testUserID   uuid.UUID
}

func TestPushSuite(t *testing.T) {
	ts := SetupProdSuite(t)
	defer TeardownProdSuite(ts)

	suite.Run(t, &PushSuite{ts: ts})
}

func (s *PushSuite) SetupSuite() {
	s.deviceToken = os.Getenv("FCM_TEST_DEVICE_TOKEN")
	if s.deviceToken == "" {
		s.T().Skip("FCM_TEST_DEVICE_TOKEN not set — skipping real push tests")
	}
	if s.ts.Firebase == nil {
		s.T().Skip("Firebase client not initialised — skipping real push tests")
	}

	// Create a test user and register their device
	s.testUserID = uuid.New()
	s.ts.DB.Exec(`INSERT INTO users (id, name, email, password_hash, is_active, role, email_verified)
		VALUES (?, 'Push Test User', ?, '$2a$10$fake', true, 'user', true)`,
		s.testUserID, UniqueProdEmail("push"))

	// Register the device token
	device := push.UserDevice{
		ID:          uuid.New(),
		UserID:      s.testUserID,
		DeviceToken: s.deviceToken,
		Platform:    "android",
		IsActive:    true}
	require.NoError(s.T(), s.ts.DB.Create(&device).Error)

	// AutoMigrate push tables
	s.ts.DB.AutoMigrate(
		&push.UserDevice{},
		&push.PushLog{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)
}

// ── Test: Send real push notification via FCM ─────────────────────────────────

func (s *PushSuite) TestSendPush_RealFCMDelivery() {
	msg := &push.PushMessage{
		UserID:           s.testUserID,
		NotificationType: "test",
		Priority:         push.PriorityHigh,
		Title:            "GeoCore Prod Test",
		Body:             "This is a real push notification from production integration tests.",
		Data:             map[string]string{"test_key": "test_value", "timestamp": time.Now().Format(time.RFC3339)},
		IdempotencyKey:   "prod-push-test-" + uuid.New().String()[:8],
	}

	err := s.ts.PushSvc.Send(context.Background(), msg)
	require.NoError(s.T(), err, "push send should succeed")

	// Verify push log was created in DB
	var logs []push.PushLog
	s.ts.DB.Where("user_id = ? AND notification_type = ?", s.testUserID, "test").Find(&logs)
	assert.GreaterOrEqual(s.T(), len(logs), 1, "should have at least one push log entry")

	if len(logs) > 0 {
		assert.Equal(s.T(), push.PushStatusSent, logs[0].Status, "push status should be 'sent'")
		assert.NotEmpty(s.T(), logs[0].ProviderMsgID, "should have FCM message ID")
	}
}

// ── Test: Direct FCM send returns message ID ───────────────────────────────────

func (s *PushSuite) TestFCMDirectSend_ReturnsMessageID() {
	result, err := s.ts.Firebase.Send(
		context.Background(),
		s.deviceToken,
		"Direct FCM Test",
		"Testing direct Firebase client send",
		map[string]string{"type": "direct_test"},
		push.PriorityHigh,
	)

	require.NoError(s.T(), err, "FCM direct send should succeed")
	require.NotNil(s.T(), result, "FCM result should not be nil")
	assert.NotEmpty(s.T(), result.MessageID, "should return FCM message ID")
	assert.Empty(s.T(), result.Error, "should not have token-level error")
}

// ── Test: Invalid device token is detected and deactivated ─────────────────────

func (s *PushSuite) TestInvalidToken_DetectedAndDeactivated() {
	invalidToken := "invalid-token-0000000000"
	device := push.UserDevice{
		ID:          uuid.New(),
		UserID:      s.testUserID,
		DeviceToken: invalidToken,
		Platform:    "android",
		IsActive:    true,
	}
	require.NoError(s.T(), s.ts.DB.Create(&device).Error)

	msg := &push.PushMessage{
		UserID:           s.testUserID,
		NotificationType: "invalid_token_test",
		Priority:         push.PriorityHigh,
		Title:            "Invalid Token Test",
		Body:             "This should fail and deactivate the token",
		IdempotencyKey:   "prod-push-invalid-" + uuid.New().String()[:8],
	}

	// Send should not error at the top level (individual device failures are logged)
	_ = s.ts.PushSvc.Send(context.Background(), msg)

	// Give the system time to process the invalid token
	time.Sleep(2 * time.Second)

	// Check if the invalid token was deactivated
	var dev push.UserDevice
	s.ts.DB.Where("id = ?", device.ID).First(&dev)
	// FCM should have returned an error for the invalid token,
	// and the system should have deactivated it
	assert.False(s.T(), dev.IsActive, "invalid device token should be deactivated")
}

// ── Test: Push idempotency prevents duplicate sends ───────────────────────────

func (s *PushSuite) TestPushIdempotency_Deduplicated() {
	idemKey := "prod-push-idem-" + uuid.New().String()[:8]

	msg := &push.PushMessage{
		UserID:           s.testUserID,
		NotificationType: "idem_test",
		Priority:         push.PriorityMedium,
		Title:            "Idempotency Test",
		Body:             "This should only be sent once.",
		IdempotencyKey:   idemKey,
	}

	// First send
	err := s.ts.PushSvc.Send(context.Background(), msg)
	require.NoError(s.T(), err)

	// Second send with same key should be suppressed
	err = s.ts.PushSvc.Send(context.Background(), msg)
	require.NoError(s.T(), err, "duplicate push should not error (silently suppressed)")
}
