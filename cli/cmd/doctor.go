package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/devboxos/devboxos/shared/config"
	"github.com/devboxos/devboxos/shared/diagnostics"
	"github.com/devboxos/devboxos/shared/runtime/docker"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and repair environment issues",
	Long:  `Run comprehensive diagnostics on your DevBoxOS environment.`,
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	rt := docker.NewDockerRuntime()
	ctx := context.Background()
	if err := rt.Connect(ctx); err != nil {
		fmt.Println("✗ Docker daemon: not accessible")
		fmt.Println("\nMake sure Docker Desktop is installed and running.")
		return nil
	}
	defer rt.Close()

	results, suggestions := diagnostics.RunDoctor(ctx, rt, dir)

	fmt.Println("DevBoxOS Diagnostics")
	fmt.Println("────────────────────")
	fmt.Println()

	for _, r := range results {
		icon := "✓"
		if !r.Passed {
			switch r.Severity {
			case diagnostics.SeverityCritical:
				icon = "✗"
			case diagnostics.SeverityError:
				icon = "✗"
			case diagnostics.SeverityWarning:
				icon = "⚠"
			}
		}
		fmt.Printf("%s %s: %s\n", icon, r.Name, r.Message)
	}

	if len(suggestions) > 0 {
		fmt.Println()
		fmt.Println("Suggestions:")
		for i, s := range suggestions {
			fmt.Printf("  %d. %s\n", i+1, s)
		}
	}

	// Count issues
	issueCount := 0
	for _, r := range results {
		if !r.Passed {
			issueCount++
		}
	}

	if issueCount == 0 {
		fmt.Println()
		fmt.Println("✓ All checks passed")
	} else {
		fmt.Printf("\n%d check(s) with issues\n", issueCount)
	}

	// Validate config separately
	parser := config.NewParser()
	cfg, err := parser.Parse(dir)
	if err == nil {
		validator, err := config.NewValidator()
		if err == nil {
			if err := validator.Validate(cfg); err != nil {
				fmt.Println()
				fmt.Println("✗ Schema validation failed:")
				fmt.Printf("  %v\n", err)
			}
		}
	}

	return nil
}
