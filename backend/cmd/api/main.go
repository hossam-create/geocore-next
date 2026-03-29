package main

<<<<<<< HEAD
  import (
        "context"
        "fmt"
        "net/http"
        "os"
        "os/signal"
        "syscall"
        "time"

        "github.com/geocore-next/backend/internal/admin"
        "github.com/geocore-next/backend/internal/auctions"
        "github.com/geocore-next/backend/internal/auth"
        "github.com/geocore-next/backend/internal/chat"
        "github.com/geocore-next/backend/internal/images"
        "github.com/geocore-next/backend/internal/kyc"
        "github.com/geocore-next/backend/internal/listings"
        "github.com/geocore-next/backend/internal/notifications"
        "github.com/geocore-next/backend/internal/payments"
        "github.com/geocore-next/backend/internal/reviews"
        "github.com/geocore-next/backend/internal/stores"
        "github.com/geocore-next/backend/internal/users"
        "github.com/geocore-next/backend/pkg/database"
        "github.com/geocore-next/backend/pkg/middleware"
=======
import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/geocore-next/backend/internal/auctions"
	"github.com/geocore-next/backend/internal/auth"
	"github.com/geocore-next/backend/internal/chat"
	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/internal/payments"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/cloudinary"
	"github.com/geocore-next/backend/pkg/database"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/util"
>>>>>>> 1aa1121 (TASK-001: Fix build errors - add missing imports, fix bool pointers, remove unused imports)

        "github.com/gin-contrib/cors"
        "github.com/gin-gonic/gin"
        "github.com/joho/godotenv"
        "github.com/redis/go-redis/v9"
        "go.uber.org/zap"
  )

<<<<<<< HEAD
  func getenv(key, fallback string) string {
        if v := os.Getenv(key); v != "" {
                return v
        }
        return fallback
  }
=======

>>>>>>> 1aa1121 (TASK-001: Fix build errors - add missing imports, fix bool pointers, remove unused imports)

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

        rdb := redis.NewClient(&redis.Options{
                Addr:     fmt.Sprintf("%s:%s", getenv("REDIS_HOST", "localhost"), getenv("REDIS_PORT", "6379")),
                Password: os.Getenv("REDIS_PASSWORD"),
        })
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        if err := rdb.Ping(ctx).Err(); err != nil {
                logger.Fatal("Redis connect failed", zap.Error(err))
        }
        logger.Info("Redis ready")

<<<<<<< HEAD
        // AI Pricing client (non-fatal if service not running)
        aiClient := auctions.NewAIPricingClient()
        if aiClient.IsHealthy(ctx) {
                logger.Info("AI Pricing service ready")
        } else {
                logger.Warn("AI Pricing service not available — bid suggestions disabled")
        }
=======
	// ── Redis ─────────────────────────────────────────
	rdbAddr := fmt.Sprintf("%s:%s", util.Getenv("REDIS_HOST", "localhost"), util.Getenv("REDIS_PORT", "6379"))
	rdb := redis.NewClient(&redis.Options{
		Addr:     rdbAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("Redis connect failed", zap.Error(err))
	}
	logger.Info("✅ Redis connected", zap.String("addr", rdbAddr))
>>>>>>> 1aa1121 (TASK-001: Fix build errors - add missing imports, fix bool pointers, remove unused imports)

        middleware.RevocationRDB = rdb

<<<<<<< HEAD
        if os.Getenv("APP_ENV") == "production" {
                gin.SetMode(gin.ReleaseMode)
        }
=======
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{util.Getenv("FRONTEND_URL", "http://localhost:3000")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.Use(middleware.RateLimit(rdb, 100, time.Minute))
>>>>>>> 1aa1121 (TASK-001: Fix build errors - add missing imports, fix bool pointers, remove unused imports)

        r := gin.New()
        r.Use(gin.Recovery())
        corsConfig := cors.Config{
                AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
                AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
                MaxAge:       12 * time.Hour,
        }
        if os.Getenv("APP_ENV") == "production" {
                corsConfig.AllowOrigins = []string{getenv("FRONTEND_URL", "http://localhost:3000")}
                corsConfig.AllowCredentials = true
        } else {
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
        auctionHub := auctions.NewHub(rdb)
        go auctionHub.Run()
        go auctionHub.SubscribeRedis(context.Background())

<<<<<<< HEAD
        v1 := r.Group("/api/v1")
        auth.RegisterRoutes(v1, db, rdb)
        users.RegisterRoutes(v1, db, rdb)
        listings.RegisterRoutes(v1, db, rdb)
        auctions.RegisterRoutes(v1, db, rdb)
        chat.RegisterRoutes(v1, db, rdb)
        payments.RegisterRoutes(v1, db, rdb)
        images.RegisterRoutes(v1, db, rdb)
        notifHub, notifSvc := notifications.RegisterRoutes(v1, db, rdb)
        admin.RegisterRoutes(v1, db, rdb)
        kyc.RegisterRoutes(v1, db)
        reviews.RegisterRoutes(v1, db)
        stores.RegisterRoutes(v1, db)

        // Wire notification service into dependent packages
        auctions.SetNotificationService(notifSvc)
        chat.SetNotificationService(notifSvc)

        // Presigned upload URL — used by KYC and listing image uploads from the browser.
        // Auth required to prevent abuse; returns mock URL in dev when R2 is not configured.
        v1.POST("/media/upload-url", middleware.Auth(), func(c *gin.Context) {
                images.NewHandler(db).GetUploadURL(c)
        })
=======
	// ── Cloudinary (optional) ──────────────────────────
	cloud, _ := cloudinary.New()

	// ── API Routes ────────────────────────────────────
	v1 := r.Group("/api/v1")
	auth.RegisterRoutes(v1, db, rdb)
	users.RegisterRoutes(v1, db, rdb, cloud)
	listings.RegisterRoutes(v1, db, rdb, cloud)
	auctions.RegisterRoutes(v1, db, rdb)
	chat.RegisterRoutes(v1, db, rdb)
	payments.RegisterRoutes(v1, db, rdb)

	// ── Cron: expire listings at midnight UTC ───────────
	go listings.RunExpireCron(db)

	// Auction WebSocket endpoint
	r.GET("/ws/auctions/:id", func(c *gin.Context) {
		auctions.ServeWS(auctionHub, c, db)
	})

	// ── HTTP Server ───────────────────────────────────
	port := util.Getenv("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
>>>>>>> 1aa1121 (TASK-001: Fix build errors - add missing imports, fix bool pointers, remove unused imports)

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
        r.GET("/ws/auctions/:id",  func(c *gin.Context) { auctions.ServeWS(auctionHub, c, db) })
        r.POST("/webhooks/stripe", payments.WebhookHandler(db))

        port := getenv("BACKEND_PORT", getenv("PORT", "8080"))
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
        ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel2()
        _ = srv.Shutdown(ctx2)
  }
  