package cmd

import (
	"strings"
	"testing"
)

func TestRootCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "DevBoxOS") {
		t.Errorf("expected help to contain 'DevBoxOS', got %q", output)
	}
}

func TestRootCmd_NoArgsShowsHelp(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Usage:") {
		t.Errorf("expected 'Usage:', got %q", output)
	}
}

func TestVersionCmd_Format(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"version"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "DevBoxOS") {
		t.Errorf("expected 'DevBoxOS', got %q", output)
	}
}

func TestStartCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"start", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Start") {
		t.Errorf("expected help to mention 'Start', got %q", output)
	}
}

func TestStopCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"stop", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Stop") {
		t.Errorf("expected help to mention 'Stop', got %q", output)
	}
}

func TestStatusCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"status", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "status") {
		t.Errorf("expected help to mention 'status', got %q", output)
	}
}

func TestLogsCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"logs", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "logs") {
		t.Errorf("expected help to mention 'logs', got %q", output)
	}
}

func TestResetCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"reset", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "reset") {
		t.Errorf("expected help to mention 'reset', got %q", output)
	}
}

func TestBuildCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"build", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Build") {
		t.Errorf("expected help to mention 'Build', got %q", output)
	}
}

func TestExecCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"exec", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Execute") && !strings.Contains(output, "Run a command") {
		t.Errorf("expected help to mention 'Execute' or 'Run a command', got %q", output)
	}
}

func TestShellCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"shell", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Interactive") && !strings.Contains(output, "interactive shell") {
		t.Errorf("expected help to mention 'Interactive' or 'interactive shell', got %q", output)
	}
}

func TestURLCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"url", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "URL") {
		t.Errorf("expected help to mention 'URL', got %q", output)
	}
}

func TestWaitCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"wait", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Wait") && !strings.Contains(output, "Block until") {
		t.Errorf("expected help to mention 'Wait' or 'Block until', got %q", output)
	}
}

func TestCpCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"cp", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Copy") {
		t.Errorf("expected help to mention 'Copy', got %q", output)
	}
}

func TestEnvCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"env", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "environment") {
		t.Errorf("expected help to mention 'environment', got %q", output)
	}
}

func TestGraphCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"graph", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "dependency graph") {
		t.Errorf("expected help to mention 'dependency graph', got %q", output)
	}
}

func TestPushCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"push", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Push") {
		t.Errorf("expected help to mention 'Push', got %q", output)
	}
}

func TestTopCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"top", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "top") {
		t.Errorf("expected help to mention 'top', got %q", output)
	}
}

func TestSnapshotCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snapshot", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "snapshot") {
		t.Errorf("expected help to mention 'snapshot', got %q", output)
	}
}

func TestSnapshotSaveCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snapshot", "save", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Save") {
		t.Errorf("expected help to mention 'Save', got %q", output)
	}
}

func TestSnapshotLoadCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snapshot", "load", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Load") {
		t.Errorf("expected help to mention 'Load', got %q", output)
	}
}

func TestSnapshotListCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snapshot", "list", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "List") {
		t.Errorf("expected help to mention 'List', got %q", output)
	}
}

func TestSnapshotDeleteCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snapshot", "delete", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Delete") {
		t.Errorf("expected help to mention 'Delete', got %q", output)
	}
}

func TestSnapshotExportCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snapshot", "export", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Export") {
		t.Errorf("expected help to mention 'Export', got %q", output)
	}
}

func TestSnapshotImportCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snapshot", "import", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Import") {
		t.Errorf("expected help to mention 'Import', got %q", output)
	}
}

func TestSnapshotGCCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"snapshot", "gc", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "gc") || strings.Contains(output, "garbage collect") {
		// accept both "gc" and "garbage collect"
		ok := strings.Contains(output, "gc")
		if !ok {
			t.Errorf("expected help to mention 'gc', got %q", output)
		}
	}
}

func TestSecretsCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"secrets", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "secrets") {
		t.Errorf("expected help to mention 'secrets', got %q", output)
	}
}

func TestSecretsSetCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"secrets", "set", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "Store") {
		t.Errorf("expected help to mention 'Store', got %q", output)
	}
}

func TestDoctorCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"doctor", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "diagnostic") {
		t.Errorf("expected help to mention 'diagnostic', got %q", output)
	}
}

func TestInitCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"init", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "scanning") {
		t.Errorf("expected help to mention 'scanning', got %q", output)
	}
}

func TestValidateCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"validate", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "syntax error") {
		t.Errorf("expected help to mention 'syntax error', got %q", output)
	}
}

func TestConfigCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"config", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "config") {
		t.Errorf("expected help to mention 'config', got %q", output)
	}
}

func TestCompletionCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"completion", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "completion") {
		t.Errorf("expected help to mention 'completion', got %q", output)
	}
}

func TestDestroyCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"destroy", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "destroy") {
		t.Errorf("expected help to mention 'destroy', got %q", output)
	}
}

func TestPSCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"ps", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "ps") {
		t.Errorf("expected help to mention 'ps', got %q", output)
	}
}

func TestPruneCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"prune", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "prune") {
		t.Errorf("expected help to mention 'prune', got %q", output)
	}
}

func TestUpgradeCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"upgrade", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "upgrade") {
		t.Errorf("expected help to mention 'upgrade', got %q", output)
	}
}

func TestComposeImportCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"init", "compose-import", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "docker-compose") {
		t.Errorf("expected help to mention 'docker-compose', got %q", output)
	}
}

func TestComposeExportCmd_Help(t *testing.T) {
	output := captureStdout(func() {
		rootCmd.SetArgs([]string{"init", "compose-export", "--help"})
		rootCmd.Execute()
	})

	if !strings.Contains(output, "compose") {
		t.Errorf("expected help to mention 'compose', got %q", output)
	}
}
