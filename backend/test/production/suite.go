//go:build production

package production

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/geocore-next/backend/internal/payments"
	"github.com/geocore-next/backend/internal/push"
	"github.com/geocore-next/backend/pkg/email"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/kafka/consumers"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ════════════════════════════════════════════════════════════════════════════════
// Production Integration Test Suite
//
// Tests REAL external integrations (SendGrid, FCM, Stripe, Kafka).
// These tests require live credentials and are gated behind the "production"
// build tag. They are NOT run in CI by default.
//
// Run:  go test -tags=production ./test/production/ -timeout 300s
// ════════════════════════════════════════════════════════════════════════════════

// ProdSuite holds real service clients for production integration tests.
type ProdSuite struct {
	Tst       *testing.T
	DB        *gorm.DB
	RDB       *redis.Client
	Ctx       context.Context
	EmailSvc  *email.EmailService
	PushSvc   *push.PushService
	Firebase  *push.FirebaseClient
	KafkaProd *kafka.Producer
}

// SetupProdSuite connects to the production-like staging DB and Redis,
// and initialises real external service clients from environment variables.
//
// Required env vars (at minimum):
//   - DATABASE_URL       — Postgres connection string
//   - REDIS_URL          — Redis address (e.g. localhost:6379)
//   - EMAIL_PROVIDER     — "sendgrid" or "ses"
//   - SENDGRID_API_KEY   — if EMAIL_PROVIDER=sendgrid
//   - FIREBASE_SERVICE_ACCOUNT_JSON — for FCM push
//   - KAFKA_BROKERS      — comma-separated broker list
//   - STRIPE_WEBHOOK_SECRET — for webhook verification
func SetupProdSuite(t *testing.T) *ProdSuite {
	ctx := context.Background()

	// ── Database ──────────────────────────────────────────────────────────────
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping production integration tests")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	require.NoError(t, err, "failed to connect to database")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	// ── Redis ────────────────────────────────────────────────────────────────
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	_, err = rdb.Ping(ctx).Result()
	require.NoError(t, err, "failed to ping Redis")

	// ── Email service (real provider) ────────────────────────────────────────
	emailSvc := email.New(rdb)

	// ── Firebase client ──────────────────────────────────────────────────────
	firebaseClient := push.NewFirebaseClientFromEnv()

	// ── Push service ─────────────────────────────────────────────────────────
	pushSvc := push.NewPushService(db, rdb, firebaseClient, nil)

	// ── Kafka ────────────────────────────────────────────────────────────────
	kafka.SetRedis(rdb)
	kafkaProd := kafka.Init()

	return &ProdSuite{
		Tst:       t,
		DB:        db,
		RDB:       rdb,
		Ctx:       ctx,
		EmailSvc:  emailSvc,
		PushSvc:   pushSvc,
		Firebase:  firebaseClient,
		KafkaProd: kafkaProd,
	}
}

// TeardownProdSuite cleans up resources.
func TeardownProdSuite(s *ProdSuite) {
	if s.KafkaProd != nil {
		s.KafkaProd.Close()
	}
	if s.RDB != nil {
		s.RDB.Close()
	}
	if s.DB != nil {
		sqlDB, _ := s.DB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

// StartKafkaConsumers starts all consumer groups for the given context.
// Call this when testing Kafka end-to-end flows.
func (s *ProdSuite) StartKafkaConsumers(ctx context.Context) {
	consumers.StartAll(ctx, s.DB)
}

// StartOutboxWorker starts the outbox worker for the given context.
func (s *ProdSuite) StartOutboxWorker(ctx context.Context) {
	kafka.NewOutboxWorker(s.DB, 2*time.Second).Start(ctx)
}

// SeedPayment creates a Payment record in pending state for webhook tests.
func (s *ProdSuite) SeedPayment(userID uuid.UUID, stripePIID string, amount float64) uuid.UUID {
	id := uuid.New()
	p := payments.Payment{
		ID:                    id,
		UserID:                userID,
		StripePaymentIntentID: stripePIID,
		Amount:                amount,
		Currency:              "USD",
		Status:                payments.PaymentStatusPending,
		PaymentMethod:         "card",
	}
	require.NoError(s.Tst, s.DB.Create(&p).Error)
	return id
}

// UniqueProdEmail generates a unique email for production test recipients.
func UniqueProdEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@test.geocore.dev", prefix, uuid.New().String()[:8])
}
