package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/jwtkeys"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ════════════════════════════════════════════════════════════════════════════════
// Scenario 4: Payment — Wallet deposit, withdraw, escrow hold/release, balance checks
// ════════════════════════════════════════════════════════════════════════════════

type PaymentSuite struct {
	suite.Suite
	ts       *TestSuite
	r        *gin.Engine
	buyerID  uuid.UUID
	sellerID uuid.UUID
}

func TestPaymentSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &PaymentSuite{ts: ts})
}

func (s *PaymentSuite) SetupSuite() {
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
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)

	wallet.RegisterRoutes(s.r.Group("/api/v1"), ts.DB, ts.RDB)

	s.buyerID = ts.CreateUserWithEmailVerified("Buyer", UniqueEmail("buyer"))
	ts.FundWallet(s.buyerID, 5000.00)

	s.sellerID = ts.CreateUserWithEmailVerified("Seller", UniqueEmail("seller"))
	ts.FundWallet(s.sellerID, 1000.00)
}

func (s *PaymentSuite) SetupTest() {
	s.ts.ResetTest()
}

// ── Test: Get wallet balance ────────────────────────────────────────────────────

func (s *PaymentSuite) TestGetWalletBalance() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/wallet", nil)
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))
}

// ── Test: Deposit into wallet ───────────────────────────────────────────────────

func (s *PaymentSuite) TestDeposit_Success() {
	body, _ := json.Marshal(gin.H{
		"amount":   100.00,
		"currency": "USD",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	bal, avail, _ := s.ts.GetWalletBalances(s.buyerID)
	expectedBal := decimal.NewFromFloat(5100.00)
	assert.True(s.T(), bal.Equal(expectedBal), "balance should be 5100 after deposit, got %s", bal.String())
	assert.True(s.T(), avail.Equal(expectedBal), "available should be 5100 after deposit, got %s", avail.String())
}

// ── Test: Withdraw from wallet ──────────────────────────────────────────────────

func (s *PaymentSuite) TestWithdraw_Success() {
	body, _ := json.Marshal(gin.H{
		"amount":   200.00,
		"currency": "USD",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/withdraw", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	_, avail, _ := s.ts.GetWalletBalances(s.buyerID)
	expectedAvail := decimal.NewFromFloat(4800.00)
	assert.True(s.T(), avail.Equal(expectedAvail), "available should be 4800 after withdrawal, got %s", avail.String())
}

// ── Test: Withdraw more than balance fails ──────────────────────────────────────

func (s *PaymentSuite) TestWithdraw_InsufficientBalance() {
	body, _ := json.Marshal(gin.H{
		"amount":   999999.00,
		"currency": "USD",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/withdraw", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	s.r.ServeHTTP(w, req)

	assert.NotEqual(s.T(), http.StatusOK, w.Code, "should not succeed with insufficient balance")
}

// ── Test: Create escrow ─────────────────────────────────────────────────────────

func (s *PaymentSuite) TestCreateEscrow_Success() {
	body, _ := json.Marshal(gin.H{
		"seller_id":    s.sellerID.String(),
		"amount":       500.00,
		"currency":     "USD",
		"reference_id": uuid.New().String(),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/escrow", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Verify escrow record created
	var escrows []wallet.Escrow
	s.ts.DB.Where("buyer_id = ?", s.buyerID).Find(&escrows)
	assert.Equal(s.T(), 1, len(escrows), "should create one escrow record")
	assert.Equal(s.T(), "pending", escrows[0].Status)

	// Buyer's available balance should decrease by 500
	_, avail, _ := s.ts.GetWalletBalances(s.buyerID)
	expectedAvail := decimal.NewFromFloat(4500.00)
	assert.True(s.T(), avail.Equal(expectedAvail), "available should be 4500 after escrow, got %s", avail.String())
}

// ── Test: Transfer between wallets ──────────────────────────────────────────────

func (s *PaymentSuite) TestTransfer_Success() {
	body, _ := json.Marshal(gin.H{
		"recipient_id": s.sellerID.String(),
		"amount":       50.00,
		"currency":     "USD",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/wallet/transfer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Idempotent deposit returns same result ───────────────────────────────

func (s *PaymentSuite) TestDeposit_Idempotent() {
	idempotencyKey := uuid.New().String()

	body1, _ := json.Marshal(gin.H{
		"amount":          75.00,
		"currency":        "USD",
		"idempotency_key": idempotencyKey,
	})

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	req1.Header.Set("Idempotency-Key", idempotencyKey)
	s.r.ServeHTTP(w1, req1)
	assert.Equal(s.T(), http.StatusOK, w1.Code)

	balAfterFirst, _, _ := s.ts.GetWalletBalances(s.buyerID)

	// Second request with same idempotency key should return cached result
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/wallet/deposit", bytes.NewReader(body1))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	req2.Header.Set("Idempotency-Key", idempotencyKey)
	s.r.ServeHTTP(w2, req2)
	assert.Equal(s.T(), http.StatusOK, w2.Code)

	balAfterSecond, _, _ := s.ts.GetWalletBalances(s.buyerID)
	assert.True(s.T(), balAfterFirst.Equal(balAfterSecond),
		"idempotent deposit should not change balance: first=%s second=%s",
		balAfterFirst.String(), balAfterSecond.String())
}

// ── Helper: sign a test JWT ─────────────────────────────────────────────────────

func (s *PaymentSuite) signToken(userID uuid.UUID) string {
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
