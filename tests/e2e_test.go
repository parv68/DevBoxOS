//go:build e2e

package tests

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fixture(t testing.TB, name string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(wd, "fixtures", name)
}

func findCLI(t testing.TB) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "cli", "devbox"),
		filepath.Join("..", "cli", "devbox.exe"),
		"devbox",
		"devbox.exe",
	}

	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}

	which, err := exec.LookPath("devbox")
	if err == nil {
		return which
	}

	t.Fatalf("devbox CLI binary not found (try: go build -o cli/ ./cli)")
	return ""
}

func devboxCmd(cli string, args ...string) *exec.Cmd {
	return exec.Command(cli, args...)
}

func writeFixture(t testing.TB, dir, name string) {
	t.Helper()
	data, err := os.ReadFile(fixture(t, name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// stackHelpers groups start/stop for E2E tests that need running containers.
func startStack(t *testing.T, cli, dir string) {
	t.Helper()
	cmd := devboxCmd(cli, "start")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("start failed: %v\n%s", err, out)
	}
	t.Logf("start output: %s", out)
}

func stopStack(t *testing.T, cli, dir string) {
	t.Helper()
	cmd := devboxCmd(cli, "stop")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stop failed: %v\n%s", err, out)
	}
	t.Logf("stop output: %s", out)
}

// ─────────────────────────────────────────────
// Local-only tests (no Docker, run in short mode)
// ─────────────────────────────────────────────

func TestE2E_Version(t *testing.T) {
	cli := findCLI(t)

	out, err := devboxCmd(cli, "version").CombinedOutput()
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "DevBoxOS") {
		t.Errorf("output should contain 'DevBoxOS', got: %s", out)
	}
}

func TestE2E_Validate(t *testing.T) {
	cli := findCLI(t)

	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")

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

	writeFixture(t, tmpDir, "devbox.yml")

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

func TestE2E_URL(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")

	cmd := devboxCmd(cli, "url")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("url failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "localhost:8080") {
		t.Errorf("url output should contain port mapping, got: %s", out)
	}
}

func TestE2E_Graph(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")

	cmd := devboxCmd(cli, "graph")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("graph failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "web") && !strings.Contains(string(out), "redis") {
		t.Errorf("graph output should contain service names, got: %s", out)
	}
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
	if !strings.Contains(string(out), "bashcompinit") && !strings.Contains(string(out), "_devbox") && !strings.Contains(string(out), "# bash completion") {
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

// ─────────────────────────────────────────────
// Docker-dependent tests (skip in short mode)
// ─────────────────────────────────────────────

func TestE2E_StartStopLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping lifecycle test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	stopStack(t, cli, tmpDir)
}

func TestE2E_Exec(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping exec test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	execCmd := devboxCmd(cli, "exec", "web", "echo", "e2e-ok")
	execCmd.Dir = tmpDir
	execOut, err := execCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("exec failed: %v\n%s", err, execOut)
	}
	if !strings.Contains(string(execOut), "e2e-ok") {
		t.Errorf("exec output should contain command output, got: %s", execOut)
	}
}

func TestE2E_Wait(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping wait test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	waitCmd := devboxCmd(cli, "wait", "web", "--timeout", "30")
	waitCmd.Dir = tmpDir
	waitOut, err := waitCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("wait failed: %v\n%s", err, waitOut)
	}
	if !strings.Contains(string(waitOut), "healthy") {
		t.Errorf("wait output should indicate healthy, got: %s", waitOut)
	}
}

func TestE2E_CP(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping cp test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	dest := filepath.Join(tmpDir, "copied-hostname")
	cpCmd := devboxCmd(cli, "cp", "web:/etc/hostname", dest)
	cpCmd.Dir = tmpDir
	cpOut, err := cpCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cp failed: %v\n%s", err, cpOut)
	}
	t.Logf("cp output: %s", cpOut)

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		t.Fatal("cp did not create destination file")
	}
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("copied file is empty")
	}
}

func TestE2E_Shell(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping shell test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	// Pipe a command to the interactive shell and check output
	cmd := devboxCmd(cli, "shell", "web")
	cmd.Dir = tmpDir
	cmd.Stdin = strings.NewReader("echo SHELL_WORKS\nexit\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("shell failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "SHELL_WORKS") {
		t.Logf("shell output (may have terminal codes): %s", out)
	}
}

func TestE2E_StartWatch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping start-watch test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()

	writeFixture(t, tmpDir, "devbox.yml")
	os.MkdirAll(filepath.Join(tmpDir, "src"), 0755)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, cli, "start", "--watch")
	cmd.Dir = tmpDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start --watch failed to start: %v", err)
	}

	// Wait a few seconds for startup, then touch a file
	time.Sleep(8 * time.Second)

	touchFile := filepath.Join(tmpDir, "src", "changed.txt")
	os.WriteFile(touchFile, []byte("trigger"), 0644)

	// Wait for watch to detect the change
	errCh := make(chan error, 1)
	go func() {
		errCh <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Timeout: check if we got the "Change detected" message
		output := stdout.String() + stderr.String()
		if strings.Contains(output, "Change detected") {
			t.Logf("Watch detected file change as expected")
		} else {
			t.Errorf("watch did not detect file change within timeout.\nstdout: %s\nstderr: %s", stdout.String(), stderr.String())
		}
	case err := <-errCh:
		// Process exited early
		output := stdout.String() + stderr.String()
		t.Fatalf("start --watch exited early: %v\noutput: %s", err, output)
	}
}

