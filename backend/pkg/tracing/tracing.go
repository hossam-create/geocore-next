// Package tracing provides OpenTelemetry initialization and Gin middleware
// for distributed tracing across the GeoCore platform.
//
// All no-op when OTEL_EXPORTER_OTLP_ENDPOINT is not set — zero overhead.
package tracing

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/geocore-next/backend/pkg/remediation"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

var (
	tp       *sdktrace.TracerProvider
	initOnce sync.Once
	enabled  bool
)

const (
	serviceName = "geocore-api"
	tracerName  = "github.com/geocore-next/backend"

	// TraceIDHeader is the HTTP header for distributed trace propagation.
	TraceIDHeader = "X-Trace-ID"
	// UserIDHeader is the HTTP header for user context propagation.
	UserIDHeader = "X-User-ID"
)

// Init initializes the OpenTelemetry SDK. No-op when OTEL_EXPORTER_OTLP_ENDPOINT is not set.
func Init() {
	initOnce.Do(func() {
		endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		if endpoint == "" {
			slog.Info("tracing: disabled (OTEL_EXPORTER_OTLP_ENDPOINT not set)")
			return
		}

		ctx := context.Background()

		exporter, err := otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err != nil {
			slog.Error("tracing: failed to create exporter", "error", err)
			return
		}

		res, err := sdkresource.New(ctx,
			sdkresource.WithAttributes(
				attribute.String("service.name", serviceName),
				attribute.String("service.version", os.Getenv("APP_VERSION")),
				attribute.String("deployment.environment", os.Getenv("APP_ENV")),
			),
		)
		if err != nil {
			slog.Error("tracing: failed to create resource", "error", err)
			return
		}

		tp = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exporter),
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.ParentBased(
				&financialRouteSampler{baseRate: 0.1}, // 100% financial, 10% other
			)),
		)

		otel.SetTracerProvider(tp)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))

		enabled = true
		slog.Info("tracing: enabled", "endpoint", endpoint)
	})
}

// Shutdown flushes pending spans. Call on graceful shutdown.
func Shutdown(ctx context.Context) {
	if tp != nil {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("tracing: shutdown error", "error", err)
		}
	}
}

// IsEnabled returns whether OTEL tracing is active.
func IsEnabled() bool {
	return enabled
}

// Tracer returns the global tracer for manual span creation.
func Tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// ── Gin Middleware ──────────────────────────────────────────────────────────

// GinMiddleware creates a Gin middleware that starts a span for each request,
// propagates trace context, and enriches spans with request metadata.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}

		// Extract propagated context from incoming headers
		ctx := otel.GetTextMapPropagator().Extract(
			c.Request.Context(),
			&headerCarrier{c.Request.Header},
		)

		// Start span
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		if spanName == "GET " {
			spanName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}

		ctx, span := Tracer().Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.route", c.FullPath()),
				attribute.String("http.user_agent", c.Request.UserAgent()),
			),
		)

		// Propagate trace ID to response header and gin context
		traceID := span.SpanContext().TraceID().String()
		c.Header(TraceIDHeader, traceID)
		c.Set("trace_id", traceID)

		// Attach user_id if available
		if uid, ok := c.Get("userID"); ok {
			span.SetAttributes(attribute.String("user.id", fmt.Sprintf("%v", uid)))
		}

		// Attach request_id
		if rid := c.GetString("request_id"); rid != "" {
			span.SetAttributes(attribute.String("request.id", rid))
		}

		// Replace request context with traced context
		c.Request = c.Request.WithContext(ctx)

		// Process request
		c.Next()

		// Record result
		status := c.Writer.Status()
		span.SetAttributes(attribute.Int("http.status_code", status))

		if status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(status))
		} else {
			span.SetStatus(codes.Ok, "")
		}

		span.End()
	}
}

// ── Kafka Trace Propagation ────────────────────────────────────────────────

// SpanFromContext returns the current span from a context.
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// TraceIDFromContext extracts the current trace ID as a string.
// Returns empty string if no active span.
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	return span.SpanContext().TraceID().String()
}

// ContextWithSpan creates a child span for a specific operation (e.g. Kafka publish).
func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if !enabled {
		return ctx, trace.SpanFromContext(ctx)
	}
	return Tracer().Start(ctx, name,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(attrs...),
	)
}

