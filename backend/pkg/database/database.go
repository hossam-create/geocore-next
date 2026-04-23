package database

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/geocore-next/backend/internal/addons"
	"github.com/geocore-next/backend/internal/admin"
	adminsettings "github.com/geocore-next/backend/internal/admin/settings"
	"github.com/geocore-next/backend/internal/ads"
	"github.com/geocore-next/backend/internal/analytics"
	"github.com/geocore-next/backend/internal/auctions"
	"github.com/geocore-next/backend/internal/auth"
	"github.com/geocore-next/backend/internal/backup"
	"github.com/geocore-next/backend/internal/cancellation"
	"github.com/geocore-next/backend/internal/chargebacks"
	"github.com/geocore-next/backend/internal/chat"
	"github.com/geocore-next/backend/internal/cms"
	"github.com/geocore-next/backend/internal/compliance"
	"github.com/geocore-next/backend/internal/deals"
	"github.com/geocore-next/backend/internal/engagement"
	"github.com/geocore-next/backend/internal/experiments"
	"github.com/geocore-next/backend/internal/fees"
	"github.com/geocore-next/backend/internal/forex"
	"github.com/geocore-next/backend/internal/fraud"
	"github.com/geocore-next/backend/internal/geoscore"
	"github.com/geocore-next/backend/internal/growth"
	"github.com/geocore-next/backend/internal/images"
	"github.com/geocore-next/backend/internal/invite"
	"github.com/geocore-next/backend/internal/kyc"
	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/internal/livestream"
	"github.com/geocore-next/backend/internal/messaging"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/ops"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/internal/payments"
	"github.com/geocore-next/backend/internal/pricing"
	"github.com/geocore-next/backend/internal/protection"
	"github.com/geocore-next/backend/internal/push"
	"github.com/geocore-next/backend/internal/recommendations"
	"github.com/geocore-next/backend/internal/redteam"
	"github.com/geocore-next/backend/internal/reviews"
	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/internal/settlement"
	"github.com/geocore-next/backend/internal/stores"
	"github.com/geocore-next/backend/internal/support"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/internal/waitlist"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/pkg/chaos"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/reputation"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBPair struct {
	Write *gorm.DB
	Read  *gorm.DB
}

func Connect() (*gorm.DB, error) {
	db, err := connectWithDSN(writeDSN())
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	instrumentQueryMetrics(db)
	sql, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	configurePool(sql)
	return db, nil
}

func ConnectReadWrite() (*DBPair, error) {
	writeDB, err := Connect()
	if err != nil {
		return nil, err
	}

	readDSN := os.Getenv("DATABASE_REPLICA_URL")
	if readDSN == "" {
		return &DBPair{Write: writeDB, Read: writeDB}, nil
	}

	readDB, err := connectWithDSN(readDSN)
	if err != nil {
		return nil, fmt.Errorf("connect replica db: %w", err)
	}
	instrumentQueryMetrics(readDB)
	sql, err := readDB.DB()
	if err == nil {
		configurePool(sql)
	}

	return &DBPair{Write: writeDB, Read: readDB}, nil
}

// configurePool applies production-safe connection pool settings.
//
// MaxOpenConns:  caps total connections to prevent DB exhaustion.
//   - default 50 (override via DB_MAX_OPEN_CONNS)
//   - PostgreSQL default max_connections = 100; leave headroom for other clients
//
// MaxIdleConns:  50% of MaxOpenConns to avoid churn (opening/closing connections).
//
// ConnMaxLifetime:  recycle connections after 30 minutes to prevent stale connections
//
//	and allow DB server to rotate credentials / restart gracefully.
//
// ConnMaxIdleTime:  close idle connections after 5 minutes to release resources
//
//	when traffic drops.
func configurePool(sql *sql.DB) {
	maxOpen := 50
	if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxOpen = n
		}
	}
	maxIdle := maxOpen / 2

	sql.SetMaxOpenConns(maxOpen)
	sql.SetMaxIdleConns(maxIdle)
	sql.SetConnMaxLifetime(30 * time.Minute)
	sql.SetConnMaxIdleTime(5 * time.Minute)

	// ── Production DB protection: session-level timeouts ──────────────────
	// These are set per-connection so they survive connection recycling.
	// statement_timeout: kills queries running longer than 5s.
	//   Financial transactions should complete well within this; if they don't,
	//   something is wrong (lock contention, missing index, etc.).
	// lock_timeout: cancels a statement if it waits > 2s for a lock.
	//   Prevents cascading blockage when a long-held lock backs up other txns.
	// idle_in_transaction_session_timeout: kills transactions that sit idle
	//   for > 60s without a query. Prevents leaked connections from holding locks.
	//
	// These can be overridden per-session for migrations or admin operations.
	applySessionTimeouts(sql)
}

