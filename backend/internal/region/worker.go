package region

import (
	"context"
	"log/slog"
	"time"
)

// StartHealthWorker runs a background goroutine that probes all regions
// every 5 seconds, caches the results in Redis, and logs failover events
// when a region transitions between healthy/unhealthy states.
func StartHealthWorker(ctx context.Context, store *Store, regions []RegionStatus) {
	ticker := time.NewTicker(5 * time.Second)

	// Track previous state for failover detection
	prevHealthy := make(map[string]bool)

	go func() {
		slog.Info("region: health worker started", "regions", len(regions))

		// Initial check immediately
		var initial []RegionStatus
		for _, r := range regions {
			checked := CheckHealth(r)
			initial = append(initial, checked)
			prevHealthy[checked.Name] = checked.Healthy
		}
		_ = store.SetStatus(ctx, initial)

		for {
			select {
			case <-ctx.Done():
				slog.Info("region: health worker stopped")
				return
			case <-ticker.C:
				var updated []RegionStatus
				for _, r := range regions {
					checked := CheckHealth(r)
					updated = append(updated, checked)

					// Detect state transitions
					wasHealthy, known := prevHealthy[checked.Name]
					if known {
						if wasHealthy && !checked.Healthy {
							slog.Error("region: FAILOVER — region went DOWN",
								"region", checked.Name,
								"latency_ms", checked.LatencyMs,
							)
						} else if !wasHealthy && checked.Healthy {
							slog.Info("region: RECOVERY — region back UP",
								"region", checked.Name,
								"latency_ms", checked.LatencyMs,
							)
						}
					}
					prevHealthy[checked.Name] = checked.Healthy
				}
				if err := store.SetStatus(ctx, updated); err != nil {
					slog.Warn("region: failed to cache status", "error", err)
				}
			}
		}
	}()
}
