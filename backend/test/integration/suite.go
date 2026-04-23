package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ════════════════════════════════════════════════════════════════════════════════
// Shared Integration Test Infrastructure
// ════════════════════════════════════════════════════════════════════════════════

// TestSuite holds shared infrastructure for all integration tests.
type TestSuite struct {
	Tst    *testing.T
	DB     *gorm.DB
	RDB    *redis.Client
	PgC    *postgres.PostgresContainer
	RedisC *tcredis.RedisContainer
	Ctx    context.Context
}

// SetupSuite starts Postgres + Redis containers and returns a ready TestSuite.
func SetupSuite(t *testing.T) *TestSuite {
	ctx := context.Background()

	pgC, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase("geocore_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
	)
	require.NoError(t, err, "failed to start postgres container")

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(pgdriver.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err, "failed to connect to postgres")

	db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	redisC, err := tcredis.Run(ctx, "redis:7-alpine")
	require.NoError(t, err, "failed to start redis container")

	redisAddr, err := redisC.ConnectionString(ctx)
	require.NoError(t, err)

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	_, err = rdb.Ping(ctx).Result()
	require.NoError(t, err, "failed to ping redis")

	return &TestSuite{
		Tst:    t,
		DB:     db,
		RDB:    rdb,
		PgC:    pgC,
		RedisC: redisC,
		Ctx:    ctx,
	}
}

// TeardownSuite terminates containers.
func (s *TestSuite) TeardownSuite() {
	if s.RDB != nil {
		s.RDB.Close()
	}
	if s.PgC != nil {
		s.PgC.Terminate(s.Ctx)
	}
	if s.RedisC != nil {
		s.RedisC.Terminate(s.Ctx)
	}
}

// ResetTest flushes Redis between tests.
func (s *TestSuite) ResetTest() {
	s.RDB.FlushDB(s.Ctx)
}

// AutoMigrateAll runs AutoMigrate for all models used across test scenarios.
func (s *TestSuite) AutoMigrateAll(models ...interface{}) {
	err := s.DB.AutoMigrate(models...)
	require.NoError(s.Tst, err, "AutoMigrate failed")
}

