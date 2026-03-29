package wallet

import (
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewHandler(db)

	w := r.Group("/wallet")
	w.Use(middleware.Auth())
	{
		w.POST("", h.CreateWallet)
		w.GET("", h.GetWallet)
		w.GET("/balance/:currency", h.GetBalance)
		w.POST("/deposit", h.Deposit)
		w.POST("/withdraw", h.Withdraw)
		w.GET("/transactions", h.GetTransactions)
	}

	// Escrow
	e := r.Group("/escrow")
	e.Use(middleware.Auth())
	{
		e.POST("", h.CreateEscrow)
		e.POST("/:id/release", h.ReleaseEscrow)
	}

	// Price Plans
	p := r.Group("/plans")
	{
		p.GET("", h.GetPricePlans)
	}

	// Subscriptions
	s := r.Group("/subscriptions")
	s.Use(middleware.Auth())
	{
		s.POST("", h.Subscribe)
		s.GET("", h.GetSubscription)
	}
}
