package compliance

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// auditedRoute describes a financial / dispute action that must be appended
// to the immutable audit chain whenever the handler completes successfully
// (HTTP 2xx). Using FullPath() keeps matching O(1).
type auditedRoute struct {
	category string
	action   string
}

// Registry of financial / dispute endpoints that require regulatory audit.
// Add routes here — the middleware picks them up automatically.
var auditedRoutes = map[string]auditedRoute{
	// Exchange (Sprint 19)
	"POST /api/v1/exchange/requests":                  {CategoryExchange, "request_created"},
	"POST /api/v1/exchange/requests/:id/match":        {CategoryExchange, "request_matched"},
	"DELETE /api/v1/exchange/requests/:id":            {CategoryExchange, "request_cancelled"},
	"POST /api/v1/exchange/:id/upload-proof":          {CategoryExchange, "proof_uploaded"},
	"POST /api/v1/exchange/:id/verify":                {CategoryExchange, "proof_verified"},
	"POST /api/v1/exchange/:id/dispute":               {CategoryDispute, "exchange_disputed"},

	// Payouts / Withdrawals
	"POST /api/v1/payments/withdraw/request":          {CategoryPayout, "withdraw_requested"},
	"DELETE /api/v1/payments/withdraw/:id/cancel":     {CategoryPayout, "withdraw_cancelled"},
	"POST /api/v1/agent/withdraw/:id/complete":        {CategoryPayout, "withdraw_completed"},

	// Disputes (standalone)
	"POST /api/v1/disputes":                           {CategoryDispute, "dispute_opened"},
	"POST /api/v1/disputes/:id/resolve":               {CategoryDispute, "dispute_resolved"},

	// Admin financial overrides
	"POST /api/v1/exchange/admin/matches/:id/auto-resolve": {CategoryAdmin, "exchange_auto_resolve"},
	"POST /api/v1/exchange/admin/users/:user_id/tier":      {CategoryAdmin, "vip_tier_changed"},
}

// AuditMiddleware appends an immutable audit row to the compliance chain for
// every successful (2xx/3xx) call to a registered financial endpoint.
//
// It must run BEFORE the handler so it can inspect the final status via
// c.Next() → c.Writer.Status(). The write is fire-and-forget (goroutine) so
// audit-store latency never slows the user response.
func AuditMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only act on registered routes, and only on success.
		key := c.Request.Method + " " + c.FullPath()
		route, ok := auditedRoutes[key]
		if !ok {
			return
		}
		status := c.Writer.Status()
		if status < 200 || status >= 400 {
			return
		}

		userID := parseUUID(c.GetString("user_id"))
		resourceID := firstParam(c, "id", "user_id")

		payload := map[string]any{
			"method":     c.Request.Method,
			"path":       c.FullPath(),
			"status":     status,
			"resource":   resourceID,
			"user_agent": c.Request.UserAgent(),
		}

		go func() {
			_, _ = LogComplianceEvent(db,
				route.category, route.action,
				userID, userID, resourceID, c.ClientIP(),
				payload)
		}()
	}
}

// parseUUID returns nil for empty/invalid strings so we never insert garbage.
func parseUUID(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &u
}

// firstParam returns the first non-empty URL parameter among the given keys.
func firstParam(c *gin.Context, keys ...string) string {
	for _, k := range keys {
		if v := c.Param(k); v != "" {
			return v
		}
	}
	return ""
}
