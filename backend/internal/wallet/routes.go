package wallet

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	h := NewHandler(db, rdb)

	// Financial kill switch — blocks all money-moving routes when paused
	fGuard := middleware.FinancialGuard(db)

	// Per-user rate limits for money-moving operations
	rl := middleware.NewRateLimiter(rdb)

	w := r.Group("/wallet")
	w.Use(middleware.Auth())
	{
		w.POST("", h.CreateWallet)
		w.GET("", h.GetWallet)
		w.GET("/balance/:currency", h.GetBalance)
		w.POST("/deposit", fGuard,
			rl.LimitByUser(10, time.Hour, "wallet:deposit:user"),
			h.Deposit)
		w.POST("/withdraw", fGuard,
			rl.LimitByUser(5, time.Hour, "wallet:withdraw:user"),
			h.Withdraw)
		w.POST("/transfer", fGuard,
			rl.LimitByUser(10, time.Hour, "wallet:transfer:user"),
			h.Transfer)
		w.GET("/transactions", h.GetTransactions)
	}

	// Escrow — user operations
	e := r.Group("/escrow")
	e.Use(middleware.Auth(), fGuard)
	{
		e.POST("",
			rl.LimitByUser(10, time.Hour, "escrow:create:user"),
			h.CreateEscrow)
	}

	// Escrow — admin operations (IDOR guard: only admins may release funds)
	eAdmin := r.Group("/escrow")
	eAdmin.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly(), fGuard)
	{
		eAdmin.POST("/:id/release", h.ReleaseEscrow)
		eAdmin.POST("/:id/cancel", h.CancelEscrow)
	}

	walletAdmin := r.Group("/admin/wallet")
	walletAdmin.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		walletAdmin.GET("/reconcile", h.Reconcile)
	}

}
