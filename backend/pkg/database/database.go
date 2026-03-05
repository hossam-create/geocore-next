package database

import (
	"fmt"
	"os"

	"github.com/geocore-next/backend/internal/auctions"
	"github.com/geocore-next/backend/internal/chat"
	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/internal/users"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		getenv("DB_HOST", "localhost"),
		getenv("DB_USER", "geocore"),
		getenv("DB_PASSWORD", "geocore_secret"),
		getenv("DB_NAME", "geocore_dev"),
		getenv("DB_PORT", "5432"),
		getenv("DB_SSLMODE", "disable"),
	)

	lvl := logger.Silent
	if os.Getenv("APP_ENV") != "production" {
		lvl = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(lvl),
	})
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	sql, _ := db.DB()
	sql.SetMaxIdleConns(10)
	sql.SetMaxOpenConns(100)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "pg_trgm"`)

	err := db.AutoMigrate(
		&users.User{},
		&listings.Category{},
		&listings.Listing{},
		&listings.ListingImage{},
		&listings.Favorite{},
		&auctions.Auction{},
		&auctions.Bid{},
		&chat.Conversation{},
		&chat.ConversationMember{},
		&chat.Message{},
	)
	if err != nil {
		return err
	}

	// Seed default data
	listings.SeedCategories(db)
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
