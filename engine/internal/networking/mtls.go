package networking

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// MTLSManager manages mTLS certificates for services.
type MTLSManager struct {
	caCert     *x509.Certificate
	caKey      *ecdsa.PrivateKey
	caCertPEM  []byte
	caKeyPEM   []byte
	certDir    string
	mu         sync.Mutex
}

// NewMTLSManager creates a new mTLS manager.
func NewMTLSManager(projectName string) (*MTLSManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	certDir := filepath.Join(homeDir, ".devbox", "certs", projectName)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return nil, fmt.Errorf("create cert directory: %w", err)
	}

	m := &MTLSManager{
		certDir: certDir,
	}

	// Try to load existing CA
	if err := m.loadCA(); err != nil {
		// Generate new CA
		if err := m.generateCA(); err != nil {
			return nil, fmt.Errorf("generate CA: %w", err)
		}
	}

	return m, nil
}

// generateCA creates a new root CA for the environment.
func (m *MTLSManager) generateCA() error {
	// Generate CA private key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("generate CA key: %w", err)
	}

	// Generate CA certificate
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("generate serial: %w", err)
	}

	caCert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"DevBoxOS"},
			CommonName:   "DevBoxOS Local CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // 1 year
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertBytes, err := x509.CreateCertificate(rand.Reader, caCert, caCert, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("create CA certificate: %w", err)
	}

	m.caCert = caCert
	m.caKey = caKey

	// Encode to PEM
	m.caCertPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caCertBytes,
	})

	caKeyBytes, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return fmt.Errorf("marshal CA key: %w", err)
	}
	m.caKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: caKeyBytes,
	})

	// Save to disk
	if err := os.WriteFile(filepath.Join(m.certDir, "ca.pem"), m.caCertPEM, 0600); err != nil {
		return fmt.Errorf("write CA cert: %w", err)
	}
	if err := os.WriteFile(filepath.Join(m.certDir, "ca-key.pem"), m.caKeyPEM, 0600); err != nil {
		return fmt.Errorf("write CA key: %w", err)
	}

	return nil
}

// loadCA loads an existing CA from disk.
func (m *MTLSManager) loadCA() error {
	certPath := filepath.Join(m.certDir, "ca.pem")
	keyPath := filepath.Join(m.certDir, "ca-key.pem")

	certBytes, err := os.ReadFile(certPath)
	if err != nil {
		return err
	}

	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}

	// Parse certificate
	block, _ := pem.Decode(certBytes)
	if block == nil {
		return fmt.Errorf("failed to decode CA certificate")
	}

	caCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA certificate: %w", err)
	}

	// Parse key
	block, _ = pem.Decode(keyBytes)
	if block == nil {
		return fmt.Errorf("failed to decode CA key")
	}

	caKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse CA key: %w", err)
	}

	m.caCert = caCert
	m.caKey = caKey
	m.caCertPEM = certBytes
	m.caKeyPEM = keyBytes

	return nil
}

// GenerateServiceCert generates a TLS certificate for a service.
func (m *MTLSManager) GenerateServiceCert(serviceName string, dnsNames []string) (tls.Certificate, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate private key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate key: %w", err)
	}

	// Generate certificate
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate serial: %w", err)
	}

	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"DevBoxOS"},
			CommonName:   serviceName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour), // 24h TTL
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:    dnsNames,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, m.caCert, &key.PublicKey, m.caKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	// Create TLS certificate
	tlsCert := tls.Certificate{
		Certificate: [][]byte{certBytes, m.caCert.Raw},
		PrivateKey:  key,
	}

	return tlsCert, nil
}

// CACertPEM returns the CA certificate in PEM format.
func (m *MTLSManager) CACertPEM() []byte {
	return m.caCertPEM
}

// Cleanup removes all certificates for this project.
func (m *MTLSManager) Cleanup() error {
	return os.RemoveAll(m.certDir)
}
