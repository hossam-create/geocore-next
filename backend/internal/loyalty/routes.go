package loyalty

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	l := r.Group("/loyalty")
	l.Use(middleware.Auth())
	{
		l.GET("/account", h.GetAccount)
		l.GET("/transactions", h.GetTransactions)
		l.POST("/daily-bonus", h.ClaimDailyBonus)
		l.POST("/referral", h.ApplyReferral)
		l.GET("/badges", h.GetBadges)
		l.GET("/leaderboard", h.GetLeaderboard)

		// Rewards
		l.GET("/rewards", h.GetRewards)
		l.POST("/rewards/:id/redeem", h.RedeemReward)
		l.GET("/redemptions", h.GetMyRedemptions)
	}

	// Internal API for earning points (called by other services)
	internal := r.Group("/internal/loyalty")
	{
		internal.POST("/earn", h.EarnPoints)
	}
}