// ── Header Carrier (for OTEL propagation) ──────────────────────────────────

type headerCarrier struct {
	carrier http.Header
}

func (h *headerCarrier) Get(key string) string {
	return h.carrier.Get(key)
}

func (h *headerCarrier) Set(key, value string) {
	h.carrier.Set(key, value)
}

func (h *headerCarrier) Keys() []string {
	keys := make([]string, 0, len(h.carrier))
	for k := range h.carrier {
		keys = append(keys, k)
	}
	return keys
}

// ── Helper: Generate trace-aware event metadata ────────────────────────────

// EventTraceMeta returns metadata map with trace context for Kafka events.
func EventTraceMeta(ctx context.Context) map[string]interface{} {
	meta := map[string]interface{}{
		"source": "api-service",
	}
	if !enabled {
		return meta
	}
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		meta["trace_id"] = span.SpanContext().TraceID().String()
		meta["span_id"] = span.SpanContext().SpanID().String()
	}
	if rid, ok := ctx.Value("request_id").(string); ok && rid != "" {
		meta["trace_id"] = rid // fallback to request_id
	}
	return meta
}

// EnsureTraceID returns a trace ID string — either from the span or generates one.
func EnsureTraceID(ctx context.Context) string {
	if enabled {
		span := trace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			return span.SpanContext().TraceID().String()
		}
	}
	return uuid.New().String()
}

// InjectKafkaHeaders injects trace context into Kafka message headers.
func InjectKafkaHeaders(ctx context.Context, headers map[string]string) {
	if !enabled {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, &mapCarrier{m: headers})
}

type mapCarrier struct {
	m map[string]string
}

func (mc *mapCarrier) Get(key string) string {
	return mc.m[key]
}

func (mc *mapCarrier) Set(key, value string) {
	// Normalize header keys for Kafka
	normalized := strings.ReplaceAll(strings.ToLower(key), "-", "_")
	mc.m[normalized] = value
}

func (mc *mapCarrier) Keys() []string {
	keys := make([]string, 0, len(mc.m))
	for k := range mc.m {
		keys = append(keys, k)
	}
	return keys
}

// ── Adaptive Tracing: Financial Route Sampler ──────────────────────────────

// financialRouteSampler forces 100% trace sampling on financial routes
// (wallet, payments, escrow, webhooks) and applies baseRate to everything else.
type financialRouteSampler struct {
	baseRate float64 // e.g. 0.1 = 10% for non-financial routes
}

var financialRoutePrefixes = []string{
	"/api/v1/wallet",
	"/api/v1/payments",
	"/api/v1/escrow",
	"/api/v1/webhooks",
	"/admin/escrow",
	"/admin/wallet",
}

func (s *financialRouteSampler) ShouldSample(p sdktrace.SamplingParameters) sdktrace.SamplingResult {
	ts := trace.SpanContextFromContext(p.ParentContext).TraceState()
	// Check if any span attribute indicates a financial route
	for _, attr := range p.Attributes {
		if attr.Key == "http.route" {
			route := attr.Value.AsString()
			for _, prefix := range financialRoutePrefixes {
				if strings.HasPrefix(route, prefix) {
					return sdktrace.SamplingResult{
						Decision:   sdktrace.RecordAndSample,
						Tracestate: ts,
					}
				}
			}
		}
	}
	// Non-financial route — apply base sampling rate
	if sdktrace.TraceIDRatioBased(s.baseRate).ShouldSample(p).Decision == sdktrace.RecordAndSample {
		return sdktrace.SamplingResult{
			Decision:   sdktrace.RecordAndSample,
			Tracestate: ts,
		}
	}
	return sdktrace.SamplingResult{
		Decision:   sdktrace.Drop,
		Tracestate: ts,
	}
}

func (s *financialRouteSampler) Description() string {
	return fmt.Sprintf("financialRouteSampler{baseRate=%.2f,financial=1.0}", s.baseRate)
}

// ── Redis Memory Monitor + Risk Control ─────────────────────────────────────

