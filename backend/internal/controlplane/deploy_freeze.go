package controlplane

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// DeployFreeze controls whether deployments are allowed.
// Used during incidents or maintenance windows.
type DeployFreeze struct {
	rdb *redis.Client
}

// NewDeployFreeze creates a deploy freeze controller backed by Redis.
func NewDeployFreeze(rdb *redis.Client) *DeployFreeze {
	return &DeployFreeze{rdb: rdb}
}

// Freeze blocks all deployments with an optional reason and duration.
func (d *DeployFreeze) Freeze(ctx context.Context, reason string, duration time.Duration) error {
	if d.rdb == nil {
		return nil
	}
	if err := d.rdb.Set(ctx, "deploy:freeze", "1", duration).Err(); err != nil {
		return err
	}
	if err := d.rdb.Set(ctx, "deploy:freeze:reason", reason, duration).Err(); err != nil {
		return err
	}
	slog.Error("controlplane: deployment FROZEN", "reason", reason, "duration", duration)
	return nil
}

// Unfreeze allows deployments to proceed.
func (d *DeployFreeze) Unfreeze(ctx context.Context) error {
	if d.rdb == nil {
		return nil
	}
	d.rdb.Del(ctx, "deploy:freeze", "deploy:freeze:reason")
	slog.Info("controlplane: deployment unfrozen")
	return nil
}

// IsDeploymentFrozen returns true if deployments are currently blocked.
func (d *DeployFreeze) IsDeploymentFrozen(ctx context.Context) bool {
	if d.rdb == nil {
		return false
	}
	val, err := d.rdb.Get(ctx, "deploy:freeze").Result()
	if err != nil {
		return false
	}
	return val == "1"
}

// FreezeReason returns the reason for the current deployment freeze.
func (d *DeployFreeze) FreezeReason(ctx context.Context) string {
	if d.rdb == nil {
		return ""
	}
	val, err := d.rdb.Get(ctx, "deploy:freeze:reason").Result()
	if err != nil {
		return ""
	}
	return val
}
