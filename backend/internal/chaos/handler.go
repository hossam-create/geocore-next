package chaos

import (
	"net/http"
	"strconv"
	"time"

	chaosstate "github.com/geocore-next/backend/pkg/chaos"
	"github.com/gin-gonic/gin"
)

var engineRef *ChaosEngine

// SetEngine stores the chaos engine reference for route handlers.
func SetEngine(e *ChaosEngine) {
	engineRef = e
}

// RegisterRoutes mounts chaos injection endpoints.
// MUST be gated by APP_ENV check — never expose in production.
func RegisterRoutes(r *gin.Engine) {
	g := r.Group("/chaos")

	// ── Redis ────────────────────────────────────────────────────────────────
	g.POST("/redis/down", func(c *gin.Context) {
		chaosstate.SetRedisDown(true)
		c.JSON(http.StatusOK, gin.H{"status": "redis forced down"})
	})
	g.POST("/redis/up", func(c *gin.Context) {
		chaosstate.SetRedisDown(false)
		c.JSON(http.StatusOK, gin.H{"status": "redis restored"})
	})

	// ── Kafka ────────────────────────────────────────────────────────────────
	g.POST("/kafka/down", func(c *gin.Context) {
		chaosstate.SetKafkaDown(true)
		c.JSON(http.StatusOK, gin.H{"status": "kafka forced down"})
	})
	g.POST("/kafka/up", func(c *gin.Context) {
		chaosstate.SetKafkaDown(false)
		c.JSON(http.StatusOK, gin.H{"status": "kafka restored"})
	})

	// ── DB Latency ───────────────────────────────────────────────────────────
	g.POST("/db/slow", func(c *gin.Context) {
		chaosstate.SetDBLatency(2 * time.Second)
		c.JSON(http.StatusOK, gin.H{"status": "db latency injected", "latency": "2s"})
	})
	g.POST("/db/slow/:ms", func(c *gin.Context) {
		ms, _ := strconv.Atoi(c.Param("ms"))
		if ms <= 0 {
			ms = 2000
		}
		chaosstate.SetDBLatency(time.Duration(ms) * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"status": "db latency injected", "latency_ms": ms})
	})
	g.POST("/db/normal", func(c *gin.Context) {
		chaosstate.SetDBLatency(0)
		c.JSON(http.StatusOK, gin.H{"status": "db restored"})
	})

	// ── Region ───────────────────────────────────────────────────────────────
	g.POST("/region/down", func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name query param required"})
			return
		}
		chaosstate.SetRegionDown(name, true)
		c.JSON(http.StatusOK, gin.H{"status": "region forced down", "region": name})
	})
	g.POST("/region/up", func(c *gin.Context) {
		name := c.Query("name")
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name query param required"})
			return
		}
		chaosstate.SetRegionDown(name, false)
		c.JSON(http.StatusOK, gin.H{"status": "region restored", "region": name})
	})

	// ── Engine Rates (probabilistic injection) ──────────────────────────────
	g.POST("/engine/set", func(c *gin.Context) {
		if engineRef == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "chaos engine not initialized"})
			return
		}
		key := c.Query("key")
		pct, _ := strconv.Atoi(c.Query("pct"))
		if key == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "key query param required"})
			return
		}
		if err := engineRef.SetRate(c.Request.Context(), key, pct); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "rate set", "key": key, "pct": pct})
	})
	g.GET("/engine/rates", func(c *gin.Context) {
		if engineRef == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "chaos engine not initialized"})
			return
		}
		c.JSON(http.StatusOK, engineRef.AllRates(c.Request.Context()))
	})
	g.POST("/engine/reset", func(c *gin.Context) {
		if engineRef == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "chaos engine not initialized"})
			return
		}
		_ = engineRef.ResetAll(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"status": "engine rates reset"})
	})

	// ── Reset All ────────────────────────────────────────────────────────────
	g.POST("/reset", func(c *gin.Context) {
		chaosstate.Reset()
		if engineRef != nil {
			_ = engineRef.ResetAll(c.Request.Context())
		}
		c.JSON(http.StatusOK, gin.H{"status": "all chaos reset"})
	})

	// ── Status ───────────────────────────────────────────────────────────────
	g.GET("/status", func(c *gin.Context) {
		status := gin.H{
			"redis_down":   chaosstate.IsRedisDown(),
			"kafka_down":   chaosstate.IsKafkaDown(),
			"db_latency":   chaosstate.DBLatency().String(),
			"down_regions": chaosstate.DownRegions(),
		}
		if engineRef != nil {
			status["engine_rates"] = engineRef.AllRates(c.Request.Context())
		}
		c.JSON(http.StatusOK, status)
	})
}
