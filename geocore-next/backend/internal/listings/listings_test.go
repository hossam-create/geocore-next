package listings_test

import (
        "bytes"
        "encoding/json"
        "fmt"
        "net/http"
        "net/http/httptest"
        "testing"
        "time"

        "github.com/geocore-next/backend/internal/listings"
        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "github.com/stretchr/testify/assert"
        "github.com/stretchr/testify/require"
        "gorm.io/driver/sqlite"
        "gorm.io/gorm"
        "gorm.io/gorm/logger"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func setupListingsDB(t *testing.T) *gorm.DB {
        t.Helper()
        db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
                Logger: logger.Default.LogMode(logger.Silent),
        })
        require.NoError(t, err)

        // Force a single DB connection so goroutines (e.g. fraud detector) share
        // the same in-memory SQLite database instead of getting a fresh empty one.
        sqlDB, err := db.DB()
        require.NoError(t, err)
        sqlDB.SetMaxOpenConns(1)

        // Create a minimal users table compatible with SQLite (no partial indexes).
        // We only need id, email_verified, deleted_at for the listings handler + email_verified guard.
        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS users (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL DEFAULT '',
                email TEXT UNIQUE NOT NULL DEFAULT '',
                phone TEXT,
                password_hash TEXT,
                avatar_url TEXT,
                bio TEXT,
                location TEXT,
                language TEXT DEFAULT 'en',
                currency TEXT DEFAULT 'USD',
                rating REAL DEFAULT 0,
                review_count INTEGER DEFAULT 0,
                sold_count INTEGER DEFAULT 0,
                is_verified INTEGER DEFAULT 0,
                is_active INTEGER DEFAULT 1,
                is_banned INTEGER DEFAULT 0,
                ban_reason TEXT,
                role TEXT DEFAULT 'user',
                balance REAL DEFAULT 0,
                email_verified INTEGER DEFAULT 0,
                verification_token TEXT,
                verification_token_expires_at DATETIME,
                google_id TEXT,
                apple_id TEXT,
                facebook_id TEXT,
                auth_provider TEXT DEFAULT 'email',
                password_reset_token TEXT,
                password_reset_expires_at DATETIME,
                password_changed_at DATETIME,
                stripe_customer_id TEXT,
                subscription_tier TEXT DEFAULT 'basic',
                subscription_expires_at DATETIME,
                created_at DATETIME,
                updated_at DATETIME,
                deleted_at DATETIME
        )`).Error)

        // Use raw DDL for listing tables — GORM AutoMigrate generates PostgreSQL-specific
        // syntax (e.g. GIN indexes) that SQLite does not understand.
        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS categories (
                id TEXT PRIMARY KEY,
                parent_id TEXT,
                name_en TEXT NOT NULL DEFAULT '',
                name_ar TEXT,
                slug TEXT UNIQUE NOT NULL DEFAULT '',
                icon TEXT,
                sort_order INTEGER DEFAULT 0,
                is_active INTEGER DEFAULT 1
        )`).Error)

        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS listings (
                id TEXT PRIMARY KEY,
                user_id TEXT NOT NULL,
                category_id TEXT NOT NULL,
                title TEXT NOT NULL DEFAULT '',
                description TEXT,
                price REAL,
                currency TEXT DEFAULT 'USD',
                price_type TEXT DEFAULT 'fixed',
                condition TEXT,
                status TEXT DEFAULT 'active',
                type TEXT DEFAULT 'sell',
                country TEXT,
                city TEXT,
                address TEXT,
                latitude REAL,
                longitude REAL,
                view_count INTEGER DEFAULT 0,
                favorite_count INTEGER DEFAULT 0,
                is_featured INTEGER DEFAULT 0,
                featured_until DATETIME,
                expires_at DATETIME,
                sold_at DATETIME,
                created_at DATETIME,
                updated_at DATETIME,
                deleted_at DATETIME
        )`).Error)

        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS listing_images (
                id TEXT PRIMARY KEY,
                listing_id TEXT NOT NULL,
                url TEXT NOT NULL DEFAULT '',
                sort_order INTEGER DEFAULT 0,
                is_cover INTEGER DEFAULT 0
        )`).Error)

        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS favorites (
                id TEXT PRIMARY KEY,
                user_id TEXT NOT NULL,
                listing_id TEXT NOT NULL,
                created_at DATETIME
        )`).Error)

        // Minimal bids table so the fraud detector's bidVelocity signal can query
        // it without "no such table" errors when fired asynchronously from Create.
        require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS bids (
                id TEXT PRIMARY KEY,
                auction_id TEXT NOT NULL,
                user_id TEXT NOT NULL,
                amount REAL NOT NULL,
                placed_at DATETIME,
                created_at DATETIME
        )`).Error)

        return db
}

