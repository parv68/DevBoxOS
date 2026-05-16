package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strconv"
)

const (
	defaultSecretLength = 32
	minSecretLength       = 8
	maxSecretLength       = 256
)

// GenerateProvider creates random secrets.
type GenerateProvider struct{}

// NewGenerateProvider creates a new secret generator.
func NewGenerateProvider() *GenerateProvider {
	return &GenerateProvider{}
}

// Name returns the provider name.
func (g *GenerateProvider) Name() string {
	return "generate"
}

// Resolve generates a random secret.
// Source format: "length" or "length:charset"
// Examples: "32", "64:hex", "16:alphanumeric"
func (g *GenerateProvider) Resolve(source string) (string, error) {
	if source == "" {
		return g.generate(defaultSecretLength, "base64")
	}

	parts := splitSource(source)
	length := defaultSecretLength
	charset := "base64"

	if len(parts) >= 1 {
		var err error
		length, err = strconv.Atoi(parts[0])
		if err != nil {
			return "", fmt.Errorf("invalid secret length: %s", parts[0])
		}
	}

	if len(parts) >= 2 {
		charset = parts[1]
	}

	if length < minSecretLength {
		return "", fmt.Errorf("secret length %d is too short (min %d)", length, minSecretLength)
	}

	if length > maxSecretLength {
		return "", fmt.Errorf("secret length %d is too long (max %d)", length, maxSecretLength)
	}

	return g.generate(length, charset)
}

func (g *GenerateProvider) generate(length int, charset string) (string, error) {
	switch charset {
	case "base64":
		bytes := make([]byte, length)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("generate random bytes: %w", err)
		}
		return base64.URLEncoding.EncodeToString(bytes)[:length], nil

	case "hex":
		bytes := make([]byte, length/2+1)
		if _, err := rand.Read(bytes); err != nil {
			return "", fmt.Errorf("generate random bytes: %w", err)
		}
		return fmt.Sprintf("%x", bytes)[:length], nil

	case "alphanumeric":
		const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		result := make([]byte, length)
		maxIdx := big.NewInt(int64(len(chars)))
		for i := range result {
			n, err := rand.Int(rand.Reader, maxIdx)
			if err != nil {
				return "", fmt.Errorf("generate random character: %w", err)
			}
			result[i] = chars[n.Int64()]
		}
		return string(result), nil

	default:
		return "", fmt.Errorf("unsupported charset: %s (supported: base64, hex, alphanumeric)", charset)
	}
}

func splitSource(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
