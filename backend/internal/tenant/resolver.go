package tenant

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/authz"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	CtxTenantID   = "tenant_id"
	CtxTenantPlan = "tenant_plan"
	CtxTenantRole = "tenant_role"
)

// Resolver is a Gin middleware that extracts tenant identity from incoming requests.
// It is backward-compatible: requests without X-API-Key proceed without tenant context.
type Resolver struct {
	db    *gorm.DB
	quota *QuotaEnforcer
}

// NewResolver creates a Resolver with optional quota enforcement.
func NewResolver(db *gorm.DB, quota *QuotaEnforcer) *Resolver {
	return &Resolver{db: db, quota: quota}
}

// Middleware returns the Gin handler that resolves tenant identity and enforces quotas.
func (r *Resolver) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, planID, role := "", "starter", "owner"

		// Strategy 1: X-API-Key header (primary SaaS auth path)
		if raw := c.GetHeader("X-API-Key"); raw != "" && strings.HasPrefix(raw, "gc_") {
			hash := hashRaw(raw)
			tid, rl := authz.LookupByHash(r.db, hash)
			if tid != "" {
				tenantID = tid
				role = rl
				// Resolve plan from tenant record
				var t Tenant
				if err := r.db.Select("plan").First(&t, "id = ?", tid).Error; err == nil {
					planID = t.Plan
				}
				// Update last_used_at asynchronously
				go r.db.Exec("UPDATE api_keys SET last_used_at = ? WHERE key_hash = ?", time.Now(), hash)
			}
		}

		// Strategy 2: X-Tenant-ID header (internal / dev mode)
		if tenantID == "" {
			if id := c.GetHeader("X-Tenant-ID"); id != "" {
				tenantID = id
				if p := c.GetHeader("X-Tenant-Plan"); p != "" {
					planID = p
				}
			}
		}

		if tenantID != "" {
			c.Set(CtxTenantID, tenantID)
			c.Set(CtxTenantPlan, planID)
			c.Set(CtxTenantRole, role)

			// Quota enforcement
			if r.quota != nil && !r.quota.Allow(c.Request.Context(), tenantID, planID) {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error":       "daily request quota exceeded for your plan",
					"plan":        planID,
					"upgrade_url": "https://geocore.app/pricing",
				})
				return
			}

			// Record usage (fire-and-forget)
			go RecordRequest(tenantID, 1)
		}

		c.Next()
	}
}

// GetTenantID extracts the resolved tenant ID from a Gin context (empty = single-tenant mode).
func GetTenantID(c *gin.Context) string { return c.GetString(CtxTenantID) }

// GetTenantPlan extracts the tenant's plan from a Gin context.
func GetTenantPlan(c *gin.Context) string { return c.GetString(CtxTenantPlan) }

// GetScope builds an isolation Scope from the current Gin context.
func GetScope(c *gin.Context) Scope {
	return New(GetTenantID(c), GetTenantPlan(c))
}

func hashRaw(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}
