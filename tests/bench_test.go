//go:build e2e

package tests

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ─────────────────────────────────────────────
// Local-only benchmarks (no Docker, run in all modes)
// ─────────────────────────────────────────────

func BenchmarkCLI_Build(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping build benchmark in short mode")
	}

	cli := findCLI(b)

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		data, err := os.ReadFile(fixture(b, "build-stack/Dockerfile"))
		if err != nil {
			b.Fatalf("read build fixture: %v", err)
		}
		if err := os.WriteFile(filepath.Join(tmpDir, "Dockerfile"), data, 0644); err != nil {
			b.Fatalf("write Dockerfile: %v", err)
		}

		b.ResetTimer()

		cmd := exec.Command(cli, "build")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("build failed: %v", err)
		}
	}
}

func BenchmarkCLI_Validate(b *testing.B) {
	cli := findCLI(b)
	tmpDir := b.TempDir()
	writeFixture(b, tmpDir, "devbox.yml")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(cli, "validate")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("validate failed: %v", err)
		}
	}
}

func BenchmarkCLI_Init(b *testing.B) {
	cli := findCLI(b)

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		writeScannerFixture(b, tmpDir, "node-express")

		b.ResetTimer()
		cmd := exec.Command(cli, "init")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("init failed: %v", err)
		}
	}
}

func BenchmarkCLI_Graph(b *testing.B) {
	cli := findCLI(b)
	tmpDir := b.TempDir()
	writeFixture(b, tmpDir, "devbox.yml")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(cli, "graph")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("graph failed: %v", err)
		}
	}
}

func BenchmarkCLI_ComposeImport(b *testing.B) {
	cli := findCLI(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		absCompose := fixture(b, "docker-compose.yml")
		cmd := exec.Command(cli, "init", "compose-import", absCompose)
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("compose-import failed: %v", err)
		}
	}
}

func BenchmarkCLI_ComposeExport(b *testing.B) {
	cli := findCLI(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		writeFixture(b, tmpDir, "devbox.yml")
		cmd := exec.Command(cli, "init", "compose-export")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("compose-export failed: %v", err)
		}
	}
}

func BenchmarkCLI_SecretsSet(b *testing.B) {
	cli := findCLI(b)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		cmd := exec.Command(cli, "secrets", "set", "bench-key", "bench-value")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("secrets set failed: %v", err)
		}
	}
}

func BenchmarkCLI_SecretsGet(b *testing.B) {
	cli := findCLI(b)

	// Set up a known secret once
	tmpDir := b.TempDir()
	setup := exec.Command(cli, "secrets", "set", "bench-key", "bench-value")
	setup.Dir = tmpDir
	if out, err := setup.CombinedOutput(); err != nil {
		b.Fatalf("secrets setup failed: %v\n%s", err, out)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(cli, "secrets", "get", "bench-key")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("secrets get failed: %v", err)
		}
	}
}

// ─────────────────────────────────────────────
// Docker-dependent benchmarks (skip in short mode)
// ─────────────────────────────────────────────

func BenchmarkCLI_Start(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping start benchmark in short mode")
	}

	cli := findCLI(b)

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		writeFixture(b, tmpDir, "devbox.yml")

		b.StopTimer()
		stopStackB(b, cli, tmpDir) // cleanup from prior run if any
		b.StartTimer()

		cmd := exec.Command(cli, "start")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("start failed: %v", err)
		}

		b.StopTimer()
		stopStackB(b, cli, tmpDir)
		b.StartTimer()
	}
}

func BenchmarkCLI_Stop(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping stop benchmark in short mode")
	}

	cli := findCLI(b)

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		writeFixture(b, tmpDir, "devbox.yml")

		startStackB(b, cli, tmpDir)

		b.ResetTimer()

		cmd := exec.Command(cli, "stop")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("stop failed: %v", err)
		}
	}
}

func BenchmarkCLI_Exec(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping exec benchmark in short mode")
	}

	cli := findCLI(b)
	tmpDir := b.TempDir()
	writeFixture(b, tmpDir, "devbox.yml")
	startStackB(b, cli, tmpDir)
	defer stopStackB(b, cli, tmpDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(cli, "exec", "web", "echo", "bench-ok")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("exec failed: %v", err)
		}
	}
}

func BenchmarkCLI_Wait(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping wait benchmark in short mode")
	}

	cli := findCLI(b)
	tmpDir := b.TempDir()
	writeFixture(b, tmpDir, "devbox.yml")
	startStackB(b, cli, tmpDir)
	defer stopStackB(b, cli, tmpDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(cli, "wait", "web", "--timeout", "10")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("wait failed: %v", err)
		}
	}
}

func BenchmarkCLI_CP(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping cp benchmark in short mode")
	}

	cli := findCLI(b)
	tmpDir := b.TempDir()
	writeFixture(b, tmpDir, "devbox.yml")
	startStackB(b, cli, tmpDir)
	defer stopStackB(b, cli, tmpDir)

	destDir := filepath.Join(tmpDir, "cp-bench")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dest := filepath.Join(destDir, "hostname")
		cmd := exec.Command(cli, "cp", "web:/etc/hostname", dest)
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("cp failed: %v", err)
		}
	}
}

