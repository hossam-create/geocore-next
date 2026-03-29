package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders adds security-related HTTP headers to all responses.
// These headers help protect against common web vulnerabilities:
// - XSS attacks
// - Clickjacking
// - MIME type sniffing
// - Man-in-the-middle attacks
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")
		
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")
		
		// Enable XSS filter in browsers
		c.Header("X-XSS-Protection", "1; mode=block")
		
		// Force HTTPS for 1 year (only in production)
		if c.Request.URL.Scheme == "https" || c.Request.Header.Get("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}
		
		// Control referrer information
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Prevent browser from caching sensitive data
		c.Header("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		
		// Permissions Policy (formerly Feature Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		
		c.Next()
	}
}

// ContentSecurityPolicy adds CSP headers for additional XSS protection.
// This is more restrictive and should be customized based on your frontend needs.
func ContentSecurityPolicy() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic CSP - customize based on your frontend requirements
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' https://fonts.gstatic.com; " +
			"img-src 'self' data: https: blob:; " +
			"connect-src 'self' https://api.stripe.com https://api.posthog.com; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"
		
		c.Header("Content-Security-Policy", csp)
		c.Next()
	}
}

// NoSniff prevents browsers from interpreting files as a different MIME type
func NoSniff() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}
