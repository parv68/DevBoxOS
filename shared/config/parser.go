package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devboxos/devboxos/shared/types"
	"gopkg.in/yaml.v3"
)

const (
	defaultConfigFile = "devbox.yml"
)

// Parser handles parsing of devbox.yml configuration files.
type Parser struct{}

// NewParser creates a new config parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse reads and parses a devbox.yml file from the given directory.
func (p *Parser) Parse(dir string) (*types.Config, error) {
	configPath := filepath.Join(dir, defaultConfigFile)

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("devbox.yml not found in %s — run 'devbox init' to create one", dir)
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// ParseFile reads and parses a devbox.yml file from an explicit path.
func (p *Parser) ParseFile(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg types.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// Generate creates a default devbox.yml in the given directory.
func (p *Parser) Generate(dir string, name string) error {
	configPath := filepath.Join(dir, defaultConfigFile)

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("devbox.yml already exists in %s", dir)
	}

	cfg := &types.Config{
		Name:    name,
		Version: "1.0",
		Services: map[string]types.Service{
			"api": {
				Image:   "node:18",
				Command: "npm run dev",
				Port:    "3000",
			},
		},
		Networking: types.Networking{
			Discovery: true,
			Expose:    []int{3000},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
