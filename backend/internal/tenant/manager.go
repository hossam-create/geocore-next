package tenant

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Tenant is the top-level SaaS tenant record.
type Tenant struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name      string    `gorm:"not null"                                       json:"name"`
	Slug      string    `gorm:"uniqueIndex;not null"                           json:"slug"`
	Plan      string    `gorm:"not null;default:'starter'"                     json:"plan"`
	Status    string    `gorm:"not null;default:'active'"                      json:"status"`
	Email     string    `gorm:"not null"                                       json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

// Manager handles tenant lifecycle operations.
type Manager struct{ db *gorm.DB }

// NewManager constructs a Manager with the given DB connection.
func NewManager(db *gorm.DB) *Manager { return &Manager{db: db} }

// Create provisions a new tenant with the given plan.
func (m *Manager) Create(name, email, planID string) (*Tenant, error) {
	slug := slugRe.ReplaceAllString(strings.ToLower(name), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) < 3 {
		return nil, errors.New("tenant name too short — minimum 3 characters after normalisation")
	}
	t := &Tenant{Name: name, Slug: slug, Plan: planID, Status: "active", Email: email}
	if err := m.db.Create(t).Error; err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return t, nil
}

// Get retrieves a tenant by UUID.
func (m *Manager) Get(id string) (*Tenant, error) {
	var t Tenant
	if err := m.db.First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// GetBySlug retrieves a tenant by URL-friendly slug.
func (m *Manager) GetBySlug(slug string) (*Tenant, error) {
	var t Tenant
	if err := m.db.First(&t, "slug = ?", slug).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

// List returns tenants with basic pagination.
func (m *Manager) List(limit, offset int) ([]Tenant, int64, error) {
	var tenants []Tenant
	var count int64
	m.db.Model(&Tenant{}).Count(&count)
	if err := m.db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&tenants).Error; err != nil {
		return nil, 0, err
	}
	return tenants, count, nil
}

// Suspend disables a tenant's access without deleting their data.
func (m *Manager) Suspend(id string) error {
	return m.db.Model(&Tenant{}).Where("id = ?", id).Update("status", "suspended").Error
}

// UpdatePlan changes the subscription tier for a tenant.
func (m *Manager) UpdatePlan(id, planID string) error {
	return m.db.Model(&Tenant{}).Where("id = ?", id).Update("plan", planID).Error
}
