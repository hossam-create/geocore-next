package images_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geocore-next/backend/internal/images"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func setupUploadURLDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db
}

func setupUploadURLRouter(t *testing.T, userID string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	db := setupUploadURLDB(t)

	// NewHandler calls NewR2ClientFromEnv() which returns nil when R2 env vars
	// are not set. In that case, GetUploadURL falls back to returning mock URLs —
	// exactly what we want for test coverage.
	h := images.NewHandler(db)

	v1 := r.Group("/api/v1")
	media := v1.Group("/media")
	media.Use(func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Next()
	})
	media.POST("/upload-url", h.GetUploadURL)

	return r
}

func jsonBodyImg(t *testing.T, payload any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	return bytes.NewBuffer(b)
}

// ── GetUploadURL tests ────────────────────────────────────────────────────────

func TestGetUploadURL_JPEG(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "photo.jpg",
		"content_type": "image/jpeg",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]any)
	assert.NotEmpty(t, data["upload_url"])
	assert.NotEmpty(t, data["public_url"])
	assert.NotEmpty(t, data["key"])
}

func TestGetUploadURL_PNG(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "photo.png",
		"content_type": "image/png",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]any)
	assert.Contains(t, data["key"].(string), ".png")
}

func TestGetUploadURL_WebP(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "photo.webp",
		"content_type": "image/webp",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))
	data := resp["data"].(map[string]any)
	assert.Contains(t, data["key"].(string), ".webp")
}

func TestGetUploadURL_GIF(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "anim.gif",
		"content_type": "image/gif",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUploadURL_CustomFolder(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "image.jpg",
		"content_type": "image/jpeg",
		"folder":       "listings/2026",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Contains(t, data["key"].(string), "listings/2026/")
}

func TestGetUploadURL_DefaultFolder(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "image.jpg",
		"content_type": "image/jpeg",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Contains(t, data["key"].(string), "uploads/")
}

func TestGetUploadURL_UnsupportedContentType(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "video.mp4",
		"content_type": "video/mp4",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
}

func TestGetUploadURL_UnsupportedPDF(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "doc.pdf",
		"content_type": "application/pdf",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetUploadURL_MissingFilename(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"content_type": "image/jpeg",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetUploadURL_MissingContentType(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename": "photo.jpg",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetUploadURL_FileTooLarge(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "huge.jpg",
		"content_type": "image/jpeg",
		"size":         int64(20 * 1024 * 1024), // 20 MB — exceeds 10 MB limit
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
}

func TestGetUploadURL_AcceptableSize(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "photo.jpg",
		"content_type": "image/jpeg",
		"size":         int64(5 * 1024 * 1024), // 5 MB — within 10 MB limit
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUploadURL_ZeroSize(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "photo.jpg",
		"content_type": "image/jpeg",
		"size":         int64(0), // 0 size should be treated as unset, allowed
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetUploadURL_ResponseShape(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "listing-photo.jpg",
		"content_type": "image/jpeg",
		"folder":       "listings",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]any)
	// Verify all required fields are present
	assert.NotEmpty(t, data["upload_url"], "upload_url must be present")
	assert.NotEmpty(t, data["public_url"], "public_url must be present")
	assert.NotEmpty(t, data["key"], "key must be present")
	assert.NotNil(t, data["expires_in"], "expires_in must be present")
	assert.NotNil(t, data["max_size_bytes"], "max_size_bytes must be present")
	assert.Equal(t, float64(300), data["expires_in"])
	assert.Equal(t, float64(10*1024*1024), data["max_size_bytes"])
}

// TestGetUploadURL_KeyUniqueness verifies that each upload URL request returns
// a unique key — preventing collisions between concurrent uploads.
func TestGetUploadURL_KeyUniqueness(t *testing.T) {
	r := setupUploadURLRouter(t, "test-user-id")

	makeRequest := func() string {
		body := jsonBodyImg(t, map[string]any{
			"filename":     "photo.jpg",
			"content_type": "image/jpeg",
		})
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
		data := resp["data"].(map[string]any)
		return data["key"].(string)
	}

	key1 := makeRequest()
	key2 := makeRequest()
	key3 := makeRequest()

	assert.NotEqual(t, key1, key2, "each upload should generate a unique key")
	assert.NotEqual(t, key2, key3, "each upload should generate a unique key")
	assert.NotEqual(t, key1, key3, "each upload should generate a unique key")
}

// TestGetUploadURL_PublicURLLinkedToListing verifies the complete workflow:
// the public_url from GetUploadURL is suitable for use in listing image_urls.
// The returned URL is non-empty and can be stored as a listing image reference.
func TestGetUploadURL_PublicURLLinkedToListing(t *testing.T) {
	r := setupUploadURLRouter(t, "some-user-id")

	body := jsonBodyImg(t, map[string]any{
		"filename":     "item.jpg",
		"content_type": "image/jpeg",
		"folder":       "listings",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/media/upload-url", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var uploadResp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &uploadResp))
	data := uploadResp["data"].(map[string]any)
	publicURL := data["public_url"].(string)

	// The public_url must be a valid non-empty string suitable for storage on a listing
	assert.NotEmpty(t, publicURL, "public_url should be non-empty")
	assert.Contains(t, publicURL, "http", "public_url should be a valid URL")

	uploadURL := data["upload_url"].(string)
	assert.NotEmpty(t, uploadURL, "upload_url should be non-empty")

	// In test mode (no R2 configured), mock response is returned
	_, hasMock := data["_mock"]
	if hasMock {
		// Mock mode: URLs are placeholder but structurally valid
		assert.Contains(t, uploadURL, "mock", "dev mode upload_url should indicate mock")
	} else {
		// Production mode: real presigned URL
		assert.Contains(t, uploadURL, "https://", "presigned URL should use HTTPS")
	}
}