func BenchmarkCLI_Shell(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping shell benchmark in short mode")
	}

	cli := findCLI(b)
	tmpDir := b.TempDir()
	writeFixture(b, tmpDir, "devbox.yml")
	startStackB(b, cli, tmpDir)
	defer stopStackB(b, cli, tmpDir)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(cli, "shell", "web")
		cmd.Dir = tmpDir
		cmd.Stdin = strings.NewReader("echo BENCH_OK\nexit\n")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("shell failed: %v", err)
		}
	}
}

func BenchmarkCLI_SnapshotSave(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping snapshot-save benchmark in short mode")
	}

	cli := findCLI(b)

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		writeFixture(b, tmpDir, "devbox.yml")
		startStackB(b, cli, tmpDir)

		b.ResetTimer()

		cmd := exec.Command(cli, "snapshot", "save", "bench-snap")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("snapshot save failed: %v", err)
		}

		b.StopTimer()
		stopStackB(b, cli, tmpDir)
		b.StartTimer()
	}
}

func BenchmarkCLI_SnapshotLoad(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping snapshot-load benchmark in short mode")
	}

	cli := findCLI(b)
	tmpDir := b.TempDir()
	writeFixture(b, tmpDir, "devbox.yml")
	startStackB(b, cli, tmpDir)
	defer stopStackB(b, cli, tmpDir)

	// Create a snapshot once
	setup := exec.Command(cli, "snapshot", "save", "bench-load-snap")
	setup.Dir = tmpDir
	if out, err := setup.CombinedOutput(); err != nil {
		b.Fatalf("snapshot setup failed: %v\n%s", err, out)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		cmd := exec.Command(cli, "snapshot", "load", "bench-load-snap")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("snapshot load failed: %v", err)
		}
	}
}

func BenchmarkCLI_SnapshotExport(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping snapshot-export benchmark in short mode")
	}

	cli := findCLI(b)
	tmpDir := b.TempDir()
	writeFixture(b, tmpDir, "devbox.yml")
	startStackB(b, cli, tmpDir)
	defer stopStackB(b, cli, tmpDir)

	// Create a snapshot once
	setup := exec.Command(cli, "snapshot", "save", "bench-export-snap")
	setup.Dir = tmpDir
	if out, err := setup.CombinedOutput(); err != nil {
		b.Fatalf("snapshot setup failed: %v\n%s", err, out)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		dest := filepath.Join(tmpDir, "snap-export.tar")
		cmd := exec.Command(cli, "snapshot", "export", "bench-export-snap", dest)
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("snapshot export failed: %v", err)
		}
	}
}

func BenchmarkCLI_SnapshotImport(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping snapshot-import benchmark in short mode")
	}

	cli := findCLI(b)

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		writeFixture(b, tmpDir, "devbox.yml")
		startStackB(b, cli, tmpDir)

		// Create and export a snapshot
		save := exec.Command(cli, "snapshot", "save", "bench-import-snap")
		save.Dir = tmpDir
		if out, err := save.CombinedOutput(); err != nil {
			b.Fatalf("snapshot save failed: %v\n%s", err, out)
		}
		exportPath := filepath.Join(tmpDir, "snap-export.tar")
		exp := exec.Command(cli, "snapshot", "export", "bench-import-snap", exportPath)
		exp.Dir = tmpDir
		if out, err := exp.CombinedOutput(); err != nil {
			b.Fatalf("snapshot export failed: %v\n%s", err, out)
		}

		b.ResetTimer()

		cmd := exec.Command(cli, "snapshot", "import", exportPath)
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		if err := cmd.Run(); err != nil {
			b.Fatalf("snapshot import failed: %v", err)
		}

		b.StopTimer()
		stopStackB(b, cli, tmpDir)
		b.StartTimer()
	}
}

func BenchmarkCLI_StartWatch(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping start-watch benchmark in short mode")
	}

	cli := findCLI(b)

	for i := 0; i < b.N; i++ {
		tmpDir := b.TempDir()
		writeFixture(b, tmpDir, "devbox.yml")
		touchDir := filepath.Join(tmpDir, "src")
		os.MkdirAll(touchDir, 0755)

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)

		cmd := exec.CommandContext(ctx, cli, "start", "--watch")
		cmd.Dir = tmpDir
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard

		if err := cmd.Start(); err != nil {
			cancel()
			b.Fatalf("start --watch failed: %v", err)
		}

		// Let the stack start
		time.Sleep(8 * time.Second)

		b.ResetTimer()

		// Touch a file to trigger change detection
		touchFile := filepath.Join(touchDir, "trigger.txt")
		os.WriteFile(touchFile, []byte("bench"), 0644)

		// Wait briefly for detection
		time.Sleep(3 * time.Second)

		b.StopTimer()
		cancel()
		cmd.Wait()
		b.StartTimer()
	}
}

// ─────────────────────────────────────────────
// Bench helpers (accept testing.TB)
// ─────────────────────────────────────────────

func startStackB(b testing.TB, cli, dir string) {
	b.Helper()
	cmd := exec.Command(cli, "start")
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		b.Fatalf("start failed: %v", err)
	}
}

func stopStackB(b testing.TB, cli, dir string) {
	b.Helper()
	cmd := exec.Command(cli, "stop")
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		b.Fatalf("stop failed: %v", err)
	}
}
