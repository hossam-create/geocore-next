package tenant

import (
	"fmt"

	"gorm.io/gorm"
)

// Scope provides tenant-scoped namespacing across all infrastructure layers.
// A zero-value Scope (empty TenantID) is the single-tenant no-op mode.
type Scope struct {
	TenantID string
	Plan     string
}

// New builds a Scope from raw context values set by the resolver middleware.
func New(tenantID, plan string) Scope { return Scope{TenantID: tenantID, Plan: plan} }

// IsMultiTenant returns true when operating in multi-tenant mode.
func (s Scope) IsMultiTenant() bool { return s.TenantID != "" }

// DB scopes a GORM query to this tenant via row-level filtering.
// No-op in single-tenant mode.
func (s Scope) DB(db *gorm.DB) *gorm.DB {
	if !s.IsMultiTenant() {
		return db
	}
	return db.Where("tenant_id = ?", s.TenantID)
}

// RedisKey returns a tenant-namespaced Redis key: tenant:{id}:{key}
func (s Scope) RedisKey(key string) string {
	if !s.IsMultiTenant() {
		return key
	}
	return fmt.Sprintf("tenant:%s:%s", s.TenantID, key)
}

// KafkaTopic returns a tenant-namespaced Kafka topic: tenant.{id}.{base}
func (s Scope) KafkaTopic(base string) string {
	if !s.IsMultiTenant() {
		return base
	}
	return fmt.Sprintf("tenant.%s.%s", s.TenantID, base)
}

// MetricsLabel returns the Prometheus label value for this tenant.
func (s Scope) MetricsLabel() string {
	if !s.IsMultiTenant() {
		return "default"
	}
	return s.TenantID
}
