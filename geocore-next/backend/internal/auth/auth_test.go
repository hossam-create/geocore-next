package auth_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/geocore-next/backend/internal/auth"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── helpers ─────────────────────────────────────────────────────────────────

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&users.User{}))
	return db
}

func setupRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	auth.RegisterRoutes(v1, db, nil)
	return r
}

func setupRouterWithRedis(db *gorm.DB, rdb *redis.Client) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/api/v1")
	auth.RegisterRoutes(v1, db, rdb)
	return r
}

func jsonBody(t *testing.T, payload any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// post is a convenience helper for POST requests.
func post(t *testing.T, r *gin.Engine, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, path, jsonBody(t, payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

// postWithAuth is a convenience helper for authenticated POST requests.
func postWithAuth(t *testing.T, r *gin.Engine, path, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodPost, path, jsonBody(t, payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	return w
}

// getWithAuth is a convenience helper for authenticated GET requests.
func getWithAuth(t *testing.T, r *gin.Engine, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, path, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	return w
}

// parseData parses the "data" field from a standard API response.
func parseData(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var resp map[string]any
	require.NoError(t, json.Unmarshal(body, &resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "expected 'data' map in response; got: %s", string(body))
	return data
}

// registerUser registers a user and returns its access_token.
func registerUser(t *testing.T, r *gin.Engine, email, name, password string) string {
	t.Helper()
	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": name, "email": email, "password": password, "phone": "+971501234567",
	})
	require.Equal(t, http.StatusCreated, w.Code, "register failed: %s", w.Body.String())
	data := parseData(t, w.Body.Bytes())
	token, ok := data["access_token"].(string)
	require.True(t, ok && token != "", "expected non-empty access_token; got: %s", w.Body.String())
	return token
}

// makeExpiredToken signs a JWT that has already expired (for middleware tests).
func makeExpiredToken(t *testing.T, userID, email string) string {
	t.Helper()
	secret := os.Getenv("JWT_SECRET")
	claims := middleware.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	require.NoError(t, err)
	return signed
}

// ── Register tests ───────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "Ahmed Al-Farsi", "email": "ahmed@test.com",
		"password": "Secure123!", "phone": "+971501234567",
	})

	assert.Equal(t, http.StatusCreated, w.Code)
	data := parseData(t, w.Body.Bytes())
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
	assert.NotEmpty(t, data["message"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	payload := map[string]string{
		"name": "User A", "email": "dup@test.com",
		"password": "Password1!", "phone": "+971501234567",
	}
	w1 := post(t, r, "/api/v1/auth/register", payload)
	assert.Equal(t, http.StatusCreated, w1.Code)

	payload["name"] = "User B"
	payload["phone"] = "+971501234568"
	w2 := post(t, r, "/api/v1/auth/register", payload)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestRegister_WeakPassword_TooShort(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "User", "email": "user@test.com",
		"password": "weak", "phone": "+971501234567",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_MissingName(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"email": "noname@test.com", "password": "Secure123!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_MissingEmail(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "No Email", "password": "Secure123!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_InvalidEmail(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "Bad Email", "email": "not-an-email", "password": "Secure123!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_NameTooShort(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "A", "email": "short@test.com", "password": "Secure123!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_ResponseContainsTokens(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "New User", "email": "newuser@test.com",
		"password": "Secure123!", "phone": "+971501234567",
	})
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := parseData(t, w.Body.Bytes())
	assert.NotEmpty(t, data["access_token"], "access_token missing from register response")
	assert.NotEmpty(t, data["refresh_token"], "refresh_token missing from register response")
}

// ── Login tests ──────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "ali@test.com", "Ali Hassan", "Secure123!")

	w := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "ali@test.com", "password": "Secure123!",
	})
	assert.Equal(t, http.StatusOK, w.Code)
	data := parseData(t, w.Body.Bytes())
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["refresh_token"])
}

