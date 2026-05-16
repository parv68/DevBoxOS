package cmd

import (
	"fmt"
	"os"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose and repair environment issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}

		conn, err := client.New()
		if err != nil {
			return fmt.Errorf("connect to engine: %w", err)
		}
		defer conn.Close()

		result, err := conn.Doctor(dir)
		if err != nil {
			return fmt.Errorf("run diagnostics: %w", err)
		}

		output.DoctorProto(result)
		return nil
	},
}
