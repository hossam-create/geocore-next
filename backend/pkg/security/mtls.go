// Package security provides mTLS and service-to-service authentication.
// Phase 5 — Zero Trust Security Model.
package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
)

// MTLSConfig holds mTLS configuration for inter-service communication.
type MTLSConfig struct {
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
	CAFile   string `json:"ca_file"`
}

// LoadTLSConfig creates a tls.Config for mutual TLS.
func LoadTLSConfig(cfg MTLSConfig) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("security: load keypair: %w", err)
	}

	caCert, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("security: read CA: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("security: invalid CA cert")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ClientCAs:    caPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
