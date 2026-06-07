package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/devboxos/devboxos/cli/internal/client"
	"github.com/devboxos/devboxos/cli/internal/output"
	"github.com/spf13/cobra"
)

var engineCmd = &cobra.Command{
	Use:   "engine",
	Short: "Manage the DevBoxOS engine daemon",
	Long:  `Start, stop, and restart the DevBoxOS engine daemon process.`,
}

var engineStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the engine daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ping existing engine
		existing, err := tryConnect()
		if err == nil {
			_, _ = existing.Ping()
			existing.Close()
			output.Info("Engine is already running")
			return nil
		}

		binPath, err := client.EngineBinPath()
		if err != nil {
			return fmt.Errorf("locate engine binary: %w", err)
		}

		if _, err := os.Stat(binPath); err != nil {
			return fmt.Errorf("engine binary not found at %s: %w", binPath, err)
		}

		c := exec.Command(binPath, "--daemon")
		c.Stdout = os.Stderr
		c.Stderr = os.Stderr
		if err := c.Start(); err != nil {
			return fmt.Errorf("start engine: %w", err)
		}

		// Wait for engine to become responsive
		time.Sleep(1 * time.Second)
		conn, err := client.New()
		if err != nil {
			return fmt.Errorf("engine started but not responding: %w", err)
		}
		conn.Close()

		output.Success("Engine started")
		return nil
	},
}

var engineStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the engine daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		conn, err := client.New()
		if err != nil {
			return fmt.Errorf("connect to engine: %w", err)
		}
		defer conn.Close()

		output.Info("Stopping engine daemon...")
		if err := conn.Shutdown(); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}

		// Wait for process to exit
		time.Sleep(2 * time.Second)
		output.Success("Engine stopped")
		return nil
	},
}

var engineRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the engine daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Try stopping existing engine
		conn, err := client.New()
		if err == nil {
			output.Info("Stopping engine daemon...")
			_ = conn.Shutdown()
			conn.Close()
			time.Sleep(2 * time.Second)
		}

		binPath, err := client.EngineBinPath()
		if err != nil {
			return fmt.Errorf("locate engine binary: %w", err)
		}

		if _, err := os.Stat(binPath); err != nil {
			return fmt.Errorf("engine binary not found at %s: %w", binPath, err)
		}

		c := exec.Command(binPath, "--daemon")
		c.Stdout = os.Stderr
		c.Stderr = os.Stderr
		if err := c.Start(); err != nil {
			return fmt.Errorf("start engine: %w", err)
		}

		time.Sleep(1 * time.Second)
		newConn, err := client.New()
		if err != nil {
			return fmt.Errorf("engine restarted but not responding: %w", err)
		}
		newConn.Close()

		output.Success("Engine restarted")
		return nil
	},
}

// tryConnect attempts a quick gRPC dial without auto-starting.
func tryConnect() (*client.Client, error) {
	return client.New()
}

func init() {
	engineCmd.AddCommand(engineStartCmd)
	engineCmd.AddCommand(engineStopCmd)
	engineCmd.AddCommand(engineRestartCmd)
	rootCmd.AddCommand(engineCmd)
}
