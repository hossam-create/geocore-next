// Package jwtkeys manages RSA key pairs for RS256 JWT signing.
// Keys are loaded from environment variables JWT_PRIVATE_KEY and JWT_PUBLIC_KEY
// (PEM-encoded). In development, a transient in-memory key pair is generated
// automatically when neither env var is set.
package jwtkeys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log/slog"
	"os"
	"sync"
)

var (
	once       sync.Once
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
)

// Private returns the RSA private key used to sign access tokens.
func Private() *rsa.PrivateKey {
	once.Do(load)
	return privateKey
}

// Public returns the RSA public key used to verify access tokens.
func Public() *rsa.PublicKey {
	once.Do(load)
	return publicKey
}

func load() {
	privPEM := os.Getenv("JWT_PRIVATE_KEY")
	pubPEM := os.Getenv("JWT_PUBLIC_KEY")

	if privPEM != "" && pubPEM != "" {
		loadFromEnv(privPEM, pubPEM)
		slog.Info("jwtkeys: loaded RSA key pair from environment")
		return
	}

	// Development fallback — generate a transient key pair
	slog.Warn("jwtkeys: JWT_PRIVATE_KEY/JWT_PUBLIC_KEY not set — generating ephemeral RSA-2048 key pair (UNSAFE for production)")
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("jwtkeys: failed to generate RSA key pair: " + err.Error())
	}
	privateKey = priv
	publicKey = &priv.PublicKey
}

func loadFromEnv(privPEM, pubPEM string) {
	// Private key
	privBlock, _ := pem.Decode([]byte(privPEM))
	if privBlock == nil {
		panic("jwtkeys: JWT_PRIVATE_KEY is not valid PEM")
	}
	priv, err := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	if err != nil {
		// Try PKCS1
		privRSA, err2 := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
		if err2 != nil {
			panic("jwtkeys: cannot parse JWT_PRIVATE_KEY (tried PKCS8 and PKCS1): " + err.Error())
		}
		privateKey = privRSA
	} else {
		rsaKey, ok := priv.(*rsa.PrivateKey)
		if !ok {
			panic("jwtkeys: JWT_PRIVATE_KEY is not an RSA key")
		}
		privateKey = rsaKey
	}

	// Public key
	pubBlock, _ := pem.Decode([]byte(pubPEM))
	if pubBlock == nil {
		panic("jwtkeys: JWT_PUBLIC_KEY is not valid PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		panic("jwtkeys: cannot parse JWT_PUBLIC_KEY: " + err.Error())
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		panic("jwtkeys: JWT_PUBLIC_KEY is not an RSA key")
	}
	publicKey = rsaPub
}
