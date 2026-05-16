package secrets

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
)

// AgeCrypto handles encryption/decryption using age.
type AgeCrypto struct {
	identity       *age.X25519Identity
	recipient      age.Recipient
	recipientStr   string
}

// NewAgeCrypto creates a new age encryption handler.
func NewAgeCrypto() (*AgeCrypto, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, fmt.Errorf("generate age identity: %w", err)
	}

	recipient := identity.Recipient()

	return &AgeCrypto{
		identity:     identity,
		recipient:    recipient,
		recipientStr: recipient.String(),
	}, nil
}

// NewAgeCryptoFromKey creates an age handler from an existing identity string.
func NewAgeCryptoFromKey(identityStr string) (*AgeCrypto, error) {
	identity, err := age.ParseX25519Identity(identityStr)
	if err != nil {
		return nil, fmt.Errorf("parse age identity: %w", err)
	}

	recipient := identity.Recipient()

	return &AgeCrypto{
		identity:     identity,
		recipient:    recipient,
		recipientStr: recipient.String(),
	}, nil
}
func (a *AgeCrypto) Encrypt(plaintext []byte) ([]byte, error) {
	var buf bytes.Buffer

	w, err := age.Encrypt(&buf, a.recipient)
	if err != nil {
		return nil, fmt.Errorf("create age encryptor: %w", err)
	}

	if _, err := w.Write(plaintext); err != nil {
		return nil, fmt.Errorf("write to age encryptor: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close age encryptor: %w", err)
	}

	return buf.Bytes(), nil
}

// Decrypt decrypts ciphertext and returns the plaintext.
func (a *AgeCrypto) Decrypt(ciphertext []byte) ([]byte, error) {
	r, err := age.Decrypt(bytes.NewReader(ciphertext), a.identity)
	if err != nil {
		return nil, fmt.Errorf("create age decryptor: %w", err)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, fmt.Errorf("read from age decryptor: %w", err)
	}

	return buf.Bytes(), nil
}

// IdentityString returns the identity string for persistence.
func (a *AgeCrypto) IdentityString() string {
	return a.identity.String()
}

// RecipientString returns the public key recipient string.
func (a *AgeCrypto) RecipientString() string {
	return a.recipientStr
}

// LoadOrCreateKey loads an existing key or creates a new one.
func LoadOrCreateKey(keyPath string) (*AgeCrypto, error) {
	dir := filepath.Dir(keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create key directory: %w", err)
	}

	if data, err := os.ReadFile(keyPath); err == nil {
		return NewAgeCryptoFromKey(string(data))
	}

	crypto, err := NewAgeCrypto()
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(keyPath, []byte(crypto.IdentityString()), 0600); err != nil {
		return nil, fmt.Errorf("write age key: %w", err)
	}

	return crypto, nil
}
