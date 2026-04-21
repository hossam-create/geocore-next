package metrics

import (
	"database/sql"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	once sync.Once

	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests by method, route, and status code.",
		},
		[]string{"method", "route", "status_code"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds by method and route.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	HTTPLatencyMs = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_latency_ms",
			Help:    "HTTP request latency in milliseconds by method and route.",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2000, 5000},
		},
		[]string{"method", "route"},
	)

	DBConnectionsOpen = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Number of open DB connections.",
		},
	)
	DBConnectionsIdle = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle DB connections.",
		},
	)
	DBConnectionsInUse = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_in_use",
			Help: "Number of in-use DB connections.",
		},
	)
	DBConnectionsWaitCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_wait_count",
			Help: "Total number of connections waited for.",
		},
	)

	WalletInvariantViolation = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "wallet_invariant_violation",
			Help: "Total number of wallet invariant violations detected.",
		},
	)

	ReconcileMismatch = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "reconcile_mismatch",
			Help: "Total number of reconciliation runs that detected a mismatch.",
		},
	)

	GuestOrdersTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "guest_orders_total",
			Help: "Total number of created guest orders.",
		},
	)

	ModerationBlocksTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "moderation_blocks_total",
			Help: "Total number of moderation block decisions.",
		},
	)

	DisputesSLABreachedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "disputes_sla_breached_total",
			Help: "Total number of disputes marked as SLA breached.",
		},
	)

	MatchingRequestsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "matching_requests_total",
			Help: "Total number of crowdshipping matching requests.",
		},
	)

	WalletOpsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wallet_ops_total",
			Help: "Total number of wallet operations by operation and status.",
		},
		[]string{"operation", "status"},
	)

	WalletErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "wallet_errors_total",
			Help: "Total number of wallet operation errors.",
		},
	)

	DBQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration",
			Help:    "Database query duration in seconds by operation.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	CacheHitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total cache hits by namespace.",
		},
		[]string{"namespace"},
	)

	CacheMissesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total cache misses by namespace.",
		},
		[]string{"namespace"},
	)

	CircuitBreakerOpenTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_open_total",
			Help: "Total number of times a circuit breaker transitioned to open state.",
		},
		[]string{"service"},
	)

	CircuitBreakerFailuresTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Total number of failed calls through circuit breakers by service.",
		},
		[]string{"service"},
	)

	DLQSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "dlq_size",
			Help: "Current number of jobs in the Dead Letter Queue.",
		},
	)

	// ── FinOps: Cost Intelligence Metrics ────────────────────────────────────
	// These enable "cost per X" analysis: divide AWS bill by these counters.

	BusinessEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "business_events_total",
			Help: "Total business events by type (order, payment, escrow, wallet). Used for cost-per-event analysis.",
		},
		[]string{"event_type"},
	)

	OrdersCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "Total orders created. Divide AWS monthly cost by this for cost-per-order.",
		},
	)

	PaymentsProcessedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "payments_processed_total",
			Help: "Total payments processed. Divide AWS monthly cost by this for cost-per-payment.",
		},
	)

	WalletTransactionsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "wallet_transactions_total",
			Help: "Total wallet transactions. Divide AWS monthly cost by this for cost-per-tx.",
		},
	)

	KafkaOutboxPending = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "kafka_outbox_pending",
			Help: "Number of pending events in the Kafka outbox table awaiting delivery.",
		},
	)

	KafkaEventsPublished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_events_published_total",
			Help: "Total Kafka events published by topic.",
		},
		[]string{"topic"},
	)

	KafkaEventsFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_events_failed_total",
			Help: "Total Kafka events that failed delivery by topic.",
		},
		[]string{"topic"},
	)

	// ── Fraud Engine Metrics ────────────────────────────────────────────────────

	FraudDecisionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fraud_decisions_total",
			Help: "Total fraud decisions by decision type and event type.",
		},
		[]string{"decision", "event_type"},
	)

	FraudScoreDistribution = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "fraud_score_distribution",
			Help:    "Distribution of fraud risk scores (0-100).",
			Buckets: []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		},
	)

	FraudThresholdFallbacks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fraud_threshold_fallbacks_total",
			Help: "Number of times fraud thresholds fell back to defaults (Redis unavailable).",
		},
		[]string{"threshold_key"},
	)

	// ── Kafka Consumer Lag ──────────────────────────────────────────────────────

	KafkaConsumerLag = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kafka_consumer_lag",
			Help: "Current consumer lag (messages behind) per topic and consumer group.",
		},
		[]string{"topic", "consumer_group"},
	)

	// ── Redis Memory & Eviction ────────────────────────────────────────────────

	RedisMemoryUsedBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_memory_used_bytes",
			Help: "Current Redis memory usage in bytes.",
		},
	)
	RedisMemoryMaxBytes = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_memory_max_bytes",
			Help: "Redis maxmemory limit in bytes (0 = no limit).",
		},
	)
	RedisEvictedKeysTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "redis_evicted_keys_total",
			Help: "Total number of Redis keys evicted due to maxmemory policy.",
		},
	)
	RedisConnectedClients = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_connected_clients",
			Help: "Number of connected Redis clients.",
		},
	)
	RedisHitRate = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_hit_rate",
			Help: "Redis keyspace hit rate (hits / (hits + misses)).",
		},
	)

	// ── Goroutine count ─────────────────────────────────────────────────────────

	GoroutineCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutine_count",
			Help: "Current number of goroutines.",
		},
	)
)

