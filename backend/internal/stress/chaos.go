package stress

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"sync"
	"time"
)

// ChaosEvent records a single chaos injection for the final report.
type ChaosEvent struct {
	Type       ChaosType `json:"type"`
	Action     string    `json:"action"`
	InjectedAt time.Time `json:"injected_at"`
}

// ChaosEngine injects controlled failures via HTTP calls to the running API.
// All injections are safe: no data corruption, no process termination.
type ChaosEngine struct {
	targetURL string
	client    *http.Client

	mu     sync.Mutex
	events []ChaosEvent
}

func newChaosEngine() *ChaosEngine {
	target := os.Getenv("STRESS_TARGET_URL")
	if target == "" {
		target = "http://localhost:8080"
	}
	return &ChaosEngine{
		targetURL: target,
		client:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *ChaosEngine) Events() []ChaosEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]ChaosEvent, len(c.events))
	copy(out, c.events)
	return out
}

func (c *ChaosEngine) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = nil
}

// InjectForScenario executes the chaos actions matching a scenario's chaos types.
func (c *ChaosEngine) InjectForScenario(ctx context.Context, chaosTypes []ChaosType) {
	for _, ct := range chaosTypes {
		switch ct {
		case ChaosDBPressure:
			c.injectDBPressure(ctx)
		case ChaosKafkaBreakdown:
			c.injectKafkaBreakdown(ctx)
		case ChaosRedisEviction:
			c.injectRedisEviction(ctx)
		case ChaosLatencySpike:
			c.injectLatencySpike(ctx)
		case ChaosCascading:
			c.injectRedisEviction(ctx)
			c.injectDBPressure(ctx)
			c.injectKafkaBreakdown(ctx)
		}
	}
}

// InjectRandom injects a random subset of failures (used for ad-hoc chaos).
func (c *ChaosEngine) InjectRandom(ctx context.Context) {
	if rand.Float64() < 0.3 {
		c.injectDBPressure(ctx)
	}
	if rand.Float64() < 0.2 {
		c.injectKafkaBreakdown(ctx)
	}
	if rand.Float64() < 0.15 {
		c.injectRedisEviction(ctx)
	}
}

// ── Injection strategies ─────────────────────────────────────────────────────

// injectDBPressure fires 60 concurrent read-heavy DB requests in parallel.
func (c *ChaosEngine) injectDBPressure(ctx context.Context) {
	c.record(ChaosDBPressure, "60 concurrent heavy DB reads (listings?limit=100)")
	var wg sync.WaitGroup
	for i := 0; i < 60; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
				c.targetURL+"/api/v1/listings?page=1&limit=100&sort=created_at", nil)
			if req == nil {
				return
			}
			resp, _ := c.client.Do(req)
			if resp != nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()
}

// injectKafkaBreakdown synthesizes a Kafka lag incident via the AIOps analyze endpoint.
func (c *ChaosEngine) injectKafkaBreakdown(ctx context.Context) {
	c.record(ChaosKafkaBreakdown, "synthetic Kafka lag incident (6000 lag vs 100 baseline)")
	c.triggerSyntheticIncident(ctx, "kafka", "kafka_consumer_lag", 6000, 100, "P0")
}

// injectRedisEviction synthesizes a Redis memory pressure incident.
func (c *ChaosEngine) injectRedisEviction(ctx context.Context) {
	c.record(ChaosRedisEviction, "synthetic Redis memory pressure incident (95% vs 50% baseline)")
	c.triggerSyntheticIncident(ctx, "redis", "redis_memory_used_bytes", 0.95, 0.50, "P1")
}

// injectLatencySpike synthesizes an API latency spike incident.
func (c *ChaosEngine) injectLatencySpike(ctx context.Context) {
	c.record(ChaosLatencySpike, "synthetic API latency spike (1200ms vs 300ms baseline)")
	c.triggerSyntheticIncident(ctx, "api", "api_latency_p95", 1200, 300, "P1")
}

// triggerSyntheticIncident posts a manual incident to the AIOps layer.
// This validates that AIOps detects and responds to the injected condition.
// The request is best-effort — auth failure is acceptable in stress context.
func (c *ChaosEngine) triggerSyntheticIncident(
	ctx context.Context,
	service, metric string,
	value, baseline float64,
	severity string,
) {
	payload := map[string]interface{}{
		"service":  service,
		"metric":   metric,
		"value":    value,
		"baseline": baseline,
		"severity": severity,
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.targetURL+"/api/v1/aiops/analyze", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, _ := c.client.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
}

func (c *ChaosEngine) record(t ChaosType, action string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, ChaosEvent{
		Type:       t,
		Action:     fmt.Sprintf("[CHAOS:%s] %s", t, action),
		InjectedAt: time.Now(),
	})
}
