package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

// Manager handles local state persistence via SQLite.
type Manager struct {
	db *sql.DB
}

// NewManager creates a new state manager, initializing the database if needed.
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	devboxDir := filepath.Join(homeDir, ".devbox")
	if err := os.MkdirAll(devboxDir, 0755); err != nil {
		return nil, fmt.Errorf("create devbox directory: %w", err)
	}

	dbPath := filepath.Join(devboxDir, "state.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Set connection options
	db.SetMaxOpenConns(1) // SQLite requires single writer
	db.SetConnMaxLifetime(time.Hour)

	m := &Manager{db: db}
	if err := m.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize schema: %w", err)
	}

	return m, nil
}

// initSchema creates the database tables if they don't exist.
func (m *Manager) initSchema() error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS environments (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			path TEXT NOT NULL UNIQUE,
			version TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'stopped',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS services (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL REFERENCES environments(id),
			name TEXT NOT NULL,
			container_id TEXT,
			status TEXT NOT NULL DEFAULT 'stopped',
			port INTEGER,
			health_status TEXT,
			last_check DATETIME,
			restart_count INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL REFERENCES environments(id),
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			hash_sha256 TEXT NOT NULL,
			signature TEXT,
			metadata JSON,
			created_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS locks (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL REFERENCES environments(id),
			operation TEXT NOT NULL,
			acquired_at DATETIME NOT NULL,
			expires_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS telemetry (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			command TEXT,
			duration_ms INTEGER,
			os TEXT,
			arch TEXT,
			version TEXT,
			timestamp DATETIME NOT NULL
		)`,
	}

	for _, schema := range schemas {
		if _, err := m.db.Exec(schema); err != nil {
			return fmt.Errorf("exec schema: %w", err)
		}
	}

	return nil
}

// Close closes the database connection.
func (m *Manager) Close() error {
	return m.db.Close()
}

// DB returns the underlying database connection.
func (m *Manager) DB() *sql.DB {
	return m.db
}
