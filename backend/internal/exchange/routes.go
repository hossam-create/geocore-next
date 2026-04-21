package exchange

// routes.go — Sprint 19/20 Exchange route registration.
//
// Legal Compliance Middleware:
//   Every exchange endpoint injects a disclaimer header and response field:
//   "The platform does not hold funds and only facilitates matching."

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const legalDisclaimer = "The platform does not hold funds and only facilitates matching."

// LegalDisclaimerMiddleware adds the compliance disclaimer to every response
// in this domain, both as an HTTP header and as a context value.
func LegalDisclaimerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Exchange-Disclaimer", legalDisclaimer)
		c.Set("exchange_disclaimer", legalDisclaimer)
		c.Next()
	}
}

// disclaimerResponse wraps any response with the legal disclaimer field.
func disclaimerResponse(c *gin.Context, code int, data gin.H) {
	data["disclaimer"] = legalDisclaimer
	c.JSON(code, data)
}

// RegisterRoutes wires all exchange endpoints under /api/v1/exchange.
// authMiddleware should be the existing JWT Auth() middleware.
func RegisterRoutes(rg *gin.RouterGroup, db *gorm.DB, rdb *redis.Client, authMiddleware gin.HandlerFunc) {
	h := NewHandler(db, rdb)

	ex := rg.Group("/exchange")
	ex.Use(LegalDisclaimerMiddleware())

	// Public (read-only)
	ex.GET("/requests", h.ListRequests)
	ex.GET("/fee-estimate", h.FeeEstimate)
	ex.GET("/tiers", h.TierInfo)
	ex.GET("/liquidity", h.LiquidityInsight) // Part 2: liquidity insight
	ex.GET("/rate-hint", h.RateHintEndpoint) // Part 3: advisory rate guidance

	// Info endpoint
	ex.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"description": "Community Exchange (VIP) — non-custodial P2P FX layer",
			"model":       "trust-and-match",
			"disclaimer":  legalDisclaimer,
			"tiers":       []string{TierFree, TierVIP, TierPro},
		})
	})

	// Authenticated
	authed := ex.Group("")
	authed.Use(authMiddleware)

	authed.POST("/requests", h.CreateRequest)
	authed.POST("/requests/:id/match", h.MatchRequest)
	authed.DELETE("/requests/:id", h.CancelRequest)

	authed.POST("/:id/upload-proof", h.UploadProof)
	authed.POST("/:id/verify", h.VerifyProof) // admin/trust-engine only in prod
	authed.POST("/:id/dispute", h.RaiseDispute)

	// VIP tier
	authed.GET("/me/tier", h.MyTier)

	// Risk profile (Part 6)
	authed.GET("/risk/me", h.RiskProfile)

	// Admin-only
	admin := ex.Group("/admin")
	admin.Use(authMiddleware)
	admin.POST("/users/:user_id/tier", h.AdminSetTier)            // VIP tier management
	admin.POST("/seed", h.AdminSeedLiquidity)                     // Part 2: seed low-liquidity pairs
	admin.POST("/matches/:id/auto-resolve", h.AutoResolveHandler) // Part 5: dispute auto-resolve
}

// HealthCheck returns a simple status; used by the monitor.
func HealthCheck(c *gin.Context) {
	disclaimerResponse(c, http.StatusOK, gin.H{"status": "ok", "service": "exchange"})
}