const (
	// RedisMemoryRiskThreshold — when memory usage exceeds this fraction of
	// maxmemory, the system enters degraded mode to prevent silent collapse.
	RedisMemoryRiskThreshold = 0.85

	// RedisEvictionAlertPerMin — eviction rate above this triggers P1 alert.
	RedisEvictionAlertPerMin = 10
)

var redisDegraded bool

// IsRedisDegraded returns whether Redis risk control has activated degraded mode.
func IsRedisDegraded() bool { return redisDegraded }

// StartRedisMonitor starts a background goroutine that scrapes Redis INFO
// every 15s and updates Prometheus gauges for memory, eviction, and hit rate.
// When memory >= 85% of maxmemory or eviction rate > 10/min, it enables
// degraded mode (serve stale cache, skip non-critical DB reads) via the
// remediation package. Auto-recovers when conditions normalize.
func StartRedisMonitor(rdb *redis.Client) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		var lastEvicted int64
		var lastEvictTime time.Time
		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			info, err := rdb.Info(ctx, "memory", "stats", "keyspace").Result()
			cancel()
			if err != nil {
				continue
			}
			usedMem, maxMem, evicted, clients := parseRedisInfo(info)
			metrics.RedisMemoryUsedBytes.Set(float64(usedMem))
			metrics.RedisMemoryMaxBytes.Set(float64(maxMem))
			metrics.RedisConnectedClients.Set(float64(clients))
			if evicted > lastEvicted {
				metrics.RedisEvictedKeysTotal.Add(float64(evicted - lastEvicted))
			}

			// Keyspace hit rate
			hits, misses := parseRedisKeyspace(info)
			total := hits + misses
			if total > 0 {
				metrics.RedisHitRate.Set(float64(hits) / float64(total))
			}

			// ── Risk Control ─────────────────────────────────────────────────
			atRisk := false
			reason := ""
			if maxMem > 0 && float64(usedMem)/float64(maxMem) >= RedisMemoryRiskThreshold {
				atRisk = true
				reason = "memory_pressure"
			}
			if !lastEvictTime.IsZero() && evicted > lastEvicted {
				elapsedMin := time.Since(lastEvictTime).Minutes()
				if elapsedMin > 0 {
					evictPerMin := float64(evicted-lastEvicted) / elapsedMin
					if evictPerMin >= float64(RedisEvictionAlertPerMin) {
						atRisk = true
						reason = "eviction_spike"
					}
				}
			}
			lastEvicted = evicted
			lastEvictTime = time.Now()

			if atRisk && !redisDegraded {
				slog.Error("redis: RISK DETECTED — enabling degraded mode",
					"reason", reason,
					"memory_pct", fmt.Sprintf("%.1f%%", float64(usedMem)/float64(maxMem)*100))
				redisDegraded = true
				remediation.EnableDBReadOnly("redis_risk:" + reason)
			} else if !atRisk && redisDegraded {
				slog.Info("redis: risk cleared — restoring normal mode")
				redisDegraded = false
				remediation.DisableDBReadOnly()
			}
		}
	}()
}

func parseRedisInfo(info string) (usedMem, maxMem, evicted, clients int64) {
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "used_memory:") {
			fmt.Sscanf(line, "used_memory:%d", &usedMem)
		} else if strings.HasPrefix(line, "maxmemory:") {
			fmt.Sscanf(line, "maxmemory:%d", &maxMem)
		} else if strings.HasPrefix(line, "evicted_keys:") {
			fmt.Sscanf(line, "evicted_keys:%d", &evicted)
		} else if strings.HasPrefix(line, "connected_clients:") {
			fmt.Sscanf(line, "connected_clients:%d", &clients)
		}
	}
	return
}

func parseRedisKeyspace(info string) (hits, misses int64) {
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "keyspace_hits:") {
			fmt.Sscanf(line, "keyspace_hits:%d", &hits)
		} else if strings.HasPrefix(line, "keyspace_misses:") {
			fmt.Sscanf(line, "keyspace_misses:%d", &misses)
		}
	}
	return
}

// ── Goroutine Count Reporter ───────────────────────────────────────────────

// StartGoroutineReporter starts a background goroutine that reports the
// current goroutine count to Prometheus every 10 seconds.
func StartGoroutineReporter() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			metrics.SetGoroutineCount(float64(runtime.NumGoroutine()))
		}
	}()
}
