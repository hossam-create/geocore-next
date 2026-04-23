package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/fraud"
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
// Scenario 7: Fraud — Risk analysis, scoring, alerts, Kafka event emission
// ════════════════════════════════════════════════════════════════════════════════

type FraudSuite struct {
	suite.Suite
	ts     *TestSuite
	r      *gin.Engine
	userID uuid.UUID
}

func TestFraudSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &FraudSuite{ts: ts})
}

func (s *FraudSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(
		&users.User{},
		&fraud.FraudAlert{},
		&fraud.FraudRule{},
		&fraud.UserRiskProfile{},
		&fraud.FraudFeedback{},
		&fraud.UserRiskSnapshot{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)

	fraud.RegisterRoutes(s.r.Group("/api/v1"), ts.DB)

	s.userID = ts.CreateUserWithEmailVerified("Fraud User", UniqueEmail("fraud"))
}

func (s *FraudSuite) SetupTest() {
	s.ts.ResetTest()
	s.ts.DB.Exec("DELETE FROM fraud_alerts")
	s.ts.DB.Exec("DELETE FROM user_risk_profiles")
	s.ts.DB.Exec("DELETE FROM outbox_events")
}

// ── Test: Analyze low-risk transaction ──────────────────────────────────────────

func (s *FraudSuite) TestAnalyze_LowRiskTransaction() {
	body, _ := json.Marshal(gin.H{
		"user_id":           s.userID.String(),
		"amount":            50.00,
		"total_orders":      10,
		"avg_order_value":   100.00,
		"account_age_hours": 720.0, // 30 days
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/fraud/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)

	riskScore, _ := data["risk_score"].(float64)
	assert.Less(s.T(), riskScore, 70.0, "low-risk transaction should have score < 70")
}

// ── Test: Analyze high-risk transaction creates alert ───────────────────────────

func (s *FraudSuite) TestAnalyze_HighRiskTransaction_CreatesAlert() {
	body, _ := json.Marshal(gin.H{
		"user_id":           s.userID.String(),
		"amount":            50000.00, // very large amount
		"total_orders":      1,        // new account
		"avg_order_value":   50.00,    // way above average
		"account_age_hours": 2.0,      // 2 hours old
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/fraud/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)

	riskScore, _ := data["risk_score"].(float64)
	assert.GreaterOrEqual(s.T(), riskScore, 70.0, "high-risk transaction should have score >= 70")

	// Verify fraud alert was created
	var alerts []fraud.FraudAlert
	s.ts.DB.Where("target_id = ?", s.userID).Find(&alerts)
	assert.GreaterOrEqual(s.T(), len(alerts), 1, "should create fraud alert for high-risk transaction")
}

// ── Test: Analyze writes Kafka outbox event ─────────────────────────────────────

func (s *FraudSuite) TestAnalyze_WritesOutboxEvent() {
	body, _ := json.Marshal(gin.H{
		"user_id":           s.userID.String(),
		"amount":            100.00,
		"total_orders":      5,
		"avg_order_value":   80.00,
		"account_age_hours": 240.0,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/fraud/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	// Verify outbox event was written
	var outboxEvents []kafka.OutboxEvent
	s.ts.DB.Order("created_at DESC").Limit(5).Find(&outboxEvents)
	assert.GreaterOrEqual(s.T(), len(outboxEvents), 1, "should write at least one outbox event")
}

// ── Test: Scoring engine with low-velocity user ────────────────────────────────

func (s *FraudSuite) TestScoringEngine_LowVelocityUser() {
	result := fraud.AnalyzeTransaction(100.00, 20, 85.00, 2160.0) // 90 days
	assert.Less(s.T(), result.RiskScore, 70.0, "low-velocity user should have low risk score")
	assert.Equal(s.T(), "approved", result.Decision)
}

// ── Test: Scoring engine with high-velocity new account ─────────────────────────

func (s *FraudSuite) TestScoringEngine_HighVelocityNewAccount() {
	result := fraud.AnalyzeTransaction(10000.00, 0, 0, 1.0) // 1 hour old, first order, $10k
	assert.GreaterOrEqual(s.T(), result.RiskScore, 70.0, "high-velocity new account should have high risk score")
	assert.Equal(s.T(), "declined", result.Decision)
}

// ── Test: Scoring engine with moderate risk ─────────────────────────────────────

func (s *FraudSuite) TestScoringEngine_ModerateRisk() {
	result := fraud.AnalyzeTransaction(500.00, 3, 100.00, 48.0) // 2 days, 3 orders, $500
	assert.GreaterOrEqual(s.T(), result.RiskScore, 30.0, "moderate risk should have score >= 30")
	assert.Less(s.T(), result.RiskScore, 70.0, "moderate risk should have score < 70")
	assert.Equal(s.T(), "review", result.Decision)
}

// ── Test: Missing user_id returns error ─────────────────────────────────────────

func (s *FraudSuite) TestAnalyze_MissingUserID() {
	body, _ := json.Marshal(gin.H{
		"amount":            100.00,
		"total_orders":      5,
		"avg_order_value":   80.00,
		"account_age_hours": 240.0,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/fraud/analyze", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.userID))
	s.r.ServeHTTP(w, req)

	// Should return bad request since user_id is required
	assert.NotEqual(s.T(), http.StatusOK, w.Code)
}

// ── Helper: sign a test JWT ─────────────────────────────────────────────────────

func (s *FraudSuite) signToken(userID uuid.UUID) string {
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
