package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/chat"
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
// Scenario 8: Chat — Conversations, messages, WebSocket tokens, unread counts
// ════════════════════════════════════════════════════════════════════════════════

type ChatSuite struct {
	suite.Suite
	ts      *TestSuite
	r       *gin.Engine
	userAID uuid.UUID
	userBID uuid.UUID
	convID  uuid.UUID
}

func TestChatSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &ChatSuite{ts: ts})
}

func (s *ChatSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(
		&users.User{},
		&chat.Conversation{},
		&chat.ConversationMember{},
		&chat.Message{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)

	chat.RegisterRoutes(s.r.Group("/api/v1"), ts.DB, ts.RDB)

	s.userAID = ts.CreateUserWithEmailVerified("Chat User A", UniqueEmail("chat-a"))
	s.userBID = ts.CreateUserWithEmailVerified("Chat User B", UniqueEmail("chat-b"))
}

func (s *ChatSuite) SetupTest() {
	s.ts.ResetTest()
	s.ts.DB.Exec("DELETE FROM messages")
	s.ts.DB.Exec("DELETE FROM conversation_members")
	s.ts.DB.Exec("DELETE FROM conversations")
}

// ── Test: Generate WebSocket token ──────────────────────────────────────────────

func (s *ChatSuite) TestGenerateWSToken() {
	body, _ := json.Marshal(gin.H{})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/chat/ws-token", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userAID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)
	assert.NotEmpty(s.T(), data["token"], "should return a WebSocket token")
}

// ── Test: Create or get conversation ────────────────────────────────────────────

func (s *ChatSuite) TestCreateOrGetConversation() {
	body, _ := json.Marshal(gin.H{
		"participant_ids": []string{s.userAID.String(), s.userBID.String()},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userAID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)
	assert.NotEmpty(s.T(), data["id"], "should return conversation ID")

	s.convID, _ = uuid.Parse(data["id"].(string))
}

// ── Test: Send message ──────────────────────────────────────────────────────────

func (s *ChatSuite) TestSendMessage() {
	// Create conversation first
	convID := s.createConversation()

	body, _ := json.Marshal(gin.H{
		"content": "Hello from integration test!",
		"type":    "text",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/chat/conversations/"+convID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userAID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)

	// Verify message was persisted
	var msgs []chat.Message
	s.ts.DB.Where("conversation_id = ?", convID).Find(&msgs)
	assert.Equal(s.T(), 1, len(msgs), "should persist one message")
	assert.Equal(s.T(), "Hello from integration test!", msgs[0].Content)
}

// ── Test: Get conversations list ────────────────────────────────────────────────

func (s *ChatSuite) TestGetConversations() {
	s.createConversation()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/chat/conversations", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userAID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].([]interface{})
	require.True(s.T(), ok)
	assert.GreaterOrEqual(s.T(), len(data), 1, "should return at least one conversation")
}

// ── Test: Get messages for conversation ──────────────────────────────────────────

func (s *ChatSuite) TestGetMessages() {
	convID := s.createConversation()
	s.sendMessage(convID, s.userAID, "First message")
	s.sendMessage(convID, s.userBID, "Second message")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/chat/conversations/"+convID.String()+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userAID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Send message to non-member conversation fails ─────────────────────────

func (s *ChatSuite) TestSendMessage_NonMember() {
	convID := s.createConversation()
	otherUserID := s.ts.CreateUserWithEmailVerified("Other User", UniqueEmail("other"))

	body, _ := json.Marshal(gin.H{
		"content": "I shouldn't be able to send this",
		"type":    "text",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/chat/conversations/"+convID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(otherUserID))
	s.r.ServeHTTP(w, req)

	assert.NotEqual(s.T(), http.StatusCreated, w.Code, "non-member should not be able to send message")
}

// ── Helpers ─────────────────────────────────────────────────────────────────────

func (s *ChatSuite) createConversation() uuid.UUID {
	body, _ := json.Marshal(gin.H{
		"participant_ids": []string{s.userAID.String(), s.userBID.String()},
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/chat/conversations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userAID))
	s.r.ServeHTTP(w, req)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	id, _ := uuid.Parse(data["id"].(string))
	return id
}

func (s *ChatSuite) sendMessage(convID, senderID uuid.UUID, content string) {
	body, _ := json.Marshal(gin.H{
		"content": content,
		"type":    "text",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/chat/conversations/"+convID.String()+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(senderID))
	s.r.ServeHTTP(w, req)
}

func (s *ChatSuite) signToken(userID uuid.UUID) string {
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