func TestLogin_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "fatima@test.com", "Fatima", "RightPass1!")

	w := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "fatima@test.com", "password": "WrongPass1!",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_NonexistentUser(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "ghost@test.com", "password": "Secure123!",
	})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_MissingPassword(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "missing@test.com",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_MissingEmail(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/login", map[string]string{
		"password": "Secure123!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_UnverifiedAccount_IncludesWarning(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "unverified@test.com", "Unverified User", "Secure123!")

	w := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "unverified@test.com", "password": "Secure123!",
	})
	require.Equal(t, http.StatusOK, w.Code)
	data := parseData(t, w.Body.Bytes())
	// Unverified users can still login but should get a warning
	assert.NotEmpty(t, data["access_token"])
	assert.NotEmpty(t, data["warning"], "expected warning field for unverified account")
}

// ── Me endpoint (auth required) ──────────────────────────────────────────────

func TestMe_Success(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	token := registerUser(t, r, "me@test.com", "Me User", "Secure123!")

	w := getWithAuth(t, r, "/api/v1/auth/me", token)
	assert.Equal(t, http.StatusOK, w.Code)
	data := parseData(t, w.Body.Bytes())
	assert.Equal(t, "me@test.com", data["email"])
}

func TestMe_NoToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMe_MalformedToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := getWithAuth(t, r, "/api/v1/auth/me", "not.a.valid.jwt")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMe_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "expiry@test.com", "Expiry User", "Secure123!")
	expiredTok := makeExpiredToken(t, "some-uuid", "expiry@test.com")

	w := getWithAuth(t, r, "/api/v1/auth/me", expiredTok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMe_InvalidAuthHeader_NoBearerPrefix(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	token := registerUser(t, r, "bearer@test.com", "Bearer User", "Secure123!")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Token "+token) // wrong scheme
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMe_InvalidAuthHeader_EmptyBearer(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── Email verification tests ─────────────────────────────────────────────────

func TestVerifyEmail_Success(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "verify@test.com", "Verify User", "Secure123!")

	// Retrieve the token from the DB (in production this comes via email)
	var user users.User
	require.NoError(t, db.Where("email = ?", "verify@test.com").First(&user).Error)
	require.NotEmpty(t, user.VerificationToken)

	w := post(t, r, "/api/v1/auth/verify-email", map[string]string{
		"token": user.VerificationToken,
	})
	assert.Equal(t, http.StatusOK, w.Code)
	data := parseData(t, w.Body.Bytes())
	assert.NotEmpty(t, data["access_token"], "fresh access_token should be returned on verification")
	assert.NotEmpty(t, data["refresh_token"], "fresh refresh_token should be returned on verification")

	// Confirm DB state was updated
	var updated users.User
	require.NoError(t, db.Where("email = ?", "verify@test.com").First(&updated).Error)
	assert.True(t, updated.EmailVerified)
	assert.Empty(t, updated.VerificationToken, "token should be consumed after use")
}

func TestVerifyEmail_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/verify-email", map[string]string{
		"token": "completely-invalid-token",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyEmail_AlreadyVerified(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "already@test.com", "Already Verified", "Secure123!")

	var user users.User
	require.NoError(t, db.Where("email = ?", "already@test.com").First(&user).Error)
	token := user.VerificationToken

	// First verification — should succeed
	w1 := post(t, r, "/api/v1/auth/verify-email", map[string]string{"token": token})
	require.Equal(t, http.StatusOK, w1.Code)

	// Second attempt with same token — token was consumed, so it's now invalid
	w2 := post(t, r, "/api/v1/auth/verify-email", map[string]string{"token": token})
	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestVerifyEmail_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "expired@test.com", "Expired Token", "Secure123!")

	// Manually expire the token by backdating the expiry
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, db.Model(&users.User{}).
		Where("email = ?", "expired@test.com").
		Update("verification_token_expires_at", past).Error)

	var user users.User
	require.NoError(t, db.Where("email = ?", "expired@test.com").First(&user).Error)

	w := post(t, r, "/api/v1/auth/verify-email", map[string]string{
		"token": user.VerificationToken,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyEmail_MissingToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/verify-email", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Password reset flow tests ─────────────────────────────────────────────────

func TestForgotPassword_RegisteredEmail(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "reset@test.com", "Reset User", "Secure123!")

	w := post(t, r, "/api/v1/auth/forgot-password", map[string]string{
		"email": "reset@test.com",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	// Confirm token was generated in the DB
	var user users.User
	require.NoError(t, db.Where("email = ?", "reset@test.com").First(&user).Error)
	assert.NotEmpty(t, user.PasswordResetToken)
	assert.NotNil(t, user.PasswordResetExpiresAt)
}

func TestForgotPassword_UnregisteredEmail(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	// Should return 200 even for unknown emails (prevents email enumeration)
	w := post(t, r, "/api/v1/auth/forgot-password", map[string]string{
		"email": "nobody@test.com",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestForgotPassword_InvalidEmail(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/forgot-password", map[string]string{
		"email": "not-an-email",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestForgotPassword_PerEmailRateLimit_WithRedis(t *testing.T) {
	// Use miniredis to simulate Redis and verify per-email rate limiting (SetNX).
	// The per-email limit (15 min) is enforced inside the handler independently of
	// the IP-level sliding-window middleware (3/hr). This test isolates the handler
	// logic by bypassing the IP limiter (it triggers on the 4th request; we only
	// make 2 here).
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	db := setupTestDB(t)
	r := setupRouterWithRedis(db, rdb)

	registerUser(t, r, "rl-email@test.com", "Per Email RL", "Secure123!")

	// First request — should succeed and set a 15-minute Redis key for this email
	w1 := post(t, r, "/api/v1/auth/forgot-password", map[string]string{
		"email": "rl-email@test.com",
	})
	assert.Equal(t, http.StatusOK, w1.Code, "first forgot-password request should succeed")

	// Second request for same email within 15 minutes — per-email SetNX rejects it
	w2 := post(t, r, "/api/v1/auth/forgot-password", map[string]string{
		"email": "rl-email@test.com",
	})
	assert.Equal(t, http.StatusTooManyRequests, w2.Code,
		"second forgot-password for same email within 15 min should be rate-limited")
}

func TestValidateResetToken_ValidToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "validate@test.com", "Validate User", "Secure123!")
	post(t, r, "/api/v1/auth/forgot-password", map[string]string{
		"email": "validate@test.com",
	})

	var user users.User
	require.NoError(t, db.Where("email = ?", "validate@test.com").First(&user).Error)
	require.NotEmpty(t, user.PasswordResetToken)

	w := post(t, r, "/api/v1/auth/validate-reset-token", map[string]string{
		"token": user.PasswordResetToken,
	})
	assert.Equal(t, http.StatusOK, w.Code)
	data := parseData(t, w.Body.Bytes())
	assert.Equal(t, true, data["valid"])
	assert.NotEmpty(t, data["email"], "masked email should be returned")
}

func TestValidateResetToken_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/validate-reset-token", map[string]string{
		"token": "invalid-token-xyz",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestValidateResetToken_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "expired-reset@test.com", "Expired Reset", "Secure123!")

	// Directly insert an expired reset token
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, db.Model(&users.User{}).
		Where("email = ?", "expired-reset@test.com").
		Updates(map[string]any{
			"password_reset_token":      "expired-test-token-abc123",
			"password_reset_expires_at": past,
		}).Error)

	w := post(t, r, "/api/v1/auth/validate-reset-token", map[string]string{
		"token": "expired-test-token-abc123",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_Success(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "pwreset@test.com", "PW Reset", "OldPass1!")

	// Set a known reset token directly (simulating forgot-password without Redis)
	future := time.Now().Add(time.Hour)
	require.NoError(t, db.Model(&users.User{}).
		Where("email = ?", "pwreset@test.com").
		Updates(map[string]any{
			"password_reset_token":      "valid-reset-token-abc123",
			"password_reset_expires_at": future,
		}).Error)

	w := post(t, r, "/api/v1/auth/reset-password", map[string]string{
		"token":            "valid-reset-token-abc123",
		"new_password":     "NewPass1!",
		"confirm_password": "NewPass1!",
	})
	assert.Equal(t, http.StatusOK, w.Code)

	// Confirm token was cleared
	var user users.User
	require.NoError(t, db.Where("email = ?", "pwreset@test.com").First(&user).Error)
	assert.Empty(t, user.PasswordResetToken, "token should be consumed after use")

	// Confirm new password works
	wl := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "pwreset@test.com", "password": "NewPass1!",
	})
	assert.Equal(t, http.StatusOK, wl.Code, "should be able to login with new password")

	// Confirm old password no longer works
	wl2 := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "pwreset@test.com", "password": "OldPass1!",
	})
	assert.Equal(t, http.StatusUnauthorized, wl2.Code, "old password should be rejected")
}