func Init() {
	once.Do(func() {
		prometheus.MustRegister(HTTPRequestsTotal)
		prometheus.MustRegister(HTTPRequestDuration)
		prometheus.MustRegister(HTTPLatencyMs)
		prometheus.MustRegister(DBConnectionsOpen)
		prometheus.MustRegister(DBConnectionsIdle)
		prometheus.MustRegister(DBConnectionsInUse)
		prometheus.MustRegister(DBConnectionsWaitCount)
		prometheus.MustRegister(WalletInvariantViolation)
		prometheus.MustRegister(ReconcileMismatch)
		prometheus.MustRegister(GuestOrdersTotal)
		prometheus.MustRegister(ModerationBlocksTotal)
		prometheus.MustRegister(DisputesSLABreachedTotal)
		prometheus.MustRegister(MatchingRequestsTotal)
		prometheus.MustRegister(WalletOpsTotal)
		prometheus.MustRegister(WalletErrorsTotal)
		prometheus.MustRegister(DBQueryDuration)
		prometheus.MustRegister(CacheHitsTotal)
		prometheus.MustRegister(CacheMissesTotal)
		prometheus.MustRegister(CircuitBreakerOpenTotal)
		prometheus.MustRegister(CircuitBreakerFailuresTotal)
		prometheus.MustRegister(DLQSize)
		prometheus.MustRegister(BusinessEventsTotal)
		prometheus.MustRegister(OrdersCreatedTotal)
		prometheus.MustRegister(PaymentsProcessedTotal)
		prometheus.MustRegister(WalletTransactionsTotal)
		prometheus.MustRegister(KafkaOutboxPending)
		prometheus.MustRegister(KafkaEventsPublished)
		prometheus.MustRegister(KafkaEventsFailed)
		prometheus.MustRegister(FraudDecisionsTotal)
		prometheus.MustRegister(FraudScoreDistribution)
		prometheus.MustRegister(FraudThresholdFallbacks)
		prometheus.MustRegister(KafkaConsumerLag)
		prometheus.MustRegister(RedisMemoryUsedBytes)
		prometheus.MustRegister(RedisMemoryMaxBytes)
		prometheus.MustRegister(RedisEvictedKeysTotal)
		prometheus.MustRegister(RedisConnectedClients)
		prometheus.MustRegister(RedisHitRate)
		prometheus.MustRegister(GoroutineCount)
	})
}

func ObserveHTTPRequest(method, route string, statusCode int, duration time.Duration) {
	Init()
	HTTPRequestsTotal.WithLabelValues(method, route, strconv.Itoa(statusCode)).Inc()
	HTTPRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
	HTTPLatencyMs.WithLabelValues(method, route).Observe(float64(duration.Milliseconds()))
}

func ObserveDBConnections(db *sql.DB) {
	if db == nil {
		return
	}
	Init()
	stats := db.Stats()
	DBConnectionsOpen.Set(float64(stats.OpenConnections))
	DBConnectionsIdle.Set(float64(stats.Idle))
	DBConnectionsInUse.Set(float64(stats.InUse))
	DBConnectionsWaitCount.Set(float64(stats.WaitCount))
}

func IncWalletInvariantViolation() {
	Init()
	WalletInvariantViolation.Inc()
}

func IncReconcileMismatch() {
	Init()
	ReconcileMismatch.Inc()
}

