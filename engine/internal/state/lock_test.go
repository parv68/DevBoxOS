package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLock_AcquireRelease(t *testing.T) {
	tmpDir := t.TempDir()
	l := NewLock(tmpDir)

	err := l.Acquire("test-op")
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	err = l.Release()
	if err != nil {
		t.Fatalf("Release() failed: %v", err)
	}
}

func TestLock_AcquireTwiceFails(t *testing.T) {
	tmpDir := t.TempDir()
	l := NewLock(tmpDir)

	if err := l.Acquire("op1"); err != nil {
		t.Fatalf("first Acquire() failed: %v", err)
	}

	err := l.Acquire("op2")
	if err == nil {
		t.Error("expected error on second Acquire(), got nil")
	}

	l.Release()
}

func TestLock_StaleLock(t *testing.T) {
	tmpDir := t.TempDir()
	l := NewLock(tmpDir)

	if err := l.Acquire("stale-op"); err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Force lock file to be older than 5 minutes
	lockPath := l.path
	oldTime := time.Now().Add(-10 * time.Minute)
	os.Chtimes(lockPath, oldTime, oldTime)

	if err := l.Acquire("fresh-op"); err != nil {
		t.Errorf("Acquire() should succeed on stale lock, got: %v", err)
	}

	l.Release()
}

func TestLock_ForceRelease(t *testing.T) {
	tmpDir := t.TempDir()
	l := NewLock(tmpDir)

	if err := l.Acquire("op"); err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	if err := l.ForceRelease(); err != nil {
		t.Fatalf("ForceRelease() failed: %v", err)
	}

	if _, err := os.Stat(l.path); !os.IsNotExist(err) {
		t.Error("lock file should not exist after ForceRelease()")
	}
}

func TestLock_ReleaseNoLock(t *testing.T) {
	tmpDir := t.TempDir()
	l := NewLock(tmpDir)

	err := l.Release()
	if err == nil {
		t.Error("expected error releasing non-existent lock, got nil")
	}
}

func TestNewLockFilepath(t *testing.T) {
	l := NewLock("/some/project/path")
	if !filepath.IsAbs(l.path) {
		t.Errorf("expected absolute lock path, got %s", l.path)
	}
	if !filepath.HasPrefix(l.path, filepath.Join(os.TempDir(), ".devbox")) {
		t.Logf("lock path: %s", l.path)
	}
}
