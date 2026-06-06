package autodetect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devboxos/devboxos/shared/types"
	"gopkg.in/yaml.v3"
)

// AutoDetect scans a project directory and infers required runtimes and services.
func AutoDetect(dir string) (*types.Config, error) {
	cfg := &types.Config{
		Services: make(map[string]types.Service),
		Runtimes: make(map[string]string),
	}

	cfg.Name = filepath.Base(dir)
	cfg.Version = "1.0"

	// Scan for package.json (Node.js)
	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		runtime, version := detectNodeRuntime(filepath.Join(dir, "package.json"))
		if runtime != "" {
			cfg.Runtimes[runtime] = version
		}

		cfg.Services["api"] = types.Service{
			Command: "npm run dev",
			Port:    "3000",
		}
	}

	// Scan for requirements.txt or pyproject.toml (Python)
	if _, err := os.Stat(filepath.Join(dir, "requirements.txt")); err == nil {
		cfg.Runtimes["python"] = "3.11"
		cfg.Services["worker"] = types.Service{
			Runtime: "python311",
			Command: "python worker.py",
		}
	} else if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
		cfg.Runtimes["python"] = "3.11"
		cfg.Services["worker"] = types.Service{
			Runtime: "python311",
			Command: "python -m app",
		}
	}

	// Scan for go.mod (Go)
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		cfg.Runtimes["go"] = "1.22"
		cfg.Services["api"] = types.Service{
			Runtime: "go122",
			Command: "go run main.go",
			Port:    "8080",
		}
	}

	// Scan for Cargo.toml (Rust)
	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
		cfg.Runtimes["rust"] = "1.75"
		cfg.Services["api"] = types.Service{
			Runtime: "rust175",
			Command: "cargo run",
			Port:    "8080",
		}
	}

	// Scan for docker-compose.yml
	if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err == nil {
		if err := importDockerCompose(dir, cfg); err != nil {
			_ = fmt.Sprintf("Warning: could not import docker-compose.yml: %v", err)
		}
	}

	// Scan for Dockerfile
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err == nil {
		cfg.Services["app"] = types.Service{
			Build: &types.BuildConfig{
				Context:    ".",
				Dockerfile: "Dockerfile",
			},
			Port: "8080",
		}
	}

	// Default networking
	cfg.Networking = types.Networking{
		Discovery: true,
		Egress:    "default-deny",
	}

	// Set exposed ports
	for _, svc := range cfg.Services {
		if svc.Port != "" {
			var port int
			fmt.Sscanf(svc.Port, "%d", &port)
			if port > 0 {
				cfg.Networking.Expose = append(cfg.Networking.Expose, port)
			}
		}
	}

	return cfg, nil
}

func detectNodeRuntime(packageJSONPath string) (string, string) {
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return "", ""
	}

	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", ""
	}

	if pkg.Engines.Node != "" {
		version := pkg.Engines.Node
		for len(version) > 0 && (version[0] == '^' || version[0] == '~' || version[0] == '>' || version[0] == '=' || version[0] == '<') {
			version = version[1:]
		}
		if len(version) > 0 {
			return "node", version[:2]
		}
	}

	return "node", "18"
}

func importDockerCompose(dir string, cfg *types.Config) error {
	data, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		return err
	}

	var compose struct {
		Services map[string]struct {
			Image       string            `yaml:"image"`
			Build       interface{}       `yaml:"build"`
			Ports       []string          `yaml:"ports"`
			Environment map[string]string `yaml:"environment"`
			DependsOn   []string          `yaml:"depends_on"`
			Volumes     []string          `yaml:"volumes"`
		} `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &compose); err != nil {
		return err
	}

	for name, svc := range compose.Services {
		dbSvc := types.Service{
			Image:     svc.Image,
			Env:       svc.Environment,
			DependsOn: svc.DependsOn,
			Volumes:   svc.Volumes,
		}

		if len(svc.Ports) > 0 {
			dbSvc.Port = svc.Ports[0]
		}

		cfg.Services[name] = dbSvc
	}

	return nil
}
