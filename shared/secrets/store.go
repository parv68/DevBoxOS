package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// SecretEntry represents a stored secret.
type SecretEntry struct {
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	Source    string    `json:"source"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Store manages encrypted secret storage.
type Store struct {
	crypto    *AgeCrypto
	storePath string
	secrets   map[string]SecretEntry
}

// NewStore creates a new secret store.
func NewStore(crypto *AgeCrypto, storePath string) *Store {
	return &Store{
		crypto:    crypto,
		storePath: storePath,
		secrets:   make(map[string]SecretEntry),
	}
}

// Load reads and decrypts the secret store from disk.
func (s *Store) Load() error {
	dir := filepath.Dir(s.storePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create store directory: %w", err)
	}

	data, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read store file: %w", err)
	}

	plaintext, err := s.crypto.Decrypt(data)
	if err != nil {
		return fmt.Errorf("decrypt store: %w", err)
	}

	if err := json.Unmarshal(plaintext, &s.secrets); err != nil {
		return fmt.Errorf("unmarshal secrets: %w", err)
	}

	return nil
}

// Save encrypts and writes the secret store to disk.
func (s *Store) Save() error {
	plaintext, err := json.Marshal(s.secrets)
	if err != nil {
		return fmt.Errorf("marshal secrets: %w", err)
	}

	ciphertext, err := s.crypto.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt secrets: %w", err)
	}

	if err := os.WriteFile(s.storePath, ciphertext, 0600); err != nil {
		return fmt.Errorf("write store file: %w", err)
	}

	return nil
}

// Set adds or updates a secret.
func (s *Store) Set(name, value, source string) error {
	now := time.Now()

	if existing, ok := s.secrets[name]; ok {
		existing.Value = value
		existing.Source = source
		existing.UpdatedAt = now
		s.secrets[name] = existing
	} else {
		s.secrets[name] = SecretEntry{
			Name:      name,
			Value:     value,
			Source:    source,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}

	return s.Save()
}

// Get retrieves a secret by name.
func (s *Store) Get(name string) (SecretEntry, error) {
	entry, ok := s.secrets[name]
	if !ok {
		return SecretEntry{}, fmt.Errorf("secret %s not found", name)
	}
	return entry, nil
}

// List returns all secret entries (values masked).
func (s *Store) List() []SecretEntry {
	result := make([]SecretEntry, 0, len(s.secrets))
	for _, entry := range s.secrets {
		entry.Value = "****"
		result = append(result, entry)
	}
	return result
}

// Delete removes a secret by name.
func (s *Store) Delete(name string) error {
	if _, ok := s.secrets[name]; !ok {
		return fmt.Errorf("secret %s not found", name)
	}

	delete(s.secrets, name)
	return s.Save()
}

// Exists checks if a secret exists.
func (s *Store) Exists(name string) bool {
	_, ok := s.secrets[name]
	return ok
}