// fakeAuth injects user_id and email_verified directly into the Gin context
// without JWT validation — suitable only for tests.
func fakeAuth(userID string, emailVerified bool) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.Set("user_id", userID)
                c.Set("user_email", "test@example.com")
                c.Set("user_role", "user")
                c.Set("email_verified", emailVerified)
                c.Next()
        }
}

func emailVerifiedGuard(db *gorm.DB) gin.HandlerFunc {
        return func(c *gin.Context) {
                userID := c.GetString("user_id")
                var emailVerified bool
                db.Raw("SELECT email_verified FROM users WHERE id = ? AND deleted_at IS NULL", userID).Scan(&emailVerified)
                if !emailVerified {
                        c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                                "success": false,
                                "message": "Please verify your email address before performing this action",
                                "code":    "EMAIL_NOT_VERIFIED",
                        })
                        return
                }
                c.Next()
        }
}

func setupListingsRouter(db *gorm.DB, actingUserID string, emailVerified bool) *gin.Engine {
        gin.SetMode(gin.TestMode)
        r := gin.New()
        h := listings.NewHandler(db, nil)

        v1 := r.Group("/api/v1")

        v1.GET("/categories", h.GetCategories)
        v1.GET("/listings/search", h.Search)
        v1.GET("/listings/suggestions", h.Suggestions)

        lst := v1.Group("/listings")
        lst.GET("", h.List)
        lst.GET("/:id", h.Get)

        authed := lst.Group("")
        authed.Use(fakeAuth(actingUserID, emailVerified))
        authed.GET("/me", h.GetMyListings)
        authed.PUT("/:id", h.Update)
        authed.DELETE("/:id", h.Delete)
        authed.POST("/:id/favorite", h.ToggleFavorite)

        verified := lst.Group("")
        verified.Use(fakeAuth(actingUserID, emailVerified), emailVerifiedGuard(db))
        verified.POST("", h.Create)

        return r
}

func jsonBodyL(t *testing.T, payload any) *bytes.Buffer {
        t.Helper()
        b, err := json.Marshal(payload)
        require.NoError(t, err)
        return bytes.NewBuffer(b)
}

// createTestUser inserts a user directly via raw SQL and returns their ID string.
// Raw SQL is used to avoid importing users.User which has SQLite-incompatible partial indexes.
func createTestUser(t *testing.T, db *gorm.DB, emailVerified bool) string {
        t.Helper()
        id := uuid.New()
        email := fmt.Sprintf("user-%s@test.com", uuid.New().String()[:8])
        now := time.Now().UTC().Format("2006-01-02 15:04:05")
        emailVerifiedInt := 0
        if emailVerified {
                emailVerifiedInt = 1
        }
        require.NoError(t, db.Exec(
                `INSERT INTO users (id, name, email, email_verified, role, created_at, updated_at)
                 VALUES (?, 'Test User', ?, ?, 'user', ?, ?)`,
                id.String(), email, emailVerifiedInt, now, now,
        ).Error)
        return id.String()
}

