package secrets

import (
	"fmt"
	"strings"
)

// Provider defines the interface for secret resolution.
type Provider interface {
	Name() string
	Resolve(source string) (string, error)
}

// ProviderFunc is a function that implements Provider.
type ProviderFunc func(source string) (string, error)

func (f ProviderFunc) Name() string {
	return "function"
}

func (f ProviderFunc) Resolve(source string) (string, error) {
	return f(source)
}

// Registry holds all available secret providers.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Resolve parses the source string and delegates to the appropriate provider.
// Source format: "provider:args" (e.g., "env:DB_PASSWORD", "file:.secrets/key", "generate:32")
func (r *Registry) Resolve(source string) (string, error) {
	parts := strings.SplitN(source, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid secret source format: %s (expected provider:args)", source)
	}

	providerName := parts[0]
	args := parts[1]

	provider, ok := r.providers[providerName]
	if !ok {
		return "", fmt.Errorf("unknown secret provider: %s", providerName)
	}

	return provider.Resolve(args)
}
