package secrets

import (
	"fmt"
	"path/filepath"

	"github.com/devboxos/devboxos/shared/types"
)

// Resolver orchestrates secret resolution from multiple providers.
type Resolver struct {
	registry *Registry
	store    *Store
	crypto   *AgeCrypto
	baseDir  string
}

// NewResolver creates a new secret resolver.
func NewResolver(baseDir string, keyPath string, storePath string) (*Resolver, error) {
	crypto, err := LoadOrCreateKey(keyPath)
	if err != nil {
		return nil, fmt.Errorf("load age key: %w", err)
	}

	store := NewStore(crypto, storePath)
	if err := store.Load(); err != nil {
		return nil, fmt.Errorf("load secret store: %w", err)
	}

	registry := NewRegistry()
	registry.Register(NewEnvProvider())
	registry.Register(NewFileProvider(baseDir))
	registry.Register(NewGenerateProvider())

	return &Resolver{
		registry: registry,
		store:    store,
		crypto:   crypto,
		baseDir:  baseDir,
	}, nil
}

// Resolve resolves a secret reference to its value.
// Checks stored secrets first, then resolves from providers.
func (r *Resolver) Resolve(ref types.SecretRef) (string, error) {
	if ref.Name == "" {
		return "", fmt.Errorf("secret name cannot be empty")
	}

	if ref.Source == "" {
		return "", fmt.Errorf("secret %s has no source defined", ref.Name)
	}

	if r.store.Exists(ref.Name) {
		entry, err := r.store.Get(ref.Name)
		if err != nil {
			return "", err
		}

		if entry.Source == ref.Source {
			return entry.Value, nil
		}
	}

	value, err := r.registry.Resolve(ref.Source)
	if err != nil {
		return "", fmt.Errorf("resolve secret %s from %s: %w", ref.Name, ref.Source, err)
	}

	if err := r.store.Set(ref.Name, value, ref.Source); err != nil {
		return "", fmt.Errorf("store resolved secret %s: %w", ref.Name, err)
	}

	return value, nil
}

// Set manually stores a secret.
func (r *Resolver) Set(name, value string) error {
	return r.store.Set(name, value, "manual")
}

// Get retrieves a stored secret.
func (r *Resolver) Get(name string) (string, error) {
	entry, err := r.store.Get(name)
	if err != nil {
		return "", err
	}
	return entry.Value, nil
}

// List returns all stored secrets (masked).
func (r *Resolver) List() []SecretEntry {
	return r.store.List()
}

// Delete removes a stored secret.
func (r *Resolver) Delete(name string) error {
	return r.store.Delete(name)
}

// Rotate regenerates a secret and stores it.
func (r *Resolver) Rotate(name string) error {
	entry, err := r.store.Get(name)
	if err != nil {
		return err
	}

	if entry.Source == "manual" {
		return fmt.Errorf("cannot rotate manually set secret %s", name)
	}

	value, err := r.registry.Resolve(entry.Source)
	if err != nil {
		return fmt.Errorf("resolve secret %s for rotation: %w", name, err)
	}

	return r.store.Set(name, value, entry.Source)
}

// KeyPath returns the path to the encryption key.
func (r *Resolver) KeyPath() string {
	return filepath.Join(r.baseDir, ".devbox", "secrets.key")
}

// StorePath returns the path to the encrypted store.
func (r *Resolver) StorePath() string {
	return filepath.Join(r.baseDir, ".devbox", "secrets.enc")
}