func TestResetPassword_PasswordMismatch(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "mismatch@test.com", "Mismatch", "Secure123!")
	future := time.Now().Add(time.Hour)
	require.NoError(t, db.Model(&users.User{}).
		Where("email = ?", "mismatch@test.com").
		Updates(map[string]any{
			"password_reset_token":      "mismatch-token-xyz",
			"password_reset_expires_at": future,
		}).Error)

	w := post(t, r, "/api/v1/auth/reset-password", map[string]string{
		"token":            "mismatch-token-xyz",
		"new_password":     "NewPass1!",
		"confirm_password": "Different1!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_WeakNewPassword(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "weaknew@test.com", "Weak New", "Secure123!")
	future := time.Now().Add(time.Hour)
	require.NoError(t, db.Model(&users.User{}).
		Where("email = ?", "weaknew@test.com").
		Updates(map[string]any{
			"password_reset_token":      "weaknew-token-xyz",
			"password_reset_expires_at": future,
		}).Error)

	cases := []struct {
		name     string
		password string
	}{
		{"too short", "Ab1!"},
		{"no uppercase", "alllower1!"},
		{"no lowercase", "ALLUPPER1!"},
		{"no digit", "NoDigitHere!"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := post(t, r, "/api/v1/auth/reset-password", map[string]string{
				"token":            "weaknew-token-xyz",
				"new_password":     tc.password,
				"confirm_password": tc.password,
			})
			assert.Equal(t, http.StatusBadRequest, w.Code, "expected rejection for weak password: %s", tc.password)
		})
	}
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "expiredreset@test.com", "Expired Reset", "Secure123!")
	past := time.Now().Add(-2 * time.Hour)
	require.NoError(t, db.Model(&users.User{}).
		Where("email = ?", "expiredreset@test.com").
		Updates(map[string]any{
			"password_reset_token":      "expired-reset-token-xyz",
			"password_reset_expires_at": past,
		}).Error)

	w := post(t, r, "/api/v1/auth/reset-password", map[string]string{
		"token":            "expired-reset-token-xyz",
		"new_password":     "NewPass1!",
		"confirm_password": "NewPass1!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := post(t, r, "/api/v1/auth/reset-password", map[string]string{
		"token":            "nonexistent-token-xyz",
		"new_password":     "NewPass1!",
		"confirm_password": "NewPass1!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestResetPassword_TokenConsumedAfterUse(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	registerUser(t, r, "consumed@test.com", "Consumed Token", "Secure123!")
	future := time.Now().Add(time.Hour)
	require.NoError(t, db.Model(&users.User{}).
		Where("email = ?", "consumed@test.com").
		Updates(map[string]any{
			"password_reset_token":      "one-time-token-xyz",
			"password_reset_expires_at": future,
		}).Error)

	// First use — should succeed
	w1 := post(t, r, "/api/v1/auth/reset-password", map[string]string{
		"token":            "one-time-token-xyz",
		"new_password":     "NewPass1!",
		"confirm_password": "NewPass1!",
	})
	require.Equal(t, http.StatusOK, w1.Code)

	// Second use — token should be consumed and rejected
	w2 := post(t, r, "/api/v1/auth/reset-password", map[string]string{
		"token":            "one-time-token-xyz",
		"new_password":     "AnotherNew1!",
		"confirm_password": "AnotherNew1!",
	})
	assert.Equal(t, http.StatusBadRequest, w2.Code, "token should be rejected after first use")
}

func TestResetPassword_MissingFields(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	// Missing confirm_password
	w := post(t, r, "/api/v1/auth/reset-password", map[string]string{
		"token":        "some-token",
		"new_password": "NewPass1!",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── JWT middleware rejection tests ───────────────────────────────────────────

func TestJWTMiddleware_MissingHeader(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["error"])
}

func TestJWTMiddleware_GarbageToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	w := getWithAuth(t, r, "/api/v1/auth/me", "garbage.token.value")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	tok := makeExpiredToken(t, "00000000-0000-0000-0000-000000000001", "expired@test.com")
	w := getWithAuth(t, r, "/api/v1/auth/me", tok)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_WrongAlgorithm(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	// Sign with RS256 (asymmetric) — server expects HS256
	privKey, err := generateRSAKey()
	require.NoError(t, err)

	claims := middleware.Claims{
		UserID: "some-user-id",
		Email:  "alg@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(privKey)
	require.NoError(t, err)

	w := getWithAuth(t, r, "/api/v1/auth/me", signed)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTMiddleware_WrongSecret(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	// Sign with a different secret
	claims := middleware.Claims{
		UserID: "some-user-id",
		Email:  "wrongsecret@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte("wrong-secret-key"))
	require.NoError(t, err)

	// Only matters if JWT_SECRET env is non-empty; skip if blank (both would use empty)
	if os.Getenv("JWT_SECRET") == "" {
		t.Skip("JWT_SECRET is empty; wrong-secret test is only meaningful with a set secret")
	}

	w := getWithAuth(t, r, "/api/v1/auth/me", signed)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ── Full password reset flow (end-to-end) ────────────────────────────────────

func TestPasswordResetFlow_EndToEnd(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	// 1. Register
	registerUser(t, r, "flow@test.com", "Flow User", "Original1!")

	// 2. Inject a reset token directly (since Redis is nil, forgot-password skips rate-limiting)
	future := time.Now().Add(time.Hour)
	resetToken := "e2e-flow-token-abc123"
	require.NoError(t, db.Model(&users.User{}).
		Where("email = ?", "flow@test.com").
		Updates(map[string]any{
			"password_reset_token":      resetToken,
			"password_reset_expires_at": future,
		}).Error)

	// 3. Validate token
	wv := post(t, r, "/api/v1/auth/validate-reset-token", map[string]string{
		"token": resetToken,
	})
	require.Equal(t, http.StatusOK, wv.Code, "validate-reset-token should succeed")

	// 4. Reset password
	wr := post(t, r, "/api/v1/auth/reset-password", map[string]string{
		"token":            resetToken,
		"new_password":     "NewFlow1!",
		"confirm_password": "NewFlow1!",
	})
	require.Equal(t, http.StatusOK, wr.Code, "reset-password should succeed")

	// 5. Login with new password
	wl := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "flow@test.com", "password": "NewFlow1!",
	})
	assert.Equal(t, http.StatusOK, wl.Code, "login with new password should succeed")

	// 6. Old password should fail
	wl2 := post(t, r, "/api/v1/auth/login", map[string]string{
		"email": "flow@test.com", "password": "Original1!",
	})
	assert.Equal(t, http.StatusUnauthorized, wl2.Code, "old password should be rejected")
}

// ── Rate limit middleware ─────────────────────────────────────────────────────

func TestRateLimiter_FailOpenWhenRedisNil(t *testing.T) {
	// With nil Redis the sliding-window middleware fails open — requests pass through.
	db := setupTestDB(t)
	r := setupRouter(db)

	email := fmt.Sprintf("rl%d@test.com", time.Now().UnixNano())
	w1 := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "Rate Test", "email": email, "password": "RateTest1!",
	})
	assert.NotEqual(t, http.StatusTooManyRequests, w1.Code,
		"rate limiter should fail open (allow request) when Redis is nil")
}

func TestRateLimiter_Login_IPRateLimit_WithRedis(t *testing.T) {
	// Verify that the IP-based sliding-window rate limiter blocks login after
	// exceeding the configured limit (10 requests per 15 minutes).
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	db := setupTestDB(t)
	r := setupRouterWithRedis(db, rdb)

	registerUser(t, r, "ip-rl@test.com", "IP Rate Limit", "Secure123!")

	// Exhaust the 10-request-per-15-min login rate limit
	var lastCode int
	for i := 0; i < 12; i++ {
		w := post(t, r, "/api/v1/auth/login", map[string]string{
			"email": "ip-rl@test.com", "password": "Secure123!",
		})
		lastCode = w.Code
		if w.Code == http.StatusTooManyRequests {
			break
		}
	}
	assert.Equal(t, http.StatusTooManyRequests, lastCode,
		"login should be rate-limited after exceeding 10 requests in 15 minutes")
}

func TestRateLimiter_ForgotPassword_IPRateLimit_WithRedis(t *testing.T) {
	// Verify the IP-level rate limiter on /forgot-password (3 requests per hour).
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	db := setupTestDB(t)
	r := setupRouterWithRedis(db, rdb)

	// Use a different email each time so the per-email rate limit doesn't trigger first
	var lastCode int
	for i := 0; i < 5; i++ {
		w := post(t, r, "/api/v1/auth/forgot-password", map[string]string{
			"email": fmt.Sprintf("noone%d@test.com", i),
		})
		lastCode = w.Code
		if w.Code == http.StatusTooManyRequests {
			break
		}
	}
	assert.Equal(t, http.StatusTooManyRequests, lastCode,
		"forgot-password should be rate-limited after exceeding 3 IP requests per hour")
}

// ── Token refresh endpoint ───────────────────────────────────────────────────

func TestRefresh_Success(t *testing.T) {
	// With nil Redis the refresh token is issued but not stored, so re-use will
	// fail. Here we verify the endpoint exists and returns 200 with a Redis-backed
	// token store.
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	db := setupTestDB(t)
	r := setupRouterWithRedis(db, rdb)

	// Register and capture refresh_token
	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "Refresh User", "email": "refresh@test.com",
		"password": "Secure123!", "phone": "+971501234567",
	})
	require.Equal(t, http.StatusCreated, w.Code)
	data := parseData(t, w.Body.Bytes())
	refreshToken, ok := data["refresh_token"].(string)
	require.True(t, ok && refreshToken != "", "refresh_token must be non-empty")

	// Exchange for new token pair
	wr := post(t, r, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	})
	assert.Equal(t, http.StatusOK, wr.Code, "refresh should succeed: %s", wr.Body.String())
	refreshData := parseData(t, wr.Body.Bytes())
	assert.NotEmpty(t, refreshData["access_token"], "new access_token expected")
	assert.NotEmpty(t, refreshData["refresh_token"], "new refresh_token expected (rotation)")
}

