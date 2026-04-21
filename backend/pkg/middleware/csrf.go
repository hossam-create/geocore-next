package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "gc_csrf_token"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenTTL   = 24 * time.Hour
)

// EnsureCSRFCookie issues a CSRF cookie if absent and returns the current token.
func EnsureCSRFCookie(c *gin.Context) string {
	if tok, err := c.Cookie(csrfCookieName); err == nil && tok != "" {
		return tok
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	token := hex.EncodeToString(b)
	secure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(csrfCookieName, token, int(csrfTokenTTL.Seconds()), "/", "", secure, false)
	return token
}

// CSRF enforces double-submit CSRF protection for cookie-authenticated, state-changing requests.
// API Bearer-token requests are skipped (not vulnerable to browser CSRF in this form).
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			c.Next()
			return
		}

		// No cookies present -> skip (non-browser clients).
		if c.GetHeader("Cookie") == "" {
			c.Next()
			return
		}

		cookieToken, err := c.Cookie(csrfCookieName)
		if err != nil || cookieToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token missing"})
			return
		}
		headToken := c.GetHeader(csrfHeaderName)
		if headToken == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token header missing"})
			return
		}

		if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "csrf token invalid"})
			return
		}

		c.Next()
	}
}