func applySessionTimeouts(sql *sql.DB) {
	timeouts := []string{
		// 5s — generous for OLTP. Financial txns with SELECT FOR UPDATE
		// should complete in <1s under normal conditions.
		`SET statement_timeout = '5s'`,
		// 2s — fail fast on lock contention rather than queuing.
		// The application-level retry with backoff handles the retry.
		`SET lock_timeout = '2s'`,
		// 60s — catch leaked transactions (forgotten COMMIT/ROLLBACK).
		`SET idle_in_transaction_session_timeout = '60s'`,
	}
	for _, stmt := range timeouts {
		if _, err := sql.Exec(stmt); err != nil {
			// Non-fatal: may fail on older Postgres or if superuser-only.
			slog.Warn("DB: failed to set session timeout (non-fatal)", "stmt", stmt, "error", err)
		}
	}
	slog.Info("DB: session timeouts applied",
		"statement_timeout", "5s",
		"lock_timeout", "2s",
		"idle_in_transaction_session_timeout", "60s")
}

// StartPoolCollector starts a background goroutine that reports DB pool
// stats to Prometheus every 10 seconds. Call once at startup.
func StartPoolCollector(db *gorm.DB) {
	sql, err := db.DB()
	if err != nil {
		return
	}
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			metrics.ObserveDBConnections(sql)
		}
	}()
}

