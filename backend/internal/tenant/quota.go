package tenant

import (
	"context"
	"fmt"
	"time"

	"github.com/geocore-next/backend/internal/billing"
	"github.com/redis/go-redis/v9"
)

// QuotaEnforcer validates per-tenant request quotas using Redis counters.
// Fails open (allows) when Redis is unavailable.
type QuotaEnforcer struct {
	rdb *redis.Client
}

// NewQuotaEnforcer creates a QuotaEnforcer backed by Redis.
// Pass nil to disable quota enforcement (single-tenant / dev mode).
func NewQuotaEnforcer(rdb *redis.Client) *QuotaEnforcer {
	return &QuotaEnforcer{rdb: rdb}
}

// Allow returns true if the tenant is within their daily request quota.
func (q *QuotaEnforcer) Allow(ctx context.Context, tenantID, planID string) bool {
	if q.rdb == nil || tenantID == "" {
		return true
	}
	plan := billing.Get(billing.PlanID(planID))
	if plan.MaxRequests == 0 {
		return true // unlimited plan
	}
	key := fmt.Sprintf("quota:daily:%s:%s", tenantID, today())
	count, err := q.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true // fail-open on Redis error
	}
	if count == 1 {
		q.rdb.Expire(ctx, key, 25*time.Hour)
	}
	return count <= plan.MaxRequests
}

// CanUseFeature returns true when the plan includes a given feature.
func CanUseFeature(planID, feature string) bool {
	plan := billing.Get(billing.PlanID(planID))
	switch feature {
	case "aiops":
		return plan.AIOpsEnabled
	case "chaos":
		return plan.ChaosEnabled
	case "reslab":
		return plan.ResLabEnabled
	case "multi_region":
		return plan.MultiRegion
	default:
		return true
	}
}

func today() string { return time.Now().UTC().Format("2006-01-02") }
