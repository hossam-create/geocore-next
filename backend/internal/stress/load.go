package stress

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

// loadEndpoints are the backend API paths exercised during load generation.
// These are all read-only/idempotent to avoid side effects during testing.
var loadEndpoints = []string{
	"/health",
	"/api/v1/listings",
	"/api/v1/listings?page=1&limit=20",
	"/api/v1/listings?page=2&limit=20",
	"/api/v1/auctions",
	"/api/v1/search?q=laptop",
	"/api/v1/search?q=phone",
	"/api/v1/search?q=furniture",
}

// LoadGenerator fires concurrent HTTP requests against the target URL.
// Set STRESS_TARGET_URL to point at a non-production instance.
// Defaults to http://localhost:8080 (dev server).
type LoadGenerator struct {
	targetURL string
	client    *http.Client
	collector *MetricsCollector
}

func newLoadGenerator(c *MetricsCollector) *LoadGenerator {
	target := os.Getenv("STRESS_TARGET_URL")
	if target == "" {
		target = "http://localhost:8080"
	}
	return &LoadGenerator{
		targetURL: target,
		client:    &http.Client{Timeout: 15 * time.Second},
		collector: c,
	}
}

// Wave maintains `concurrency` concurrent virtual users firing requests for `duration`.
// The semaphore pattern ensures we never exceed `concurrency` parallel goroutines.
func (g *LoadGenerator) Wave(ctx context.Context, concurrency int, duration time.Duration) {
	const maxConcurrency = 500
	if concurrency > maxConcurrency {
		concurrency = maxConcurrency
	}
	deadline := time.Now().Add(duration)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := 0; time.Now().Before(deadline) && ctx.Err() == nil; i++ {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break
		}
		wg.Add(1)
		go func(idx int) {
			defer func() {
				<-sem
				wg.Done()
			}()
			endpoint := loadEndpoints[idx%len(loadEndpoints)]
			g.fire(ctx, endpoint)
		}(i)
	}
	wg.Wait()
}

func (g *LoadGenerator) fire(ctx context.Context, endpoint string) {
	start := time.Now()
	url := fmt.Sprintf("%s%s", g.targetURL, endpoint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		g.collector.record(requestSample{
			duration: time.Since(start), endpoint: endpoint, isError: true,
		})
		return
	}

	resp, err := g.client.Do(req)
	dur := time.Since(start)
	isError := err != nil
	statusCode := 0
	if resp != nil {
		statusCode = resp.StatusCode
		resp.Body.Close()
		if statusCode >= 500 {
			isError = true
		}
	}

	g.collector.record(requestSample{
		duration: dur, statusCode: statusCode, endpoint: endpoint, isError: isError,
	})
}
