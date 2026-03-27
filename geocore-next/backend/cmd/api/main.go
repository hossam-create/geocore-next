package main

import (
        "context"
        "fmt"
        "net/http"
        "os"
        "os/signal"
        "strings"
        "syscall"
        "time"

        "github.com/geocore-next/backend/internal/admin"
        "github.com/geocore-next/backend/internal/auctions"
        "github.com/geocore-next/backend/internal/auth"
        "github.com/geocore-next/backend/internal/chat"
        "github.com/geocore-next/backend/internal/images"
        "github.com/geocore-next/backend/internal/kyc"
        "github.com/geocore-next/backend/internal/listings"
        "github.com/geocore-next/backend/internal/monetization"
        "github.com/geocore-next/backend/internal/notifications"
        "github.com/geocore-next/backend/internal/payments"
        "github.com/geocore-next/backend/internal/reviews"
        "github.com/geocore-next/backend/internal/stores"
        "github.com/geocore-next/backend/internal/users"
        "github.com/geocore-next/backend/pkg/database"
        "github.com/geocore-next/backend/pkg/middleware"
        "github.com/geocore-next/backend/pkg/util"

        "github.com/gin-contrib/cors"
        "github.com/gin-gonic/gin"
        "github.com/joho/godotenv"
        "github.com/redis/go-redis/v9"
        "go.uber.org/zap"
)

const (
        redisMaxRetries = 5
        redisRetryDelay = 2 * time.Second
)

