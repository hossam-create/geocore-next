package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/auth"
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
// Scenario 1: Auth — Register, Login, Verify Email, Refresh Token
// ════════════════════════════════════════════════════════════════════════════════

type AuthSuite struct {
	suite.Suite
	ts *TestSuite
	r  *gin.Engine
	h  *auth.Handler
	v1 *gin.RouterGroup
}

func TestAuthSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &AuthSuite{ts: ts})
}

func (s *AuthSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(
		&users.User{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)

	s.h = auth.NewHandler(ts.DB, ts.RDB)
	s.v1 = s.r.Group("/api/v1")
	auth.RegisterRoutes(s.v1, ts.DB, ts.RDB)
}

func (s *AuthSuite) SetupTest() {
	s.ts.ResetTest()
	// Clean user table between tests
	s.ts.DB.Exec("DELETE FROM users")
	s.ts.DB.Exec("DELETE FROM outbox_events")
}

// ── Test: Register with valid data ──────────────────────────────────────────────

func (s *AuthSuite) TestRegister_Success() {
	email := UniqueEmail("auth")
	body, _ := json.Marshal(gin.H{
		"name":     "Test User",
		"email":    email,
		"password": "Str0ng!Pass#1",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok, "response should have data object")
	assert.NotEmpty(s.T(), data["access_token"])
	assert.NotEmpty(s.T(), data["refresh_token"])

	userData, ok := data["user"].(map[string]interface{})
	require.True(s.T(), ok)
	assert.Equal(s.T(), email, userData["email"])
	assert.Equal(s.T(), false, userData["email_verified"])
}

// ── Test: Register with duplicate email ─────────────────────────────────────────

func (s *AuthSuite) TestRegister_DuplicateEmail() {
	email := UniqueEmail("dup")
	body1, _ := json.Marshal(gin.H{
		"name":     "First User",
		"email":    email,
		"password": "Str0ng!Pass#1",
	})
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w1, req1)
	assert.Equal(s.T(), http.StatusCreated, w1.Code)

	body2, _ := json.Marshal(gin.H{
		"name":     "Second User",
		"email":    email,
		"password": "Str0ng!Pass#2",
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w2, req2)
	assert.Equal(s.T(), http.StatusConflict, w2.Code)
}

// ── Test: Register with weak password ───────────────────────────────────────────

func (s *AuthSuite) TestRegister_WeakPassword() {
	email := UniqueEmail("weak")
	body, _ := json.Marshal(gin.H{
		"name":     "Weak Pass User",
		"email":    email,
		"password": "123456",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// ── Test: Login with valid credentials ──────────────────────────────────────────

func (s *AuthSuite) TestLogin_Success() {
	email := UniqueEmail("login")
	// Register first
	regBody, _ := json.Marshal(gin.H{
		"name":     "Login User",
		"email":    email,
		"password": "Str0ng!Pass#1",
	})
	wReg := httptest.NewRecorder()
	reqReg, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(regBody))
	reqReg.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(wReg, reqReg)
	require.Equal(s.T(), http.StatusCreated, wReg.Code)

	// Now login
	loginBody, _ := json.Marshal(gin.H{
		"email":    email,
		"password": "Str0ng!Pass#1",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)
	assert.NotEmpty(s.T(), data["access_token"])
}

// ── Test: Login with wrong password ─────────────────────────────────────────────

func (s *AuthSuite) TestLogin_WrongPassword() {
	email := UniqueEmail("wrongpass")
	regBody, _ := json.Marshal(gin.H{
		"name":     "Wrong Pass User",
		"email":    email,
		"password": "Str0ng!Pass#1",
	})
	wReg := httptest.NewRecorder()
	reqReg, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(regBody))
	reqReg.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(wReg, reqReg)
	require.Equal(s.T(), http.StatusCreated, wReg.Code)

	loginBody, _ := json.Marshal(gin.H{
		"email":    email,
		"password": "WrongPass999!",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// ── Test: Verify email with valid token ─────────────────────────────────────────

func (s *AuthSuite) TestVerifyEmail_ValidToken() {
	token := "valid-test-token-abc123"
	expiresAt := time.Now().Add(24 * time.Hour)
	userID := s.ts.CreateUserWithVerificationToken("Verify User", UniqueEmail("verify"), token, expiresAt)

	verifyBody, _ := json.Marshal(gin.H{"token": token})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/verify-email", bytes.NewReader(verifyBody))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Verify user is now email_verified
	var user users.User
	require.NoError(s.T(), s.ts.DB.First(&user, "id = ?", userID).Error)
	assert.True(s.T(), user.EmailVerified)
	assert.Empty(s.T(), user.VerificationToken)
}

// ── Test: Verify email with expired token ────────────────────────────────────────

func (s *AuthSuite) TestVerifyEmail_ExpiredToken() {
	token := "expired-test-token-xyz789"
	expiresAt := time.Now().Add(-1 * time.Hour) // expired 1 hour ago
	s.ts.CreateUserWithVerificationToken("Expired User", UniqueEmail("expired"), token, expiresAt)

	verifyBody, _ := json.Marshal(gin.H{"token": token})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/verify-email", bytes.NewReader(verifyBody))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// ── Test: Verify email with invalid token ───────────────────────────────────────

func (s *AuthSuite) TestVerifyEmail_InvalidToken() {
	verifyBody, _ := json.Marshal(gin.H{"token": "nonexistent-token"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/verify-email", bytes.NewReader(verifyBody))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// ── Test: Resend verification rate limited ──────────────────────────────────────

func (s *AuthSuite) TestResendVerification_RateLimited() {
	userID := s.ts.CreateUser("Resend User", UniqueEmail("resend"))

	// First resend should succeed
	resendBody, _ := json.Marshal(gin.H{})
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/auth/resend-verification", bytes.NewReader(resendBody))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+s.generateToken(userID))
	s.r.ServeHTTP(w1, req1)
	assert.Equal(s.T(), http.StatusOK, w1.Code)

	// Immediate second resend should be rate limited
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/auth/resend-verification", bytes.NewReader(resendBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+s.generateToken(userID))
	s.r.ServeHTTP(w2, req2)
	assert.Equal(s.T(), http.StatusBadRequest, w2.Code)
}

// ── Helper: generate a test JWT ─────────────────────────────────────────────────

func (s *AuthSuite) generateToken(userID uuid.UUID) string {
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
