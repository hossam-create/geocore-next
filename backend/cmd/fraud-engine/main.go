package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/fraud"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/server"
	"github.com/geocore-next/backend/pkg/tracing"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load()

	slog.Info("fraud-engine: starting")

	// ── DB ────────────────────────────────────────────────────────────────
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		env("DB_HOST", "localhost"),
		env("DB_USER", "postgres"),
		env("DB_PASSWORD", "postgres"),
		env("DB_NAME", "geocore_dev"),
		env("DB_PORT", "5432"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("fraud-engine: db connection failed", "error", err)
		os.Exit(1)
	}

	// ── Redis ─────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:         env("REDIS_ADDR", "localhost:6379"),
		Password:     env("REDIS_PASSWORD", ""),
		DB:           0,
		PoolSize:     30, // fraud engine needs moderate Redis connections
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   3,
		PoolTimeout:  4 * time.Second,
	})

	// ── Observability ─────────────────────────────────────────────────────
	metrics.Init()
	tracing.Init()

	// ── Fraud Engine ──────────────────────────────────────────────────────
	store := fraud.NewFeatureStore(rdb)
	engine := fraud.NewEngine(store, fraud.WithDB(db), fraud.WithThresholds(fraud.NewThresholdStore(rdb)))
	fraudConsumer := fraud.NewConsumer(engine)

	// ── Kafka ─────────────────────────────────────────────────────────────
	kafka.SetRedis(rdb)
	kafka.Init()

	// Subscribe to financial event topics
	topics := []string{
		"orders.events",
		"wallet.events",
		"escrow.events",
		"payments.events",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumers := make([]*kafka.Consumer, 0, len(topics))
	for _, topic := range topics {
		c := kafka.NewConsumer(topic, "fraud-service")
		consumers = append(consumers, c)
		go c.Run(ctx, fraudConsumer.HandleEvent)
	}

	// ── Graceful shutdown ──────────────────────────────────────────────────
	server.WaitForCancel(cancel, func(ctx context.Context) {
		for _, c := range consumers {
			_ = c.Close()
		}
		_ = db
	})
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
