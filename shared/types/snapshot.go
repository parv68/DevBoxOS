package types

import "time"

// Snapshot represents a saved environment state.
type Snapshot struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	SizeBytes int64     `json:"size_bytes"`
	HashSHA256 string   `json:"hash_sha256"`
	Signature string    `json:"signature,omitempty"`
	Metadata  string    `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
}

// SnapshotManifest represents the manifest.json inside a .devbox archive.
type SnapshotManifest struct {
	Version     string              `json:"version"`
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Environment SnapshotEnvironment `json:"environment"`
	Services    []SnapshotService   `json:"services"`
	CreatedAt   string              `json:"created_at"`
	OS          string              `json:"os"`
	Arch        string              `json:"arch"`
	DevboxVersion string            `json:"devbox_version"`
}

// SnapshotEnvironment represents environment metadata in a snapshot.
type SnapshotEnvironment struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

// SnapshotService represents a service's state in a snapshot.
type SnapshotService struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
	Port   int    `json:"port,omitempty"`
}

// SnapshotIntegrity represents the integrity.json file in a snapshot.
type SnapshotIntegrity struct {
	Files []FileHash `json:"files"`
}

// FileHash represents a single file's hash in the integrity manifest.
type FileHash struct {
	Path string `json:"path"`
	SHA256 string `json:"sha256"`
}
