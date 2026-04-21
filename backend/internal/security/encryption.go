package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	argon2Time    uint32 = 1
	argon2Memory  uint32 = 64 * 1024
	argon2Threads uint8  = 4
	argon2KeyLen  uint32 = 32
	argon2SaltLen        = 16
)

// EncryptField encrypts sensitive values for at-rest storage using XChaCha20-Poly1305.
func EncryptField(plaintext string, key []byte) (string, error) {
	if len(key) != chacha20poly1305.KeySize {
		return "", fmt.Errorf("invalid key length: expected %d bytes", chacha20poly1305.KeySize)
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptField decrypts values encrypted by EncryptField.
func DecryptField(ciphertextB64 string, key []byte) (string, error) {
	if len(key) != chacha20poly1305.KeySize {
		return "", fmt.Errorf("invalid key length: expected %d bytes", chacha20poly1305.KeySize)
	}
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < chacha20poly1305.NonceSizeX {
		return "", errors.New("ciphertext too short")
	}

	nonce := ciphertext[:chacha20poly1305.NonceSizeX]
	enc := ciphertext[chacha20poly1305.NonceSizeX:]
	plaintext, err := aead.Open(nil, nonce, enc, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// HashPassword hashes plaintext passwords with Argon2id.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Time,
		argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword verifies argon2id hashes and keeps bcrypt compatibility for legacy users.
func VerifyPassword(storedHash, password string) bool {
	if strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$") || strings.HasPrefix(storedHash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil
	}

	timeCost, memoryCost, parallelism, salt, expectedHash, err := decodeArgon2Hash(storedHash)
	if err != nil {
		return false
	}

	actualHash := argon2.IDKey([]byte(password), salt, timeCost, memoryCost, uint8(parallelism), uint32(len(expectedHash)))
	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}

func decodeArgon2Hash(encoded string) (timeCost uint32, memoryCost uint32, parallelism uint32, salt []byte, hash []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		err = errors.New("invalid argon2 hash format")
		return
	}
	if parts[1] != "argon2id" {
		err = errors.New("unsupported hash algorithm")
		return
	}

	if _, scanErr := fmt.Sscanf(parts[2], "v=%d", new(int)); scanErr != nil {
		err = scanErr
		return
	}

	var mem, timeV, par uint32
	if _, scanErr := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &timeV, &par); scanErr != nil {
		err = scanErr
		return
	}
	memoryCost = mem
	timeCost = timeV
	parallelism = par

	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return
	}
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	return
}
