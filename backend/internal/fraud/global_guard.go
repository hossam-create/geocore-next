package fraud

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Sprint 24 protected routes — checked before the handler executes.
// Matches Gin's FullPath() so the engine stays cheap (one map lookup).
var sprint24ProtectedRoutes = map[string]string{
	"POST /api/v1/exchange/requests":                         "create_request",
	"POST /api/v1/exchange/requests/:id/match":               "match_request",
	"POST /api/v1/livestream/:id/items/:itemId/bid":          "place_bid",
	"POST /api/v1/livestream/:id/items/:itemId/quick-bid":    "quick_bid",
	"POST /api/v1/livestream/:id/items/:itemId/bid/priority": "priority_bid",
	"POST /api/v1/payments/withdraw/request":                 "withdraw",
}

// GlobalGuard returns a Gin middleware that enforces PredictRisk decisions
// on the four Sprint 24 critical actions. Mount after middleware.Auth() so
// `user_id` is available on the context.
//
//   - PredictBlock  → 403 Forbidden (action refused)
//   - PredictLimit  → allowed but flagged (header + context key) for handlers
//                      that want to apply lower per-hour ceilings
//   - PredictSoft   → allowed, header set so the FE can show a captcha prompt
//   - PredictAllow  → passthrough with zero extra cost
func GlobalGuard(p *Predictor) gin.HandlerFunc {
	return func(c *gin.Context) {
		action, ok := sprint24ProtectedRoutes[c.Request.Method+" "+c.FullPath()]
		if !ok {
			c.Next()
			return
		}
		uidStr := c.GetString("user_id")
		if uidStr == "" {
			c.Next()
			return
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			c.Next()
			return
		}

		res := p.PredictRisk(c.Request.Context(), uid)

		// Expose decision on context for downstream handlers.
		c.Set("fraud_score", res.Score)
		c.Set("fraud_decision", string(res.Decision))
		c.Set("fraud_action", action)
		c.Header("X-Risk-Score", itoa(res.Score))
		c.Header("X-Risk-Decision", string(res.Decision))

		switch res.Decision {
		case PredictBlock:
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":      "fraud_prevention_block",
				"message":    "Action blocked by fraud prevention. Contact support if you believe this is wrong.",
				"risk_score": res.Score,
				"action":     action,
			})
			return
		case PredictLimit:
			c.Header("X-Risk-Limit", "true")
		case PredictSoft:
			c.Header("X-Risk-Challenge", "captcha")
		}
		c.Next()
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
