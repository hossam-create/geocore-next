package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/livestream"
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
// Scenario 9: Live Commerce — Session CRUD, join/leave, listing items, feed ranking
// ════════════════════════════════════════════════════════════════════════════════

type LiveSuite struct {
	suite.Suite
	ts     *TestSuite
	r      *gin.Engine
	hostID uuid.UUID
}

func TestLiveSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &LiveSuite{ts: ts})
}

func (s *LiveSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(
		&users.User{},
		&livestream.Session{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)

	livestream.RegisterRoutes(s.r.Group("/api/v1"), ts.DB, ts.RDB)

	s.hostID = ts.CreateUserWithEmailVerified("Live Host", UniqueEmail("live-host"))
}

func (s *LiveSuite) SetupTest() {
	s.ts.ResetTest()
	s.ts.DB.Exec("DELETE FROM livestream_sessions")
}

// ── Test: Create session ────────────────────────────────────────────────────────

func (s *LiveSuite) TestCreateSession_Success() {
	body, _ := json.Marshal(gin.H{
		"title":       "Test Live Session",
		"description": "Integration test live commerce session",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/livestream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.hostID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)
	assert.NotEmpty(s.T(), data["id"])

	// Verify session persisted
	var sessions []livestream.Session
	s.ts.DB.Where("host_id = ?", s.hostID).Find(&sessions)
	assert.Equal(s.T(), 1, len(sessions))
	assert.Equal(s.T(), livestream.StatusScheduled, sessions[0].Status)
}

// ── Test: Create session missing title ──────────────────────────────────────────

func (s *LiveSuite) TestCreateSession_MissingTitle() {
	body, _ := json.Marshal(gin.H{
		"description": "No title session",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/livestream", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.hostID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// ── Test: List sessions ────────────────────────────────────────────────────────

func (s *LiveSuite) TestListSessions() {
	s.createTestSession()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/livestream", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Get session by ID ────────────────────────────────────────────────────

func (s *LiveSuite) TestGetSession() {
	sessionID := s.createTestSession()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/livestream/"+sessionID.String(), nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Start session ────────────────────────────────────────────────────────

func (s *LiveSuite) TestStartSession() {
	sessionID := s.createTestSession()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/livestream/"+sessionID.String()+"/start", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.hostID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var session livestream.Session
	s.ts.DB.First(&session, "id = ?", sessionID)
	assert.Equal(s.T(), livestream.StatusLive, session.Status)
	assert.NotNil(s.T(), session.StartedAt)
}

// ── Test: End session ──────────────────────────────────────────────────────────

func (s *LiveSuite) TestEndSession() {
	sessionID := s.createTestLiveSession()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/livestream/"+sessionID.String()+"/end", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.hostID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var session livestream.Session
	s.ts.DB.First(&session, "id = ?", sessionID)
	assert.Equal(s.T(), livestream.StatusEnded, session.Status)
	assert.NotNil(s.T(), session.EndedAt)
}

// ── Test: Join session ─────────────────────────────────────────────────────────

func (s *LiveSuite) TestJoinSession() {
	sessionID := s.createTestLiveSession()
	viewerID := s.ts.CreateUserWithEmailVerified("Viewer", UniqueEmail("viewer"))

	body, _ := json.Marshal(gin.H{
		"display_name": "TestViewer",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/livestream/"+sessionID.String()+"/join", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(viewerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Cancel session ───────────────────────────────────────────────────────

func (s *LiveSuite) TestCancelSession() {
	sessionID := s.createTestSession()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/livestream/"+sessionID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.hostID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var session livestream.Session
	s.ts.DB.Unscoped().First(&session, "id = ?", sessionID)
	assert.Equal(s.T(), livestream.StatusCancelled, session.Status)
}

// ── Test: Feed endpoint returns results ─────────────────────────────────────────

func (s *LiveSuite) TestFeedEndpoint() {
	s.createTestLiveSession()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/livestream/feed", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Helpers ─────────────────────────────────────────────────────────────────────

func (s *LiveSuite) createTestSession() uuid.UUID {
	session := livestream.Session{
		ID:       uuid.New(),
		HostID:   s.hostID,
		Title:    "Test Session " + uuid.New().String()[:8],
		RoomName: "room-" + uuid.New().String()[:8],
		Status:   livestream.StatusScheduled,
	}
	require.NoError(s.T(), s.ts.DB.Create(&session).Error)
	return session.ID
}

func (s *LiveSuite) createTestLiveSession() uuid.UUID {
	now := time.Now()
	session := livestream.Session{
		ID:        uuid.New(),
		HostID:    s.hostID,
		Title:     "Live Session " + uuid.New().String()[:8],
		RoomName:  "room-" + uuid.New().String()[:8],
		Status:    livestream.StatusLive,
		StartedAt: &now,
	}
	require.NoError(s.T(), s.ts.DB.Create(&session).Error)
	return session.ID
}

func (s *LiveSuite) signToken(userID uuid.UUID) string {
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
