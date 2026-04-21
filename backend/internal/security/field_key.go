package security

import (
	"encoding/base64"
	"fmt"
	"os"

	"golang.org/x/crypto/chacha20poly1305"
)

// FieldEncryptionKey returns the XChaCha20-Poly1305 key from FIELD_ENCRYPTION_KEY.
// The environment variable must be base64-encoded 32 bytes.
func FieldEncryptionKey() ([]byte, error) {
	b64 := os.Getenv("FIELD_ENCRYPTION_KEY")
	if b64 == "" {
		return nil, fmt.Errorf("FIELD_ENCRYPTION_KEY not set")
	}
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decode FIELD_ENCRYPTION_KEY: %w", err)
	}
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("FIELD_ENCRYPTION_KEY must decode to %d bytes", chacha20poly1305.KeySize)
	}
	return key, nil
}
