package monetization

import (
	"os"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v79"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all monetization endpoints onto the given router group.
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	// Init Stripe key (idempotent with payments.InitStripe, but we set it here
	// so this package can run standalone in tests).
	if key := os.Getenv("STRIPE_SECRET_KEY"); key != "" {
		stripe.Key = key
	}

	// Seed default platform settings (commission rate, boost fee) if absent.
	SeedDefaultSettings(db)

	h := NewHandler(db)

	// ── Listing boost ─────────────────────────────────────────────────────────
	listings := r.Group("/listings")
	listings.Use(middleware.Auth())
	{
		listings.POST("/:id/boost",         h.BoostListing)
		listings.POST("/:id/boost/confirm", h.ConfirmBoost)
	}

	// ── Seller subscriptions ──────────────────────────────────────────────────
	subs := r.Group("/subscriptions")
	subs.Use(middleware.Auth())
	{
		subs.GET("/me",       h.GetMySubscription)
		subs.POST("/upgrade", h.UpgradeSubscription)
		subs.POST("/confirm", h.ConfirmSubscription)
	}
}
