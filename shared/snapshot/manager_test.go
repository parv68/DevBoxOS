package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	m := NewManager(nil, "/tmp/test")
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestNewSecretsHandler(t *testing.T) {
	h := NewSecretsHandler("/path/key", "/path/store")
	if h == nil {
		t.Fatal("NewSecretsHandler() returned nil")
	}
}

func TestManager_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(nil, tmpDir)

	m.store.SaveManifest(&Manifest{ID: "test-id", Name: "test", CreatedAt: time.Now()})
	os.MkdirAll(m.store.SnapshotDir("test-id"), 0755)

	if err := m.Delete("test-id"); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if _, err := m.store.LoadManifest("test-id"); err == nil {
		t.Error("expected manifest to be deleted")
	}
}

func TestManager_DeleteNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(nil, tmpDir)

	m.store.EnsureDir()

	err := m.Delete("nonexistent")
	if err == nil {
		t.Log("Delete() nonexistent returned nil (file was already gone)")
	}
}

func TestManager_List_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(nil, tmpDir)

	infos, err := m.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected empty list, got %d", len(infos))
	}
}

func TestManager_List_WithManifests(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(nil, tmpDir)

	m.store.SaveManifest(&Manifest{ID: "id1", Name: "snap1", CreatedAt: time.Now(), SizeBytes: 100})
	m.store.SaveManifest(&Manifest{ID: "id2", Name: "snap2", CreatedAt: time.Now(), SizeBytes: 200})

	infos, err := m.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}
	if len(infos) != 2 {
		t.Errorf("expected 2, got %d", len(infos))
	}
}

func TestCalculateSnapshotDir(t *testing.T) {
	m := &Manager{}
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("world"), 0644)

	size, hash, err := m.calculateSnapshotDir(tmpDir)
	if err != nil {
		t.Fatalf("calculateSnapshotDir() failed: %v", err)
	}

	if size != 10 {
		t.Errorf("expected size 10, got %d", size)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}
}
