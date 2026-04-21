package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the HTTP header used to propagate the request ID.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the gin context key for the request ID.
	RequestIDKey = "request_id"
)

// RequestID generates a unique request ID for every incoming request if one
// is not already provided via the X-Request-ID header. The ID is stored in
// the gin context and set as a response header so callers can correlate
// logs, jobs, and DB queries to a single request.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader(RequestIDHeader)
		if rid == "" {
			rid = uuid.New().String()
		}
		c.Set(RequestIDKey, rid)
		c.Header(RequestIDHeader, rid)
		c.Next()
	}
}

// GetRequestID extracts the request ID from the gin context.
func GetRequestID(c *gin.Context) string {
	if v, ok := c.Get(RequestIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
