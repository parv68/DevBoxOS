package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devboxos/devboxos/cli/internal/autodetect"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/devboxos/devboxos/shared/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	initFromGit   string
	initTemplate  string
	initBranch    string
	initDryRun    bool
	initMaxDepth  int
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new DevBoxOS project",
	Long: `Generate a devbox.yml configuration file by scanning the current project.

Examples:
  devbox init
  devbox init my-project
  devbox init --dry-run
  devbox init --max-depth 4
  devbox init --from-git https://github.com/user/project.git
  devbox init --template react-express-postgres`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initFromGit, "from-git", "", "Clone a repository and initialize from it")
	initCmd.Flags().StringVar(&initTemplate, "template", "", "Use a predefined project template")
	initCmd.Flags().StringVar(&initBranch, "branch", "", "Git branch to clone (requires --from-git)")
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false, "Print generated configuration to stdout without writing files")
	initCmd.Flags().IntVar(&initMaxDepth, "max-depth", 2, "Maximum subdirectory depth for monorepo scanning")
}

func runInit(cmd *cobra.Command, args []string) error {
	if initFromGit != "" {
		return runInitFromGit(args)
	}

	if initTemplate != "" {
		return runInitWithTemplate(args)
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	projectName := filepath.Base(dir)
	if len(args) > 0 && args[0] != "" {
		projectName = args[0]
	}

	configPath := filepath.Join(dir, "devbox.yml")
	if !initDryRun {
		if _, err := os.Stat(configPath); err == nil {
			output.Warning("devbox.yml already exists in %s", dir)
			return nil
		}
	}

	output.Info("Scanning project...")

	cfg, err := autodetect.AutoDetectWithDepth(dir, initMaxDepth)
	if err != nil {
		return fmt.Errorf("auto-detect: %w", err)
	}
	cfg.Name = projectName

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if initDryRun {
		output.Success("Generated configuration for %s:", projectName)
		fmt.Println(string(data))
		return nil
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	output.Success("Created devbox.yml in %s", dir)
	output.Info("Detected %d service(s):", len(cfg.Services))
	for name := range cfg.Services {
		output.Info("  - %s", name)
	}
	checkRuntimeAvailability(cfg)
	output.Info("Run 'devbox start' to launch your environment")
	return nil
}

func checkRuntimeAvailability(cfg *types.Config) {
	runtimeCmds := map[string]string{
		"node": "node", "go": "go", "python": "python3",
		"rust": "cargo", "java": "java", "ruby": "ruby", "php": "php",
	}
	for rt := range cfg.Runtimes {
		cmd, ok := runtimeCmds[rt]
		if !ok {
			continue
		}
		if _, err := exec.LookPath(cmd); err != nil {
			output.Warning("%s detected but %q not found on PATH", rt, cmd)
		}
	}
}

func runInitFromGit(args []string) error {
	tmpDir, err := os.MkdirTemp("", "devbox-git-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	output.Info("Cloning %s...", initFromGit)

	cloneArgs := []string{"clone", "--depth", "1"}
	if initBranch != "" {
		cloneArgs = append(cloneArgs, "--branch", initBranch)
	}
	cloneArgs = append(cloneArgs, initFromGit, tmpDir)

	gitCmd := exec.Command("git", cloneArgs...)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	output.Info("Detecting project configuration...")

	cfg, err := autodetect.AutoDetectWithDepth(tmpDir, initMaxDepth)
	if err != nil {
		return fmt.Errorf("auto-detect: %w", err)
	}

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	projectName := filepath.Base(initFromGit)
	projectName = strings.TrimSuffix(projectName, ".git")
	projectDir = filepath.Join(projectDir, projectName)

	if err := os.Rename(tmpDir, projectDir); err != nil {
		if err := os.RemoveAll(projectDir); os.IsExist(err) {
			os.RemoveAll(projectDir)
		}
		if err := os.Rename(tmpDir, projectDir); err != nil {
			return fmt.Errorf("move project to %s: %w", projectDir, err)
		}
	}

	cfg.Name = projectName
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if initDryRun {
		output.Success("Generated configuration for %s:", projectName)
		fmt.Println(string(data))
		return nil
	}

	configPath := filepath.Join(projectDir, "devbox.yml")
	if _, err := os.Stat(configPath); err == nil {
		output.Warning("devbox.yml already exists in cloned repository")
	} else {
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("write config file: %w", err)
		}
	}

	output.Success("Project initialized from %s", initFromGit)
	output.Info("Project directory: %s", projectDir)
	output.Info("Detected %d service(s):", len(cfg.Services))
	for name := range cfg.Services {
		output.Info("  - %s", name)
	}
	checkRuntimeAvailability(cfg)
	output.Info("Run 'cd %s && devbox start' to launch your environment", projectName)
	return nil
}

func runInitWithTemplate(args []string) error {
	tmpl, ok := templates[initTemplate]
	if !ok {
		return fmt.Errorf("unknown template %q. Available: %s", initTemplate, availableTemplates())
	}

	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	projectName := filepath.Base(dir)
	if len(args) > 0 && args[0] != "" {
		projectName = args[0]
	}

	if initDryRun {
		output.Success("Template %s would create %d files:", initTemplate, len(tmpl.Files))
		for name := range tmpl.Files {
			output.Info("  - %s", name)
		}
		return nil
	}

	output.Info("Creating %s template in %s...", initTemplate, dir)

	for filename, content := range tmpl.Files {
		filePath := filepath.Join(dir, filename)
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return fmt.Errorf("create directory for %s: %w", filename, err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
		output.Info("  Created: %s", filename)
	}

	output.Success("Initialized %s template for %s", initTemplate, projectName)
	output.Info("Run 'cd %s && devbox start' to launch your environment", projectName)
	return nil
}

func availableTemplates() string {
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}
