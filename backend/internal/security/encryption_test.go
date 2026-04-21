package security

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncryptFieldRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	ct, err := EncryptField("hello", key)
	require.NoError(t, err)
	require.NotEmpty(t, ct)

	pt, err := DecryptField(ct, key)
	require.NoError(t, err)
	require.Equal(t, "hello", pt)
}

func TestFieldEncryptionKey_InvalidOrMissing(t *testing.T) {
	// Ensure we don't panic when unset; we should return a clear error.
	t.Setenv("FIELD_ENCRYPTION_KEY", "")
	_, err := FieldEncryptionKey()
	require.Error(t, err)
}

func TestFieldEncryptionKey_Valid(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(255 - i)
	}

	b64 := base64.StdEncoding.EncodeToString(key)
	t.Setenv("FIELD_ENCRYPTION_KEY", b64)

	decoded, err := FieldEncryptionKey()
	require.NoError(t, err)
	require.Equal(t, key, decoded)
}
