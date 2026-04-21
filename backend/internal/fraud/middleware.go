package fraud

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DecisionMiddleware blocks requests that the fraud engine has flagged as BLOCK.
// Place this after auth middleware so user_id is available.
//
// Usage: r.Use(fraud.DecisionMiddleware(engine))
func DecisionMiddleware(engine *Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		if userID == "" {
			c.Next()
			return
		}

		amountStr, _ := c.GetQuery("amount")
		var amount float64
		if amountStr != "" {
			_, _ = fmt.Sscanf(amountStr, "%f", &amount)
		}

		req := ScoreRequest{
			UserID:    userID,
			Amount:    amount,
			EventType: "api." + c.FullPath(),
			IP:        c.ClientIP(),
			RequestID: c.GetString("request_id"),
			TraceID:   c.GetString("trace_id"),
		}

		result := engine.Score(c.Request.Context(), req)

		switch result.Decision {
		case DecisionBlock:
			c.JSON(http.StatusForbidden, gin.H{
				"error":      "Transaction blocked by fraud engine",
				"risk_score": result.RiskScore,
				"risk_level": result.RiskLevel,
			})
			c.Abort()
			return
		case DecisionChallenge:
			c.Set("fraud_challenge", true)
			c.Set("fraud_risk_score", result.RiskScore)
		}

		c.Next()
	}
}
