package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileProvider resolves secrets from files.
type FileProvider struct {
	baseDir string
}

// NewFileProvider creates a new file-based secret provider.
func NewFileProvider(baseDir string) *FileProvider {
	return &FileProvider{baseDir: baseDir}
}

// Name returns the provider name.
func (f *FileProvider) Name() string {
	return "file"
}

// Resolve reads the secret from the specified file.
func (f *FileProvider) Resolve(source string) (string, error) {
	if source == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	path := source
	if !filepath.IsAbs(path) && f.baseDir != "" {
		path = filepath.Join(f.baseDir, source)
	}

	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("secret file %s does not exist", path)
		}
		return "", fmt.Errorf("stat secret file %s: %w", path, err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("secret file %s is a directory", path)
	}

	if info.Size() > 1024*1024 {
		return "", fmt.Errorf("secret file %s is too large (max 1MB)", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read secret file %s: %w", path, err)
	}

	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("secret file %s is empty", path)
	}

	return value, nil
}
