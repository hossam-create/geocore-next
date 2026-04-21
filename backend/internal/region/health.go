package region

import (
	"net/http"
	"time"

	"github.com/geocore-next/backend/pkg/chaos"
)

// RegionStatus holds the current health and latency of a region.
type RegionStatus struct {
	Name      string `json:"name"`
	BaseURL   string `json:"base_url"`
	Healthy   bool   `json:"healthy"`
	LatencyMs int64  `json:"latency_ms"`
}

// CheckHealth probes a region's /health/ready endpoint.
// Respects chaos state: if a region is marked down via chaos, returns unhealthy immediately.
func CheckHealth(r RegionStatus) RegionStatus {
	// Chaos hook: simulate region down
	if chaos.IsRegionDown(r.Name) {
		r.Healthy = false
		return r
	}

	start := time.Now()

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(r.BaseURL + "/health/ready")
	if err != nil || resp.StatusCode != 200 {
		r.Healthy = false
		if resp != nil {
			resp.Body.Close()
		}
		return r
	}
	resp.Body.Close()

	r.Healthy = true
	r.LatencyMs = time.Since(start).Milliseconds()
	return r
}
