package listings

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// context is used by StartExpiryScheduler — import in main.go to call it.

func RegisterRoutes(r *gin.RouterGroup, dbWrite *gorm.DB, dbRead *gorm.DB, rdb *redis.Client) {
	h := NewHandlerReadWrite(dbWrite, dbRead, rdb)

	rl := middleware.NewRateLimiter(rdb)

	// DegradedRead: when DB is slow, these endpoints serve stale cache
	// instead of hitting the DB. Non-critical reads only.
	r.GET("/categories", middleware.DegradedRead(), h.GetCategories)
	r.GET("/categories/:id/fields", h.GetCategoryFields)

	// ── Sprint 18: Category Tree / Slug / Breadcrumb ────────────────────────────
	// Mounted under /category/ (singular) to avoid wildcard-conflict with existing
	// /categories/:id/fields in Gin's radix tree. /categories/tree alias also added.
	r.GET("/categories/tree", middleware.DegradedRead(), h.GetCategoryTree)
	r.GET("/category/:slug", middleware.DegradedRead(), h.GetCategoryBySlug)
	r.GET("/category/:slug/breadcrumb", middleware.DegradedRead(), h.GetCategoryBreadcrumb)
	r.GET("/category/:slug/listings", middleware.DegradedRead(), rl.Limit(60, time.Minute, "listings:cat:ip"), h.GetCategoryListings)

	// ── Search endpoints (public, rate-limited) ──────────────────────────────────
	r.GET("/listings/search", middleware.DegradedRead(), rl.Limit(30, time.Minute, "listings:search:ip"), h.Search)
	r.GET("/listings/suggestions", middleware.DegradedRead(), rl.Limit(60, time.Minute, "listings:suggest:ip"), h.Suggestions)

	// ── Sprint 18: Saved Searches (auth-only) ───────────────────────────────────
	saved := r.Group("/search")
	saved.Use(middleware.Auth())
	{
		saved.POST("/save", h.CreateSavedSearch)
		saved.GET("/saved", h.ListSavedSearches)
		saved.DELETE("/saved/:id", h.DeleteSavedSearch)
	}

	listings := r.Group("/listings")
	{
		// Public — anyone can browse listings (rate-limited)
		listings.GET("", middleware.DegradedRead(), rl.Limit(60, time.Minute, "listings:list:ip"), h.List)
		listings.GET("/:id", middleware.DegradedRead(), h.Get)
		listings.POST("/:id/view", h.RecordView)

		// Urgency signals (Sprint 4 — public)
		listings.GET("/:id/urgency", h.GetUrgency)

		// Boost info (Sprint 4 — public)
		listings.GET("/:id/boost", h.GetBoostInfo)

		// Auth required
		authed := listings.Group("")
		authed.Use(middleware.Auth())
		{
			authed.GET("/me", h.GetMyListings)
			authed.GET("/recent-searches", h.RecentSearches)
			authed.GET("/export", h.ExportCSV)
			authed.GET("/export/template", h.ExportTemplate)
			authed.POST("/import", h.ImportCSV)
			authed.PUT("/:id", h.Update)
			authed.DELETE("/:id", h.Delete)
			authed.POST("/:id/favorite", h.ToggleFavorite)
			authed.GET("/:id/views", h.GetViewAnalytics)
			authed.GET("/views/summary", h.GetMyViewsSummary)
			authed.GET("/views/export", h.ExportViewsCSV)

			// Watchlist (Sprint 4)
			authed.POST("/:id/watch", h.AddToWatchlist)
			authed.DELETE("/:id/watch", h.RemoveFromWatchlist)
			authed.GET("/watchlist", h.GetWatchlist)

			// Seller tier info (Sprint 4)
			authed.GET("/seller-tier", h.GetSellerTierInfoHandler)

			// Recently viewed
			authed.GET("/recently-viewed", h.RecentlyViewed)

			// Listing Q&A
			authed.POST("/:id/questions", h.AskQuestion)
			authed.POST("/:id/questions/:qaId/answer", h.AnswerQuestion)

			// Listing Feedback
			authed.POST("/:id/feedback", h.CreateFeedback)

			// Listing Variants (seller only)
			authed.POST("/:id/variants", h.CreateVariant)
			authed.PUT("/:id/variants/:variantId", h.UpdateVariant)
			authed.DELETE("/:id/variants/:variantId", h.DeleteVariant)
		}

		// Public — variants, Q&A, feedback (read-only)
		listings.GET("/:id/variants", h.ListVariants)
		listings.GET("/:id/questions", h.ListQA)
		listings.GET("/:id/feedback", h.ListFeedback)

		// Auth + email verified — posting requires verified email
		verified := listings.Group("")
		verified.Use(middleware.Auth(), middleware.EmailVerified(dbWrite))
		{
			verified.POST("", h.Create)

			// Boost listing (Sprint 4 — paid boost)
			verified.POST("/:id/boost", h.ApplyBoost)
		}

		// ── Trading endpoints (auth + email verified) ──────────────────────────────
		trade := listings.Group("")
		trade.Use(middleware.Auth(), middleware.EmailVerified(dbWrite))
		{
			// Buy Now — instant purchase
			trade.POST("/:id/buy-now", h.BuyNow)

			// Negotiation — make offer / respond
			trade.POST("/:id/offer", h.SubmitOffer)
			trade.POST("/:id/offer/respond", h.RespondOffer)
			trade.GET("/:id/negotiations", h.ListNegotiationThreads)
			trade.GET("/:id/negotiation/:thread_id", h.GetNegotiationThread)

			// Order conversion — accepted offer → order + escrow
			trade.POST("/:id/negotiation/:thread_id/convert", h.ConvertOfferToOrder)

			// Payment retry — retry escrow on PAYMENT_FAILED thread
			trade.POST("/:id/negotiation/:thread_id/retry-payment", h.RetryPayment)

			// Auction — create auction for listing
			trade.POST("/:id/auction", h.CreateAuctionForListing)
			trade.POST("/:id/auction/convert", h.AuctionBidToOrder)

			// Trade info — listing trading config + status
			trade.GET("/:id/trade", h.GetTradeInfo)
		}

		// ── Background schedulers (call from main.go) ──────────────────────────────
		// listings.RegisterAuctionAutoConvert(dbWrite)  — auto-convert high bids
		// listings.StartExpiryScheduler(ctx, dbWrite)   — expire stale offers every 5m
	}
}
