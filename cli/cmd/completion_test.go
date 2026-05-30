package cmd

import (
	"strings"
	"testing"
)

func TestCompletionCmd_InvalidShell(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"completion", "invalid"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "completion") {
		t.Errorf("expected completion-related output, got %q", output)
	}
}

func TestCompletionCmd_Bash(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"completion", "bash"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "bash") {
		t.Errorf("expected bash completion output, got %q", output)
	}
}

func TestCompletionCmd_Zsh(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"completion", "zsh"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "completion") {
		t.Errorf("expected completion-related output, got %q", output)
	}
}
