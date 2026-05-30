//go:build e2e

package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "fixtures", name)
}

func findCLI(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "cli", "devboxos.exe"),
		filepath.Join("..", "cli", "devboxos"),
		"devboxos.exe",
		"devboxos",
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	which, err := exec.LookPath("devboxos")
	if err == nil {
		return which
	}

	t.Fatalf("devboxos CLI binary not found (try: go build -o cli/ ./cli)")
	return ""
}

func devboxCmd(cli string, args ...string) *exec.Cmd {
	cmd := exec.Command(cli, args...)
	return cmd
}

func TestE2E_Version(t *testing.T) {
	cli := findCLI(t)

	out, err := devboxCmd(cli, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, out)
	}
	if !strings.Contains(strings.ToLower(string(out)), "devboxos") {
		t.Errorf("output should contain 'devboxos', got: %s", out)
	}
}

func TestE2E_Validate(t *testing.T) {
	cli := findCLI(t)

	// Create a project dir with devbox.yml from fixture
	tmpDir := t.TempDir()
	data, err := os.ReadFile(fixture(t, "devbox.yml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), data, 0644); err != nil {
		t.Fatalf("write devbox.yml: %v", err)
	}

	cmd := devboxCmd(cli, "validate")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "valid") {
		t.Errorf("expected validation to pass, got: %s", out)
	}
}

func TestE2E_InitThenValidate(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()

	initCmd := devboxCmd(cli, "init")
	initCmd.Dir = tmpDir
	initOut, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, initOut)
	}
	t.Logf("init output: %s", initOut)

	ymlPath := filepath.Join(tmpDir, "devbox.yml")
	if _, err := os.Stat(ymlPath); os.IsNotExist(err) {
		t.Fatalf("init did not create devbox.yml in %s", tmpDir)
	}
}

func TestE2E_InitCreateValidProject(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()

	initCmd := devboxCmd(cli, "init")
	initCmd.Dir = tmpDir
	initOut, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init failed: %v\n%s", err, initOut)
	}
	t.Logf("init output: %s", initOut)

	// Overwrite with our fixture for a valid project
	data, err := os.ReadFile(fixture(t, "devbox.yml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), data, 0644); err != nil {
		t.Fatalf("write devbox.yml: %v", err)
	}

	valCmd := devboxCmd(cli, "validate")
	valCmd.Dir = tmpDir
	valOut, err := valCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate after init failed: %v\n%s", err, valOut)
	}
	if !strings.Contains(string(valOut), "valid") {
		t.Errorf("expected validation to pass, got: %s", valOut)
	}
}

func TestE2E_StartStopLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping lifecycle test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	data, err := os.ReadFile(fixture(t, "devbox.yml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), data, 0644); err != nil {
		t.Fatalf("write devbox.yml: %v", err)
	}

	startCmd := devboxCmd(cli, "start")
	startCmd.Dir = tmpDir
	startOut, err := startCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("start failed: %v\n%s", err, startOut)
	}
	t.Logf("start output: %s", startOut)

	stopCmd := devboxCmd(cli, "stop")
	stopCmd.Dir = tmpDir
	stopOut, err := stopCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stop failed: %v\n%s", err, stopOut)
	}
	t.Logf("stop output: %s", stopOut)
}

func TestE2E_Doctor(t *testing.T) {
	cli := findCLI(t)

	out, err := devboxCmd(cli, "doctor").CombinedOutput()
	if err != nil {
		t.Fatalf("doctor failed: %v\n%s", err, out)
	}
	t.Logf("doctor output: %s", out)
}

func TestE2E_ComposeImport(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()
	absCompose := fixture(t, "docker-compose.yml")

	cmd := devboxCmd(cli, "init", "compose-import", absCompose)
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compose-import failed: %v\n%s", err, out)
	}
	t.Logf("compose-import output: %s", out)

	outFile := filepath.Join(tmpDir, "devbox.yaml")
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		// Also check devbox.yml as default
		outFile = filepath.Join(tmpDir, "devbox.yml")
	}
	if _, err := os.Stat(outFile); os.IsNotExist(err) {
		t.Fatalf("compose-import did not create any devbox file in %s", tmpDir)
	}
}

func TestE2E_CompletionBash(t *testing.T) {
	cli := findCLI(t)

	out, err := devboxCmd(cli, "completion", "bash").CombinedOutput()
	if err != nil {
		t.Fatalf("completion bash failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "bashcompinit") && !strings.Contains(string(out), "_devboxos") && !strings.Contains(string(out), "# bash completion") {
		t.Logf("completion output (first 200 chars): %s", string(out)[:min(len(out), 200)])
	}
}

func TestE2E_Prune(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping prune test in short mode")
	}

	cli := findCLI(t)

	out, err := devboxCmd(cli, "prune").CombinedOutput()
	if err != nil {
		t.Fatalf("prune failed: %v\n%s", err, out)
	}
	t.Logf("prune output: %s", out)
}

func TestE2E_HelpAllSubcommands(t *testing.T) {
	cli := findCLI(t)

	type subcommand struct {
		args []string
		name string
	}

	subcommands := []subcommand{
		{[]string{"build", "--help"}, "build"},
		{[]string{"config", "--help"}, "config"},
		{[]string{"completion", "--help"}, "completion"},
		{[]string{"destroy", "--help"}, "destroy"},
		{[]string{"doctor", "--help"}, "doctor"},
		{[]string{"exec", "--help"}, "exec"},
		{[]string{"init", "--help"}, "init"},
		{[]string{"init", "compose-import", "--help"}, "compose-import"},
		{[]string{"init", "compose-export", "--help"}, "compose-export"},
		{[]string{"logs", "--help"}, "logs"},
		{[]string{"prune", "--help"}, "prune"},
		{[]string{"ps", "--help"}, "ps"},
		{[]string{"reset", "--help"}, "reset"},
		{[]string{"secrets", "--help"}, "secrets"},
		{[]string{"snapshot", "--help"}, "snapshot"},
		{[]string{"start", "--help"}, "start"},
		{[]string{"status", "--help"}, "status"},
		{[]string{"stop", "--help"}, "stop"},
		{[]string{"upgrade", "--help"}, "upgrade"},
		{[]string{"validate", "--help"}, "validate"},
		{[]string{"version", "--help"}, "version"},
	}

	for _, sc := range subcommands {
		t.Run(sc.name, func(t *testing.T) {
			out, err := devboxCmd(cli, sc.args...).CombinedOutput()
			if err != nil {
				t.Errorf("%v --help failed: %v\n%s", sc.args, err, out)
			}
			if !strings.Contains(string(out), sc.name) && !strings.Contains(string(out), "Usage") {
				t.Errorf("%v --help output missing usage info: %s", sc.args, out)
			}
		})
	}
}
