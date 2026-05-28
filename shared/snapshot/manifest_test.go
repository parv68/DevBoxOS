package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	s := NewStore("/tmp/test")
	if s == nil {
		t.Fatal("NewStore() returned nil")
	}
}

func TestStore_SaveAndLoadManifest(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir)

	manifest := &Manifest{
		ID:          "test-id",
		Name:        "test-snapshot",
		ProjectName: "test-project",
		CreatedAt:   time.Now(),
		Version:     "1.0",
		SizeBytes:   1024,
		HashSHA256:  "abc123",
		Services: []ServiceSnapshot{
			{Name: "web", Image: "nginx:alpine"},
		},
	}

	if err := s.SaveManifest(manifest); err != nil {
		t.Fatalf("SaveManifest() failed: %v", err)
	}

	loaded, err := s.LoadManifest("test-id")
	if err != nil {
		t.Fatalf("LoadManifest() failed: %v", err)
	}

	if loaded.ID != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", loaded.ID)
	}
	if loaded.Name != "test-snapshot" {
		t.Errorf("expected Name 'test-snapshot', got '%s'", loaded.Name)
	}
	if loaded.ProjectName != "test-project" {
		t.Errorf("expected ProjectName 'test-project', got '%s'", loaded.ProjectName)
	}
	if len(loaded.Services) != 1 || loaded.Services[0].Name != "web" {
		t.Errorf("unexpected services: %v", loaded.Services)
	}
}

func TestStore_LoadManifest_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir)

	_, err := s.LoadManifest("nonexistent-id")
	if err == nil {
		t.Fatal("expected error for nonexistent manifest")
	}
}

func TestStore_DeleteManifest(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir)

	manifest := &Manifest{ID: "test-id", Name: "test", CreatedAt: time.Now()}
	s.SaveManifest(manifest)

	if err := s.DeleteManifest("test-id"); err != nil {
		t.Fatalf("DeleteManifest() failed: %v", err)
	}

	_, err := s.LoadManifest("test-id")
	if err == nil {
		t.Error("expected error after deleting manifest")
	}
}

func TestStore_ListManifests(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir)

	infos, err := s.ListManifests()
	if err != nil {
		t.Fatalf("ListManifests() failed on empty dir: %v", err)
	}
	if len(infos) != 0 {
		t.Errorf("expected 0 manifests, got %d", len(infos))
	}

	s.SaveManifest(&Manifest{ID: "id1", Name: "snap1", CreatedAt: time.Now(), SizeBytes: 100})
	s.SaveManifest(&Manifest{ID: "id2", Name: "snap2", CreatedAt: time.Now(), SizeBytes: 200})

	infos, err = s.ListManifests()
	if err != nil {
		t.Fatalf("ListManifests() failed: %v", err)
	}
	if len(infos) != 2 {
		t.Errorf("expected 2 manifests, got %d", len(infos))
	}
}

func TestStore_SnapshotDir(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir)

	dir := s.SnapshotDir("test-id")
	expected := filepath.Join(tmpDir, ".devbox", "snapshots", "test-id")
	if dir != expected {
		t.Errorf("SnapshotDir() = %s, want %s", dir, expected)
	}
}

func TestStore_EnsureSnapshotDir(t *testing.T) {
	tmpDir := t.TempDir()
	s := NewStore(tmpDir)

	if err := s.EnsureSnapshotDir("test-id"); err != nil {
		t.Fatalf("EnsureSnapshotDir() failed: %v", err)
	}

	dir := s.SnapshotDir("test-id")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("snapshot directory not created")
	}
}

func TestSecretsHandler_CopyAndRestore(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "secrets.key")
	storePath := filepath.Join(tmpDir, "secrets.enc")

	os.WriteFile(keyPath, []byte("test-key-data"), 0600)
	os.WriteFile(storePath, []byte("test-encrypted-data"), 0600)

	handler := NewSecretsHandler(keyPath, storePath)

	snapshotDir := filepath.Join(tmpDir, "snap")
	os.MkdirAll(snapshotDir, 0755)

	if err := handler.CopyToSnapshot(snapshotDir); err != nil {
		t.Fatalf("CopyToSnapshot() failed: %v", err)
	}

	// Verify files were copied
	copiedKey, _ := os.ReadFile(filepath.Join(snapshotDir, "secrets", "secrets.key"))
	if string(copiedKey) != "test-key-data" {
		t.Errorf("unexpected copied key: %s", string(copiedKey))
	}

	// Now restore to a different directory
	restoreDir := t.TempDir()
	restoreKeyPath := filepath.Join(restoreDir, "secrets.key")
	restoreStorePath := filepath.Join(restoreDir, "secrets.enc")

	restoreHandler := NewSecretsHandler(restoreKeyPath, restoreStorePath)
	if err := restoreHandler.RestoreFromSnapshot(snapshotDir); err != nil {
		t.Fatalf("RestoreFromSnapshot() failed: %v", err)
	}

	restoredKey, _ := os.ReadFile(restoreKeyPath)
	if string(restoredKey) != "test-key-data" {
		t.Errorf("unexpected restored key: %s", string(restoredKey))
	}
}

func TestSecretsHandler_RestoreNoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	handler := NewSecretsHandler(
		filepath.Join(tmpDir, "secrets.key"),
		filepath.Join(tmpDir, "secrets.enc"),
	)

	snapshotDir := filepath.Join(tmpDir, "snap-no-secrets")
	os.MkdirAll(snapshotDir, 0755)

	err := handler.RestoreFromSnapshot(snapshotDir)
	if err != nil {
		t.Fatalf("RestoreFromSnapshot() with no secrets files failed: %v", err)
	}
}
