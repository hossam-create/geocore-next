package main

import (
	"context"
	"errors"
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
	"github.com/geocore-next/backend/internal/disputes"
	"github.com/geocore-next/backend/internal/images"
	"github.com/geocore-next/backend/internal/kyc"
	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/internal/loyalty"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/payments"
	"github.com/geocore-next/backend/internal/reviews"
	"github.com/geocore-next/backend/internal/stores"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/database"
	"github.com/geocore-next/backend/pkg/i18n"
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func validateEnv() error {
	// JWT_SECRET is critical for authentication
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return errors.New("JWT_SECRET environment variable is required")
	}
	if len(jwtSecret) < 32 {
		return errors.New("JWT_SECRET must be at least 32 characters for security")
	}

	// Warn if using placeholder values in production
	if os.Getenv("APP_ENV") == "production" {
		if jwtSecret == "change_this_to_a_secure_random_string_min_32_chars" {
			return errors.New("JWT_SECRET must not use placeholder value in production")
		}
	}

	return nil
}

func main() {
	_ = godotenv.Load()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Validate critical environment variables
	if err := validateEnv(); err != nil {
		logger.Fatal("Environment validation failed", zap.Error(err))
	}

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

	middleware.RevocationRDB = rdb

	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	// Security headers
	r.Use(middleware.SecurityHeaders())

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

	// i18n middleware
	r.Use(i18n.Middleware())

	// Rate limiting
	rl := middleware.NewRateLimiter(rdb)
	r.Use(rl.Limit(100, time.Minute, "api"))

	// Start background job queue
	jobQueue := jobs.NewJobQueue(rdb)
	jobs.RegisterDefaultHandlers(jobQueue, &jobs.HandlerDependencies{})
	jobQueue.Start(4) // 4 workers
	defer jobQueue.Stop()

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
	wallet.RegisterRoutes(v1, db)
	disputes.RegisterRoutes(v1, db)
	loyalty.RegisterRoutes(v1, db)

	auctions.SetNotificationService(notifSvc)
	chat.SetNotificationService(notifSvc)

	v1.POST("/media/upload-url", middleware.Auth(), func(c *gin.Context) {
		images.NewHandler(db).GetUploadURL(c)
	})

	r.GET("/ws/notifications", func(c *gin.Context) { notifications.ServeWS(notifHub, c) })
	r.GET("/ws/auctions/:id", func(c *gin.Context) { auctions.ServeWS(auctionHub, c, db) })
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
