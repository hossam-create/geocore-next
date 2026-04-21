// Binary worker runs Kafka consumers and async job processors.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/kafka/consumers"
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

	slog.Info("worker: starting")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		env("DB_HOST", "localhost"),
		env("DB_USER", "postgres"),
		env("DB_PASSWORD", "postgres"),
		env("DB_NAME", "geocore_dev"),
		env("DB_PORT", "5432"),
	)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("worker: db connection failed", "error", err)
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:         env("REDIS_ADDR", "localhost:6379"),
		Password:     env("REDIS_PASSWORD", ""),
		PoolSize:     50, // workers need fewer Redis connections than API
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   3,
		PoolTimeout:  4 * time.Second,
	})

	metrics.Init()
	tracing.Init()

	kafka.SetRedis(rdb)
	kafka.Init()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kafka.NewOutboxWorker(db, 2*time.Second).Start(ctx)
	consumers.StartAll(ctx, db)

	server.WaitForCancel(cancel, func(ctx context.Context) {
		kafka.Global().Close()
	})
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
