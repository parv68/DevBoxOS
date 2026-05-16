package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Manifest represents snapshot metadata.
type Manifest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	ProjectName string            `json:"project_name"`
	CreatedAt   time.Time         `json:"created_at"`
	Version     string            `json:"version"`
	Services    []ServiceSnapshot `json:"services"`
	Networks    []NetworkSnapshot `json:"networks"`
	Secrets     bool              `json:"secrets_included"`
	SizeBytes   int64             `json:"size_bytes"`
	HashSHA256  string            `json:"hash_sha256"`
}

// ServiceSnapshot represents a service's snapshot data.
type ServiceSnapshot struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Built       bool              `json:"built"`
	ContainerID string            `json:"container_id,omitempty"`
	Volumes     []VolumeSnapshot  `json:"volumes"`
	Env         map[string]string `json:"env,omitempty"`
}

// VolumeSnapshot represents a volume's snapshot data.
type VolumeSnapshot struct {
	Name     string `json:"name"`
	Exported bool   `json:"exported"`
	FileName string `json:"file_name"`
}

// NetworkSnapshot represents a network's snapshot data.
type NetworkSnapshot struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Subnet string `json:"subnet"`
}

// Info represents snapshot listing info.
type Info struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages snapshot metadata on disk.
type Store struct {
	dir string
}

// NewStore creates a new snapshot store.
func NewStore(projectPath string) *Store {
	return &Store{
		dir: filepath.Join(projectPath, ".devbox", "snapshots"),
	}
}

// EnsureDir creates the snapshot directory if needed.
func (s *Store) EnsureDir() error {
	return os.MkdirAll(s.dir, 0755)
}

// SaveManifest writes a snapshot manifest.
func (s *Store) SaveManifest(manifest *Manifest) error {
	if err := s.EnsureDir(); err != nil {
		return err
	}

	path := filepath.Join(s.dir, manifest.ID+".json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// LoadManifest reads a snapshot manifest.
func (s *Store) LoadManifest(id string) (*Manifest, error) {
	path := filepath.Join(s.dir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("snapshot %s not found", id)
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("unmarshal manifest: %w", err)
	}

	return &manifest, nil
}

// ListManifests returns all snapshot manifests.
func (s *Store) ListManifests() ([]Info, error) {
	if err := s.EnsureDir(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read snapshot dir: %w", err)
	}

	var infos []Info
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			continue
		}

		var manifest Manifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		infos = append(infos, Info{
			ID:        manifest.ID,
			Name:      manifest.Name,
			SizeBytes: manifest.SizeBytes,
			CreatedAt: manifest.CreatedAt,
		})
	}

	return infos, nil
}

// DeleteManifest removes a snapshot manifest.
func (s *Store) DeleteManifest(id string) error {
	path := filepath.Join(s.dir, id+".json")
	return os.Remove(path)
}

// SnapshotDir returns the directory for a specific snapshot.
func (s *Store) SnapshotDir(id string) string {
	return filepath.Join(s.dir, id)
}

// EnsureSnapshotDir creates the snapshot data directory.
func (s *Store) EnsureSnapshotDir(id string) error {
	dir := s.SnapshotDir(id)
	return os.MkdirAll(dir, 0755)
}
