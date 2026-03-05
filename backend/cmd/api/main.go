package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"fmt"

	"github.com/geocore-next/backend/internal/auctions"
	"github.com/geocore-next/backend/internal/auth"
	"github.com/geocore-next/backend/internal/chat"
	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/internal/payments"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/database"

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

func main() {
	_ = godotenv.Load()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// ── Database ──────────────────────────────────────
	db, err := database.Connect()
	if err != nil {
		logger.Fatal("DB connect failed", zap.Error(err))
	}
	if err := database.AutoMigrate(db); err != nil {
		logger.Fatal("AutoMigrate failed", zap.Error(err))
	}
	logger.Info("✅ Database ready")

	// ── Redis ─────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", getenv("REDIS_HOST", "localhost"), getenv("REDIS_PORT", "6379")),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("Redis connect failed", zap.Error(err))
	}
	logger.Info("✅ Redis ready")

	// ── Gin ───────────────────────────────────────────
	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{getenv("FRONTEND_URL", "http://localhost:3000")},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now()})
	})

	// ── WebSocket Hubs ────────────────────────────────
	chatHub := chat.NewHub(rdb)
	go chatHub.Run()

	auctionHub := auctions.NewHub(rdb)
	go auctionHub.Run()

	// ── API Routes ────────────────────────────────────
	v1 := r.Group("/api/v1")
	auth.RegisterRoutes(v1, db, rdb)
	users.RegisterRoutes(v1, db, rdb)
	listings.RegisterRoutes(v1, db, rdb)
	auctions.RegisterRoutes(v1, db, rdb)
	chat.RegisterRoutes(v1, db, rdb)
	payments.RegisterRoutes(v1, db, rdb)

	// Auction WebSocket endpoint
	r.GET("/ws/auctions/:id", func(c *gin.Context) {
		auctions.ServeWS(auctionHub, c, db)
	})

	// ── HTTP Server ───────────────────────────────────
	port := getenv("PORT", "8080")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("🚀 GeoCore Next API running", zap.String("port", port))
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