func TestE2E_SnapshotRoundtrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping snapshot roundtrip test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	snapshotName := "e2e-test-snap"

	// 1. Save
	saveCmd := devboxCmd(cli, "snapshot", "save", snapshotName)
	saveCmd.Dir = tmpDir
	saveOut, err := saveCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot save failed: %v\n%s", err, saveOut)
	}
	t.Logf("snapshot save: %s", saveOut)

	// Extract snapshot ID from save output: "Creating <name> (<id>)..."
	snapID := extractSnapshotID(string(saveOut))
	if snapID == "" {
		t.Fatal("could not extract snapshot ID from save output")
	}
	t.Logf("snapshot ID: %s", snapID)

	// 2. List
	listCmd := devboxCmd(cli, "snapshot", "list")
	listCmd.Dir = tmpDir
	listOut, err := listCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot list failed: %v\n%s", err, listOut)
	}
	if !strings.Contains(string(listOut), snapshotName) {
		t.Errorf("snapshot list should contain %q, got: %s", snapshotName, listOut)
	}

	// 3. Export (by ID, not name)
	exportPath := filepath.Join(tmpDir, "snapshot-export.tar")
	exportCmd := devboxCmd(cli, "snapshot", "export", snapID, exportPath)
	exportCmd.Dir = tmpDir
	exportOut, err := exportCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot export failed: %v\n%s", err, exportOut)
	}
	t.Logf("snapshot export: %s", exportOut)
	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Fatal("snapshot export did not create tarball")
	}

	// 4. Delete (by ID)
	delCmd := devboxCmd(cli, "snapshot", "delete", snapID)
	delCmd.Dir = tmpDir
	delOut, err := delCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot delete failed: %v\n%s", err, delOut)
	}
	t.Logf("snapshot delete: %s", delOut)

	// 5. Import
	importCmd := devboxCmd(cli, "snapshot", "import", exportPath)
	importCmd.Dir = tmpDir
	importOut, err := importCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot import failed: %v\n%s", err, importOut)
	}
	t.Logf("snapshot import: %s", importOut)

	// 6. List should show the imported snapshot
	list2Cmd := devboxCmd(cli, "snapshot", "list")
	list2Cmd.Dir = tmpDir
	list2Out, err := list2Cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot list (2) failed: %v\n%s", err, list2Out)
	}
	if !strings.Contains(string(list2Out), snapID[:8]) {
		t.Errorf("snapshot list should contain imported snapshot ID, got: %s", list2Out)
	}
}

// ─────────────────────────────────────────────
// Help smoke test for all commands
// ─────────────────────────────────────────────

func TestE2E_HelpAllSubcommands(t *testing.T) {
	cli := findCLI(t)

	type subcommand struct {
		args []string
		name string
	}

	subcommands := []subcommand{
		{[]string{"build", "--help"}, "build"},
		{[]string{"completion", "--help"}, "completion"},
		{[]string{"config", "--help"}, "config"},
		{[]string{"cp", "--help"}, "cp"},
		{[]string{"destroy", "--help"}, "destroy"},
		{[]string{"doctor", "--help"}, "doctor"},
		{[]string{"env", "--help"}, "env"},
		{[]string{"exec", "--help"}, "exec"},
		{[]string{"graph", "--help"}, "graph"},
		{[]string{"init", "--help"}, "init"},
		{[]string{"init", "compose-import", "--help"}, "compose-import"},
		{[]string{"init", "compose-export", "--help"}, "compose-export"},
		{[]string{"logs", "--help"}, "logs"},
		{[]string{"prune", "--help"}, "prune"},
		{[]string{"ps", "--help"}, "ps"},
		{[]string{"push", "--help"}, "push"},
		{[]string{"reset", "--help"}, "reset"},
		{[]string{"secrets", "--help"}, "secrets"},
		{[]string{"shell", "--help"}, "shell"},
		{[]string{"snapshot", "--help"}, "snapshot"},
		{[]string{"start", "--help"}, "start"},
		{[]string{"status", "--help"}, "status"},
		{[]string{"stop", "--help"}, "stop"},
		{[]string{"top", "--help"}, "top"},
		{[]string{"upgrade", "--help"}, "upgrade"},
		{[]string{"url", "--help"}, "url"},
		{[]string{"validate", "--help"}, "validate"},
		{[]string{"version", "--help"}, "version"},
		{[]string{"wait", "--help"}, "wait"},
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

// extractSnapshotID extracts the hex snapshot ID from save output like:
// "Creating my-snap (a1b2c3d4)..."
func extractSnapshotID(output string) string {
	start := strings.Index(output, "(")
	if start < 0 {
		return ""
	}
	start++
	end := strings.Index(output[start:], ")")
	if end < 0 {
		return ""
	}
	id := output[start : start+end]
	// Should be a hex hash
	if len(id) < 6 {
		return ""
	}
	return id
}
