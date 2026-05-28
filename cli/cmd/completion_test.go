package cmd

import (
	"strings"
	"testing"
)

func TestCompletionCmd_InvalidShell(t *testing.T) {
	err := func() error {
		rootCmd.SetArgs([]string{"completion", "invalid-shell"})
		return rootCmd.Execute()
	}()

	if err == nil {
		t.Error("expected error for invalid shell, got nil")
	}
}

func TestCompletionCmd_Bash(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"completion", "bash"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "bash") && len(output) < 50 {
		t.Errorf("bash completion too short or missing, got %d bytes", len(output))
	}
}

func TestCompletionCmd_Zsh(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"completion", "zsh"})
		rootCmd.Execute()
	})
	if !strings.Contains(output, "#compdef") && len(output) < 50 {
		t.Errorf("zsh completion too short or missing, got %d bytes", len(output))
	}
}
