package authz

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// APIKey represents a tenant authentication credential.
type APIKey struct {
	ID         string     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID   string     `gorm:"type:uuid;not null;index"                       json:"tenant_id"`
	Name       string     `gorm:"not null"                                       json:"name"`
	KeyHash    string     `gorm:"uniqueIndex;not null"                           json:"-"`
	KeyPrefix  string     `gorm:"not null"                                       json:"key_prefix"` // first 10 chars for display
	Role       string     `gorm:"not null;default:'dev'"                         json:"role"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateKeyResult wraps the persisted APIKey with the one-time raw key value.
type CreateKeyResult struct {
	APIKey
	RawKey string `json:"key"` // shown once — never stored
}

// CreateKey generates a new API key for a tenant.
// The raw key is returned in the result and is never stored.
func CreateKey(db *gorm.DB, tenantID, name, role string) (*CreateKeyResult, error) {
	raw, err := generateKey()
	if err != nil {
		return nil, err
	}
	k := &APIKey{
		TenantID:  tenantID,
		Name:      name,
		KeyHash:   hashKey(raw),
		KeyPrefix: raw[:10],
		Role:      role,
	}
	if err := db.Create(k).Error; err != nil {
		return nil, fmt.Errorf("create api key: %w", err)
	}
	return &CreateKeyResult{APIKey: *k, RawKey: raw}, nil
}

// ListKeys returns all non-revoked API keys for a tenant.
func ListKeys(db *gorm.DB, tenantID string) ([]APIKey, error) {
	var keys []APIKey
	err := db.Where("tenant_id = ? AND revoked_at IS NULL", tenantID).
		Order("created_at DESC").
		Find(&keys).Error
	return keys, err
}

// RevokeKey marks an API key as revoked (soft-delete).
func RevokeKey(db *gorm.DB, keyID, tenantID string) error {
	now := time.Now()
	result := db.Model(&APIKey{}).
		Where("id = ? AND tenant_id = ?", keyID, tenantID).
		Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("api key not found")
	}
	return nil
}

// LookupByHash resolves a tenant ID and role from a raw API key.
// Returns "", "" if the key is invalid, revoked, or the tenant is not active.
func LookupByHash(db *gorm.DB, keyHash string) (tenantID, role string) {
	type row struct {
		TenantID string
		Role     string
	}
	var r row
	db.Table("api_keys").
		Select("api_keys.role, tenants.id as tenant_id").
		Joins("JOIN tenants ON tenants.id = api_keys.tenant_id").
		Where("api_keys.key_hash = ? AND api_keys.revoked_at IS NULL AND tenants.status = 'active'", keyHash).
		Scan(&r)
	return r.TenantID, r.Role
}

func generateKey() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate key: %w", err)
	}
	return "gc_" + hex.EncodeToString(b), nil
}

func hashKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}