func AutoMigrate(db *gorm.DB) error {
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`)
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "pg_trgm"`)
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "unaccent"`)

	err := db.AutoMigrate(
		&users.User{},
		&security.SecurityAuditLog{},
		&wallet.Wallet{},
		&wallet.WalletBalance{},
		&wallet.WalletTransaction{},
		&wallet.Escrow{},
		&wallet.PricePlan{},
		&wallet.UserSubscription{},
		&listings.Category{},
		&listings.Listing{},
		&listings.ListingImage{},
		&listings.Favorite{},
		&listings.SavedSearch{},
		&auctions.Auction{},
		&auctions.Bid{},
		&chat.Conversation{},
		&chat.ConversationMember{},
		&chat.Message{},
		&deals.Deal{},
		&payments.Payment{},
		&payments.EscrowAccount{},
		&payments.SavedPaymentMethod{},
		&payments.ProcessedStripeEvent{},
		&payments.PayMobOrder{},
		&payments.ProcessedPayMobEvent{},
		&forex.ExchangeRate{},
		&forex.ConversionRecord{},
		&settlement.Settlement{},
		&settlement.Payout{},
		&reputation.Profile{},
		&fees.FeeConfig{},
		&geoscore.GeoScore{},
		&geoscore.BehaviorEvent{},
		&analytics.RouteMetrics{},
		&wallet.IdempotentRequest{},
		&images.Image{},
		&images.ListingImageAssoc{},
		&auth.User2FA{},
		&notifications.Notification{},
		&notifications.NotificationPreference{},
		&notifications.PushToken{},
		&push.UserDevice{},
		&push.PushLog{},
		&admin.AdminLog{},
		&kyc.KYCProfile{},
		&kyc.KYCDocument{},
		&kyc.KYCAuditLog{},
		&reviews.Review{},
		&stores.Storefront{},
		&support.ContactMessage{},
		&support.SupportTicket{},
		&support.TicketMessage{},
		&order.Order{},
		&order.OrderItem{},
		&kafka.OutboxEvent{},
		&kafka.ProcessedEvent{},
		&recommendations.UserInteraction{},
		&livestream.Session{},
		&livestream.LiveItem{},
		&livestream.LiveBid{},
		&livestream.AuctionDeposit{},
		&livestream.LiveConversionEvent{},
		&livestream.LiveCommission{},
		&livestream.LiveBoost{},
		&livestream.LivePaidEntry{},
		&livestream.LivePriorityBid{},
		&livestream.LiveStreamerEarning{},
		&livestream.LiveAIEvent{},
		&livestream.LiveInvite{},
		&livestream.WinShare{},
		&livestream.GroupInvite{},
		&livestream.GroupMember{},
		&livestream.UserStreak{},
		&livestream.GrowthReward{},
		// Sprint 16: Creator Economy
		&livestream.Creator{},
		&livestream.CreatorDeal{},
		&livestream.CreatorEarning{},
		&livestream.CreatorMilestone{},
		// Sprint 17: Marketplace Brain
		&livestream.SessionScoreSnapshot{},
		// Sprint 20: Private Invite & Referral Network
		&invite.Invite{},
		&invite.InviteUsage{},
		&invite.ReferralReward{},
		// Sprint 21: Waitlist + Hype Engine
		&waitlist.WaitlistUser{},
		&waitlist.WaitlistConfig{},
		&waitlist.OnboardingState{},
		// Sprint 22: Data Protection & Incident Response
		&backup.BackupRecord{},
		// Sprint 23: Security Monitoring & Admin Observability
		&security.UserRiskProfile{},
		// Sprint 24: Fraud Prediction Engine
		&fraud.UserRiskSnapshot{},
		// Sprint 25: Red Team Simulation
		&redteam.RedTeamRun{},
		// Sprint 26: Compliance & GDPR Layer
		&compliance.ConsentRecord{},
		&compliance.ComplianceAuditLog{},
		// Admin Settings Engine (Layer 0 blueprint)
		&adminsettings.AdminSetting{},
		&adminsettings.FeatureFlag{},
		&adminsettings.SupportTicket{},
		&adminsettings.TicketMessage{},
		&adminsettings.TrustFlag{},
		// Phase 8: Banner Ads
		&ads.Ad{},
		// Phase 8: Chargebacks
		&chargebacks.Chargeback{},
		// Phase 9: Addon Marketplace
		&addons.Addon{},
		&addons.AddonVersion{},
		&addons.AddonReview{},
		// Phase 9: CMS
		&cms.HeroSlide{},
		&cms.ContentBlock{},
		&cms.MediaFile{},
		&cms.SiteSetting{},
		&cms.NavMenu{},
		// Phase 9: Listing extras (variants, Q&A, feedback)
		&listings.ListingVariant{},
		&listings.ListingQA{},
		&listings.ListingFeedback{},
		// Smart Cancellation Fee Engine
		&cancellation.CancellationPolicy{},
		&cancellation.UserCancellationStats{},
		&cancellation.UserCancellationToken{},
		&cancellation.CancellationLedger{},
		&cancellation.OrderInsurance{},
		&cancellation.UserInsuranceUsage{},
		// Travel Guarantee + Protection Engine
		&protection.OrderProtection{},
		&protection.GuaranteeClaim{},
		&protection.ABVariantAssignment{},
		&protection.ABEvent{},
		&protection.ProtectionDailyMetrics{},
		// Dynamic Insurance Pricing AI
		pricing.PricingModelConfig{},
		pricing.PricingEvent{},
		pricing.PricingABAssignment{},
		// Multi-Armed Bandit Pricing
		pricing.BanditArm{},
		pricing.BanditEvent{},
		pricing.BanditConfig{},
		// Reinforcement Learning Pricing
		pricing.RLTransition{},
		pricing.RLSession{},
		pricing.RLConfig{},
		// Hybrid Pricing Engine
		pricing.HybridConfig{},
		pricing.HybridEvent{},
		// Cross-System RL Coordinator
		pricing.CrossConfig{},
		pricing.CrossTransition{},
		pricing.CrossEvent{},
		// Feature Store + Embeddings + Retrieval
		pricing.UserFeatures{},
		pricing.ItemFeatures{},
		pricing.SessionFeatures{},
		pricing.EmbeddingVector{},
		pricing.EmbeddingEvent{},
		pricing.PipelineLatencyLog{},
		// Engagement Engine (momentum + notifications + re-engagement + timing)
		engagement.SessionMomentum{},
		engagement.NotificationEvent{},
		engagement.UserEngagementProfile{},
		engagement.PlannedTouch{},
		engagement.UserActivityHour{},
		engagement.EngagementConfig{},
		// Growth Engine (user state + dopamine + re-engagement + decisions)
		growth.UserState{},
		growth.UserActionEvent{},
		growth.DopamineEvent{},
		growth.ReEngagementLog{},
		growth.DecisionLog{},
		// Messaging (dispatcher + templates + channels + anti-spam)
		messaging.Message{},
		messaging.MessageCooldown{},
		messaging.UserMessagingPrefs{},
		// Experiments (A/B + bandits)
		experiments.Experiment{},
		experiments.ExperimentAssignment{},
		experiments.ExperimentEvent{},
		experiments.BanditArm{},
		experiments.BanditPull{},
	)
	if err != nil {
		return err
	}
	listings.SeedCategories(db)
	listings.ApplySearchIndexes(db)
	forex.SeedRates(db)
	fees.SeedDefaults(db)
	fees.Init(db)
	admin.SeedEmailTemplates(db)
	addons.SeedAddons(db)
	cms.SeedCMS(db)
	cancellation.SeedDefaults(db)
	if err := ops.AutoMigrateOps(db); err != nil {
		return err
	}
	applyPerformanceIndexes(db)
	return nil
}

// applyPerformanceIndexes creates partial/composite indexes for hot read paths.
// Uses CREATE INDEX IF NOT EXISTS to be idempotent on re-runs.
// Note: CONCURRENTLY cannot run inside a transaction; GORM raw exec is fine here.
func applyPerformanceIndexes(db *gorm.DB) {
	indexes := []string{
		// Security audit log hot queries
		`CREATE INDEX IF NOT EXISTS idx_sal_user_created ON security_audit_logs (user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sal_event_created ON security_audit_logs (event_type, created_at DESC)`,
		// Payments hot queries
		`CREATE INDEX IF NOT EXISTS idx_payments_user_status ON payments (user_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_payments_pi ON payments (stripe_payment_intent_id)`,
		// Wallet transactions hot query
		`CREATE INDEX IF NOT EXISTS idx_wallet_tx_wallet_created ON wallet_transactions (wallet_id, created_at DESC)`,
		// Escrow state queries
		`CREATE INDEX IF NOT EXISTS idx_escrows_status ON escrows (status)`,
		`CREATE INDEX IF NOT EXISTS idx_escrows_buyer ON escrows (buyer_id, status)`,
		// Webhook dedup lookup
		`CREATE INDEX IF NOT EXISTS idx_processed_stripe_event ON processed_stripe_events (stripe_event_id)`,
		// Idempotency lookup
		`CREATE INDEX IF NOT EXISTS idx_idempotent_req_lookup ON idempotent_requests (user_id, idempotency_key, expires_at)`,
		// Scaling indexes
		`CREATE INDEX IF NOT EXISTS idx_wallet_user_currency ON wallet_balances(user_id, currency)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(buyer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_disputes_deadline ON disputes(status, resolution_deadline)`,
		`CREATE INDEX IF NOT EXISTS idx_escrow_status ON escrows(status)`,
	}
	for _, sql := range indexes {
		if err := db.Exec(sql).Error; err != nil {
			// Non-fatal: log and continue (index may already exist with different name)
			_ = err
		}
	}
}

func writeDSN() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		getenv2("DB_HOST", "PGHOST", "localhost"),
		getenv2("DB_USER", "PGUSER", "geocore"),
		getenv2("DB_PASSWORD", "PGPASSWORD", "geocore_secret"),
		getenv2("DB_NAME", "PGDATABASE", "geocore_dev"),
		getenv2("DB_PORT", "PGPORT", "5432"),
		getenv("DB_SSLMODE", "disable"),
	)
}

func connectWithDSN(dsn string) (*gorm.DB, error) {
	lvl := logger.Silent
	if os.Getenv("APP_ENV") != "production" {
		lvl = logger.Info
	}
	return gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(lvl)})
}

func instrumentQueryMetrics(db *gorm.DB) {
	if db == nil {
		return
	}

	// Chaos hook: inject DB latency before each operation
	chaosLatency := func(tx *gorm.DB) {
		if d := chaos.DBLatency(); d > 0 {
			time.Sleep(d)
		}
	}
	_ = db.Callback().Query().Before("gorm:query").Register("chaos:latency", chaosLatency)
	_ = db.Callback().Create().Before("gorm:create").Register("chaos:latency", chaosLatency)
	_ = db.Callback().Update().Before("gorm:update").Register("chaos:latency", chaosLatency)
	_ = db.Callback().Delete().Before("gorm:delete").Register("chaos:latency", chaosLatency)
	_ = db.Callback().Raw().Before("gorm:raw").Register("chaos:latency", chaosLatency)

	after := func(operation string) func(*gorm.DB) {
		return func(tx *gorm.DB) {
			if v, ok := tx.InstanceGet(operation + ":start"); ok {
				if start, ok := v.(time.Time); ok {
					elapsed := time.Since(start)
					metrics.ObserveDBQueryDuration(operation, elapsed)
					if elapsed > 200*time.Millisecond {
						slog.Warn("slow query detected",
							"operation", operation,
							"duration_ms", elapsed.Milliseconds(),
							"table", tx.Statement.Table,
							"rows_affected", tx.RowsAffected,
						)
					}
				}
			}
		}
	}

	_ = db.Callback().Query().Before("gorm:query").Register("metrics:query:before", func(tx *gorm.DB) {
		tx.InstanceSet("query:start", time.Now())
	})
	_ = db.Callback().Query().After("gorm:query").Register("metrics:query:after", after("query"))

	_ = db.Callback().Create().Before("gorm:create").Register("metrics:create:before", func(tx *gorm.DB) {
		tx.InstanceSet("create:start", time.Now())
	})
	_ = db.Callback().Create().After("gorm:create").Register("metrics:create:after", after("create"))

	_ = db.Callback().Update().Before("gorm:update").Register("metrics:update:before", func(tx *gorm.DB) {
		tx.InstanceSet("update:start", time.Now())
	})
	_ = db.Callback().Update().After("gorm:update").Register("metrics:update:after", after("update"))

	_ = db.Callback().Delete().Before("gorm:delete").Register("metrics:delete:before", func(tx *gorm.DB) {
		tx.InstanceSet("delete:start", time.Now())
	})
	_ = db.Callback().Delete().After("gorm:delete").Register("metrics:delete:after", after("delete"))

	_ = db.Callback().Raw().Before("gorm:raw").Register("metrics:raw:before", func(tx *gorm.DB) {
		tx.InstanceSet("raw:start", time.Now())
	})
	_ = db.Callback().Raw().After("gorm:raw").Register("metrics:raw:after", after("raw"))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenv2(key1, key2, fallback string) string {
	if v := os.Getenv(key1); v != "" {
		return v
	}
	if v := os.Getenv(key2); v != "" {
		return v
	}
	return fallback
}
