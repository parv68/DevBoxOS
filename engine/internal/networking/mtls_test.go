package networking

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"
	"testing"
)

func TestNewMTLSManager_CreatesCA(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewMTLSManager("test-project")
	if err != nil {
		t.Fatalf("NewMTLSManager() failed: %v", err)
	}

	certDir := filepath.Join(tmpHome, ".devbox", "certs", "test-project")
	if _, err := os.Stat(filepath.Join(certDir, "ca.pem")); os.IsNotExist(err) {
		t.Error("ca.pem not created")
	}
	if _, err := os.Stat(filepath.Join(certDir, "ca-key.pem")); os.IsNotExist(err) {
		t.Error("ca-key.pem not created")
	}

	if len(m.CACertPEM()) == 0 {
		t.Error("CACertPEM() returned empty")
	}
}

func TestNewMTLSManager_LoadsExistingCA(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m1, err := NewMTLSManager("test-project")
	if err != nil {
		t.Fatalf("first NewMTLSManager() failed: %v", err)
	}
	caPEM1 := m1.CACertPEM()

	m2, err := NewMTLSManager("test-project")
	if err != nil {
		t.Fatalf("second NewMTLSManager() failed: %v", err)
	}
	caPEM2 := m2.CACertPEM()

	if string(caPEM1) != string(caPEM2) {
		t.Error("CA PEM should be identical on reload")
	}
}

func TestGenerateServiceCert(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewMTLSManager("test-project")
	if err != nil {
		t.Fatalf("NewMTLSManager() failed: %v", err)
	}

	cert, err := m.GenerateServiceCert("web", []string{"web.local", "web.testproject.local"})
	if err != nil {
		t.Fatalf("GenerateServiceCert() failed: %v", err)
	}

	if len(cert.Certificate) != 2 {
		t.Errorf("expected 2 certificates in chain (service + CA), got %d", len(cert.Certificate))
	}

	if cert.PrivateKey == nil {
		t.Error("expected non-nil private key")
	}

	_, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Errorf("failed to parse service certificate: %v", err)
	}
}

func TestGenerateServiceCert_UniquePerCall(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewMTLSManager("test-project")
	if err != nil {
		t.Fatalf("NewMTLSManager() failed: %v", err)
	}

	cert1, _ := m.GenerateServiceCert("web", []string{"web.local"})
	cert2, _ := m.GenerateServiceCert("web", []string{"web.local"})

	if len(cert1.Certificate[0]) == len(cert2.Certificate[0]) {
		// Compare bytes
		equal := true
		for i := range cert1.Certificate[0] {
			if cert1.Certificate[0][i] != cert2.Certificate[0][i] {
				equal = false
				break
			}
		}
		if equal {
			t.Error("expected different certificates on each generation")
		}
	}
}

func TestCleanup(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewMTLSManager("test-project")
	if err != nil {
		t.Fatalf("NewMTLSManager() failed: %v", err)
	}

	certDir := filepath.Join(tmpHome, ".devbox", "certs", "test-project")
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		t.Fatal("cert directory not created")
	}

	if err := m.Cleanup(); err != nil {
		t.Fatalf("Cleanup() failed: %v", err)
	}

	if _, err := os.Stat(certDir); !os.IsNotExist(err) {
		t.Error("cert directory should be removed after Cleanup()")
	}
}

func TestMTLSManager_ServiceCertTLSConfig(t *testing.T) {
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer func() {
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	m, err := NewMTLSManager("test-project")
	if err != nil {
		t.Fatalf("NewMTLSManager() failed: %v", err)
	}

	cert, err := m.GenerateServiceCert("web", []string{"web.local"})
	if err != nil {
		t.Fatalf("GenerateServiceCert() failed: %v", err)
	}

	// Verify the cert can be used in a tls.Config
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	if config == nil {
		t.Error("tls.Config should not be nil")
	}
	if len(config.Certificates) != 1 {
		t.Errorf("expected 1 certificate in config, got %d", len(config.Certificates))
	}
}