// createTestCategory inserts a category into the DB and returns its ID.
func createTestCategory(t *testing.T, db *gorm.DB) uuid.UUID {
        t.Helper()
        cat := listings.Category{
                ID:       uuid.New(),
                NameEn:   "Electronics",
                Slug:     fmt.Sprintf("electronics-%s", uuid.New().String()[:6]),
                IsActive: true,
        }
        require.NoError(t, db.Create(&cat).Error)
        return cat.ID
}

// createTestListing creates a listing directly in the DB and returns it.
func createTestListing(t *testing.T, db *gorm.DB, userID uuid.UUID, catID uuid.UUID) listings.Listing {
        t.Helper()
        price := 100.0
        listing := listings.Listing{
                ID:          uuid.New(),
                UserID:      userID,
                CategoryID:  catID,
                Title:       "Test iPhone 12",
                Description: "A great phone in good condition",
                Price:       &price,
                Currency:    "USD",
                PriceType:   "fixed",
                Condition:   "good",
                Type:        "sell",
                Country:     "UAE",
                City:        "Dubai",
                Status:      "active",
        }
        require.NoError(t, db.Create(&listing).Error)
        return listing
}

// ── Create tests ─────────────────────────────────────────────────────────────

func TestCreate_Success(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{
                "category_id": catID.String(),
                "title":       "Brand New Laptop",
                "description": "Excellent condition, barely used",
                "price":       999.99,
                "currency":    "USD",
                "price_type":  "fixed",
                "condition":   "new",
                "type":        "sell",
                "country":     "UAE",
                "city":        "Dubai",
        })
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings", body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusCreated, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        assert.True(t, resp["success"].(bool))
}

func TestCreate_MissingRequiredFields(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{
                "title": "Short",
        })
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings", body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_InvalidCondition(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{
                "category_id": catID.String(),
                "title":       "Some Item For Sale",
                "description": "This item is in great condition",
                "condition":   "broken",
                "country":     "UAE",
                "city":        "Dubai",
        })
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings", body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_InvalidType(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{
                "category_id": catID.String(),
                "title":       "Some Item For Sale",
                "description": "This item is in great condition",
                "type":        "swap",
                "country":     "UAE",
                "city":        "Dubai",
        })
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings", body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_EmailNotVerified(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, false)
        catID := createTestCategory(t, db)
        r := setupListingsRouter(db, userID, false)

        body := jsonBodyL(t, map[string]any{
                "category_id": catID.String(),
                "title":       "Brand New Laptop",
                "description": "Excellent condition, barely used",
                "country":     "UAE",
                "city":        "Dubai",
        })
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings", body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCreate_NoCategoryProvided(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{
                "title":       "Some Listing Title",
                "description": "A detailed description here",
                "country":     "UAE",
                "city":        "Dubai",
        })
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings", body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Read tests ────────────────────────────────────────────────────────────────

func TestGet_Success(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/"+listing.ID.String(), nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        assert.True(t, resp["success"].(bool))
}

