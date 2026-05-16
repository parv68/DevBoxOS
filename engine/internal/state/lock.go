package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Lock represents a file-based operation lock.
type Lock struct {
	path string
}

// NewLock creates a new lock manager for the given project path.
func NewLock(projectPath string) *Lock {
	homeDir, _ := os.UserHomeDir()
	lockDir := filepath.Join(homeDir, ".devbox", "locks")
	os.MkdirAll(lockDir, 0755)

	// Create a safe filename from the project path
	safeName := filepath.Base(projectPath)
	return &Lock{
		path: filepath.Join(lockDir, safeName+".lock"),
	}
}

// Acquire attempts to acquire a lock for the given operation.
// Returns an error if a lock already exists and hasn't expired.
func (l *Lock) Acquire(operation string) error {
	// Check if lock file exists
	info, err := os.Stat(l.path)
	if err == nil {
		// File exists — check if it's stale (older than 5 minutes)
		if time.Since(info.ModTime()) > 5*time.Minute {
			// Stale lock — remove it
			os.Remove(l.path)
		} else {
			return fmt.Errorf("operation locked: another %s is in progress for this project", operation)
		}
	}

	// Create lock file
	content := fmt.Sprintf("%s\n%d\n", operation, time.Now().Unix())
	return os.WriteFile(l.path, []byte(content), 0644)
}

// Release removes the lock file.
func (l *Lock) Release() error {
	return os.Remove(l.path)
}

// ForceRelease removes the lock file regardless of state.
func (l *Lock) ForceRelease() error {
	return os.Remove(l.path)
}
