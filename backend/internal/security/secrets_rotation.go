package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// VaultClient is an interface for secret storage backends.
// Production: HashiCorp Vault, AWS Secrets Manager, etc.
type VaultClient interface {
	Put(ctx context.Context, key, value string) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

// SecretManager handles secret rotation with a vault backend and Redis cache.
type SecretManager struct {
	vault VaultClient
	rdb   *redis.Client
}

// NewSecretManager creates a secret manager with the given vault and Redis.
func NewSecretManager(vault VaultClient, rdb *redis.Client) *SecretManager {
	return &SecretManager{vault: vault, rdb: rdb}
}

// Rotate generates a new secret, stores it in the vault, and updates the Redis cache.
func (s *SecretManager) Rotate(ctx context.Context, key string) error {
	newSecret, err := generateSecureKey()
	if err != nil {
		return err
	}

	if err := s.vault.Put(ctx, key, newSecret); err != nil {
		return err
	}

	// Update Redis cache so services pick up the new value immediately
	if s.rdb != nil {
		if err := s.rdb.Set(ctx, "secret:"+key, newSecret, 0).Err(); err != nil {
			slog.Warn("secrets: failed to update Redis cache", "key", key, "error", err)
		}
	}

	slog.Info("secrets: rotated", "key", key)
	return nil
}

// Get retrieves a secret, preferring Redis cache then falling back to vault.
func (s *SecretManager) Get(ctx context.Context, key string) (string, error) {
	// Try Redis cache first
	if s.rdb != nil {
		val, err := s.rdb.Get(ctx, "secret:"+key).Result()
		if err == nil {
			return val, nil
		}
	}

	// Fall back to vault
	return s.vault.Get(ctx, key)
}

// InMemoryVault is a development VaultClient that stores secrets in memory.
type InMemoryVault struct {
	secrets map[string]string
}

// NewInMemoryVault creates an in-memory vault for development/testing.
func NewInMemoryVault() *InMemoryVault {
	return &InMemoryVault{secrets: make(map[string]string)}
}

func (v *InMemoryVault) Put(_ context.Context, key, value string) error {
	v.secrets[key] = value
	return nil
}

func (v *InMemoryVault) Get(_ context.Context, key string) (string, error) {
	return v.secrets[key], nil
}

func (v *InMemoryVault) Delete(_ context.Context, key string) error {
	delete(v.secrets, key)
	return nil
}

func generateSecureKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
