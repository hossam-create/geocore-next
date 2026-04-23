package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/auctions"
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
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ════════════════════════════════════════════════════════════════════════════════
// Scenario 3: Auction — Create auction, place bid, buy-now, proxy bid, wallet integration
// ════════════════════════════════════════════════════════════════════════════════

type AuctionSuite struct {
	suite.Suite
	ts       *TestSuite
	r        *gin.Engine
	sellerID uuid.UUID
	buyerAID uuid.UUID
	buyerBID uuid.UUID
	catID    uuid.UUID
}

func TestAuctionSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &AuctionSuite{ts: ts})
}

func (s *AuctionSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(
		&users.User{},
		&auctions.Auction{},
		&auctions.Bid{},
		&auctions.ProxyBid{},
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
	ts.CreateManualTables()

	auctions.RegisterRoutes(s.r.Group("/api/v1"), ts.DB, ts.RDB)

	s.catID = ts.CreateCategory()
	s.sellerID = ts.CreateUserWithEmailVerified("Auction Seller", UniqueEmail("auction-seller"))
	s.buyerAID = ts.CreateUserWithEmailVerified("Buyer A", UniqueEmail("buyer-a"))
	s.buyerBID = ts.CreateUserWithEmailVerified("Buyer B", UniqueEmail("buyer-b"))

	ts.FundWallet(s.buyerAID, 5000.00)
	ts.FundWallet(s.buyerBID, 5000.00)
}

func (s *AuctionSuite) SetupTest() {
	s.ts.ResetTest()
	s.ts.DB.Exec("DELETE FROM proxy_bids")
	s.ts.DB.Exec("DELETE FROM bids")
	s.ts.DB.Exec("DELETE FROM auctions")
}

// ── Test: List auctions ─────────────────────────────────────────────────────────

func (s *AuctionSuite) TestListAuctions() {
	s.createTestAuction()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auctions", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Get auction by ID ────────────────────────────────────────────────────

func (s *AuctionSuite) TestGetAuction() {
	auctionID := s.createTestAuction()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auctions/"+auctionID.String(), nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Create auction ───────────────────────────────────────────────────────

func (s *AuctionSuite) TestCreateAuction_Success() {
	listingID := s.ts.CreateListing(s.sellerID, s.catID, "Auction Item", 100.00)
	endTime := time.Now().Add(24 * time.Hour)

	body, _ := json.Marshal(gin.H{
		"listing_id":     listingID.String(),
		"starting_price": 50.00,
		"reserve_price":  80.00,
		"buy_now_price":  150.00,
		"end_time":       endTime.Format(time.RFC3339),
		"currency":       "USD",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auctions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.sellerID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)
}

// ── Test: Place bid ─────────────────────────────────────────────────────────────

func (s *AuctionSuite) TestPlaceBid_Success() {
	auctionID := s.createTestAuction()

	body, _ := json.Marshal(gin.H{
		"amount": 60.00,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auctions/"+auctionID.String()+"/bid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerAID))
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusCreated, w.Code)

	// Verify bid persisted
	var bids []auctions.Bid
	s.ts.DB.Where("auction_id = ?", auctionID).Find(&bids)
	assert.Equal(s.T(), 1, len(bids), "should persist one bid")
}

// ── Test: Place bid below current highest fails ────────────────────────────────

func (s *AuctionSuite) TestPlaceBid_BelowCurrent() {
	auctionID := s.createTestAuction()

	// First bid at 60
	s.placeBid(auctionID, s.buyerAID, 60.00)

	// Second bid at 55 (below current) should fail
	body, _ := json.Marshal(gin.H{
		"amount": 55.00,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auctions/"+auctionID.String()+"/bid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(s.buyerBID))
	s.r.ServeHTTP(w, req)

	assert.NotEqual(s.T(), http.StatusCreated, w.Code, "bid below current should fail")
}

// ── Test: Get bids for auction ─────────────────────────────────────────────────

func (s *AuctionSuite) TestGetBids() {
	auctionID := s.createTestAuction()
	s.placeBid(auctionID, s.buyerAID, 60.00)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/auctions/"+auctionID.String()+"/bids", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Unauthenticated bid fails ───────────────────────────────────────────

func (s *AuctionSuite) TestPlaceBid_Unauthenticated() {
	auctionID := s.createTestAuction()

	body, _ := json.Marshal(gin.H{
		"amount": 60.00,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auctions/"+auctionID.String()+"/bid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusUnauthorized, w.Code)
}

// ── Helpers ─────────────────────────────────────────────────────────────────────

func (s *AuctionSuite) createTestAuction() uuid.UUID {
	listingID := s.ts.CreateListing(s.sellerID, s.catID, "Test Auction Item", 100.00)
	now := time.Now()
	reserve := 80.00
	buyNow := 150.00

	auction := auctions.Auction{
		ID:           uuid.New(),
		SellerID:     s.sellerID,
		ListingID:    listingID,
		StartPrice:   50.00,
		ReservePrice: &reserve,
		BuyNowPrice:  &buyNow,
		StartsAt:     now,
		EndsAt:       now.Add(24 * time.Hour),
		Currency:     "USD",
		Status:       auctions.StatusActive,
	}
	require.NoError(s.T(), s.ts.DB.Create(&auction).Error)
	return auction.ID
}

func (s *AuctionSuite) placeBid(auctionID, bidderID uuid.UUID, amount float64) {
	body, _ := json.Marshal(gin.H{
		"amount": amount,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auctions/"+auctionID.String()+"/bid", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.signToken(bidderID))
	s.r.ServeHTTP(w, req)
}

func (s *AuctionSuite) signToken(userID uuid.UUID) string {
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
