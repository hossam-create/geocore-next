// Package region provides multi-region routing and failover logic.
// Phase 2 — Multi-Region Active-Active Architecture.
package region

import (
	"context"
	"log/slog"
)

// Router provides latency-based region routing with sticky user affinity + automatic failover.
type Router struct {
	store *Store
}

// NewRouter creates a region router backed by the Redis store.
func NewRouter(store *Store) *Router {
	return &Router{store: store}
}

// Route selects the best region for a user request.
// Priority: 1) Sticky region (if healthy) → 2) Lowest-latency healthy → 3) Fallback.
func (r *Router) Route(ctx context.Context, userID string) RegionStatus {
	// 1. Try sticky routing — user pinned to a region
	if userID != "" {
		regionName, err := r.store.GetUserRegion(ctx, userID)
		if err == nil && regionName != "" {
			regions, _ := r.store.GetStatus(ctx)
			for _, rg := range regions {
				if rg.Name == regionName && rg.Healthy {
					return rg
				}
			}
			slog.Warn("region: sticky region unhealthy — re-routing", "user_id", userID, "region", regionName)
		}
	}

	// 2. Pick best healthy region by latency
	best := r.pickBest(ctx)
	if best.Name == "" {
		slog.Error("region: no healthy regions available")
		return RegionStatus{Name: "fallback", Healthy: true}
	}

	// 3. Persist sticky routing for this user
	if userID != "" {
		_ = r.store.SetUserRegion(ctx, userID, best.Name)
	}

	return best
}

// pickBest returns the lowest-latency healthy region.
func (r *Router) pickBest(ctx context.Context) RegionStatus {
	regions, err := r.store.GetStatus(ctx)
	if err != nil || len(regions) == 0 {
		return RegionStatus{}
	}

	var best RegionStatus
	for _, rg := range regions {
		if !rg.Healthy {
			continue
		}
		if best.Name == "" || rg.LatencyMs < best.LatencyMs {
			best = rg
		}
	}
	return best
}
