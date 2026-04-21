package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/geocore-next/backend/internal/addons"
	"github.com/geocore-next/backend/internal/admin"
	adminsettings "github.com/geocore-next/backend/internal/admin/settings"
	"github.com/geocore-next/backend/internal/ads"
	"github.com/geocore-next/backend/internal/aichat"
	"github.com/geocore-next/backend/internal/aiops"
	"github.com/geocore-next/backend/internal/analytics"
	"github.com/geocore-next/backend/internal/arpreview"
	"github.com/geocore-next/backend/internal/auctions"
	"github.com/geocore-next/backend/internal/auth"
	"github.com/geocore-next/backend/internal/autonomy"
	"github.com/geocore-next/backend/internal/backup"
	"github.com/geocore-next/backend/internal/billing"
	"github.com/geocore-next/backend/internal/blockchain"
	"github.com/geocore-next/backend/internal/bnpl"
	"github.com/geocore-next/backend/internal/cancellation"
	"github.com/geocore-next/backend/internal/cart"
	"github.com/geocore-next/backend/internal/chaos"
	"github.com/geocore-next/backend/internal/chargebacks"
	"github.com/geocore-next/backend/internal/chat"
	"github.com/geocore-next/backend/internal/cms"
	"github.com/geocore-next/backend/internal/compliance"
	"github.com/geocore-next/backend/internal/controlplane"
	"github.com/geocore-next/backend/internal/controltower"
	"github.com/geocore-next/backend/internal/country"
	"github.com/geocore-next/backend/internal/crowdshipping"
	"github.com/geocore-next/backend/internal/crypto"
	"github.com/geocore-next/backend/internal/deals"
	"github.com/geocore-next/backend/internal/disputes"
	"github.com/geocore-next/backend/internal/engagement"
	"github.com/geocore-next/backend/internal/exchange"
	"github.com/geocore-next/backend/internal/experiments"
	"github.com/geocore-next/backend/internal/forex"
	"github.com/geocore-next/backend/internal/fraud"
	"github.com/geocore-next/backend/internal/gameday"
	"github.com/geocore-next/backend/internal/growth"
	"github.com/geocore-next/backend/internal/health"
	"github.com/geocore-next/backend/internal/images"
	"github.com/geocore-next/backend/internal/invite"
	"github.com/geocore-next/backend/internal/kyc"
	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/internal/livestream"
	"github.com/geocore-next/backend/internal/loyalty"
	"github.com/geocore-next/backend/internal/matching"
	"github.com/geocore-next/backend/internal/messaging"
	"github.com/geocore-next/backend/internal/moderation"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/ops"
	"github.com/geocore-next/backend/internal/order"
	"github.com/geocore-next/backend/internal/p2p"
	"github.com/geocore-next/backend/internal/payments"
	"github.com/geocore-next/backend/internal/plugins"
	"github.com/geocore-next/backend/internal/pricing"
	"github.com/geocore-next/backend/internal/protection"
	"github.com/geocore-next/backend/internal/recommendations"
	"github.com/geocore-next/backend/internal/redteam"
	"github.com/geocore-next/backend/internal/referral"
	"github.com/geocore-next/backend/internal/region"
	"github.com/geocore-next/backend/internal/reports"
	"github.com/geocore-next/backend/internal/requests"
	"github.com/geocore-next/backend/internal/reslab"
	"github.com/geocore-next/backend/internal/reverseauctions"
	"github.com/geocore-next/backend/internal/reviews"
	"github.com/geocore-next/backend/internal/search"
	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/internal/securityops"
	"github.com/geocore-next/backend/internal/settlement"
	"github.com/geocore-next/backend/internal/singularity"
	"github.com/geocore-next/backend/internal/slo"
	"github.com/geocore-next/backend/internal/stores"
	"github.com/geocore-next/backend/internal/stress"
	"github.com/geocore-next/backend/internal/subscriptions"
	"github.com/geocore-next/backend/internal/support"
	"github.com/geocore-next/backend/internal/tenant"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/internal/waitlist"
	"github.com/geocore-next/backend/internal/wallet"
	"github.com/geocore-next/backend/internal/warroom"
	"github.com/geocore-next/backend/internal/watchlist"
	"github.com/geocore-next/backend/internal/wholesale"
	pkganalytics "github.com/geocore-next/backend/pkg/analytics"
	"github.com/geocore-next/backend/pkg/database"
	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/i18n"
	"github.com/geocore-next/backend/pkg/jobs"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/kafka/consumers"
	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/remediation"
	"github.com/geocore-next/backend/pkg/server"
	"github.com/geocore-next/backend/pkg/sms"
	"github.com/geocore-next/backend/pkg/tracing"
	"github.com/google/uuid"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	// JWT keys
	// In production we require an RSA keypair for RS256 signing.
	if os.Getenv("APP_ENV") == "production" {
		priv := os.Getenv("JWT_PRIVATE_KEY")
		pub := os.Getenv("JWT_PUBLIC_KEY")
		if priv == "" || pub == "" {
			return errors.New("JWT_PRIVATE_KEY and JWT_PUBLIC_KEY environment variables are required in production")
		}
	}

	return nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		// .env file not found is okay in production
	}
	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	if err := middleware.InitSentry(
		os.Getenv("SENTRY_DSN"),
		getenv("APP_ENV", "development"),
		getenv("APP_VERSION", "dev"),
	); err != nil {
		logger.Warn("Sentry init failed; continuing without Sentry", zap.Error(err))
	} else {
		defer middleware.FlushSentry()
	}

	// Validate critical environment variables
	if err := validateEnv(); err != nil {
		logger.Fatal("Environment validation failed", zap.Error(err))
	}

	dbs, err := database.ConnectReadWrite()
	if err != nil {
		logger.Fatal("DB connect failed", zap.Error(err))
	}
	dbWrite := dbs.Write
	dbRead := dbs.Read
	if err := database.AutoMigrate(dbWrite); err != nil {
		logger.Fatal("AutoMigrate failed", zap.Error(err))
	}
	logger.Info("Database ready")
	database.StartPoolCollector(dbWrite)

	// Sprint 18: ensure full-text search indexes + compute category tree level/path.
	// Both are idempotent and safe to run on every startup.
	listings.ApplySearchIndexes(dbWrite)
	if err := listings.BackfillCategoryTree(dbWrite); err != nil {
		logger.Warn("category tree backfill failed", zap.Error(err))
	}

	// Auto-degradation: when DB latency exceeds 200ms for 3 consecutive checks,
	// enable degraded mode so read endpoints serve stale cache instead of hitting DB.
	middleware.AutoDegraded(func() time.Duration {
		sql, err := dbWrite.DB()
		if err != nil {
			return time.Second // force degraded
		}
		start := time.Now()
		if err := sql.Ping(); err != nil {
			return time.Second // force degraded
		}
		return time.Since(start)
	}, 200*time.Millisecond, 10*time.Second)

	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", getenv("REDIS_HOST", "localhost"), getenv("REDIS_PORT", "6379")),
		Password:     os.Getenv("REDIS_PASSWORD"),
		PoolSize:     100,             // match DB pool — 100 concurrent Redis ops
		MinIdleConns: 20,              // keep 20 warm connections
		DialTimeout:  5 * time.Second, // connect timeout
		ReadTimeout:  3 * time.Second, // command timeout (prevents goroutine leak on hung Redis)
		WriteTimeout: 3 * time.Second, // write timeout
		MaxRetries:   3,               // auto-retry on transient failures
		PoolTimeout:  4 * time.Second, // wait for pool connection
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Fatal("Redis connect failed", zap.Error(err))
	}
	logger.Info("Redis ready")

	// ── Redis memory + eviction monitoring ─────────────────────────────────
	tracing.StartRedisMonitor(rdb)
	tracing.StartGoroutineReporter()

	middleware.RevocationRDB = rdb

	ops.InitConfigStore(dbWrite, rdb)
	moderation.InitStore(dbWrite, rdb)

	// Billing meter — buffers usage events and flushes to DB every 60s
	billingMeter := billing.NewMeter(dbWrite)
	billing.GlobalMeter = billingMeter

	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.RequestID())
	r.Use(tracing.GinMiddleware())
	r.Use(middleware.PrometheusMiddleware())
	r.Use(middleware.SentryMiddleware())
	r.Use(gin.Recovery())
	r.Use(controlplane.KillSwitchMiddleware())
	// Tenant resolver — extracts tenant_id from X-API-Key or X-Tenant-ID header.
	// No-op for requests without those headers (backward-compatible).
	r.Use(tenant.NewResolver(dbWrite, tenant.NewQuotaEnforcer(rdb)).Middleware())
	r.Use(middleware.CSRF())

	// Security headers
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.ContentSecurityPolicy())

	corsConfig := cors.Config{
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-CSRF-Token", "X-Request-ID", "X-Idempotency-Key"},
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
	rl.OnReject = func(c *gin.Context, key, path string, limit int, window time.Duration, retryAfter int64) {
		var uid *uuid.UUID
		if userID := c.GetString("user_id"); userID != "" {
			if parsed, err := uuid.Parse(userID); err == nil {
				uid = &parsed
			}
		}
		security.LogEvent(dbWrite, c, uid, security.EventRateLimited, map[string]any{
			"key":         key,
			"path":        path,
			"limit":       limit,
			"window_sec":  int(window.Seconds()),
			"retry_after": retryAfter,
		})
	}
	r.Use(rl.Limit(100, time.Minute, "api"))

	// Sprint 22: Intrusion Detection + Emergency Mode
	alertSvc := security.AlertServiceFromEnv()
	ids := security.NewIDS(rdb, dbWrite)
	r.Use(ids.Middleware())
	r.Use(security.EmergencyMode())
	r.Use(security.FrozenUserMiddleware(dbWrite))
	go func() {
		for {
			ids.SuspiciousPatterns(context.Background(), 30*time.Minute)
			time.Sleep(15 * time.Minute)
		}
	}()

	// Global request timeout — all downstream DB/Redis/HTTP/Kafka calls
	// that respect context will fail-fast when the deadline expires.
	// Per-route overrides can give expensive endpoints more time.
	r.Use(middleware.Timeout(10 * time.Second))

	// Load shedding: reject non-critical requests when system is saturated.
	// Critical paths (wallet, orders, payments, webhooks) always pass through.
	dbPoolStats := func() (inUse, open int) {
		sql, err := dbWrite.DB()
		if err != nil {
			return 0, 1
		}
		s := sql.Stats()
		return s.InUse, s.MaxOpenConnections
	}
	r.Use(middleware.LoadShed(
		middleware.GoroutineSignal(),
		middleware.DBPoolSignal(dbPoolStats),
	))

	// Adaptive DB pool controller: shrink pool when DB is under stress,
	// restore when pressure subsides.
	{
		sqlDB, _ := dbWrite.DB()
		if sqlDB != nil {
			maxOpen := 50
			adaptivePool := database.NewAdaptivePoolController(sqlDB, maxOpen, maxOpen/2)
			adaptivePool.Start(10 * time.Second)
		}
	}

	// Start background job queue
	jobQueue := jobs.NewJobQueue(rdb)
	jobs.RegisterDefaultHandlers(jobQueue, &jobs.HandlerDependencies{
		DB:              dbWrite,
		SMSClient:       sms.NewTwilioClient(),
		AnalyticsClient: pkganalytics.NewPostHogClient(),
	})
	jobs.SetDefaultQueue(jobQueue)
	jobQueue.Start(4) // 4 workers
	defer jobQueue.Stop()

	// Start event bus and register all domain event consumers
	events.RegisterDefaultConsumers()

	// Kafka bridge + outbox worker.
	// No-op when KAFKA_BROKERS env is not set.
	kafka.SetRedis(rdb)
	kafka.Init()
	tracing.Init()
	events.RegisterKafkaBridge(dbWrite)
	kafkaCtx, kafkaCancel := context.WithCancel(context.Background())
	defer kafkaCancel()
	billingMeter.Start(kafkaCtx) // start after kafkaCtx is available for graceful shutdown
	kafka.NewOutboxWorker(dbWrite, 2*time.Second).Start(kafkaCtx)
	consumers.StartAll(kafkaCtx, dbWrite)
	defer kafka.Global().Close()

	// ── K8s Probes ───────────────────────────────────────────────────────────
	r.GET("/health/live", health.Liveness)
	r.GET("/health/ready", health.Readiness(map[string]func() bool{
		"db": func() bool {
			sql, err := dbWrite.DB()
			if err != nil {
				return false
			}
			return sql.Ping() == nil
		},
		"redis": func() bool {
			return rdb.Ping(context.Background()).Err() == nil
		},
	}))

	r.GET("/health", func(c *gin.Context) {
		remediation.CheckRedis(rdb)
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"time":        time.Now(),
			"remediation": remediation.GetStatus(),
		})
	})

	r.GET("/remediation/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, remediation.GetStatus())
	})

	// Deploy health endpoint — used by CI/CD post-deploy verification
	r.GET("/health/deploy", func(c *gin.Context) {
		// In production, these values are read from Prometheus metrics
		// via the CanaryMonitor. For now, return a healthy placeholder.
		health := remediation.CheckDeployHealth(0, 0, 0)
		if !health.Healthy {
			c.JSON(http.StatusServiceUnavailable, health)
			return
		}
		c.JSON(http.StatusOK, health)
	})

	// ── Region Routing (multi-region) ─────────────────────────────────────────
	regionStore := region.NewStore(rdb)
	regionRouter := region.NewRouter(regionStore)
	region.StartHealthWorker(kafkaCtx, regionStore, []region.RegionStatus{
		{Name: "us-east-1", BaseURL: getenv("REGION_US_URL", "http://localhost:8080")},
		{Name: "eu-west-1", BaseURL: getenv("REGION_EU_URL", "http://localhost:8080")},
	})
	r.Use(region.RegionMiddleware(regionRouter))
	r.Use(region.InjectKafkaContext())

	r.GET("/metrics", middleware.MetricsAuth(os.Getenv("METRICS_TOKEN")), func(c *gin.Context) {
		sqlDB, err := dbWrite.DB()
		if err == nil {
			metrics.ObserveDBConnections(sqlDB)
		}
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})

	if os.Getenv("APP_ENV") != "production" {
		r.GET("/debug/sentry-test", func(c *gin.Context) {
			panic("deliberate sentry test panic")
		})

		// ── Chaos Engine (probabilistic injection) ────────────────────────────
		chaosEngine := chaos.NewChaosEngine(rdb)
		chaos.SetEngine(chaosEngine)
		r.Use(chaos.ChaosMiddleware(chaosEngine))
		chaos.RegisterRoutes(r)

		// ── GameDay Scheduler (weekly resilience drills) ──────────────────────
		gdScheduler := gameday.NewScheduler()
		gdScheduler.Start(kafkaCtx)

		// ── SLO Engine (error budget tracking) ───────────────────────────────
		sloEngine := slo.NewEngine(slo.AllSLOs(), 30*time.Second)
		go sloEngine.Start(kafkaCtx)

		// ── Autonomous Control Loop (10s cycle) ──────────────────────────────
		decisionEngine := autonomy.NewDecisionEngine(sloEngine, chaosEngine)
		controlLoop := autonomy.NewControlLoop(decisionEngine, 10*time.Second)
		controlLoop.Start(kafkaCtx)

		// ── Singularity Control Plane (2min self-optimization cycle) ──────────
		singularityCfg := singularity.DefaultConfig()
		singularityCfg.Interval = 2 * time.Minute
		singularityPlane := singularity.New(singularityCfg)
		singularityPlane.Start(kafkaCtx)
	}

	// ── SLO Engine + Autonomy + Singularity run in production too ──────────────
	if os.Getenv("APP_ENV") == "production" {
		sloEngine := slo.NewEngine(slo.AllSLOs(), 60*time.Second)
		go sloEngine.Start(kafkaCtx)

		decisionEngine := autonomy.NewDecisionEngine(sloEngine, nil)
		controlLoop := autonomy.NewControlLoop(decisionEngine, 30*time.Second)
		controlLoop.Start(kafkaCtx)

		singularityCfg := singularity.DefaultConfig()
		singularityCfg.Interval = 5 * time.Minute
		singularityPlane := singularity.New(singularityCfg)
		singularityPlane.Start(kafkaCtx)
	}

	chatHub := chat.NewHub(rdb)
	go chatHub.Run()
	auctionHub := auctions.NewHub(rdb)
	go auctionHub.Run()
	go auctionHub.SubscribeRedis(context.Background())

	v1 := r.Group("/api/v1")
	// Sprint 24: mount fraud GlobalGuard BEFORE any protected route is registered.
	// It matches c.FullPath() internally, so no-op for unrelated routes.
	predictor := fraud.NewPredictor(dbWrite, rdb)
	v1.Use(fraud.GlobalGuard(predictor))
	// Sprint 26: attach non-custodial disclaimer header to every response.
	v1.Use(compliance.DisclaimerMiddleware())
	// Sprint 26: auto-append immutable audit rows for financial / dispute endpoints.
	v1.Use(compliance.AuditMiddleware(dbWrite))
	auth.RegisterRoutes(v1, dbWrite, rdb)
	users.RegisterRoutes(v1, dbWrite, rdb)
	cart.RegisterRoutes(v1, dbWrite, rdb)
	watchlist.RegisterRoutes(v1, dbWrite)
	listings.RegisterRoutes(v1, dbWrite, dbRead, rdb)
	dutchManager := auctions.RegisterRoutes(v1, dbWrite, rdb)
	go dutchManager.RestoreOnStartup()
	order.RegisterRoutes(v1, dbWrite, rdb)
	chat.RegisterRoutes(v1, dbWrite, rdb)
	payments.RegisterRoutes(v1, dbWrite, rdb)
	images.RegisterRoutes(v1, dbWrite, rdb)
	notifHub, notifSvc := notifications.RegisterRoutes(v1, dbWrite, rdb)
	admin.RegisterRoutes(v1, dbWrite, rdb, jobQueue)
	opsCronScheduler, opsAlertEngine := ops.RegisterRoutes(v1, dbWrite, rdb, jobQueue)
	opsCronScheduler.Start()
	opsAlertEngine.Start()
	defer opsCronScheduler.Stop()
	defer opsAlertEngine.Stop()
	analytics.RegisterRoutes(v1, dbRead)
	kyc.RegisterRoutes(v1, dbWrite)
	reviews.RegisterRoutes(v1, dbWrite)
	stores.RegisterRoutes(v1, dbWrite)
	wallet.RegisterRoutes(v1, dbWrite, rdb)
	forex.RegisterRoutes(v1, dbWrite)
	settlement.RegisterRoutes(v1, dbWrite)
	disputes.RegisterRoutes(v1, dbWrite)
	loyalty.RegisterRoutes(v1, dbWrite)
	referral.RegisterRoutes(v1, dbWrite)
	requests.RegisterRoutes(v1, dbWrite)
	reports.RegisterRoutes(v1, dbWrite)
	subscriptions.RegisterRoutes(v1, dbWrite)
	adminsettings.RegisterRoutes(v1, dbWrite)
	aichat.RegisterRoutes(v1)
	bnpl.RegisterRoutes(v1, dbWrite)
	crypto.RegisterRoutes(v1, dbWrite)
	livestream.RegisterRoutes(v1, dbWrite, rdb)
	crowdshipping.RegisterRoutes(v1, dbWrite, notifSvc)
	cancellation.RegisterRoutes(v1, dbWrite)
	protection.RegisterRoutes(v1, dbWrite)
	p2p.RegisterRoutes(v1, dbWrite)
	fraud.RegisterRoutes(v1, dbWrite)
	// Sprint 24: Fraud Prediction Engine — admin endpoints.
	// GlobalGuard middleware was already mounted at group creation (see above).
	fraud.RegisterPredictRoutes(v1, dbWrite, rdb)
	arpreview.RegisterRoutes(v1, dbWrite)
	blockchain.RegisterRoutes(v1, dbWrite)
	plugins.RegisterRoutes(v1, dbWrite)
	aiops.RegisterRoutes(v1, dbWrite, kafkaCtx)
	stress.RegisterRoutes(v1, dbWrite)
	reslab.RegisterRoutes(v1, dbWrite)
	warroom.RegisterRoutes(v1, dbWrite, kafkaCtx)

	// Sprint 19: Non-Custodial P2P Exchange Layer
	exchange.AutoMigrate(dbWrite)
	exchange.RegisterRoutes(v1, dbWrite, rdb, middleware.Auth())

	// Sprint 20: Private Invite & Referral Network
	invite.RegisterRoutes(v1, dbWrite, rdb)

	// Sprint 21: Waitlist + Hype Engine
	waitlist.RegisterRoutes(v1, dbWrite, rdb)

	// Sprint 22: Backup admin endpoints
	backup.RegisterRoutes(v1, dbWrite)

	// Sprint 23: Control Tower / Admin Intelligence
	controltower.RegisterRoutes(v1, dbWrite, rdb)
	// Wire IDS auto-block events into the control tower event stream.
	ids.OnBlock = func(ip, reason string) {
		controltower.Emit("ids_block", controltower.SevWarning,
			"IP auto-blocked: "+reason, "", ip)
	}

	// Sprint 23 (cont): Security Monitoring + Admin Observability
	securityops.RegisterRoutes(v1, dbWrite, rdb, ids)

	// Sprint 25: Red Team Simulation (admin-only, gated by ENABLE_REDTEAM)
	redteam.RegisterRoutes(v1, dbWrite, rdb, ids)

	// Sprint 26: Compliance & GDPR Layer
	compliance.RegisterRoutes(v1, dbWrite)
	spikeDetector := security.NewSpikeDetector(dbWrite, alertSvc)
	spikeCtx, spikeCancel := context.WithCancel(context.Background())
	defer spikeCancel()
	go spikeDetector.Run(spikeCtx, time.Minute)

	// Track C: AI-powered trust + pricing + matching
	pricing.InitSeedModel()
	pricing.RegisterRoutes(v1, dbWrite, rdb)
	matching.RegisterRoutes(v1, dbWrite, rdb)
	analytics.RegisterRouteMetricsRoutes(v1, dbRead)
	reverseauctions.RegisterRoutes(v1, dbWrite)
	search.RegisterRoutes(v1, dbRead, rdb)
	ads.RegisterRoutes(v1, dbWrite)
	chargebacks.RegisterRoutes(v1, dbWrite, rdb)
	addons.RegisterRoutes(v1, dbWrite, rdb)
	cms.RegisterRoutes(v1, dbWrite, rdb)
	cms.RegisterStaticFiles(r)
	deals.RegisterRoutes(v1, dbWrite)
	recommendations.RegisterRoutes(v1, dbWrite, rdb)
	engagement.RegisterRoutes(v1, dbWrite)
	growth.RegisterGrowthBrainRoutes(v1, dbWrite, rdb)
	messaging.RegisterRoutes(v1, dbWrite, rdb)
	experiments.RegisterRoutes(v1, dbWrite)
	support.RegisterRoutes(v1, dbWrite)

	// Country Layer Service — per-country rules (currency, tax, KYC limits, features)
	country.RegisterRoutes(v1, dbWrite, rdb)
	country.SeedCountryConfigs(dbWrite)

	// Wholesale/B2B Service — bulk listings, tiered pricing, seller verification
	wholesale.RegisterRoutes(v1, dbWrite)
	stopWalletReconcile := wallet.StartReconcileJob(dbWrite, 5*time.Minute)
	defer stopWalletReconcile()
	stopWaitlistRecalc := waitlist.StartRecalcJob(dbWrite, 5*time.Minute)
	defer stopWaitlistRecalc()

	// Sprint 22: Backup scheduler
	stopBackupJobs := backup.StartBackupJobs(dbWrite, backup.BackupConfigFromEnv(), alertSvc.ToAlertFunc())
	defer stopBackupJobs()

	auth.SetNotificationService(notifSvc)
	auctions.SetNotificationService(notifSvc)
	chat.SetNotificationService(notifSvc)
	disputes.StartSLAWorker(dbWrite, notifSvc)
	crowdshipping.StartOfferExpiryScheduler(kafkaCtx, dbWrite, notifSvc)
	go notifications.RetryFailedNotificationsWorker(dbWrite, notifSvc)

	// Sprint 18.2 — Saved search notifier (7-min tick, production-grade).
	// Uses NotifyFunc injection for dispatch; failure logging is done via
	// direct notifications.LogFailedNotification call (no cycle: notifications
	// does not import listings).
	go listings.NewSavedSearchNotifier(
		dbWrite, rdb,
		func(userID uuid.UUID, notifType, title, body string, data map[string]string) {
			notifSvc.Notify(notifications.NotifyInput{
				UserID: userID,
				Type:   notifType,
				Title:  title,
				Body:   body,
				Data:   data,
			})
		},
	).Start(kafkaCtx)

	v1.POST("/media/upload-url", middleware.Auth(), func(c *gin.Context) {
		images.NewHandler(dbWrite).GetUploadURL(c)
	})

	r.GET("/ws/notifications", func(c *gin.Context) { notifications.ServeWS(notifHub, c) })
	r.GET("/ws/auctions/:id", func(c *gin.Context) { auctions.ServeWS(auctionHub, c, dbWrite) })
	r.POST("/webhooks/stripe", payments.WebhookHandler(dbWrite))

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

	server.WaitForShutdown(srv, func(ctx context.Context) {
		logger.Info("Cleanup complete")
	})
}
