package auth

  import (
        "time"

        "github.com/geocore-next/backend/pkg/middleware"
        "github.com/gin-gonic/gin"
        "github.com/redis/go-redis/v9"
        "gorm.io/gorm"
  )

  // RegisterRoutes mounts all /auth endpoints onto the given router group.
  //
  // Rate limits applied (sliding window, Redis-backed):
  //
  //    Endpoint                  Limit   Window   Key scope
  //    ──────────────────────── ─────── ──────── ──────────────────────────────────
  //    POST /register            5 req   1 hour   per IP
  //    POST /login               10 req  15 min   per IP
  //    POST /social              10 req  15 min   per IP
  //    POST /forgot-password     3 req   1 hour   per IP
  //    POST /reset-password      5 req   1 hour   per IP
  //    POST /resend-verification 5 req   1 hour   per user (auth-required)
  func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, rdb *redis.Client) {
        h  := NewHandler(db, rdb)
        rl := middleware.NewRateLimiter(rdb)

        a := r.Group("/auth")
        {
                // ── Public endpoints ──────────────────────────────────────────────────

                a.POST("/register",
                        rl.Limit(5, time.Hour, "auth:register:ip"),
                        h.Register,
                )

                a.POST("/login",
                        rl.Limit(10, 15*time.Minute, "auth:login:ip"),
                        h.Login,
                )

                a.POST("/refresh",
                        rl.Limit(20, 15*time.Minute, "auth:refresh:ip"),
                        h.Refresh,
                )

                a.POST("/verify-email", h.VerifyEmail) // token-based; no extra rate limit needed

                a.POST("/social",
                        rl.Limit(10, 15*time.Minute, "auth:social:ip"),
                        h.SocialLogin,
                )

                // ── Password reset ──────────────────────────────────────────────────────
                // /forgot-password has its own internal Redis rate limit (1 per 15 min per
                // email), but we add an IP-level limit here as a first line of defence
                // against scripted attacks from a single IP.
                a.POST("/forgot-password",
                        rl.Limit(3, time.Hour, "auth:forgot:ip"),
                        h.ForgotPassword,
                )

                a.POST("/validate-reset-token", h.ValidateResetToken) // read-only check

                a.POST("/reset-password",
                        rl.Limit(5, time.Hour, "auth:reset:ip"),
                        h.ResetPassword,
                )

                // ── Auth-required endpoints ───────────────────────────────────────────
                authed := a.Group("")
                authed.Use(middleware.Auth())
                {
                        authed.GET("/me", h.Me)
                        authed.POST("/logout", h.Logout)

                        // Per-user limit: prevent spamming verification emails
                        authed.POST("/resend-verification",
                                rl.LimitByUser(5, time.Hour, "auth:resend:user"),
                                h.ResendVerification,
                        )
                }
        }
  }
  