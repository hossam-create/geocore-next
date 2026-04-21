// Package security provides zero-trust security primitives.
// Phase 5 — Zero Trust Security Model.
package security

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"log/slog"
	"time"
)

// ServiceIdentity represents a verified service in the mesh.
type ServiceIdentity struct {
	ServiceID   string    `json:"service_id"`
	PublicKey   string    `json:"public_key"`
	Roles       []string  `json:"roles"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// IdentityRegistry manages service identities and verification.
type IdentityRegistry struct {
	identities map[string]*ServiceIdentity
	keys       map[string]ed25519.PublicKey
}

// NewIdentityRegistry creates a new service identity registry.
func NewIdentityRegistry() *IdentityRegistry {
	return &IdentityRegistry{
		identities: make(map[string]*ServiceIdentity),
		keys:       make(map[string]ed25519.PublicKey),
	}
}

// RegisterService creates a new service identity with ed25519 keypair.
func (r *IdentityRegistry) RegisterService(serviceID string, roles []string) (privateKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}

	id := &ServiceIdentity{
		ServiceID: serviceID,
		PublicKey: base64.StdEncoding.EncodeToString(pub),
		Roles:     roles,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	r.identities[serviceID] = id
	r.keys[serviceID] = pub

	slog.Info("security: registered service identity", "service_id", serviceID, "roles", roles)
	return base64.StdEncoding.EncodeToString(priv), nil
}

// VerifyService checks if a service identity is valid and not expired.
func (r *IdentityRegistry) VerifyService(ctx context.Context, serviceID string) (*ServiceIdentity, bool) {
	id, ok := r.identities[serviceID]
	if !ok {
		return nil, false
	}
	if time.Now().After(id.ExpiresAt) {
		slog.Warn("security: expired service identity", "service_id", serviceID)
		return nil, false
	}
	return id, true
}

// HasRole checks if a service has a specific role.
func (r *IdentityRegistry) HasRole(serviceID, role string) bool {
	id, ok := r.identities[serviceID]
	if !ok {
		return false
	}
	for _, r := range id.Roles {
		if r == role {
			return true
		}
	}
	return false
}
