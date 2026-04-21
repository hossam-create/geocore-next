package payments

import (
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// RegisterRoutes mounts all /payments endpoints onto the given router group.
//
// Rate limits (per authenticated user):
//
//	POST /payments/create-payment-intent  20 req / 1 hour
//	POST /payments/request-refund          5 req / 1 hour
func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
	InitStripe()
	InitPayPal()

	h := NewHandler(db, rdb)
	rl := middleware.NewRateLimiter(rdb)

	p := r.Group("/payments")
	p.Use(middleware.Auth())
	{
		// ── Payment flow ──────────────────────────────────────────────────────
		p.POST("/create-payment-intent",
			rl.LimitByUser(20, time.Hour, "payments:create:user"),
			h.CreatePaymentIntent,
		)
		p.POST("/confirm", h.ConfirmPayment)

		// ── PayMob (MENA) ──────────────────────────────────────────────────────
		p.POST("/paymob/init", h.CreatePayMobPayment)
		p.GET("/paymob/:id/status", middleware.CriticalRead(), h.GetPayMobPaymentStatus)

		p.POST("/paypal/create", h.CreatePayPalOrder)
		p.POST("/paypal/capture", h.CapturePayPalOrder)
		p.POST("/release-escrow", h.ReleaseEscrow)
		p.POST("/request-refund",
			rl.LimitByUser(5, time.Hour, "payments:refund:user"),
			h.RequestRefund,
		)

		// ── Payment history (critical read — always primary DB)
		p.GET("", middleware.CriticalRead(), h.GetPaymentHistory)

		// ── Saved payment methods (critical read)
		p.GET("/payment-methods", middleware.CriticalRead(), h.GetPaymentMethods)
		p.POST("/add-payment-method", h.AddPaymentMethod)
		p.DELETE("/payment-methods/:id", h.DeletePaymentMethod)
	}

	// PayPal webhook/IPN verification endpoint (public)
	r.POST("/payments/paypal/webhook", h.PayPalWebhook)

	// PayMob webhook (public — HMAC verified)
	r.POST("/payments/paymob/webhook", h.PayMobWebhook)

	// ── Wallet ────────────────────────────────────────────────────────────────
	w := r.Group("/wallet")
	w.Use(middleware.Auth())
	{
		w.GET("/balance", middleware.CriticalRead(), h.GetWalletBalance)
		w.GET("/payment-transactions", middleware.CriticalRead(), h.GetWalletTransactions)
		w.POST("/top-up",
			rl.LimitByUser(10, time.Hour, "wallet:topup:user"),
			h.WalletTopUp,
		)
	}

	// ── Sprint 6: Agent Payment System ──────────────────────────────────────────

	// Deposit (user-facing)
	pay := r.Group("/payments")
	pay.Use(middleware.Auth())
	{
		pay.POST("/deposit/initiate", h.InitiateDeposit)
		pay.POST("/deposit/:id/upload-proof", h.UploadDepositProof)
		pay.GET("/deposit/:id/status", h.GetDepositStatus)
		pay.GET("/deposit/history", h.GetDepositHistory)

		// Withdraw (user-facing)
		pay.POST("/withdraw/request", h.RequestWithdraw)
		pay.DELETE("/withdraw/:id/cancel", h.CancelWithdraw)
		pay.GET("/withdraw/history", h.GetWithdrawHistory)

		// Agents (public)
		pay.GET("/agents/available", h.GetAvailableAgents)

		// Sprint 8: Dynamic Fees
		pay.GET("/fee", h.GetDynamicFeeHandler)
	}

	// Agent endpoints (agents only)
	agentRoutes := r.Group("/agent")
	agentRoutes.Use(middleware.Auth())
	{
		agentRoutes.GET("/requests", h.GetAgentPendingRequests)
		agentRoutes.POST("/deposit/:id/confirm", h.AgentConfirmDeposit)
		agentRoutes.POST("/deposit/:id/reject", h.AgentRejectDeposit)
		agentRoutes.POST("/withdraw/:id/complete", h.AgentCompleteWithdraw)
		agentRoutes.GET("/liquidity", h.GetMyLiquidity)
	}

	// Sprint 7: Payment Disputes (user-facing)
	pay.POST("/disputes", h.OpenDispute)
	pay.GET("/disputes", h.GetUserDisputes)
	pay.GET("/disputes/:id", h.GetDispute)

	// Admin
	adminPay := r.Group("/admin/payments")
	adminPay.Use(middleware.Auth(), middleware.AdminWithDB(db), middleware.AdminOnly())
	{
		adminPay.POST("/agents/register", h.RegisterAgent)
		adminPay.PUT("/agents/:id/approve", h.ApproveAgent)
		adminPay.PUT("/agents/:id/suspend", h.SuspendAgent)
		adminPay.GET("/agents", h.ListAllAgents)
		adminPay.GET("/agents/:id/utilization", h.GetAgentUtilization)
		adminPay.GET("/dashboard", h.GetPaymentDashboard)

		// VIP
		adminPay.POST("/users/:id/vip", h.UpgradeToVIP)
		adminPay.PUT("/users/:id/vip/tier", h.UpdateVIPTier)

		// Sprint 7: Matching Engine
		adminPay.POST("/matching/run", h.RunMatchingHandler)
		adminPay.GET("/matching/stats", h.GetMatchingStatsHandler)

		// Sprint 7: Liquidity
		adminPay.GET("/liquidity", h.GetSystemLiquidity)

		// Sprint 8: Dynamic Fees
		adminPay.GET("/liquidity/level", h.GetLiquidityLevelHandler)

		// Sprint 7: Disputes
		adminPay.GET("/disputes", h.ListDisputes)
		adminPay.PUT("/disputes/:id/resolve", h.ResolveDispute)
	}
}
