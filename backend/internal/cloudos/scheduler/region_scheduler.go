package scheduler

import (
	"log/slog"
)

// RegionScheduler manages multi-region traffic routing and failover.
type RegionScheduler struct {
	Primary       string
	Failover      string
	regionHealth  map[string]bool
}

// NewRegionScheduler creates a region scheduler with default regions.
func NewRegionScheduler() *RegionScheduler {
	return &RegionScheduler{
		Primary:  "us-east-1",
		Failover: "eu-west-1",
		regionHealth: map[string]bool{
			"us-east-1":      true,
			"eu-west-1":      true,
			"ap-southeast-1": true,
		},
	}
}

// ShouldFailover returns true if the primary region is unhealthy.
func (r *RegionScheduler) ShouldFailover() bool {
	return !r.regionHealth[r.Primary]
}

// ExecuteFailover switches traffic to the failover region.
func (r *RegionScheduler) ExecuteFailover() string {
	if r.ShouldFailover() {
		slog.Error("cloudos: REGION FAILOVER", "from", r.Primary, "to", r.Failover)
		return r.Failover
	}
	return r.Primary
}

// UpdateRegionHealth sets the health status for a region.
func (r *RegionScheduler) UpdateRegionHealth(region string, healthy bool) {
	r.regionHealth[region] = healthy
	if !healthy && region == r.Primary {
		slog.Error("cloudos: PRIMARY REGION DOWN", "region", region)
	}
	if healthy && region == r.Primary {
		slog.Info("cloudos: PRIMARY REGION RECOVERED", "region", region)
	}
}

// ActiveRegion returns the currently active region.
func (r *RegionScheduler) ActiveRegion() string {
	if r.regionHealth[r.Primary] {
		return r.Primary
	}
	return r.Failover
}