func IncGuestOrdersTotal() {
	Init()
	GuestOrdersTotal.Inc()
}

func IncModerationBlocksTotal() {
	Init()
	ModerationBlocksTotal.Inc()
}

func IncDisputesSLABreachedTotal() {
	Init()
	DisputesSLABreachedTotal.Inc()
}

func IncMatchingRequestsTotal() {
	Init()
	MatchingRequestsTotal.Inc()
}

func IncWalletOp(operation, status string) {
	Init()
	WalletOpsTotal.WithLabelValues(operation, status).Inc()
	if status == "error" {
		WalletErrorsTotal.Inc()
	}
}

func ObserveDBQueryDuration(operation string, duration time.Duration) {
	Init()
	DBQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func IncCacheHit(namespace string) {
	Init()
	CacheHitsTotal.WithLabelValues(namespace).Inc()
}

func IncCacheMiss(namespace string) {
	Init()
	CacheMissesTotal.WithLabelValues(namespace).Inc()
}

func IncCircuitBreakerOpen(service string) {
	Init()
	CircuitBreakerOpenTotal.WithLabelValues(service).Inc()
}

func IncCircuitBreakerFailure(service string) {
	Init()
	CircuitBreakerFailuresTotal.WithLabelValues(service).Inc()
}

// ── FinOps: Cost Intelligence Helpers ─────────────────────────────────────

// IncBusinessEvent increments the cost-per-event counter for a given event type.
// Use: metrics.IncBusinessEvent("order") → enables "cost per order" analysis.
func IncBusinessEvent(eventType string) {
	Init()
	BusinessEventsTotal.WithLabelValues(eventType).Inc()
}

func IncOrdersCreated() {
	Init()
	OrdersCreatedTotal.Inc()
	IncBusinessEvent("order")
}

func IncPaymentsProcessed() {
	Init()
	PaymentsProcessedTotal.Inc()
	IncBusinessEvent("payment")
}

func IncWalletTransaction() {
	Init()
	WalletTransactionsTotal.Inc()
	IncBusinessEvent("wallet_tx")
}

func SetKafkaOutboxPending(count float64) {
	Init()
	KafkaOutboxPending.Set(count)
}

func IncKafkaPublished(topic string) {
	Init()
	KafkaEventsPublished.WithLabelValues(topic).Inc()
}

func IncKafkaFailed(topic string) {
	Init()
	KafkaEventsFailed.WithLabelValues(topic).Inc()
}

// ── Fraud Engine Helpers ───────────────────────────────────────────────────

func IncFraudDecision(decision, eventType string) {
	Init()
	FraudDecisionsTotal.WithLabelValues(decision, eventType).Inc()
}

func ObserveFraudScore(score float64) {
	Init()
	FraudScoreDistribution.Observe(score)
}

func IncFraudThresholdFallback(key string) {
	Init()
	FraudThresholdFallbacks.WithLabelValues(key).Inc()
}

// ── Kafka Consumer Lag Helpers ─────────────────────────────────────────────

func SetKafkaConsumerLag(topic, groupID string, lag float64) {
	Init()
	KafkaConsumerLag.WithLabelValues(topic, groupID).Set(lag)
}

// ── Redis Monitoring Helpers ───────────────────────────────────────────────

// ObserveRedisInfo extracts key metrics from Redis INFO output and updates gauges.
// Call this periodically (every 15s) from a background goroutine.
func ObserveRedisInfo(usedMemory, maxMemory int64, evictedKeys int64, connectedClients int64, hits, misses int64) {
	Init()
	RedisMemoryUsedBytes.Set(float64(usedMemory))
	RedisMemoryMaxBytes.Set(float64(maxMemory))
	RedisConnectedClients.Set(float64(connectedClients))
	if evictedKeys > 0 {
		// Counter — add delta. For simplicity, we set the absolute value
		// since this is called from a periodic scraper.
	}
	total := hits + misses
	if total > 0 {
		RedisHitRate.Set(float64(hits) / float64(total))
	}
}

// SetRedisEvictedKeys sets the absolute count of evicted keys (called from scraper).
func SetRedisEvictedKeys(count float64) {
	Init()
	RedisEvictedKeysTotal.Add(count)
}

// ── Goroutine Count Helper ─────────────────────────────────────────────────

func SetGoroutineCount(count float64) {
	Init()
	GoroutineCount.Set(count)
}