func TestGet_InvalidID(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/not-a-uuid", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGet_NotFound(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/"+uuid.New().String(), nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestList_Basic(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_InvalidMinPrice(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?min_price=notanumber", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_NegativeMinPrice(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?min_price=-50", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_InvalidMaxPrice(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?max_price=abc", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_InvalidSellerID(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?seller_id=invalid-uuid", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_EmptyResults(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?city=NonExistentCity", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_Pagination(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        for i := 0; i < 5; i++ {
                createTestListing(t, db, uuid.MustParse(userID), catID)
        }
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?page=1&per_page=2", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        meta, ok := resp["meta"].(map[string]any)
        require.True(t, ok, "expected meta in response")
        assert.Equal(t, float64(2), meta["per_page"])
}

// ── Update tests ──────────────────────────────────────────────────────────────

func TestUpdate_Success(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{
                "title": "Updated iPhone 12 Title",
        })
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPut, "/api/v1/listings/"+listing.ID.String(), body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        assert.True(t, resp["success"].(bool))
}

func TestUpdate_Unauthorized_OtherUser(t *testing.T) {
        db := setupListingsDB(t)
        ownerID := createTestUser(t, db, true)
        otherID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(ownerID), catID)
        r := setupListingsRouter(db, otherID, true)

        body := jsonBodyL(t, map[string]any{"title": "Hacked title update"})
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPut, "/api/v1/listings/"+listing.ID.String(), body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdate_InvalidCondition(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{"condition": "broken"})
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPut, "/api/v1/listings/"+listing.ID.String(), body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_InvalidType(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{"type": "barter"})
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPut, "/api/v1/listings/"+listing.ID.String(), body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_InvalidStatus(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{"status": "banned"})
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPut, "/api/v1/listings/"+listing.ID.String(), body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_NegativePrice(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{"price": -50.0})
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPut, "/api/v1/listings/"+listing.ID.String(), body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdate_InvalidID(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        body := jsonBodyL(t, map[string]any{"title": "New Title"})
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPut, "/api/v1/listings/bad-id", body)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── Delete tests ──────────────────────────────────────────────────────────────

func TestDelete_Success(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodDelete, "/api/v1/listings/"+listing.ID.String(), nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestDelete_Unauthorized_OtherUser(t *testing.T) {
        db := setupListingsDB(t)
        ownerID := createTestUser(t, db, true)
        otherID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(ownerID), catID)
        r := setupListingsRouter(db, otherID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodDelete, "/api/v1/listings/"+listing.ID.String(), nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDelete_InvalidID(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodDelete, "/api/v1/listings/not-a-uuid", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDelete_NotFound(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodDelete, "/api/v1/listings/"+uuid.New().String(), nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusNotFound, w.Code)
}

// ── GetMyListings tests ───────────────────────────────────────────────────────

func TestGetMyListings_ReturnsOnlyOwn(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        otherID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        createTestListing(t, db, uuid.MustParse(otherID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/me", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data, ok := resp["data"].([]any)
        require.True(t, ok, "expected data array")
        assert.Len(t, data, 1, "should only return listings owned by the authenticated user")
}

// ── ToggleFavorite tests ──────────────────────────────────────────────────────

func TestToggleFavorite_AddAndRemove(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        listing := createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        // First call — should favorite
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings/"+listing.ID.String()+"/favorite", nil)
        r.ServeHTTP(w, req)
        assert.Equal(t, http.StatusOK, w.Code)
        var resp1 map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp1))
        data1 := resp1["data"].(map[string]any)
        assert.True(t, data1["favorited"].(bool))

        // Second call — should unfavorite
        w2 := httptest.NewRecorder()
        req2, _ := http.NewRequest(http.MethodPost, "/api/v1/listings/"+listing.ID.String()+"/favorite", nil)
        r.ServeHTTP(w2, req2)
        assert.Equal(t, http.StatusOK, w2.Code)
        var resp2 map[string]any
        require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
        data2 := resp2["data"].(map[string]any)
        assert.False(t, data2["favorited"].(bool))
}

func TestToggleFavorite_InvalidID(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings/bad-id/favorite", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ── CRUD lifecycle test ───────────────────────────────────────────────────────

func TestListingCRUDLifecycle(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        r := setupListingsRouter(db, userID, true)

        // Create
        createBody := jsonBodyL(t, map[string]any{
                "category_id": catID.String(),
                "title":       "Lifecycle Test Listing",
                "description": "A complete lifecycle test for CRUD",
                "price":       250.0,
                "currency":    "USD",
                "price_type":  "fixed",
                "condition":   "good",
                "type":        "sell",
                "country":     "UAE",
                "city":        "Abu Dhabi",
        })
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodPost, "/api/v1/listings", createBody)
        req.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w, req)
        require.Equal(t, http.StatusCreated, w.Code)

        var createResp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
        data := createResp["data"].(map[string]any)
        listingID := data["id"].(string)

        // Read
        w2 := httptest.NewRecorder()
        req2, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/"+listingID, nil)
        r.ServeHTTP(w2, req2)
        assert.Equal(t, http.StatusOK, w2.Code)

        // Update
        updateBody := jsonBodyL(t, map[string]any{
                "title":  "Updated Lifecycle Title",
                "status": "sold",
        })
        w3 := httptest.NewRecorder()
        req3, _ := http.NewRequest(http.MethodPut, "/api/v1/listings/"+listingID, updateBody)
        req3.Header.Set("Content-Type", "application/json")
        r.ServeHTTP(w3, req3)
        assert.Equal(t, http.StatusOK, w3.Code)

        // Delete
        w4 := httptest.NewRecorder()
        req4, _ := http.NewRequest(http.MethodDelete, "/api/v1/listings/"+listingID, nil)
        r.ServeHTTP(w4, req4)
        assert.Equal(t, http.StatusOK, w4.Code)

        // Confirm gone
        w5 := httptest.NewRecorder()
        req5, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/"+listingID, nil)
        r.ServeHTTP(w5, req5)
        assert.Equal(t, http.StatusNotFound, w5.Code)
}

// ── GetCategories test ────────────────────────────────────────────────────────

func TestGetCategories(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        createTestCategory(t, db)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/categories", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

// ── List price range validation tests ─────────────────────────────────────────

func TestList_MinPriceGreaterThanMaxPrice(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?min_price=500&max_price=100", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestList_ValidPriceRange(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?min_price=50&max_price=500", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestList_EqualPriceRange(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings?min_price=100&max_price=100", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

// ── Search endpoint-level tests ───────────────────────────────────────────────

func TestSearchEndpoint_Basic(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        assert.True(t, resp["success"].(bool))
        data := resp["data"].(map[string]any)
        assert.NotNil(t, data["results"])
        assert.NotNil(t, data["total"])
        assert.NotNil(t, data["page"])
        assert.NotNil(t, data["per_page"])
}

func TestSearchEndpoint_EmptyResults(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?q=zzznomatch999", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data := resp["data"].(map[string]any)
        results := data["results"].([]any)
        assert.Len(t, results, 0)
        assert.Equal(t, float64(0), data["total"])
}

func TestSearchEndpoint_FilterByCountry(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?country=UAE", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchEndpoint_FilterByCity(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?city=Dubai", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchEndpoint_FilterByPriceRange(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?min_price=50&max_price=500", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchEndpoint_FilterByCondition(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?condition=good", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchEndpoint_FilterByType(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?type=sell", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchEndpoint_FilterByStatus_Valid(t *testing.T) {
        // Only publicly safe statuses are accepted; draft and pending are owner-internal
        validStatuses := []string{"active", "sold", "reserved", "expired"}
        for _, status := range validStatuses {
                t.Run(status, func(t *testing.T) {
                        db := setupListingsDB(t)
                        userID := createTestUser(t, db, true)
                        r := setupListingsRouter(db, userID, true)

                        w := httptest.NewRecorder()
                        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?status="+status, nil)
                        r.ServeHTTP(w, req)

                        assert.Equal(t, http.StatusOK, w.Code, "status=%s should be accepted", status)
                        var resp map[string]any
                        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
                        assert.True(t, resp["success"].(bool))
                        data := resp["data"].(map[string]any)
                        assert.NotNil(t, data["results"], "results must be present for status=%s", status)
                })
        }
}

func TestSearchEndpoint_FilterByStatus_Invalid(t *testing.T) {
        // These statuses must be rejected — either unknown or owner-internal
        invalidStatuses := []string{"deleted", "draft", "pending", "published"}
        for _, status := range invalidStatuses {
                t.Run(status, func(t *testing.T) {
                        db := setupListingsDB(t)
                        userID := createTestUser(t, db, true)
                        r := setupListingsRouter(db, userID, true)

                        w := httptest.NewRecorder()
                        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?status="+status, nil)
                        r.ServeHTTP(w, req)

                        assert.Equal(t, http.StatusBadRequest, w.Code, "status=%s should be rejected from public search", status)
                })
        }
}

func TestSearchEndpoint_FilterByStatus_Default(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        // No status param — should default to active and succeed
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        assert.True(t, resp["success"].(bool))
}

func TestSearchEndpoint_InvalidCondition(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?condition=broken", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchEndpoint_InvalidType(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?type=swap", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchEndpoint_InvalidSortBy(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?sort_by=popularity", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchEndpoint_NegativeMinPrice(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?min_price=-10", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchEndpoint_MinExceedsZeroMax(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        // min_price=100 with max_price=0 must be rejected (100 > 0)
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?min_price=100&max_price=0", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchEndpoint_MinGreaterThanMax(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?min_price=1000&max_price=100", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchEndpoint_PaginationBoundary(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        for i := 0; i < 5; i++ {
                createTestListing(t, db, uuid.MustParse(userID), catID)
        }
        r := setupListingsRouter(db, userID, true)

        // Request with specific pagination parameters — response must include pagination fields
        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?page=2&per_page=2", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data := resp["data"].(map[string]any)
        // Verify pagination fields are echoed back correctly
        assert.Equal(t, float64(2), data["page"], "page should be echoed back")
        assert.Equal(t, float64(2), data["per_page"], "per_page should be echoed back")
        assert.NotNil(t, data["pages"], "pages field must be present")
        assert.NotNil(t, data["results"], "results field must be present")

        // Request a page beyond available — should return empty results, not an error
        w2 := httptest.NewRecorder()
        req2, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?page=100&per_page=20", nil)
        r.ServeHTTP(w2, req2)
        assert.Equal(t, http.StatusOK, w2.Code)
        var resp2 map[string]any
        require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
        data2 := resp2["data"].(map[string]any)
        // Results should be an empty array (not an error)
        results2, _ := data2["results"].([]any)
        assert.Len(t, results2, 0, "page beyond available results should return empty array")
}

func TestSearchEndpoint_AllFiltersCombined(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet,
                fmt.Sprintf("/api/v1/listings/search?country=UAE&city=Dubai&condition=good&type=sell&min_price=50&max_price=500&sort_by=price_asc&page=1&per_page=10&category_id=%s", catID.String()),
                nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        assert.True(t, resp["success"].(bool))
}

func TestSearchEndpoint_SortByPriceAsc(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?sort_by=price_asc", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchEndpoint_SortByDate(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?sort_by=date", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchEndpoint_SortByRelevance(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/search?q=iPhone&sort_by=relevance", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
}

func TestSearchEndpoint_Suggestions(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        catID := createTestCategory(t, db)
        createTestListing(t, db, uuid.MustParse(userID), catID)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/suggestions?q=Te", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data := resp["data"].(map[string]any)
        // Suggestions field should always be present (may be empty in test env)
        _, exists := data["suggestions"]
        assert.True(t, exists, "response should contain 'suggestions' key")
}

func TestSearchEndpoint_SuggestionsShortQuery(t *testing.T) {
        db := setupListingsDB(t)
        userID := createTestUser(t, db, true)
        r := setupListingsRouter(db, userID, true)

        w := httptest.NewRecorder()
        req, _ := http.NewRequest(http.MethodGet, "/api/v1/listings/suggestions?q=i", nil)
        r.ServeHTTP(w, req)

        assert.Equal(t, http.StatusOK, w.Code)
        var resp map[string]any
        require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
        data := resp["data"].(map[string]any)
        suggestions := data["suggestions"].([]any)
        assert.Len(t, suggestions, 0, "short query should return empty suggestions")
}
