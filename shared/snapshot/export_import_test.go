package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManager_ExportNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(nil, tmpDir)

	statusChan := make(chan string, 10)
	go func() {
		for range statusChan {
		}
	}()
	defer close(statusChan)

	err := m.Export("nonexistent-id", filepath.Join(tmpDir, "out.tar.gz"), statusChan)
	if err == nil {
		t.Error("expected error for nonexistent snapshot export")
	}
}

func TestManager_ImportInvalidTarball(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(nil, tmpDir)

	badPath := filepath.Join(tmpDir, "bad.tar.gz")
	os.WriteFile(badPath, []byte("not a tar file"), 0644)

	statusChan := make(chan string, 10)
	go func() {
		for range statusChan {
		}
	}()
	defer close(statusChan)

	err := m.Import(badPath, statusChan)
	if err == nil {
		t.Error("expected error for invalid tarball import")
	}
}

func TestManager_ExportCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(nil, tmpDir)

	m.store.SaveManifest(&Manifest{ID: "test-id", Name: "test", CreatedAt: time.Now()})
	snapDir := m.store.SnapshotDir("test-id")
	os.MkdirAll(snapDir, 0755)
	os.WriteFile(filepath.Join(snapDir, "data.txt"), []byte("snapshot data"), 0644)

	exportPath := filepath.Join(tmpDir, "export.tar.gz")
	statusChan := make(chan string, 10)
	go func() {
		for range statusChan {
		}
	}()
	defer close(statusChan)

	if err := m.Export("test-id", exportPath, statusChan); err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	if _, err := os.Stat(exportPath); os.IsNotExist(err) {
		t.Fatal("export file not created")
	}
}

func TestCalculateSnapshotDir_Empty(t *testing.T) {
	m := &Manager{}
	tmpDir := t.TempDir()

	size, hash, err := m.calculateSnapshotDir(tmpDir)
	if err != nil {
		t.Fatalf("calculateSnapshotDir() failed on empty dir: %v", err)
	}
	if size != 0 {
		t.Errorf("expected size 0 for empty dir, got %d", size)
	}
	if hash == "" {
		t.Error("expected non-empty hash even for empty dir")
	}
}

func TestCalculateSnapshotDir_Nested(t *testing.T) {
	m := &Manager{}
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "sub")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested data"), 0644)

	size, hash, err := m.calculateSnapshotDir(tmpDir)
	if err != nil {
		t.Fatalf("calculateSnapshotDir() failed: %v", err)
	}
	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
}
