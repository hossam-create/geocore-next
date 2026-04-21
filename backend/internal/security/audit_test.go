package security

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAssessRisk_SessionRevoked_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("User-Agent", "Go-http-client/1.1")

	score := assessRisk(EventSessionRevoked, map[string]any{"reason": "logout"}, c)
	assert.Equal(t, 70, score)
}

func TestAssessRisk_SessionRevoked_RefreshReuse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("User-Agent", "Go-http-client/1.1")

	score := assessRisk(EventSessionRevoked, map[string]any{"reason": "refresh_token_reuse_detected"}, c)
	assert.Equal(t, 90, score)
}