// CreateManualTables creates tables that can't be AutoMigrated due to import cycles.
func (s *TestSuite) CreateManualTables() {
	s.DB.Exec(`CREATE TABLE IF NOT EXISTS categories (
		id UUID PRIMARY KEY, parent_id UUID, name_en TEXT NOT NULL, name_ar TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE, description TEXT, icon TEXT, icon_url TEXT, image_url TEXT,
		color TEXT, sort_order INT DEFAULT 0, is_active BOOLEAN DEFAULT true, is_leaf BOOLEAN DEFAULT false,
		listing_count INT DEFAULT 0, level INT DEFAULT 0, path TEXT
	)`)
	s.DB.Exec(`CREATE TABLE IF NOT EXISTS listing_images (
		id UUID PRIMARY KEY, listing_id UUID NOT NULL, url TEXT NOT NULL,
		sort_order INT DEFAULT 0, is_cover BOOLEAN DEFAULT false
	)`)
	s.DB.Exec(`CREATE TABLE IF NOT EXISTS listings (
		id UUID PRIMARY KEY, user_id UUID NOT NULL, category_id UUID NOT NULL,
		title TEXT NOT NULL, description TEXT, price NUMERIC, currency TEXT DEFAULT 'USD',
		price_type TEXT DEFAULT 'fixed', condition TEXT, status TEXT DEFAULT 'active',
		type TEXT DEFAULT 'sell', listing_type TEXT DEFAULT 'buy_now',
		trade_config JSONB DEFAULT '{}', price_cents BIGINT DEFAULT 0,
		country TEXT, city TEXT, address TEXT, latitude DOUBLE PRECISION, longitude DOUBLE PRECISION,
		location TEXT, category TEXT,
		view_count INT DEFAULT 0, favorite_count INT DEFAULT 0, is_featured BOOLEAN DEFAULT false,
		expires_at TIMESTAMPTZ, sold_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ, deleted_at TIMESTAMPTZ,
		custom_fields JSONB DEFAULT '{}'
	)`)
	s.DB.Exec(`CREATE TABLE IF NOT EXISTS search_queries (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		query TEXT NOT NULL,
		intent_json JSONB,
		result_count INT DEFAULT 0,
		user_id UUID,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)
}

// ════════════════════════════════════════════════════════════════════════════════
// Shared Helpers
// ════════════════════════════════════════════════════════════════════════════════

// CreateUser creates a user and returns their ID.
func (s *TestSuite) CreateUser(name, email string) uuid.UUID {
	id := uuid.New()
	pwdHash := "$2a$10$fakehashforintegrationtestonly"
	user := users.User{
		ID:           id,
		Name:         name,
		Email:        email,
		PasswordHash: pwdHash,
		IsActive:     true,
		Role:         "user",
	}
	require.NoError(s.Tst, s.DB.Create(&user).Error)
	return id
}

// CreateUserWithEmailVerified creates a user with email already verified.
func (s *TestSuite) CreateUserWithEmailVerified(name, email string) uuid.UUID {
	id := uuid.New()
	pwdHash := "$2a$10$fakehashforintegrationtestonly"
	user := users.User{
		ID:            id,
		Name:          name,
		Email:         email,
		PasswordHash:  pwdHash,
		IsActive:      true,
		Role:          "user",
		EmailVerified: true,
	}
	require.NoError(s.Tst, s.DB.Create(&user).Error)
	return id
}

// CreateUserWithVerificationToken creates a user with a specific verification token.
func (s *TestSuite) CreateUserWithVerificationToken(name, email, token string, expiresAt time.Time) uuid.UUID {
	id := uuid.New()
	pwdHash := "$2a$10$fakehashforintegrationtestonly"
	user := users.User{
		ID:                         id,
		Name:                       name,
		Email:                      email,
		PasswordHash:               pwdHash,
		IsActive:                   true,
		Role:                       "user",
		VerificationToken:          token,
		VerificationTokenExpiresAt: &expiresAt,
	}
	require.NoError(s.Tst, s.DB.Create(&user).Error)
	return id
}

// FundWallet creates a wallet + USD balance row and deposits the given amount.
func (s *TestSuite) FundWallet(userID uuid.UUID, amount float64) {
	w := wallet.Wallet{
		ID:              uuid.New(),
		UserID:          userID,
		PrimaryCurrency: wallet.USD,
		DailyLimit:      decimal.NewFromInt(100000),
		MonthlyLimit:    decimal.NewFromInt(1000000),
		IsActive:        true,
	}
	require.NoError(s.Tst, s.DB.Create(&w).Error)

	amt := decimal.NewFromFloat(amount)
	bal := wallet.WalletBalance{
		ID:               uuid.New(),
		WalletID:         w.ID,
		Currency:         wallet.USD,
		Balance:          amt,
		AvailableBalance: amt,
		PendingBalance:   decimal.Zero,
	}
	require.NoError(s.Tst, s.DB.Create(&bal).Error)
}

// GetWalletBalances returns the wallet balances for a user.
func (s *TestSuite) GetWalletBalances(userID uuid.UUID) (balance, available, pending decimal.Decimal) {
	var w wallet.Wallet
	s.DB.Where("user_id = ?", userID).First(&w)
	var bal wallet.WalletBalance
	s.DB.Where("wallet_id = ? AND currency = ?", w.ID, wallet.USD).First(&bal)
	return bal.Balance, bal.AvailableBalance, bal.PendingBalance
}

// CreateCategory creates a test category and returns its ID.
func (s *TestSuite) CreateCategory() uuid.UUID {
	id := uuid.New()
	type testCategory struct {
		ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
		NameEn string    `gorm:"not null"`
		NameAr string    `gorm:"not null"`
		Slug   string    `gorm:"uniqueIndex;not null"`
	}
	cat := testCategory{
		ID:     id,
		NameEn: "Test Category",
		NameAr: "فئة اختبار",
		Slug:   "test-" + id.String()[:8],
	}
	require.NoError(s.Tst, s.DB.Table("categories").Create(&cat).Error)
	return id
}

// CreateListing creates a test listing owned by the given user and returns its ID.
func (s *TestSuite) CreateListing(userID, categoryID uuid.UUID, title string, price float64) uuid.UUID {
	id := uuid.New()
	type testListing struct {
		ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
		UserID     uuid.UUID `gorm:"not null"`
		CategoryID uuid.UUID `gorm:"not null"`
		Title      string    `gorm:"not null"`
		Price      float64
		Currency   string `gorm:"default:'USD'"`
		Status     string `gorm:"default:'active'"`
	}
	l := testListing{
		ID:         id,
		UserID:     userID,
		CategoryID: categoryID,
		Title:      title,
		Price:      price,
		Currency:   "USD",
		Status:     "active",
	}
	require.NoError(s.Tst, s.DB.Table("listings").Create(&l).Error)
	return id
}

// UniqueEmail generates a unique email for test users.
func UniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@test.geocore.dev", prefix, uuid.New().String()[:8])
}
