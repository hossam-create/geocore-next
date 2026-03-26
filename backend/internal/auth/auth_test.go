package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geocore-next/backend/internal/auth"
	"github.com/geocore-next/backend/internal/users"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ── helpers ─────────────────────────────────────────────────────────────────

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
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

func jsonBody(t *testing.T, payload any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	body := jsonBody(t, map[string]string{
		"name":     "Ahmed Al-Farsi",
		"email":    "ahmed@test.com",
		"password": "Secure123!",
		"phone":    "+971501234567",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok)
	assert.NotEmpty(t, data["token"])
}

func TestRegister_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	body := jsonBody(t, map[string]string{
		"name": "User A", "email": "dup@test.com",
		"password": "Password1!", "phone": "+971501234567",
	})

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusCreated, w1.Code)

	body = jsonBody(t, map[string]string{
		"name": "User B", "email": "dup@test.com",
		"password": "Password1!", "phone": "+971501234568",
	})
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusConflict, w2.Code)
}

func TestRegister_WeakPassword(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	body := jsonBody(t, map[string]string{
		"name": "User", "email": "user@test.com",
		"password": "weak", "phone": "+971501234567",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	// Register first
	rBody := jsonBody(t, map[string]string{
		"name": "Ali Hassan", "email": "ali@test.com",
		"password": "Secure123!", "phone": "+971501234567",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", rBody)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Login
	lBody := jsonBody(t, map[string]string{
		"email": "ali@test.com", "password": "Secure123!",
	})
	wl := httptest.NewRecorder()
	lReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", lBody)
	lReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wl, lReq)
	assert.Equal(t, http.StatusOK, wl.Code)
}

func TestLogin_WrongPassword(t *testing.T) {
	db := setupTestDB(t)
	r := setupRouter(db)

	// Register
	rBody := jsonBody(t, map[string]string{
		"name": "Fatima", "email": "fatima@test.com",
		"password": "RightPass1!", "phone": "+971501234567",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/register", rBody)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	// Login with wrong password
	lBody := jsonBody(t, map[string]string{
		"email": "fatima@test.com", "password": "WrongPass1!",
	})
	wl := httptest.NewRecorder()
	lReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", lBody)
	lReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(wl, lReq)
	assert.Equal(t, http.StatusUnauthorized, wl.Code)
}
