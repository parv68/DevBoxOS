package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var composeExportCmd = &cobra.Command{
	Use:   "compose-export [output]",
	Short: "Export DevBoxOS config to docker-compose.yml",
	Long:  `Convert a devbox.yml file into a standard docker-compose.yml format.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runComposeExport,
}

var (
	composeExportOutput string
	composeExportForce  bool
	composeVersion      string
)

func init() {
	composeExportCmd.Flags().StringVarP(&composeExportOutput, "output", "o", "docker-compose.yml", "Output file path")
	composeExportCmd.Flags().BoolVarP(&composeExportForce, "force", "f", false, "Overwrite existing output")
	composeExportCmd.Flags().StringVar(&composeVersion, "version", "3.8", "Docker Compose version")
	initCmd.AddCommand(composeExportCmd)
}

type ComposeOutput struct {
	Version  string                     `yaml:"version"`
	Services map[string]ComposeOutSvc   `yaml:"services"`
}

type ComposeOutSvc struct {
	Image       string              `yaml:"image,omitempty"`
	Build       any                 `yaml:"build,omitempty"`
	Command     any                 `yaml:"command,omitempty"`
	WorkingDir  string              `yaml:"working_dir,omitempty"`
	Ports       []string            `yaml:"ports,omitempty"`
	Environment []string            `yaml:"environment,omitempty"`
	EnvFile     string              `yaml:"env_file,omitempty"`
	Volumes     []string            `yaml:"volumes,omitempty"`
	DependsOn   any                 `yaml:"depends_on,omitempty"`
	Healthcheck *ComposeHealthcheck `yaml:"healthcheck,omitempty"`
	Restart     string              `yaml:"restart,omitempty"`
}

type ComposeHealthcheck struct {
	Test        any    `yaml:"test,omitempty"`
	Interval    string `yaml:"interval,omitempty"`
	Timeout     string `yaml:"timeout,omitempty"`
	Retries     int    `yaml:"retries,omitempty"`
	StartPeriod string `yaml:"start_period,omitempty"`
}

func runComposeExport(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	if len(args) > 0 {
		composeExportOutput = args[0]
	}

	if _, err := os.Stat(composeExportOutput); err == nil && !composeExportForce {
		return fmt.Errorf("output file exists: %s (use --force to overwrite)", composeExportOutput)
	}

	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err != nil {
		return fmt.Errorf("parse devbox config: %w", err)
	}

	compose := convertDevBoxToCompose(cfg)

	outData, err := yaml.Marshal(compose)
	if err != nil {
		return fmt.Errorf("marshal compose output: %w", err)
	}

	if err := os.WriteFile(composeExportOutput, outData, 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	fmt.Printf("✓ Exported %d services to %s\n", len(cfg.Services), composeExportOutput)
	fmt.Println("\nReview the generated docker-compose.yml and adjust as needed.")
	return nil
}

func convertDevBoxToCompose(cfg *types.Config) *ComposeOutput {
	compose := &ComposeOutput{
		Version:  composeVersion,
		Services: make(map[string]ComposeOutSvc),
	}

	for name, svc := range cfg.Services {
		outSvc := ComposeOutSvc{}

		if svc.Image != "" {
			outSvc.Image = svc.Image
		}

		if svc.Build != nil && svc.Build.Context != "" {
			outSvc.Build = convertBuildToCompose(svc.Build)
		}

		if svc.Command != "" {
			outSvc.Command = svc.Command
		}

		if svc.WorkingDir != "" {
			outSvc.WorkingDir = svc.WorkingDir
		}

		if len(svc.Ports) > 0 {
			outSvc.Ports = svc.Ports
		} else if svc.Port != "" {
			outSvc.Ports = []string{svc.Port}
		}

		if len(svc.Env) > 0 {
			outSvc.Environment = convertEnvMap(svc.Env)
		}

		if svc.EnvFile != "" {
			outSvc.EnvFile = svc.EnvFile
		}

		if len(svc.Volumes) > 0 {
			outSvc.Volumes = svc.Volumes
		}

		if len(svc.DependsOn) > 0 {
			outSvc.DependsOn = svc.DependsOn
		}

		if svc.Healthcheck != nil {
			outSvc.Healthcheck = convertHealthcheck(svc.Healthcheck)
		}

		if svc.RestartPolicy != nil {
			outSvc.Restart = convertRestartPolicy(svc.RestartPolicy)
		}

		compose.Services[name] = outSvc
	}

	return compose
}

func convertBuildToCompose(build *types.BuildConfig) any {
	if build.Dockerfile == "" && len(build.Args) == 0 && build.Target == "" {
		return build.Context
	}
	result := map[string]any{
		"context": build.Context,
	}
	if build.Dockerfile != "" {
		result["dockerfile"] = build.Dockerfile
	}
	if build.Target != "" {
		result["target"] = build.Target
	}
	if len(build.Args) > 0 {
		var args []string
		for k, v := range build.Args {
			args = append(args, fmt.Sprintf("%s=%s", k, v))
		}
		result["args"] = args
	}
	return result
}

func convertEnvMap(env map[string]string) []string {
	var result []string
	for k, v := range env {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func convertHealthcheck(hc *types.Healthcheck) *ComposeHealthcheck {
	if hc == nil {
		return nil
	}
	out := &ComposeHealthcheck{
		Interval:    hc.Interval,
		Timeout:     hc.Timeout,
		Retries:     hc.Retries,
		StartPeriod: hc.StartPeriod,
	}
	if hc.Command != "" {
		out.Test = hc.Command
	} else if hc.Path != "" {
		out.Test = []string{"CMD-SHELL", fmt.Sprintf("curl -f http://localhost%s || exit 1", hc.Path)}
	}
	return out
}

func convertRestartPolicy(rp *types.RestartPolicy) string {
	if rp == nil {
		return ""
	}
	if rp.Always {
		return "always"
	}
	if rp.OnFailure {
		return "on-failure"
	}
	return "no"
}