func main() {
        _ = godotenv.Load()
        logger, _ := zap.NewProduction()
        defer logger.Sync() //nolint:errcheck

        db, err := database.Connect()
        if err != nil {
                logger.Fatal("DB connect failed", zap.Error(err))
        }
        if err := database.AutoMigrate(db); err != nil {
                logger.Fatal("AutoMigrate failed", zap.Error(err))
        }
        logger.Info("Database ready")
        go auctions.ApplyAuctionIndexes(db)
        go listings.ApplySearchIndexes(db)

        rdb := redis.NewClient(&redis.Options{
                Addr:     fmt.Sprintf("%s:%s", util.Getenv("REDIS_HOST", "localhost"), util.Getenv("REDIS_PORT", "6379")),
                Password: os.Getenv("REDIS_PASSWORD"),
        })

        // Retry Redis connection so the API starts cleanly even when Redis is slow
        var redisErr error
        for attempt := 1; attempt <= redisMaxRetries; attempt++ {
                pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
                redisErr = rdb.Ping(pingCtx).Err()
                pingCancel()
                if redisErr == nil {
                        break
                }
                if attempt < redisMaxRetries {
                        logger.Warn("Redis connect attempt failed, retrying",
                                zap.Int("attempt", attempt),
                                zap.Int("max", redisMaxRetries),
                                zap.Error(redisErr),
                                zap.Duration("delay", redisRetryDelay),
                        )
                        time.Sleep(redisRetryDelay)
                }
        }
        if redisErr != nil {
                logger.Fatal("Redis connect failed after retries", zap.Error(redisErr))
        }
        logger.Info("Redis ready")

        // AI Pricing client (non-fatal if service not running)
        aiCtx, aiCancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer aiCancel()
        aiClient := auctions.NewAIPricingClient()
        if aiClient.IsHealthy(aiCtx) {
                logger.Info("AI Pricing service ready")
        } else {
                logger.Warn("AI Pricing service not available — bid suggestions disabled")
        }

        middleware.RevocationRDB = rdb

        if os.Getenv("APP_ENV") == "production" {
                gin.SetMode(gin.ReleaseMode)
        }

        r := gin.New()
        r.Use(gin.Recovery())
        corsConfig := cors.Config{
                AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
                AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
                MaxAge:       12 * time.Hour,
        }
        // CORS origin policy:
        //   ALLOWED_ORIGINS (comma-separated): explicit allowlist; wildcard '*' blocked in prod.
        //   No ALLOWED_ORIGINS in prod → fall back to FRONTEND_URL.
        //   No FRONTEND_URL either → fatal startup error (operator must configure before launch).
        //   Development (APP_ENV != "production"): allow all origins for local iteration.
        //
        // We never set AllowOrigins to an empty slice — gin-contrib/cors panics on startup
        // if AllowAllOrigins=false and AllowOrigins=[].  Instead we either have a non-empty
        // allowlist or we fail fast so the misconfiguration is immediately visible.
        isProd := os.Getenv("APP_ENV") == "production"
        if rawOrigins := os.Getenv("ALLOWED_ORIGINS"); rawOrigins != "" {
                var origins []string
                for _, o := range strings.Split(rawOrigins, ",") {
                        o = strings.TrimSpace(o)
                        if o == "" {
                                continue
                        }
                        // In production, wildcard '*' is forbidden — it nullifies all origin checks.
                        if isProd && o == "*" {
                                logger.Warn("CORS: wildcard '*' in ALLOWED_ORIGINS is forbidden in production and will be ignored")
                                continue
                        }
                        origins = append(origins, o)
                }
                if len(origins) > 0 {
                        corsConfig.AllowOrigins = origins
                        corsConfig.AllowCredentials = true
                        logger.Info("CORS allowlist configured", zap.Strings("origins", origins))
                } else if isProd {
                        // All entries stripped (e.g. only "*") — refuse to start.
                        logger.Fatal("CORS: ALLOWED_ORIGINS resolved to empty in production (wildcards stripped). " +
                                "Set at least one explicit origin (e.g. https://example.com).")
                } else {
                        corsConfig.AllowAllOrigins = true
                }
        } else if isProd {
                // No ALLOWED_ORIGINS: fall back to FRONTEND_URL, or abort startup.
                if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
                        logger.Warn("CORS: ALLOWED_ORIGINS not set — using FRONTEND_URL as production origin",
                                zap.String("origin", frontendURL))
                        corsConfig.AllowOrigins = []string{frontendURL}
                        corsConfig.AllowCredentials = true
                } else {
                        logger.Fatal("CORS: neither ALLOWED_ORIGINS nor FRONTEND_URL is set in production. " +
                                "Set ALLOWED_ORIGINS=https://your-domain.com before deploying.")
                }
        } else {
                // Development: allow all origins so local tools work without extra config.
                corsConfig.AllowAllOrigins = true
        }
        r.Use(cors.New(corsConfig))

        r.GET("/health", func(c *gin.Context) {
                c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
        })
        r.GET("/ready", func(c *gin.Context) {
                sql, err := db.DB()
                if err != nil {
                        c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db error", "error": err.Error()})
                        return
                }
                if err := sql.PingContext(c.Request.Context()); err != nil {
                        c.JSON(http.StatusServiceUnavailable, gin.H{"status": "db ping failed"})
                        return
                }
                if err := rdb.Ping(c.Request.Context()).Err(); err != nil {
                        c.JSON(http.StatusServiceUnavailable, gin.H{"status": "redis ping failed"})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"status": "ready"})
        })

        chatHub := chat.NewHub(rdb)
        go chatHub.Run()
        go chatHub.SubscribeRedis(context.Background())
        auctionHub := auctions.NewHub(rdb)
        go auctionHub.Run()
        go auctionHub.SubscribeRedis(context.Background())

        // Background schedulers
        schedulerCtx, cancelSchedulers := context.WithCancel(context.Background())
        go auctions.StartAuctionEndWorker(schedulerCtx, db, auctionHub)
        go listings.StartListingExpiryWorker(schedulerCtx, db)

        rl := middleware.NewRateLimiter(rdb)

        v1 := r.Group("/api/v1")
        // Global rate limit: 100 requests per minute per IP (skip OPTIONS preflight)
        v1.Use(func(c *gin.Context) {
                if c.Request.Method == http.MethodOptions {
                        c.Next()
                        return
                }
                rl.Limit(100, time.Minute, "global")(c)
        })
        auth.RegisterRoutes(v1, db, rdb)
        users.RegisterRoutes(v1, db, rdb)
        listings.RegisterRoutes(v1, db, rdb, rl)
        auctions.RegisterRoutes(v1, db, rdb, rl)
        chat.RegisterRoutes(v1, db, rdb, rl)
        payments.RegisterRoutes(v1, db, rdb)
        images.RegisterRoutes(v1, db, rdb)
        notifHub, notifSvc := notifications.RegisterRoutes(v1, db, rdb)
        admin.RegisterRoutes(v1, db, rdb)
        kyc.RegisterRoutes(v1, db)
        reviews.RegisterRoutes(v1, db)
        stores.RegisterRoutes(v1, db, rdb)
        monetization.RegisterRoutes(v1, db)

        // Wire notification service into dependent packages
        auctions.SetNotificationService(notifSvc)
        chat.SetNotificationService(notifSvc)
        payments.SetNotificationService(notifSvc)

        // Presigned upload URL — used by KYC and listing image uploads from the browser.
        // Auth required to prevent abuse; returns mock URL in dev when R2 is not configured.
        v1.POST("/media/upload-url", middleware.Auth(), func(c *gin.Context) {
                images.NewHandler(db).GetUploadURL(c)
        })

        // AI bid suggestion endpoint — proxies to Python microservice
        v1.POST("/auctions/ai-predict", middleware.Auth(), func(c *gin.Context) {
                var req auctions.BidPredictRequest
                if err := c.ShouldBindJSON(&req); err != nil {
                        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                        return
                }
                result, err := aiClient.Predict(c.Request.Context(), req)
                if err != nil {
                        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "AI service unavailable"})
                        return
                }
                c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
        })

        r.GET("/ws/notifications", func(c *gin.Context) { notifications.ServeWS(notifHub, c) })
        r.GET("/ws/auctions/:id", func(c *gin.Context) { auctions.ServeWS(auctionHub, c, db) })
        r.POST("/webhooks/stripe", payments.WebhookHandler(db))

        // ── Static file serving for production (built by render-build.sh) ────────
        if _, err := os.Stat("./web"); err == nil {
                // Admin SPA at /admin
                if _, err2 := os.Stat("./admin"); err2 == nil {
                        adminFS := http.FileServer(http.Dir("./admin"))
                        r.GET("/admin", func(c *gin.Context) { c.File("./admin/index.html") })
                        r.GET("/admin/*filepath", func(c *gin.Context) {
                                fp := c.Param("filepath")
                                if _, serr := os.Stat("./admin" + fp); os.IsNotExist(serr) {
                                        c.File("./admin/index.html")
                                        return
                                }
                                adminFS.ServeHTTP(c.Writer, c.Request)
                        })
                }
                // Web SPA — catch-all (must be last)
                webFS := http.FileServer(http.Dir("./web"))
                r.NoRoute(func(c *gin.Context) {
                        p := c.Request.URL.Path
                        if _, serr := os.Stat("./web" + p); os.IsNotExist(serr) {
                                c.File("./web/index.html")
                                return
                        }
                        webFS.ServeHTTP(c.Writer, c.Request)
                })
                logger.Info("Serving frontend static files from ./web (admin at ./admin)")
        }

        port := util.Getenv("BACKEND_PORT", util.Getenv("PORT", "8080"))
        srv := &http.Server{
                Addr:         ":" + port,
                Handler:      r,
                ReadTimeout:  15 * time.Second,
                WriteTimeout: 30 * time.Second,
                IdleTimeout:  60 * time.Second,
        }

        go func() {
                logger.Info("GeoCore Next API running", zap.String("port", port))
                if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                        logger.Fatal("Server error", zap.Error(err))
                }
        }()

        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        <-quit
        logger.Info("Shutting down gracefully...")
        cancelSchedulers()
        ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel2()
        _ = srv.Shutdown(ctx2)
}
