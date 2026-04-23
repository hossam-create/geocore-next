package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/jwtkeys"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ════════════════════════════════════════════════════════════════════════════════
// Scenario 6: Notifications — Create, list, mark read, unread count, preferences
// ════════════════════════════════════════════════════════════════════════════════

type NotificationsSuite struct {
	suite.Suite
	ts     *TestSuite
	r      *gin.Engine
	userID uuid.UUID
	svc    *notifications.Service
}

func TestNotificationsSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &NotificationsSuite{ts: ts})
}

func (s *NotificationsSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(
		&users.User{},
		&notifications.Notification{},
		&notifications.NotificationPreference{},
		&notifications.PushToken{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)

	hub, svc := notifications.RegisterRoutes(s.r.Group("/api/v1"), ts.DB, ts.RDB)
	s.svc = svc
	_ = hub

	s.userID = ts.CreateUserWithEmailVerified("Notif User", UniqueEmail("notif"))
}

func (s *NotificationsSuite) SetupTest() {
	s.ts.ResetTest()
	s.ts.DB.Exec("DELETE FROM notifications")
}

// ── Test: Service.Notify creates in-app notification ───────────────────────────

func (s *NotificationsSuite) TestNotify_CreatesInAppNotification() {
	s.svc.Notify(notifications.NotifyInput{
		UserID: s.userID,
		Type:   "order.created",
		Title:  "Order Created",
		Body:   "Your order #123 has been created",
		Data:   map[string]string{"order_id": uuid.New().String()},
	})

	var notifs []notifications.Notification
	s.ts.DB.Where("user_id = ?", s.userID).Find(&notifs)
	assert.Equal(s.T(), 1, len(notifs), "should create one in-app notification")
	assert.Equal(s.T(), "order.created", notifs[0].Type)
	assert.False(s.T(), notifs[0].Read)
}

// ── Test: List notifications ───────────────────────────────────────────────────

func (s *NotificationsSuite) TestListNotifications() {
	// Seed notifications
	s.svc.Notify(notifications.NotifyInput{
		UserID: s.userID,
		Type:   "bid.placed",
		Title:  "New Bid",
		Body:   "Someone placed a bid",
	})
	s.svc.Notify(notifications.NotifyInput{
		UserID: s.userID,
		Type:   "outbid",
		Title:  "You've been outbid",
		Body:   "Someone outbid you",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].([]interface{})
	require.True(s.T(), ok)
	assert.GreaterOrEqual(s.T(), len(data), 2, "should return at least 2 notifications")
}

// ── Test: Unread count ──────────────────────────────────────────────────────────

func (s *NotificationsSuite) TestUnreadCount() {
	s.svc.Notify(notifications.NotifyInput{
		UserID: s.userID,
		Type:   "test",
		Title:  "Test",
		Body:   "Test body",
	})
	s.svc.Notify(notifications.NotifyInput{
		UserID: s.userID,
		Type:   "test2",
		Title:  "Test 2",
		Body:   "Test body 2",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/unread-count", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)
	count, _ := data["unread_count"].(float64)
	assert.Equal(s.T(), 2.0, count, "should have 2 unread notifications")
}

// ── Test: Mark notification as read ────────────────────────────────────────────

func (s *NotificationsSuite) TestMarkRead() {
	s.svc.Notify(notifications.NotifyInput{
		UserID: s.userID,
		Type:   "test",
		Title:  "Test",
		Body:   "Test body",
	})

	var notif notifications.Notification
	s.ts.DB.Where("user_id = ?", s.userID).First(&notif)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/"+notif.ID.String()+"/read", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Verify notification is now read
	s.ts.DB.First(&notif, notif.ID)
	assert.True(s.T(), notif.Read)
}

// ── Test: Mark all as read ──────────────────────────────────────────────────────

func (s *NotificationsSuite) TestMarkAllRead() {
	s.svc.Notify(notifications.NotifyInput{UserID: s.userID, Type: "a", Title: "A", Body: "A"})
	s.svc.Notify(notifications.NotifyInput{UserID: s.userID, Type: "b", Title: "B", Body: "B"})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/mark-all-read", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var count int64
	s.ts.DB.Model(&notifications.Notification{}).Where("user_id = ? AND read = false", s.userID).Count(&count)
	assert.Equal(s.T(), int64(0), count, "all notifications should be read")
}

// ── Test: Delete notification ──────────────────────────────────────────────────

func (s *NotificationsSuite) TestDeleteNotification() {
	s.svc.Notify(notifications.NotifyInput{UserID: s.userID, Type: "del", Title: "Delete", Body: "Delete me"})

	var notif notifications.Notification
	s.ts.DB.Where("user_id = ?", s.userID).First(&notif)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/notifications/"+notif.ID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Get preferences ──────────────────────────────────────────────────────

func (s *NotificationsSuite) TestGetPreferences() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/notifications/preferences", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Update preferences ────────────────────────────────────────────────────

func (s *NotificationsSuite) TestUpdatePreferences() {
	body, _ := json.Marshal(gin.H{
		"email_new_bid": false,
		"email_outbid":  true,
		"push_new_bid":  true,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/notifications/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var prefs notifications.NotificationPreference
	s.ts.DB.First(&prefs, "user_id = ?", s.userID)
	assert.False(s.T(), prefs.EmailNewBid)
	assert.True(s.T(), prefs.EmailOutbid)
}

// ── Helper: sign a test JWT ─────────────────────────────────────────────────────

func (s *NotificationsSuite) signToken(userID uuid.UUID) string {
	claims := middleware.Claims{
		UserID: userID.String(),
		Email:  "test@test.geocore.dev",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(jwtkeys.Private())
	if err != nil {
		s.T().Fatalf("failed to sign test token: %v", err)
	}
	return signed
}
