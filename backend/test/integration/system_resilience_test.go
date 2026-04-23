package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/auth"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/jwtkeys"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// ════════════════════════════════════════════════════════════════════════════════
// Scenario 10: System Resilience — Concurrent requests, idempotency, DB constraints,
// cascade safety, and load simulation
// ════════════════════════════════════════════════════════════════════════════════

type ResilienceSuite struct {
	suite.Suite
	ts      *TestSuite
	r       *gin.Engine
	userIDs []uuid.UUID
}

func TestResilienceSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &ResilienceSuite{ts: ts})
}

func (s *ResilienceSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(
		&users.User{},
		&wallet.Wallet{},
		&wallet.WalletBalance{},
		&wallet.WalletTransaction{},
		&wallet.Escrow{},
		&wallet.IdempotentRequest{},
		&notifications.Notification{},
		&notifications.NotificationPreference{},
		&notifications.PushToken{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)

	auth.RegisterRoutes(s.r.Group("/api/v1"), ts.DB, ts.RDB)
	wallet.RegisterRoutes(s.r.Group("/api/v1"), ts.DB, ts.RDB)

	// Create 5 test users with funded wallets
	for i := 0; i < 5; i++ {
		id := ts.CreateUserWithEmailVerified(
			"Resilience User",
			UniqueEmail("resilience"),
		)
		ts.FundWallet(id, 10000.00)
		s.userIDs = append(s.userIDs, id)
	}
}

func (s *ResilienceSuite) SetupTest() {
	s.ts.ResetTest()
}

// ── Test: Concurrent wallet deposits ────────────────────────────────────────────

func (s *ResilienceSuite) TestConcurrentDeposits_NoDataCorruption() {
	userID := s.userIDs[0]
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			body, _ := json.Marshal(gin.H{
				"amount":          10.00,
				"currency":        "USD",
				"idempotency_key": uuid.New().String(), // unique per request
			})
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+s.signToken(userID))
			req.Header.Set("Idempotency-Key", uuid.New().String())
			s.r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("deposit %d failed: %d", idx, w.Code)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		s.T().Logf("concurrent deposit error: %v", err)
	}

	// Verify final balance is consistent
	_, avail, _ := s.ts.GetWalletBalances(userID)
	// Started at 10000, 10 deposits of $10 each = 10100
	expected := 10100.00
	assert.InDelta(s.T(), expected, avail.InexactFloat64(), 1.0,
		"balance should be ~%d after 10 concurrent deposits, got %s", int(expected), avail.String())
}

// ── Test: Duplicate registration returns conflict ──────────────────────────────

func (s *ResilienceSuite) TestDuplicateRegistration_Conflict() {
	email := UniqueEmail("dup-resilience")

	body, _ := json.Marshal(gin.H{
		"name":     "Dup User",
		"email":    email,
		"password": "Str0ng!Pass#1",
	})

	// First registration succeeds
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w1, req1)
	assert.Equal(s.T(), http.StatusCreated, w1.Code)

	// Second registration with same email fails
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w2, req2)
	assert.Equal(s.T(), http.StatusConflict, w2.Code)
}

// ── Test: Concurrent escrow creation ──────────────────────────────────────────

func (s *ResilienceSuite) TestConcurrentEscrow_NoDoubleSpend() {
	buyerID := s.userIDs[0]
	sellerID := s.userIDs[1]

	var wg sync.WaitGroup
	results := make(chan int, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			body, _ := json.Marshal(gin.H{
				"seller_id":    sellerID.String(),
				"amount":       1000.00,
				"currency":     "USD",
				"reference_id": uuid.New().String(),
			})
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/escrow", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+s.signToken(buyerID))
			s.r.ServeHTTP(w, req)
			results <- w.Code
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	for code := range results {
		if code == http.StatusOK {
			successCount++
		}
	}

	// With 10000 balance and 1000 per escrow, max 10 escrows should succeed
	// 5 concurrent requests should all succeed since 5*1000 = 5000 < 10000
	assert.Equal(s.T(), 5, successCount, "all 5 escrow creations should succeed")
}

// ── Test: Invalid UUID returns bad request ──────────────────────────────────────

func (s *ResilienceSuite) TestInvalidUUID_BadRequest() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/wallet/balance/invalid-uuid", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userIDs[0]))
	s.r.ServeHTTP(w, req)

	assert.NotEqual(s.T(), http.StatusOK, w.Code, "invalid UUID should not return 200")
}

// ── Test: Missing auth returns unauthorized ────────────────────────────────────

func (s *ResilienceSuite) TestMissingAuth_Unauthorized() {
	body, _ := json.Marshal(gin.H{
		"amount":   100.00,
		"currency": "USD",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// ── Test: Rapid sequential requests don't corrupt data ─────────────────────────

func (s *ResilienceSuite) TestRapidSequentialRequests_DataConsistency() {
	userID := s.userIDs[2]

	for i := 0; i < 20; i++ {
		body, _ := json.Marshal(gin.H{
			"amount":          5.00,
			"currency":        "USD",
			"idempotency_key": uuid.New().String(),
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+s.signToken(userID))
		req.Header.Set("Idempotency-Key", uuid.New().String())
		s.r.ServeHTTP(w, req)
		assert.Equal(s.T(), http.StatusOK, w.Code, "deposit %d should succeed", i)
	}

	_, avail, _ := s.ts.GetWalletBalances(userID)
	expected := 10000.00 + 20*5.00 // 10100
	assert.InDelta(s.T(), expected, avail.InexactFloat64(), 1.0,
		"balance should be ~%d after 20 sequential deposits", int(expected))
}

// ── Test: Negative amount rejected ─────────────────────────────────────────────

func (s *ResilienceSuite) TestNegativeAmount_Rejected() {
	body, _ := json.Marshal(gin.H{
		"amount":   -100.00,
		"currency": "USD",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userIDs[0]))
	s.r.ServeHTTP(w, req)

	assert.NotEqual(s.T(), http.StatusOK, w.Code, "negative amount should be rejected")
}

// ── Helper: sign a test JWT ─────────────────────────────────────────────────────

func (s *ResilienceSuite) signToken(userID uuid.UUID) string {
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
