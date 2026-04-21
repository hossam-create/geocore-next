package livestream

import (
	"context"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(v1 *gin.RouterGroup, db *gorm.DB, rdb ...*redis.Client) {
	var rdbClient *redis.Client
	if len(rdb) > 0 {
		rdbClient = rdb[0]
	}
	h := NewHandler(db, rdbClient)
	la := NewLiveAuctionHandler(db, rdbClient)

	// Public — list & get
	v1.GET("/livestream", h.ListSessions)
	v1.GET("/livestream/:id", h.GetSession)

	// Sprint 17: Marketplace Brain — public feed
	v1.GET("/livestream/feed", la.GetFeedHandler)
	v1.GET("/livestream/:id/score", la.GetSessionScoreHandler)

	// Authenticated
	ls := v1.Group("/livestream")
	ls.Use(middleware.Auth())
	{
		ls.POST("", h.CreateSession)
		ls.POST("/:id/start", h.StartSession)
		ls.POST("/:id/end", h.EndSession)
		ls.POST("/:id/join", h.JoinSession)
		ls.DELETE("/:id", h.CancelSession)
		ls.POST("/:id/leave", h.LeaveSession)
	}

	// Live Auction Items + Bidding (Sprint 9)
	// la already initialized above

	// Start background auction closer scheduler
	go la.StartAuctionCloser(context.Background())

	// Public — view items
	v1.GET("/livestream/:id/items", la.ListItems)
	v1.GET("/livestream/:id/items/:itemId/bids", la.ListBids)

	// Authenticated — manage items + bid
	lai := v1.Group("/livestream")
	lai.Use(middleware.Auth())
	{
		lai.POST("/:id/items", la.AddItem)
		lai.POST("/:id/items/:itemId/activate", la.ActivateItem)
		lai.POST("/:id/items/:itemId/bid", la.PlaceBid)
		lai.POST("/:id/items/:itemId/buy-now", la.BuyNow)
		lai.POST("/:id/items/:itemId/deposit", la.PayDeposit)
		// Sprint 11: Live Conversion Engine
		lai.POST("/:id/items/:itemId/pin", la.PinItem)
		lai.POST("/:id/items/:itemId/unpin", la.UnpinItem)
		lai.POST("/:id/items/:itemId/quick-bid", la.QuickBid)
		// Sprint 12: Monetization
		lai.POST("/:id/boost", la.PurchaseBoost)
		lai.POST("/:id/premium", la.EnablePremiumMode)
		lai.POST("/:id/entry", la.PayEntryFee)
		// Sprint 13: Revenue Flywheel
		lai.POST("/:id/items/:itemId/bid/priority", la.PriorityBid)
		// Sprint 14: AI Assistant
		lai.GET("/:id/items/:itemId/ai-insights", la.AIInsights)
		lai.POST("/:id/ai-suggestions/:suggestionId/accept", la.AcceptAISuggestion)
		// Sprint 15: Viral Growth Loops
		lai.POST("/:id/invites", la.CreateLiveInviteHandler)
		lai.POST("/invites/:code/track", la.TrackLiveInviteHandler)
		lai.POST("/items/:itemId/win-shares", la.CreateWinShareHandler)
		lai.POST("/shares/:code/attribute", la.AttributeShareJoinHandler)
		lai.POST("/:id/group-invites", la.CreateGroupInviteHandler)
		lai.POST("/group-invites/:code/join", la.JoinGroupInviteHandler)
		lai.GET("/streaks/me", la.MyStreaksHandler)
	}

	// Sprint 16: Creator Economy
	cr := v1.Group("/creators")
	cr.Use(middleware.Auth())
	{
		cr.POST("/apply", la.ApplyCreatorHandler)
		cr.GET("/me", la.MyCreatorProfileHandler)
		cr.GET("/top", la.TopCreatorsHandler)
		cr.GET("/:id", la.GetCreatorHandler)
		cr.GET("/:id/analytics", la.CreatorAnalyticsHandler)
		cr.GET("/:id/referral-code", la.CreatorReferralCodeHandler)
		// Deals
		cr.POST("/deals/invite", la.InviteCreatorHandler)
		cr.POST("/deals/:id/accept", la.AcceptDealHandler)
		cr.POST("/deals/:id/reject", la.RejectDealHandler)
		cr.GET("/deals", la.MyDealsHandler)
		// Matching
		cr.GET("/:id/match-items/:itemId", la.MatchCreatorsForItemHandler)
		// Sprint 17: Creator exposure
		cr.GET("/:id/exposure", la.GetCreatorExposureHandler)
	}
	// Seller-side deal management
	sd := v1.Group("/sellers")
	sd.Use(middleware.Auth())
	{
		sd.GET("/deals", la.SellerDealsHandler)
	}

	// Admin Live Control Panel (Sprint 10)
	admin := v1.Group("/admin/live")
	admin.Use(middleware.Auth(), middleware.AdminWithDB(db))
	{
		admin.POST("/panic", la.AdminPanic)
		admin.POST("/recover", la.AdminRecover)
		admin.POST("/:id/stop", la.AdminStopSession)
		admin.POST("/:id/pause", la.AdminPauseSession)
		admin.POST("/:id/resume", la.AdminResumeSession)
		admin.POST("/:id/user/:userId/ban", la.AdminBanUser)
		admin.POST("/:id/user/:userId/unban", la.AdminUnbanUser)
		admin.GET("/dashboard", la.AdminDashboard)
		// Sprint 11: Funnel analytics
		admin.GET("/funnel", la.AdminFunnelAnalytics)
		// Sprint 12: Monetization metrics
		admin.GET("/metrics", la.AdminMetrics)
		// Sprint 11.5: Behavioral Engine — auto funnel optimization
		admin.POST("/:id/auto-optimize", la.AdminAutoOptimize)
		// Sprint 14: AI performance analytics
		admin.GET("/ai-performance", la.AdminAIPerformance)
	}

	// Sprint 15: Growth analytics (separate admin group)
	growth := v1.Group("/admin/growth")
	growth.Use(middleware.Auth(), middleware.AdminWithDB(db))
	{
		growth.GET("/metrics", la.AdminGrowthMetrics)
	}

	// Sprint 16: Creator admin routes
	creatorAdmin := v1.Group("/admin/creators")
	creatorAdmin.Use(middleware.Auth(), middleware.AdminWithDB(db))
	{
		creatorAdmin.GET("/payouts", la.AdminCreatorPayoutsHandler)
		creatorAdmin.POST("/:id/refresh-trust", la.AdminRefreshCreatorTrustHandler)
	}

	// Sprint 17: Marketplace Brain admin routes
	brainAdmin := v1.Group("/admin/marketplace")
	brainAdmin.Use(middleware.Auth(), middleware.AdminWithDB(db))
	{
		brainAdmin.GET("/brain", la.AdminMarketplaceBrainHandler)
		brainAdmin.POST("/revenue-priority", la.AdminRevenuePriorityHandler)
		brainAdmin.POST("/snapshot", la.AdminRecordSnapshotHandler)
	}

	// Sprint 17: Traffic allocation (authenticated)
	trafficAuth := v1.Group("/livestream")
	trafficAuth.Use(middleware.Auth())
	{
		trafficAuth.GET("/:id/traffic", la.GetTrafficAllocationHandler)
		trafficAuth.POST("/:id/recover", la.RecoverSessionHandler)
	}
}
