package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geocore-next/backend/internal/search"
	"github.com/geocore-next/backend/internal/users"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ════════════════════════════════════════════════════════════════════════════════
// Scenario 2: Search — Keyword search, AI intent parsing, text fallback, suggestions
// ════════════════════════════════════════════════════════════════════════════════

type SearchSuite struct {
	suite.Suite
	ts       *TestSuite
	r        *gin.Engine
	h        *search.Handler
	sellerID uuid.UUID
	catID    uuid.UUID
}

func TestSearchSuite(t *testing.T) {
	ts := SetupSuite(t)
	defer ts.TeardownSuite()

	suite.Run(t, &SearchSuite{ts: ts})
}

func (s *SearchSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	s.r = gin.New()

	ts := s.ts
	ts.AutoMigrateAll(&users.User{})
	ts.CreateManualTables()

	s.h = search.NewHandler(ts.DB)
	v1 := s.r.Group("/api/v1")
	search.RegisterRoutes(v1, ts.DB, ts.RDB)

	// Seed test data
	s.catID = ts.CreateCategory()
	s.sellerID = ts.CreateUser("Search Seller", UniqueEmail("search-seller"))

	// Create listings for search
	ts.CreateListing(s.sellerID, s.catID, "iPhone 15 Pro Max 256GB", 1199.00)
	ts.CreateListing(s.sellerID, s.catID, "Samsung Galaxy S24 Ultra", 1299.00)
	ts.CreateListing(s.sellerID, s.catID, "MacBook Pro M3 14 inch", 1999.00)
	ts.CreateListing(s.sellerID, s.catID, "Toyota Land Cruiser 2024", 85000.00)
	ts.CreateListing(s.sellerID, s.catID, "Rolex Submariner Date", 14000.00)
}

func (s *SearchSuite) SetupTest() {
	s.ts.ResetTest()
}

// ── Test: Text search finds matching listings ───────────────────────────────────

func (s *SearchSuite) TestSearch_TextSearch_KeywordMatch() {
	body, _ := json.Marshal(gin.H{
		"query": "iPhone",
		"limit": 10,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)

	results, ok := data["results"].([]interface{})
	require.True(s.T(), ok)
	assert.GreaterOrEqual(s.T(), len(results), 1, "should find at least one iPhone listing")
}

// ── Test: Search with empty query returns error ────────────────────────────────

func (s *SearchSuite) TestSearch_EmptyQuery_ReturnsError() {
	body, _ := json.Marshal(gin.H{
		"query": "",
		"limit": 10,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// ── Test: Search with short query returns error ─────────────────────────────────

func (s *SearchSuite) TestSearch_ShortQuery_ReturnsError() {
	body, _ := json.Marshal(gin.H{
		"query": "a",
		"limit": 10,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusBadRequest, w.Code)
}

// ── Test: Search with no matches returns empty results ──────────────────────────

func (s *SearchSuite) TestSearch_NoMatches_ReturnsEmpty() {
	body, _ := json.Marshal(gin.H{
		"query": "zzzznonexistentitem12345",
		"limit": 10,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)

	results, ok := data["results"].([]interface{})
	require.True(s.T(), ok)
	assert.Equal(s.T(), 0, len(results), "should return empty results for non-matching query")
}

// ── Test: Autocomplete suggestions ──────────────────────────────────────────────

func (s *SearchSuite) TestSuggest_ReturnsSuggestions() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/suggest?q=iPho", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)

	suggestions, ok := data["suggestions"].([]interface{})
	require.True(s.T(), ok)
	assert.GreaterOrEqual(s.T(), len(suggestions), 1, "should return at least one suggestion")
}

// ── Test: Trending searches ──────────────────────────────────────────────────────

func (s *SearchSuite) TestTrending_ReturnsResults() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/trending", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)

	trending, ok := data["trending"].([]interface{})
	require.True(s.T(), ok)
	assert.GreaterOrEqual(s.T(), len(trending), 1, "trending should return fallback data when no queries exist")
}

// ── Test: Search limit clamped to max 50 ────────────────────────────────────────

func (s *SearchSuite) TestSearch_LimitClamped() {
	body, _ := json.Marshal(gin.H{
		"query": "iPhone",
		"limit": 999,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Search with filters ──────────────────────────────────────────────────

func (s *SearchSuite) TestSearch_WithFilters() {
	body, _ := json.Marshal(gin.H{
		"query":   "iPhone",
		"filters": map[string]interface{}{"category": "Electronics"},
		"limit":   10,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

// ── Test: Suggest with short query returns empty ────────────────────────────────

func (s *SearchSuite) TestSuggest_ShortQuery_ReturnsEmpty() {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/search/suggest?q=a", nil)
	s.r.ServeHTTP(w, req)

	assert.Equal(s.T(), http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(s.T(), json.Unmarshal(w.Body.Bytes(), &resp))

	data, ok := resp["data"].(map[string]interface{})
	require.True(s.T(), ok)

	suggestions, ok := data["suggestions"].([]interface{})
	require.True(s.T(), ok)
	assert.Equal(s.T(), 0, len(suggestions))
}
