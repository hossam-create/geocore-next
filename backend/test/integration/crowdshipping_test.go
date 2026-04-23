package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/crowdshipping"
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
// Scenario 5: Crowdshipping — Trip creation, delivery requests, matching, offers
// ════════════════════════════════════════════════════════════════════════════════

type CrowdshippingSuite struct {
	suite.Suite
	ts         *TestSuite
	r          *gin.Engine
	travelerID uuid.UUID
	buyerID    uuid.UUID
	notifSvc   *notifications.Service
}

func TestCrowdshippingSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &CrowdshippingSuite{ts: ts})
}

func (s *CrowdshippingSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(
		&users.User{},
		&crowdshipping.Trip{},
		&crowdshipping.DeliveryRequest{},
		&notifications.Notification{},
		&notifications.NotificationPreference{},
		&notifications.PushToken{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
	)

	// Wire notification service (real DB, mock hub/FCM)
	hub := notifications.NewHub()
	go hub.Run()
	s.notifSvc = notifications.NewService(ts.DB, hub, nil)

	crowdshipping.RegisterRoutes(s.r.Group("/api/v1"), ts.DB, s.notifSvc)

	s.travelerID = ts.CreateUserWithEmailVerified("Traveler", UniqueEmail("traveler"))
	s.buyerID = ts.CreateUserWithEmailVerified("Buyer", UniqueEmail("cs-buyer"))
}

func (s *CrowdshippingSuite) SetupTest() {
	s.ts.ResetTest()
	s.ts.DB.Exec("DELETE FROM delivery_requests")
	s.ts.DB.Exec("DELETE FROM trips")
}

// ── Test: Create trip ──────────────────────────────────────────────────────────

func (s *CrowdshippingSuite) TestCreateTrip_Success() {
	departure := time.Now().Add(48 * time.Hour)
	arrival := time.Now().Add(72 * time.Hour)

	body, _ := json.Marshal(gin.H{
		"origin_country":   "UAE",
		"origin_city":      "Dubai",
		"dest_country":     "Egypt",
		"dest_city":        "Cairo",
		"departure_date":   departure.Format(time.RFC3339),
		"arrival_date":     arrival.Format(time.RFC3339),
		"available_weight": 20.0,
		"price_per_kg":     15.0,
		"base_price":       50.0,
		"currency":         "AED",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/trips", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.travelerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)

	// Verify trip persisted
	var trips []crowdshipping.Trip
	s.ts.DB.Where("traveler_id = ?", s.travelerID).Find(&trips)
	assert.Equal(s.T(), 1, len(trips), "should create one trip")
	assert.Equal(s.T(), crowdshipping.TripStatusActive, trips[0].Status)
}

// ── Test: List trips ────────────────────────────────────────────────────────────

func (s *CrowdshippingSuite) TestListTrips() {
	s.createTestTrip()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/trips", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Create delivery request ──────────────────────────────────────────────

func (s *CrowdshippingSuite) TestCreateDeliveryRequest_Success() {
	body, _ := json.Marshal(gin.H{
		"item_name":        "iPhone 15 Pro",
		"item_description": "Brand new, sealed box",
		"item_price":       4999.00,
		"pickup_country":   "UAE",
		"pickup_city":      "Dubai",
		"delivery_country": "Egypt",
		"delivery_city":    "Cairo",
		"reward":           200.00,
		"currency":         "AED",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/delivery-requests", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)

	// Verify delivery request persisted
	var drs []crowdshipping.DeliveryRequest
	s.ts.DB.Where("buyer_id = ?", s.buyerID).Find(&drs)
	assert.Equal(s.T(), 1, len(drs), "should create one delivery request")
	assert.Equal(s.T(), crowdshipping.DeliveryPending, drs[0].Status)
}

// ── Test: List delivery requests ────────────────────────────────────────────────

func (s *CrowdshippingSuite) TestListDeliveryRequests() {
	s.createTestDeliveryRequest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/delivery-requests", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Find travelers for delivery request ───────────────────────────────────

func (s *CrowdshippingSuite) TestFindTravelers() {
	drID := s.createTestDeliveryRequest()
	s.createTestTrip()

	body, _ := json.Marshal(gin.H{})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/delivery-requests/"+drID.String()+"/find-travelers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Create trip missing required fields ──────────────────────────────────

func (s *CrowdshippingSuite) TestCreateTrip_MissingFields() {
	body, _ := json.Marshal(gin.H{
		"origin_country": "UAE",
		// missing required fields
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/trips", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.travelerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// ── Helpers ─────────────────────────────────────────────────────────────────────

func (s *CrowdshippingSuite) createTestTrip() uuid.UUID {
	departure := time.Now().Add(48 * time.Hour)
	arrival := time.Now().Add(72 * time.Hour)

	trip := crowdshipping.Trip{
		ID:              uuid.New(),
		TravelerID:      s.travelerID,
		OriginCountry:   "UAE",
		OriginCity:      "Dubai",
		DestCountry:     "Egypt",
		DestCity:        "Cairo",
		DepartureDate:   departure,
		ArrivalDate:     arrival,
		AvailableWeight: 20.0,
		PricePerKg:      15.0,
		BasePrice:       50.0,
		Currency:        "AED",
		Status:          crowdshipping.TripStatusActive,
	}
	require.NoError(s.T(), s.ts.DB.Create(&trip).Error)
	return trip.ID
}

func (s *CrowdshippingSuite) createTestDeliveryRequest() uuid.UUID {
	dr := crowdshipping.DeliveryRequest{
		ID:              uuid.New(),
		BuyerID:         s.buyerID,
		ItemName:        "Test Item",
		ItemPrice:       1000.0,
		PickupCountry:   "UAE",
		PickupCity:      "Dubai",
		DeliveryCountry: "Egypt",
		DeliveryCity:    "Cairo",
		Reward:          100.0,
		Currency:        "AED",
		DeliveryType:    crowdshipping.DeliveryTypeCrowdshipping,
		Status:          crowdshipping.DeliveryPending,
	}
	require.NoError(s.T(), s.ts.DB.Create(&dr).Error)
	return dr.ID
}

func (s *CrowdshippingSuite) signToken(userID uuid.UUID) string {
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
