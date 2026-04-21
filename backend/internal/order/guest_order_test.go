package order

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupGuestOrderTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			id TEXT PRIMARY KEY,
			buyer_id TEXT NOT NULL,
			seller_id TEXT NOT NULL,
			payment_intent_id TEXT,
			payment_id TEXT,
			status TEXT NOT NULL,
			status_history TEXT,
			subtotal REAL NOT NULL,
			platform_fee REAL NOT NULL,
			payment_fee REAL NOT NULL,
			total REAL NOT NULL,
			currency TEXT NOT NULL,
			shipping_address TEXT,
			tracking_number TEXT,
			carrier TEXT,
			shipped_at DATETIME,
			delivered_at DATETIME,
			notes TEXT,
			dispute_reason TEXT,
			dispute_evidence TEXT,
			is_guest_order BOOLEAN NOT NULL DEFAULT 0,
			guest_email TEXT,
			guest_first_name TEXT,
			guest_last_name TEXT,
			guest_phone TEXT,
			guest_token TEXT,
			guest_token_fingerprint_hash TEXT,
			delivery_type TEXT NOT NULL,
			confirmed_at DATETIME,
			completed_at DATETIME,
			cancelled_at DATETIME,
			cancelled_reason TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS order_items (
			id TEXT PRIMARY KEY,
			order_id TEXT NOT NULL,
			listing_id TEXT,
			auction_id TEXT,
			title TEXT NOT NULL,
			quantity INTEGER NOT NULL,
			unit_price REAL NOT NULL,
			total_price REAL NOT NULL,
			condition TEXT,
			created_at DATETIME
		);
	`).Error)

	r := gin.New()
	v1 := r.Group("/api/v1")
	RegisterRoutes(v1, db)
	return r
}

func makeGuestOrderPayload(sellerID uuid.UUID) map[string]any {
	return map[string]any{
		"payment_intent_id": "pi_guest_test_1",
		"seller_id":         sellerID.String(),
		"items": []map[string]any{{
			"title":      "Guest Item",
			"quantity":   1,
			"unit_price": 100,
		}},
		"subtotal":         100,
		"platform_fee":     5,
		"payment_fee":      1,
		"total":            106,
		"currency":         "AED",
		"guest_email":      "guest@example.com",
		"guest_first_name": "Guest",
		"guest_last_name":  "User",
		"guest_phone":      "+971500000000",
		"delivery_type":    string(DeliveryTypeStandard),
	}
}

func createGuestOrder(t *testing.T, r *gin.Engine) (orderID string, guestToken string) {
	t.Helper()
	sellerID := uuid.New()
	payload := makeGuestOrderPayload(sellerID)
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/orders/guest", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "guest-test-agent")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		GuestToken string `json:"guest_token"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data.ID)
	require.NotEmpty(t, resp.GuestToken)

	return resp.Data.ID, resp.GuestToken
}

func TestGuestOrderFetch_WithValidToken(t *testing.T) {
	r := setupGuestOrderTestRouter(t)
	orderID, guestToken := createGuestOrder(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderID, nil)
	req.Header.Set("X-Guest-Token", guestToken)
	req.Header.Set("User-Agent", "guest-test-agent")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

func TestGuestOrderFetch_WithoutTokenFails(t *testing.T) {
	r := setupGuestOrderTestRouter(t)
	orderID, _ := createGuestOrder(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderID, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestGuestOrderFetch_WithWrongTokenFails(t *testing.T) {
	r := setupGuestOrderTestRouter(t)
	orderID, _ := createGuestOrder(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderID, nil)
	req.Header.Set("X-Guest-Token", uuid.NewString())
	req.Header.Set("User-Agent", "guest-test-agent")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestGuestOrderFetch_WithFingerprintMismatchFails(t *testing.T) {
	r := setupGuestOrderTestRouter(t)
	orderID, guestToken := createGuestOrder(t, r)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/orders/"+orderID, nil)
	req.Header.Set("X-Guest-Token", guestToken)
	req.Header.Set("User-Agent", "guest-test-agent-different")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}
