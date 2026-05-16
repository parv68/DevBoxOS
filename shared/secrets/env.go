package secrets

import (
	"fmt"
	"os"
)

// EnvProvider resolves secrets from environment variables.
type EnvProvider struct{}

// NewEnvProvider creates a new environment variable provider.
func NewEnvProvider() *EnvProvider {
	return &EnvProvider{}
}

// Name returns the provider name.
func (e *EnvProvider) Name() string {
	return "env"
}

// Resolve retrieves the secret from the specified environment variable.
func (e *EnvProvider) Resolve(source string) (string, error) {
	if source == "" {
		return "", fmt.Errorf("environment variable name cannot be empty")
	}

	value, exists := os.LookupEnv(source)
	if !exists {
		return "", fmt.Errorf("environment variable %s is not set", source)
	}

	if value == "" {
		return "", fmt.Errorf("environment variable %s is set but empty", source)
	}

	return value, nil
}
