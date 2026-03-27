package auctions_test

import (
        "bytes"
        "context"
        "encoding/json"
        "fmt"
        "net/http"
        "net/http/httptest"
        "os"
        "strings"
        "sync"
        "testing"
        "time"

        "github.com/alicebob/miniredis/v2"
        "github.com/geocore-next/backend/internal/auctions"
        "github.com/geocore-next/backend/internal/users"
        "github.com/geocore-next/backend/pkg/middleware"
        "github.com/gin-gonic/gin"
        "github.com/golang-jwt/jwt/v5"
        "github.com/google/uuid"
        "github.com/gorilla/websocket"
        "github.com/redis/go-redis/v9"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
        "gorm.io/driver/sqlite"
        "gorm.io/gorm"
        "gorm.io/gorm/logger"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func setupDB(t *testing.T) *gorm.DB {
        t.Helper()
        // Use a unique shared-cache URI so all connections in the pool share the
        // same in-memory database. Without this, each connection from GORM's pool
        // would open an independent empty SQLite database.
        dbName := fmt.Sprintf("file:testdb_%s?mode=memory&cache=shared", uuid.New().String())
        db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
                Logger:                                   logger.Default.LogMode(logger.Silent),
                DisableForeignKeyConstraintWhenMigrating: true,
        })
        require.NoError(t, err)
        sqlDB, err := db.DB()
        require.NoError(t, err)
        // Limit to one connection so all goroutines see the same in-memory state.
        sqlDB.SetMaxOpenConns(1)
        t.Cleanup(func() { sqlDB.Close() })
        require.NoError(t, db.AutoMigrate(&users.User{}), "users.User migrate")

        // Create auction tables manually — SQLite doesn't support uuid_generate_v4() as a DEFAULT.
        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS auctions (
                id TEXT PRIMARY KEY,
                listing_id TEXT NOT NULL UNIQUE,
                seller_id TEXT NOT NULL,
                start_price REAL NOT NULL,
                reserve_price REAL,
                buy_now_price REAL,
                current_bid REAL DEFAULT 0,
                bid_count INTEGER DEFAULT 0,
                winner_id TEXT,
                status TEXT DEFAULT 'active',
                starts_at DATETIME,
                ends_at DATETIME,
                extension_count INTEGER DEFAULT 0,
                currency TEXT DEFAULT 'USD',
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                deleted_at DATETIME
        )`).Error, "create auctions table")

        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS bids (
                id TEXT PRIMARY KEY,
                auction_id TEXT NOT NULL,
                user_id TEXT NOT NULL,
                amount REAL NOT NULL,
                is_auto INTEGER DEFAULT 0,
                max_amount REAL,
                placed_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`).Error, "create bids table")

        // Listings table for test fixtures.
        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS listings (
                id TEXT PRIMARY KEY,
                user_id TEXT NOT NULL,
                category_id TEXT,
                title TEXT NOT NULL,
                price REAL,
                status TEXT DEFAULT 'active',
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )`).Error, "create listings table")

        return db
}

func setupMiniredis(t *testing.T) *redis.Client {
        t.Helper()
        mr, err := miniredis.Run()
        require.NoError(t, err)
        t.Cleanup(mr.Close)
        return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func setupRouter(t *testing.T, db *gorm.DB, rdb *redis.Client) *gin.Engine {
        t.Helper()
        gin.SetMode(gin.TestMode)
        r := gin.New()
        v1 := r.Group("/api/v1")
        auctions.RegisterRoutes(v1, db, rdb, middleware.NewRateLimiter(rdb))
        return r
}

// makeToken creates a signed JWT for a user.
func makeToken(t *testing.T, userID, email string) string {
        t.Helper()
        secret := os.Getenv("JWT_SECRET")
        if secret == "" {
                secret = "test-secret"
                os.Setenv("JWT_SECRET", secret)
        }
        claims := middleware.Claims{
                UserID: userID,
                Email:  email,
                RegisteredClaims: jwt.RegisteredClaims{
                        ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
                        IssuedAt:  jwt.NewNumericDate(time.Now()),
                },
        }
        tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        signed, err := tok.SignedString([]byte(secret))
        require.NoError(t, err)
        return signed
}

// createUser inserts a user into the DB and returns a JWT for them.
func createUser(t *testing.T, db *gorm.DB, name, email string) (uuid.UUID, string) {
        t.Helper()
        u := users.User{
                ID:            uuid.New(),
                Name:          name,
                Email:         email,
                PasswordHash:  "hash",
                EmailVerified: true,
                IsActive:      true,
        }
        require.NoError(t, db.Create(&u).Error)
        return u.ID, makeToken(t, u.ID.String(), email)
}

// createListing inserts a minimal listing row and returns its ID.
func createListing(t *testing.T, db *gorm.DB, sellerID uuid.UUID) uuid.UUID {
        t.Helper()
        id := uuid.New()
        err := db.Exec(
                `INSERT INTO listings (id, user_id, title, price, status, created_at, updated_at)
                 VALUES (?, ?, 'Test Item', 100, 'active', datetime('now'), datetime('now'))`,
                id, sellerID,
        ).Error
        require.NoError(t, err)
        return id
}

func jsonBody(t *testing.T, payload any) *bytes.Buffer {
        t.Helper()
        b, err := json.Marshal(payload)
        require.NoError(t, err)
        return bytes.NewBuffer(b)
}

func doPost(t *testing.T, r *gin.Engine, path, token string, payload any) *httptest.ResponseRecorder {
        t.Helper()
        w := httptest.NewRecorder()
        req, err := http.NewRequest(http.MethodPost, path, jsonBody(t, payload))
        require.NoError(t, err)
        req.Header.Set("Content-Type", "application/json")
        if token != "" {
                req.Header.Set("Authorization", "Bearer "+token)
        }
        r.ServeHTTP(w, req)
        return w
}

func doGet(t *testing.T, r *gin.Engine, path, token string) *httptest.ResponseRecorder {
        t.Helper()
        w := httptest.NewRecorder()
        req, err := http.NewRequest(http.MethodGet, path, nil)
        require.NoError(t, err)
        if token != "" {
                req.Header.Set("Authorization", "Bearer "+token)
        }
        r.ServeHTTP(w, req)
        return w
}

func parseData(t *testing.T, body []byte) map[string]any {
        t.Helper()
        var resp map[string]any
        require.NoError(t, json.Unmarshal(body, &resp))
        data, ok := resp["data"].(map[string]any)
        require.True(t, ok, "expected 'data' map in response; got: %s", string(body))
        return data
}

// ── Create auction tests ─────────────────────────────────────────────────────

func TestCreateAuction_Success(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, token := createUser(t, db, "Seller", "seller@test.com")
        listingID := createListing(t, db, sellerID)

        w := doPost(t, r, "/api/v1/auctions", token, map[string]any{
                "listing_id":     listingID.String(),
                "start_price":    50.0,
                "duration_hours": 24,
                "currency":       "USD",
        })

        assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())
        data := parseData(t, w.Body.Bytes())
        assert.Equal(t, "active", data["status"])
        assert.Equal(t, listingID.String(), data["listing_id"])
}

func TestCreateAuction_RequiresAuth(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        w := doPost(t, r, "/api/v1/auctions", "", map[string]any{
                "listing_id":     uuid.New().String(),
                "start_price":    50.0,
                "duration_hours": 24,
        })
        assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestCreateAuction_InvalidDuration(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, token := createUser(t, db, "Seller2", "seller2@test.com")
        listingID := createListing(t, db, sellerID)

        w := doPost(t, r, "/api/v1/auctions", token, map[string]any{
                "listing_id":     listingID.String(),
                "start_price":    50.0,
                "duration_hours": 0, // invalid: min is 1
        })
        assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── List / Get auction tests ─────────────────────────────────────────────────

func TestListAuctions(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, token := createUser(t, db, "Seller3", "seller3@test.com")
        listingID := createListing(t, db, sellerID)

        // Create an auction
        doPost(t, r, "/api/v1/auctions", token, map[string]any{
                "listing_id": listingID.String(), "start_price": 100.0, "duration_hours": 48,
        })

        w := doGet(t, r, "/api/v1/auctions", "")
        assert.Equal(t, http.StatusOK, w.Code)

        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data, ok := resp["data"]
        assert.True(t, ok)
        items, ok := data.([]any)
        assert.True(t, ok)
        assert.GreaterOrEqual(t, len(items), 1)
}

func TestGetAuction_NotFound(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        w := doGet(t, r, "/api/v1/auctions/"+uuid.New().String(), "")
        assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── PlaceBid tests ───────────────────────────────────────────────────────────

// createAuction is a helper that creates an auction in the DB directly.
func createAuction(t *testing.T, db *gorm.DB, sellerID, listingID uuid.UUID, startPrice float64, endsIn time.Duration) auctions.Auction {
        t.Helper()
        now := time.Now()
        a := auctions.Auction{
                ID:         uuid.New(),
                ListingID:  listingID,
                SellerID:   sellerID,
                StartPrice: startPrice,
                CurrentBid: 0,
                Currency:   "USD",
                Status:     "active",
                StartsAt:   now,
                EndsAt:     now.Add(endsIn),
        }
        require.NoError(t, db.Create(&a).Error)
        return a
}

func TestPlaceBid_Success(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller4", "seller4@test.com")
        bidderID, bidderToken := createUser(t, db, "Bidder", "bidder@test.com")
        _ = bidderID
        listingID := createListing(t, db, sellerID)
        a := createAuction(t, db, sellerID, listingID, 50.0, 2*time.Hour)

        w := doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), bidderToken, map[string]any{
                "amount": 60.0,
        })
        assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())

        // Verify auction current_bid updated
        var updated auctions.Auction
        db.First(&updated, "id = ?", a.ID)
        assert.Equal(t, 60.0, updated.CurrentBid)
        assert.Equal(t, 1, updated.BidCount)
}

func TestPlaceBid_CannotBidOnOwnAuction(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, sellerToken := createUser(t, db, "Seller5", "seller5@test.com")
        listingID := createListing(t, db, sellerID)
        a := createAuction(t, db, sellerID, listingID, 50.0, 2*time.Hour)

        w := doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), sellerToken, map[string]any{
                "amount": 60.0,
        })
        assert.Equal(t, http.StatusBadRequest, w.Code)
        assert.Contains(t, w.Body.String(), "own auction")
}

func TestPlaceBid_TooLow(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller6", "seller6@test.com")
        _, bidderToken := createUser(t, db, "Bidder2", "bidder2@test.com")
        listingID := createListing(t, db, sellerID)
        a := createAuction(t, db, sellerID, listingID, 100.0, 2*time.Hour)

        // Bid below start price
        w := doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), bidderToken, map[string]any{
                "amount": 50.0,
        })
        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPlaceBid_AuctionEnded(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller7", "seller7@test.com")
        _, bidderToken := createUser(t, db, "Bidder3", "bidder3@test.com")
        listingID := createListing(t, db, sellerID)
        // Create auction that already ended
        a := createAuction(t, db, sellerID, listingID, 50.0, -1*time.Hour)

        w := doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), bidderToken, map[string]any{
                "amount": 60.0,
        })
        // Should fail — auction not found (status=active but ends_at in the past)
        assert.NotEqual(t, http.StatusCreated, w.Code)
}

// ── Auto-bid tests ───────────────────────────────────────────────────────────

func TestAutoBid_TriggeredOnCounterBid(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller8", "seller8@test.com")
        _, autoBidderToken := createUser(t, db, "AutoBidder", "autobidder@test.com")
        _, manualBidderToken := createUser(t, db, "ManualBidder", "manualbidder@test.com")
        listingID := createListing(t, db, sellerID)
        a := createAuction(t, db, sellerID, listingID, 50.0, 2*time.Hour)

        // AutoBidder places an auto-bid with max_amount = 200
        maxAmt := 200.0
        w := doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), autoBidderToken, map[string]any{
                "amount":     60.0,
                "is_auto":    true,
                "max_amount": maxAmt,
        })
        require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

        // Wait for the auto-bid proxy goroutine from the first bid to finish.
        time.Sleep(100 * time.Millisecond)

        // ManualBidder outbids at 70 — should trigger auto-bid counter at 80
        w = doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), manualBidderToken, map[string]any{
                "amount": 70.0,
        })
        require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

        // Give the auto-bid goroutine time to complete
        time.Sleep(200 * time.Millisecond)

        var updated auctions.Auction
        db.First(&updated, "id = ?", a.ID)
        // Auto-bidder should have countered at 80 (70 + bidIncrement=10)
        assert.Equal(t, 80.0, updated.CurrentBid)
}

func TestAutoBid_RespectsMaxAmount(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller9", "seller9@test.com")
        _, autoBidderToken := createUser(t, db, "AutoBidder2", "autobidder2@test.com")
        _, manualBidderToken := createUser(t, db, "ManualBidder2", "manualbidder2@test.com")
        listingID := createListing(t, db, sellerID)
        a := createAuction(t, db, sellerID, listingID, 50.0, 2*time.Hour)

        // AutoBidder: max = 75 (cannot exceed this)
        maxAmt := 75.0
        w := doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), autoBidderToken, map[string]any{
                "amount":     60.0,
                "is_auto":    true,
                "max_amount": maxAmt,
        })
        require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

        // Wait for the auto-bid proxy goroutine from the first bid to finish.
        time.Sleep(100 * time.Millisecond)

        // ManualBidder bids at 70 — auto-bid counter should cap at 75
        w = doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), manualBidderToken, map[string]any{
                "amount": 70.0,
        })
        require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

        time.Sleep(200 * time.Millisecond)

        var updated auctions.Auction
        db.First(&updated, "id = ?", a.ID)
        // Counter should be 75 (capped at max_amount)
        assert.Equal(t, 75.0, updated.CurrentBid)
}

// ── Auction end / finalize tests ─────────────────────────────────────────────

func TestAuctionEndScheduler_EndsAuction(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)

        sellerID, _ := createUser(t, db, "Seller10", "seller10@test.com")
        bidderID, _ := createUser(t, db, "Bidder10", "bidder10@test.com")
        listingID := createListing(t, db, sellerID)

        // Create an auction that ended 1 second ago
        now := time.Now()
        a := auctions.Auction{
                ID:         uuid.New(),
                ListingID:  listingID,
                SellerID:   sellerID,
                StartPrice: 50.0,
                CurrentBid: 0,
                Currency:   "USD",
                Status:     "active",
                StartsAt:   now.Add(-2 * time.Hour),
                EndsAt:     now.Add(-1 * time.Second),
        }
        require.NoError(t, db.Create(&a).Error)

        // Place a bid directly so there's a winner
        bid := auctions.Bid{
                ID:        uuid.New(),
                AuctionID: a.ID,
                UserID:    bidderID,
                Amount:    75.0,
                PlacedAt:  now.Add(-30 * time.Minute),
        }
        require.NoError(t, db.Create(&bid).Error)
        db.Model(&a).Updates(map[string]any{"current_bid": 75.0, "bid_count": 1})

        // Call ProcessEndedAuctions directly (avoids the 60-second ticker delay).
        hub := auctions.NewHub(rdb)
        go hub.Run()
        auctions.ProcessEndedAuctions(db, hub)

        // Use Unscoped to bypass GORM soft-delete filter (sqlite NULL handling edge case).
        var finalized auctions.Auction
        require.NoError(t, db.Unscoped().Where("id = ?", a.ID).First(&finalized).Error)

        assert.Equal(t, "ended", finalized.Status)
        require.NotNil(t, finalized.WinnerID)
        assert.Equal(t, bidderID, *finalized.WinnerID)
}

func TestAuctionEndScheduler_NoWinner(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)

        sellerID, _ := createUser(t, db, "Seller11", "seller11@test.com")
        listingID := createListing(t, db, sellerID)

        now := time.Now()
        a := auctions.Auction{
                ID:         uuid.New(),
                ListingID:  listingID,
                SellerID:   sellerID,
                StartPrice: 50.0,
                CurrentBid: 0,
                Currency:   "USD",
                Status:     "active",
                StartsAt:   now.Add(-2 * time.Hour),
                EndsAt:     now.Add(-1 * time.Second),
        }
        require.NoError(t, db.Create(&a).Error)

        hub := auctions.NewHub(rdb)
        go hub.Run()
        auctions.ProcessEndedAuctions(db, hub)

        var finalized auctions.Auction
        require.NoError(t, db.Unscoped().Where("id = ?", a.ID).First(&finalized).Error)

        assert.Equal(t, "ended", finalized.Status)
        assert.Nil(t, finalized.WinnerID)
}

// ── Search / Complex query tests ─────────────────────────────────────────────

func TestSearchAuctions_StatusFilter(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller12", "seller12@test.com")
        listingID1 := createListing(t, db, sellerID)
        listingID2 := createListing(t, db, sellerID)

        // Create an active auction and an ended one (each with a distinct listing_id).
        active := createAuction(t, db, sellerID, listingID1, 50.0, 24*time.Hour)
        _ = active
        ended := auctions.Auction{
                ID:         uuid.New(),
                ListingID:  listingID2,
                SellerID:   sellerID,
                StartPrice: 50.0,
                CurrentBid: 60.0,
                Currency:   "USD",
                Status:     "ended",
                StartsAt:   time.Now().Add(-48 * time.Hour),
                EndsAt:     time.Now().Add(-1 * time.Hour),
        }
        require.NoError(t, db.Create(&ended).Error)

        w := doGet(t, r, "/api/v1/auctions/search?status=ended", "")
        assert.Equal(t, http.StatusOK, w.Code)

        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data := resp["data"].(map[string]any)
        total := data["total"].(float64)
        assert.GreaterOrEqual(t, int(total), 1)
}

func TestSearchAuctions_SortByBids(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller13", "seller13@test.com")
        listingID1 := createListing(t, db, sellerID)
        listingID2 := createListing(t, db, sellerID)

        a1 := createAuction(t, db, sellerID, listingID1, 50.0, 24*time.Hour)
        a2 := createAuction(t, db, sellerID, listingID2, 50.0, 24*time.Hour)

        // Give a2 more bids
        db.Model(&a1).Update("bid_count", 2)
        db.Model(&a2).Update("bid_count", 10)

        w := doGet(t, r, "/api/v1/auctions/search?sort_by=bids_desc&status=all", "")
        assert.Equal(t, http.StatusOK, w.Code)

        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data := resp["data"].(map[string]any)
        results := data["results"].([]any)
        require.GreaterOrEqual(t, len(results), 2)

        // First result should have more bids
        first := results[0].(map[string]any)
        second := results[1].(map[string]any)
        firstBids := first["bid_count"].(float64)
        secondBids := second["bid_count"].(float64)
        assert.GreaterOrEqual(t, firstBids, secondBids)
}

func TestSearchAuctions_PriceFilter(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller14", "seller14@test.com")
        listingID1 := createListing(t, db, sellerID)
        listingID2 := createListing(t, db, sellerID)

        a1 := createAuction(t, db, sellerID, listingID1, 100.0, 24*time.Hour)
        a2 := createAuction(t, db, sellerID, listingID2, 500.0, 24*time.Hour)
        db.Model(&a1).Update("current_bid", 120.0)
        db.Model(&a2).Update("current_bid", 550.0)

        // Filter: max_price = 200 — should return a1 only
        w := doGet(t, r, "/api/v1/auctions/search?max_price=200", "")
        assert.Equal(t, http.StatusOK, w.Code)

        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data := resp["data"].(map[string]any)
        results := data["results"].([]any)

        for _, item := range results {
                auction := item.(map[string]any)
                // current_bid <= 200 or (bid_count=0 and start_price <= 200)
                currentBid := auction["current_bid"].(float64)
                bidCount := auction["bid_count"].(float64)
                startPrice := auction["start_price"].(float64)
                if bidCount > 0 {
                        assert.LessOrEqual(t, currentBid, 200.0)
                } else {
                        assert.LessOrEqual(t, startPrice, 200.0)
                }
        }
}

// ── WebSocket hub concurrency test ────────────────────────────────────────────

func TestAuctionHub_ConcurrentBroadcast(t *testing.T) {
        rdb := setupMiniredis(t)
        hub := auctions.NewHub(rdb)
        go hub.Run()

        // Verify the hub doesn't panic or deadlock under concurrent broadcasts
        var wg sync.WaitGroup
        auctionID := uuid.New().String()

        for i := 0; i < 50; i++ {
                wg.Add(1)
                go func(n int) {
                        defer wg.Done()
                        msg := &auctions.BroadcastMsg{
                                AuctionID: auctionID,
                                Data:      []byte(fmt.Sprintf(`{"bid": %d}`, n*10)),
                        }
                        hub.Broadcast(msg)
                }(i)
        }

        done := make(chan struct{})
        go func() {
                wg.Wait()
                close(done)
        }()

        select {
        case <-done:
                // success
        case <-time.After(3 * time.Second):
                t.Fatal("concurrent broadcast timed out — possible deadlock")
        }
}

// setupServerWithHub starts a real HTTP test server with the WS endpoint
// registered and returns the server, the hub, and the HTTP router (for REST).
func setupServerWithHub(t *testing.T, db *gorm.DB, rdb *redis.Client) (*httptest.Server, *auctions.Hub, *gin.Engine) {
        t.Helper()
        gin.SetMode(gin.TestMode)
        r := gin.New()
        v1 := r.Group("/api/v1")
        auctions.RegisterRoutes(v1, db, rdb, middleware.NewRateLimiter(rdb))

        hub := auctions.NewHub(rdb)
        go hub.Run()
        // Start the Redis subscriber so bid events published by the handler
        // are forwarded to connected WebSocket clients.
        ctx, cancel := context.WithCancel(context.Background())
        go hub.SubscribeRedis(ctx)
        t.Cleanup(cancel)

        r.GET("/ws/auctions/:id", func(c *gin.Context) {
                auctions.ServeWS(hub, c, db)
        })

        srv := httptest.NewServer(r)
        t.Cleanup(srv.Close)
        return srv, hub, r
}

// wsURL converts an HTTP test-server URL to a WS URL.
func wsURL(httpURL, path string) string {
        return "ws" + strings.TrimPrefix(httpURL, "http") + path
}

// ── WebSocket end-to-end tests ────────────────────────────────────────────────

// TestWS_BidEventDelivered verifies that placing a bid causes a JSON payload
// with the bid amount to be delivered to a subscribed WebSocket client.
func TestWS_BidEventDelivered(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        srv, _, r := setupServerWithHub(t, db, rdb)

        sellerID, _ := createUser(t, db, "WSSeller1", "wsseller1@test.com")
        _, bidderToken := createUser(t, db, "WSBidder1", "wsbidder1@test.com")
        listingID := createListing(t, db, sellerID)
        a := createAuction(t, db, sellerID, listingID, 50.0, 2*time.Hour)

        // Connect a WebSocket client to the auction room.
        wsConn, _, err := websocket.DefaultDialer.Dial(
                wsURL(srv.URL, fmt.Sprintf("/ws/auctions/%s", a.ID)), nil,
        )
        require.NoError(t, err, "WebSocket dial")
        defer wsConn.Close()

        // Give the hub time to register the client.
        time.Sleep(50 * time.Millisecond)

        // Place a bid via the REST API.
        w := doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), bidderToken, map[string]any{
                "amount": 65.0,
        })
        require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

        // The bid handler publishes to Redis; the hub's SubscribeRedis loop is NOT
        // started here (that's for multi-node). Instead the handler publishes to Redis,
        // and since we're in single-node test mode, we verify via the hub Broadcast
        // that's called directly by the handler through the registered hub.
        // Read the WebSocket message with a timeout.
        wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
        _, raw, err := wsConn.ReadMessage()
        require.NoError(t, err, "expected WS message after bid")

        var payload map[string]any
        require.NoError(t, json.Unmarshal(raw, &payload))
        assert.Equal(t, 65.0, payload["bid"].(float64), "bid field in WS payload")
}

// TestWS_AuctionEndedEventDelivered verifies that when the scheduler finalizes
// an auction, an "auction_ended" event is broadcast to WebSocket subscribers.
func TestWS_AuctionEndedEventDelivered(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        srv, hub, _ := setupServerWithHub(t, db, rdb)

        sellerID, _ := createUser(t, db, "WSSeller2", "wsseller2@test.com")
        bidderID, _ := createUser(t, db, "WSBidder2", "wsbidder2@test.com")
        listingID := createListing(t, db, sellerID)

        now := time.Now()
        a := auctions.Auction{
                ID:         uuid.New(),
                ListingID:  listingID,
                SellerID:   sellerID,
                StartPrice: 50.0,
                CurrentBid: 0,
                Currency:   "USD",
                Status:     "active",
                StartsAt:   now.Add(-2 * time.Hour),
                EndsAt:     now.Add(-1 * time.Second),
        }
        require.NoError(t, db.Create(&a).Error)

        bid := auctions.Bid{
                ID:        uuid.New(),
                AuctionID: a.ID,
                UserID:    bidderID,
                Amount:    80.0,
                PlacedAt:  now.Add(-30 * time.Minute),
        }
        require.NoError(t, db.Create(&bid).Error)
        db.Model(&a).Updates(map[string]any{"current_bid": 80.0, "bid_count": 1})

        // Connect WebSocket client.
        wsConn, _, err := websocket.DefaultDialer.Dial(
                wsURL(srv.URL, fmt.Sprintf("/ws/auctions/%s", a.ID)), nil,
        )
        require.NoError(t, err, "WebSocket dial")
        defer wsConn.Close()

        // Give the hub time to register the client.
        time.Sleep(50 * time.Millisecond)

        // Finalize the auction — broadcasts "auction_ended" directly via hub.Broadcast.
        auctions.ProcessEndedAuctions(db, hub)

        // Read the WebSocket message.
        wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
        _, raw, err := wsConn.ReadMessage()
        require.NoError(t, err, "expected WS message after auction ended")

        var payload map[string]any
        require.NoError(t, json.Unmarshal(raw, &payload))
        assert.Equal(t, "auction_ended", payload["event"].(string))
        assert.Equal(t, a.ID.String(), payload["auction_id"].(string))
        assert.Equal(t, 80.0, payload["final_bid"].(float64))
}

// ── GetBids test ─────────────────────────────────────────────────────────────

func TestGetBids(t *testing.T) {
        db := setupDB(t)
        rdb := setupMiniredis(t)
        r := setupRouter(t, db, rdb)

        sellerID, _ := createUser(t, db, "Seller15", "seller15@test.com")
        bidderID, bidderToken := createUser(t, db, "Bidder15", "bidder15@test.com")
        listingID := createListing(t, db, sellerID)
        a := createAuction(t, db, sellerID, listingID, 50.0, 2*time.Hour)

        // Place a bid
        w := doPost(t, r, fmt.Sprintf("/api/v1/auctions/%s/bid", a.ID), bidderToken, map[string]any{
                "amount": 60.0,
        })
        require.Equal(t, http.StatusCreated, w.Code)
        _ = bidderID

        // Get bids
        w = doGet(t, r, fmt.Sprintf("/api/v1/auctions/%s/bids", a.ID), "")
        assert.Equal(t, http.StatusOK, w.Code)

        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        bids := resp["data"].([]any)
        assert.Len(t, bids, 1)
        bid := bids[0].(map[string]any)
        assert.Equal(t, 60.0, bid["amount"].(float64))
}
