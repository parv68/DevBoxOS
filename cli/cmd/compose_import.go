package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devboxos/devboxos/shared/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var composeImportCmd = &cobra.Command{
	Use:   "compose-import [file]",
	Short: "Import docker-compose.yml into DevBoxOS config",
	Long:  `Convert a docker-compose.yml file into a DevBoxOS devbox.yaml configuration.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runComposeImport,
}

var (
	composeOutput string
	composeOverwrite bool
)

func init() {
	composeImportCmd.Flags().StringVarP(&composeOutput, "output", "o", "devbox.yaml", "Output file path")
	composeImportCmd.Flags().BoolVarP(&composeOverwrite, "force", "f", false, "Overwrite existing devbox.yaml")
	initCmd.AddCommand(composeImportCmd)
}

type ComposeFile struct {
	Version  string                     `yaml:"version"`
	Services map[string]ComposeService  `yaml:"services"`
	Networks map[string]any             `yaml:"networks,omitempty"`
	Volumes  map[string]any             `yaml:"volumes,omitempty"`
}

type ComposeService struct {
	Image       string            `yaml:"image,omitempty"`
	Build       any               `yaml:"build,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Environment []string          `yaml:"environment,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	DependsOn   any               `yaml:"depends_on,omitempty"`
	Command     any               `yaml:"command,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	EnvFile     any               `yaml:"env_file,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

func runComposeImport(cmd *cobra.Command, args []string) error {
	composeFile := "docker-compose.yml"
	if len(args) > 0 {
		composeFile = args[0]
	}

	if _, err := os.Stat(composeFile); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s", composeFile)
	}

	if _, err := os.Stat(composeOutput); err == nil && !composeOverwrite {
		return fmt.Errorf("output file exists: %s (use --force to overwrite)", composeOutput)
	}

	data, err := os.ReadFile(composeFile)
	if err != nil {
		return fmt.Errorf("read compose file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return fmt.Errorf("parse compose file: %w", err)
	}

	devboxConfig := convertComposeToDevBox(&compose)

	outData, err := yaml.Marshal(devboxConfig)
	if err != nil {
		return fmt.Errorf("marshal devbox config: %w", err)
	}

	if err := os.WriteFile(composeOutput, outData, 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("✓ Imported %d services from %s\n", len(compose.Services), composeFile)
	fmt.Printf("✓ Generated %s\n", composeOutput)
	fmt.Println("\nReview the generated config and adjust as needed:")
	fmt.Println("  devbox validate    # Check configuration")
	fmt.Println("  devbox start       # Start environment")

	return nil
}

func convertComposeToDevBox(compose *ComposeFile) *types.Config {
	cfg := &types.Config{
		Name:     filepath.Base(filepath.Dir(".")),
		Services: make(map[string]types.Service),
	}

	for name, svc := range compose.Services {
		devboxSvc := types.Service{
			Env: convertEnvList(svc.Environment),
		}

		if svc.Image != "" {
			devboxSvc.Image = svc.Image
		}

		if svc.Build != nil {
			devboxSvc.Build = convertBuild(svc.Build)
		}

		if len(svc.Ports) > 0 {
			devboxSvc.Ports = svc.Ports
		}

		if svc.DependsOn != nil {
			devboxSvc.DependsOn = convertDependsOn(svc.DependsOn)
		}

		if svc.Command != nil {
			devboxSvc.Command = convertCommandString(svc.Command)
		}

		if svc.WorkingDir != "" {
			devboxSvc.WorkingDir = svc.WorkingDir
		}

		if len(svc.Volumes) > 0 {
			devboxSvc.Volumes = svc.Volumes
		}

		cfg.Services[name] = devboxSvc
	}

	return cfg
}

func convertEnvList(env []string) map[string]string {
	result := make(map[string]string)
	for _, e := range env {
		if idx := strings.Index(e, "="); idx >= 0 {
			result[e[:idx]] = e[idx+1:]
		} else {
			result[e] = ""
		}
	}
	return result
}

func convertBuild(build any) *types.BuildConfig {
	switch b := build.(type) {
	case string:
		return &types.BuildConfig{Context: b}
	case map[string]any:
		cfg := &types.BuildConfig{}
		if ctx, ok := b["context"].(string); ok {
			cfg.Context = ctx
		}
		if dockerfile, ok := b["dockerfile"].(string); ok {
			cfg.Dockerfile = dockerfile
		}
		if args, ok := b["args"].([]any); ok {
			cfg.Args = make(map[string]string)
			for _, arg := range args {
				if s, ok := arg.(string); ok {
					if idx := strings.Index(s, "="); idx >= 0 {
						cfg.Args[s[:idx]] = s[idx+1:]
					}
				}
			}
		}
		return cfg
	default:
		return nil
	}
}

func convertDependsOn(dependsOn any) []string {
	switch d := dependsOn.(type) {
	case []any:
		var result []string
		for _, item := range d {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case map[string]any:
		var result []string
		for k := range d {
			result = append(result, k)
		}
		return result
	case string:
		return []string{d}
	default:
		return nil
	}
}

func convertCommandString(cmd any) string {
	switch c := cmd.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, item := range c {
			if s, ok := item.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return ""
	}
}
