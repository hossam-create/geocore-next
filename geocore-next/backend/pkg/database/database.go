package database

import (
        "fmt"
        "os"
        "time"

        "github.com/geocore-next/backend/internal/admin"
        "github.com/geocore-next/backend/internal/auctions"
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
        "github.com/geocore-next/backend/pkg/util"
        "gorm.io/driver/postgres"
        "gorm.io/gorm"
        "gorm.io/gorm/logger"
)

const (
        maxRetries  = 5
        retryDelay  = 2 * time.Second
)

func Connect() (*gorm.DB, error) {
        var dsn string

        if url := os.Getenv("DATABASE_URL"); url != "" {
                dsn = url
        } else {
                dsn = fmt.Sprintf(
                        "host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
                        util.Getenv2("DB_HOST", "PGHOST", "localhost"),
                        util.Getenv2("DB_USER", "PGUSER", "geocore"),
                        util.Getenv2("DB_PASSWORD", "PGPASSWORD", "geocore_secret"),
                        util.Getenv2("DB_NAME", "PGDATABASE", "geocore_dev"),
                        util.Getenv2("DB_PORT", "PGPORT", "5432"),
                        util.Getenv("DB_SSLMODE", "disable"),
                )
        }

        lvl := logger.Silent
        if os.Getenv("APP_ENV") != "production" {
                lvl = logger.Info
        }

        var db *gorm.DB
        var err error
        for attempt := 1; attempt <= maxRetries; attempt++ {
                db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(lvl)})
                if err == nil {
                        // Verify the underlying connection is actually reachable
                        sqlDB, pingErr := db.DB()
                        if pingErr == nil {
                                pingErr = sqlDB.Ping()
                        }
                        if pingErr == nil {
                                break
                        }
                        err = pingErr
                }
                if attempt < maxRetries {
                        fmt.Printf("DB connect attempt %d/%d failed: %v — retrying in %s\n", attempt, maxRetries, err, retryDelay)
                        time.Sleep(retryDelay)
                }
        }
        if err != nil {
                return nil, fmt.Errorf("connect db after %d attempts: %w", maxRetries, err)
        }

        sql, _ := db.DB()
        sql.SetMaxIdleConns(10)
        sql.SetMaxOpenConns(100)
        sql.SetConnMaxLifetime(5 * time.Minute)
        sql.SetConnMaxIdleTime(2 * time.Minute)

        // Warm the connection pool: open one connection so the first real query
        // doesn't pay the dial latency. QueryRow is lighter than Ping (Ping closes
        // the connection immediately; QueryRow keeps it in the pool).
        if _, warmErr := sql.Exec("SELECT 1"); warmErr != nil {
                return nil, fmt.Errorf("db pool warm-up query failed: %w", warmErr)
        }

        return db, nil
}

func AutoMigrate(db *gorm.DB) error {
        db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
        db.Exec(`CREATE EXTENSION IF NOT EXISTS "pg_trgm"`)
        db.Exec(`CREATE EXTENSION IF NOT EXISTS "unaccent"`)

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
                &payments.Payment{},
                &payments.EscrowAccount{},
                &payments.SavedPaymentMethod{},
                &images.Image{},
                &notifications.Notification{},
                &notifications.NotificationPreference{},
                &notifications.PushToken{},
                &admin.AdminLog{},
                &kyc.KYCProfile{},
                &kyc.KYCDocument{},
                &kyc.KYCAuditLog{},
                &reviews.Review{},
                &stores.Storefront{},
		&monetization.PlatformSettings{},
		&monetization.PlatformCommission{},
		&monetization.SellerSubscription{},
        )
        if err != nil {
                return err
        }
        listings.SeedCategories(db)
        listings.ApplySearchIndexes(db)
        return nil
}

