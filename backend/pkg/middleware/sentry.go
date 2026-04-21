package middleware

import (
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

// InitSentry initializes Sentry SDK
// Returns nil if DSN is empty (safe for development)
func InitSentry(dsn, environment, release string) error {
	if dsn == "" {
		return nil // Sentry disabled
	}

	return sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		Release:          release,
		TracesSampleRate: 0.2, // 20% of transactions
		AttachStacktrace: true,
	})
}

// SentryMiddleware returns a Gin middleware that captures panics and errors
func SentryMiddleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{
		Repanic: false,
		Timeout: 5 * time.Second,
	})
}

// CaptureError captures an error to Sentry manually
func CaptureError(err error) {
	if sentry.CurrentHub().Client() != nil {
		sentry.CaptureException(err)
	}
}

// CaptureMessage captures a message to Sentry manually
func CaptureMessage(msg string) {
	if sentry.CurrentHub().Client() != nil {
		sentry.CaptureMessage(msg)
	}
}

// SetUserContext sets user context for Sentry events
func SetUserContext(userID, email, name string) {
	if sentry.CurrentHub().Client() != nil {
		sentry.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetUser(sentry.User{
				ID:    userID,
				Email: email,
				Name:  name,
			})
		})
	}
}

// FlushSentry flushes Sentry events before shutdown
func FlushSentry() {
	sentry.Flush(2 * time.Second)
}

// RecoveryMiddleware is a custom recovery middleware that also reports to Sentry
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Report to Sentry
				if sentry.CurrentHub().Client() != nil {
					sentry.CurrentHub().Recover(err)
					sentry.Flush(2 * time.Second)
				}

				// Return error response
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
			}
		}()
		c.Next()
	}
}
