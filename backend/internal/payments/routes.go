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
  //   POST /payments/create-payment-intent  20 req / 1 hour
  //   POST /payments/request-refund          5 req / 1 hour
  func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
        InitStripe()

        h  := NewHandler(db)
        rl := middleware.NewRateLimiter(rdb)

        p := r.Group("/payments")
        p.Use(middleware.Auth())
        {
                // ── Payment flow ──────────────────────────────────────────────────────
                p.POST("/create-payment-intent",
                        rl.LimitByUser(20, time.Hour, "payments:create:user"),
                        h.CreatePaymentIntent,
                )
                p.POST("/confirm",        h.ConfirmPayment)
                p.POST("/release-escrow", h.ReleaseEscrow)
                p.POST("/request-refund",
                        rl.LimitByUser(5, time.Hour, "payments:refund:user"),
                        h.RequestRefund,
                )

                // ── Payment history ───────────────────────────────────────────────────
                p.GET("", h.GetPaymentHistory)

                // ── Saved payment methods ─────────────────────────────────────────────
                p.GET("/payment-methods",        h.GetPaymentMethods)
                p.POST("/add-payment-method",    h.AddPaymentMethod)
                p.DELETE("/payment-methods/:id", h.DeletePaymentMethod)
        }

        // ── Wallet ────────────────────────────────────────────────────────────────
        w := r.Group("/wallet")
        w.Use(middleware.Auth())
        {
                w.GET("/balance",      h.GetWalletBalance)
                w.GET("/transactions", h.GetWalletTransactions)
                w.POST("/top-up",
                        rl.LimitByUser(10, time.Hour, "wallet:topup:user"),
                        h.WalletTopUp,
                )
        }
  }
  