func TestRefresh_InvalidToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	wr := post(t, r, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": "invalid.jwt.token",
	})
	assert.Equal(t, http.StatusUnauthorized, wr.Code)
}

func TestRefresh_ExpiredToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	expiredRefresh := makeExpiredToken(t, "some-uuid", "expiredrefresh@test.com")
	wr := post(t, r, "/api/v1/auth/refresh", map[string]string{
		"refresh_token": expiredRefresh,
	})
	assert.Equal(t, http.StatusUnauthorized, wr.Code)
}

func TestRefresh_MissingToken(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	wr := post(t, r, "/api/v1/auth/refresh", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, wr.Code)
}

func TestRefresh_OldTokenRejectedAfterRotation(t *testing.T) {
	// After a successful token rotation, re-using the original refresh token
	// should be treated as theft and return 401.
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	db := setupTestDB(t)
	r := setupRouterWithRedis(db, rdb)

	w := post(t, r, "/api/v1/auth/register", map[string]string{
		"name": "Rotate User", "email": "rotate@test.com",
		"password": "Secure123!", "phone": "+971501234567",
	})
	require.Equal(t, http.StatusCreated, w.Code)
	data := parseData(t, w.Body.Bytes())
	oldRefresh := data["refresh_token"].(string)

	// First rotation — succeeds, old token is consumed
	wr1 := post(t, r, "/api/v1/auth/refresh", map[string]string{"refresh_token": oldRefresh})
	require.Equal(t, http.StatusOK, wr1.Code)

	// Re-using the old token — should be treated as theft
	wr2 := post(t, r, "/api/v1/auth/refresh", map[string]string{"refresh_token": oldRefresh})
	assert.Equal(t, http.StatusUnauthorized, wr2.Code,
		"re-using a rotated refresh token should be rejected as token theft")
}

// ── Helper: RSA key for wrong-algorithm test ─────────────────────────────────

func generateRSAKey() (interface{}, error) {
	rsaCrypto, err := rsa.GenerateKey(rand.Reader, 2048)
	return rsaCrypto, err
}

// Ensure helpers used in other tests are not flagged as unused
var _ = postWithAuth
