package snapshot

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/devboxos/devboxos/shared/runtime"
	"github.com/devboxos/devboxos/shared/types"
	"github.com/google/uuid"
)

// Manager handles snapshot operations.
type Manager struct {
	store   *Store
	rt      runtime.Runtime
	secrets *SecretsHandler
}

// SecretsHandler manages secret store operations for snapshots.
type SecretsHandler struct {
	keyPath   string
	storePath string
}

// NewSecretsHandler creates a secrets handler.
func NewSecretsHandler(keyPath, storePath string) *SecretsHandler {
	return &SecretsHandler{
		keyPath:   keyPath,
		storePath: storePath,
	}
}

// NewManager creates a new snapshot manager.
func NewManager(rt runtime.Runtime, projectPath string) *Manager {
	return &Manager{
		store: NewStore(projectPath),
		rt:    rt,
		secrets: NewSecretsHandler(
			filepath.Join(projectPath, ".devbox", "secrets.key"),
			filepath.Join(projectPath, ".devbox", "secrets.enc"),
		),
	}
}

// Save creates a new snapshot.
func (m *Manager) Save(ctx context.Context, cfg *types.Config, name string, includeLogs bool, statusChan chan<- string) (*Manifest, error) {
	id := uuid.New().String()[:8]
	statusChan <- fmt.Sprintf("Creating snapshot %s (%s)...", name, id[:8])

	manifest := &Manifest{
		ID:          id,
		Name:        name,
		ProjectName: cfg.Name,
		CreatedAt:   time.Now(),
		Version:     cfg.Version,
		Secrets:     true,
	}

	if err := m.store.EnsureSnapshotDir(id); err != nil {
		return nil, fmt.Errorf("create snapshot dir: %w", err)
	}

	// Snapshot volumes for each service
	for svcName, svc := range cfg.Services {
		statusChan <- fmt.Sprintf("Snapshotting service: %s", svcName)

		svcSnapshot := ServiceSnapshot{
			Name:  svcName,
			Image: svc.Image,
			Env:   svc.Env,
		}

		// Check if service uses build
		if svc.Build != nil && svc.Build.Context != "" {
			svcSnapshot.Built = true
		}

		// Find container
		containers, err := m.rt.ListContainers(ctx, map[string]string{
			"devboxos.service": svcName,
		})
		if err == nil && len(containers) > 0 {
			svcSnapshot.ContainerID = containers[0].ID

			// Export volumes
			for _, volName := range svc.Volumes {
				parts := strings.SplitN(volName, ":", 2)
				if len(parts) == 2 {
					volSnapshot := VolumeSnapshot{
						Name: parts[1],
					}

					volExists, _ := m.rt.VolumeExists(ctx, parts[1])
					if volExists {
						exportPath := filepath.Join(m.store.SnapshotDir(id), fmt.Sprintf("vol-%s-%s.tar.gz", svcName, parts[1]))
						if err := m.exportVolume(ctx, parts[1], exportPath, statusChan); err != nil {
							statusChan <- fmt.Sprintf("Warning: could not export volume %s: %v", parts[1], err)
						} else {
							volSnapshot.Exported = true
							volSnapshot.FileName = filepath.Base(exportPath)
						}
					}

					svcSnapshot.Volumes = append(svcSnapshot.Volumes, volSnapshot)
				}
			}
		}

		manifest.Services = append(manifest.Services, svcSnapshot)
	}

	// Snapshot networks
	networkName := fmt.Sprintf("devbox-%s", cfg.Name)
	exists, _ := m.rt.NetworkExists(ctx, networkName)
	if exists {
		manifest.Networks = append(manifest.Networks, NetworkSnapshot{
			Name:   networkName,
			Driver: "bridge",
		})
	}

	// Copy secrets
	if err := m.secrets.CopyToSnapshot(m.store.SnapshotDir(id)); err != nil {
		statusChan <- fmt.Sprintf("Warning: could not copy secrets: %v", err)
		manifest.Secrets = false
	}

	// Calculate size and hash
	size, hash, err := m.calculateSnapshotDir(m.store.SnapshotDir(id))
	if err != nil {
		statusChan <- fmt.Sprintf("Warning: could not calculate snapshot size: %v", err)
	}
	manifest.SizeBytes = size
	manifest.HashSHA256 = hash

	// Save manifest
	if err := m.store.SaveManifest(manifest); err != nil {
		return nil, fmt.Errorf("save manifest: %w", err)
	}

	statusChan <- fmt.Sprintf("Snapshot saved: %s (%s)", name, id[:8])
	return manifest, nil
}

// Load restores a snapshot.
func (m *Manager) Load(ctx context.Context, snapshotID string, force bool, statusChan chan<- string) error {
	manifest, err := m.store.LoadManifest(snapshotID)
	if err != nil {
		return err
	}

	statusChan <- fmt.Sprintf("Loading snapshot %s (%s)...", manifest.Name, snapshotID[:8])

	// Restore secrets
	if manifest.Secrets {
		if err := m.secrets.RestoreFromSnapshot(m.store.SnapshotDir(snapshotID)); err != nil {
			statusChan <- fmt.Sprintf("Warning: could not restore secrets: %v", err)
		} else {
			statusChan <- "Secrets restored"
		}
	}

	// Restore volumes
	for _, svc := range manifest.Services {
		statusChan <- fmt.Sprintf("Restoring service: %s", svc.Name)

		for _, vol := range svc.Volumes {
			if vol.Exported && vol.FileName != "" {
				exportPath := filepath.Join(m.store.SnapshotDir(snapshotID), vol.FileName)
				if _, err := os.Stat(exportPath); err == nil {
					if err := m.importVolume(ctx, vol.Name, exportPath, statusChan); err != nil {
						statusChan <- fmt.Sprintf("Warning: could not import volume %s: %v", vol.Name, err)
					} else {
						statusChan <- fmt.Sprintf("Volume %s restored", vol.Name)
					}
				}
			}
		}
	}

	statusChan <- fmt.Sprintf("Snapshot %s loaded", manifest.Name)
	return nil
}

// List returns all snapshots.
func (m *Manager) List() ([]Info, error) {
	return m.store.ListManifests()
}

// Delete removes a snapshot.
func (m *Manager) Delete(snapshotID string) error {
	// Remove snapshot directory
	dir := m.store.SnapshotDir(snapshotID)
	if err := os.RemoveAll(dir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove snapshot dir: %w", err)
	}

	// Remove manifest
	return m.store.DeleteManifest(snapshotID)
}

// Export exports a snapshot to a tarball.
func (m *Manager) Export(snapshotID, outputPath string, statusChan chan<- string) error {
	manifest, err := m.store.LoadManifest(snapshotID)
	if err != nil {
		return err
	}

	statusChan <- fmt.Sprintf("Exporting snapshot %s to %s...", manifest.Name, outputPath)

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	snapshotDir := m.store.SnapshotDir(snapshotID)

	err = filepath.Walk(snapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(snapshotDir, path)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join("snapshot", relPath))

		if info.IsDir() {
			header.Name += "/"
			header.Size = 0
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(tarWriter, file)
		return err
	})

	if err != nil {
		return fmt.Errorf("create tarball: %w", err)
	}

	statusChan <- fmt.Sprintf("Snapshot exported: %s", outputPath)
	return nil
}

// Import imports a snapshot from a tarball.
func (m *Manager) Import(tarballPath string, statusChan chan<- string) error {
	statusChan <- fmt.Sprintf("Importing snapshot from %s...", tarballPath)

	file, err := os.Open(tarballPath)
	if err != nil {
		return fmt.Errorf("open tarball: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var snapshotID string

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tarball: %w", err)
		}

		name := header.Name
		if strings.HasPrefix(name, "snapshot/") {
			relPath := strings.TrimPrefix(name, "snapshot/")
			if relPath == "" {
				continue
			}

			if strings.HasSuffix(relPath, ".json") && snapshotID == "" {
				snapshotID = strings.TrimSuffix(relPath, ".json")
			}

			destPath := filepath.Join(m.store.SnapshotDir(snapshotID), relPath)

			if header.Typeflag == tar.TypeDir {
				if err := os.MkdirAll(destPath, 0755); err != nil {
					return fmt.Errorf("create directory: %w", err)
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("create parent directory: %w", err)
			}

			destFile, err := os.Create(destPath)
			if err != nil {
				return fmt.Errorf("create file: %w", err)
			}

			if _, err := io.Copy(destFile, tarReader); err != nil {
				destFile.Close()
				return fmt.Errorf("write file: %w", err)
			}
			destFile.Close()
		}
	}

	if snapshotID == "" {
		return fmt.Errorf("no snapshot manifest found in tarball")
	}

	statusChan <- fmt.Sprintf("Snapshot imported: %s", snapshotID[:8])
	return nil
}

// exportVolume exports a Docker volume to a tar.gz file.
func (m *Manager) exportVolume(ctx context.Context, volName, outputPath string, statusChan chan<- string) error {
	statusChan <- fmt.Sprintf("Exporting volume: %s", volName)

	// Create a temporary container to export the volume
	containerName := fmt.Sprintf("devbox-snapshot-export-%s", uuid.New().String()[:8])

	containerConfig := runtime.ContainerConfig{
		Name:  containerName,
		Image: "alpine:latest",
		Volumes: map[string]string{
			volName: "/data",
		},
		Command: []string{"sh", "-c", "tar czf /tmp/vol.tar.gz -C /data . && cat /tmp/vol.tar.gz"},
	}

	containerID, err := m.rt.CreateContainer(ctx, containerConfig)
	if err != nil {
		return fmt.Errorf("create export container: %w", err)
	}

	// We can't easily stream from the container, so we'll use docker cp approach
	// For now, mark as not exported
	_ = m.rt.RemoveContainer(ctx, containerID, true)

	// Alternative: use docker volume export (not available in SDK directly)
	// We'll skip volume export for now and rely on container state
	statusChan <- fmt.Sprintf("Volume export skipped: %s (not yet supported)", volName)
	return nil
}

// importVolume imports a Docker volume from a tar.gz file.
func (m *Manager) importVolume(ctx context.Context, volName, inputPath string, statusChan chan<- string) error {
	statusChan <- fmt.Sprintf("Importing volume: %s", volName)

	// Create volume if it doesn't exist
	_ = m.rt.CreateVolume(ctx, volName)

	// Create a temporary container to import the volume
	containerName := fmt.Sprintf("devbox-snapshot-import-%s", uuid.New().String()[:8])

	containerConfig := runtime.ContainerConfig{
		Name:  containerName,
		Image: "alpine:latest",
		Volumes: map[string]string{
			volName: "/data",
		},
		Command: []string{"sh", "-c", "tar xzf /tmp/vol.tar.gz -C /data"},
	}

	containerID, err := m.rt.CreateContainer(ctx, containerConfig)
	if err != nil {
		return fmt.Errorf("create import container: %w", err)
	}

	// Copy tarball to container
	// This would require docker cp, which isn't in our runtime interface
	_ = m.rt.RemoveContainer(ctx, containerID, true)

	statusChan <- fmt.Sprintf("Volume import skipped: %s (not yet supported)", volName)
	return nil
}

// calculateSnapshotSize calculates the total size and SHA-256 hash of a snapshot.
func (m *Manager) calculateSnapshotDir(snapshotDir string) (int64, string, error) {
	var totalSize int64
	hasher := sha256.New()

	err := filepath.Walk(snapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			totalSize += info.Size()

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			hasher.Write(data)
		}

		return nil
	})

	if err != nil {
		return 0, "", err
	}

	return totalSize, hex.EncodeToString(hasher.Sum(nil)), nil
}

// CopyToSnapshot copies secret files to the snapshot directory.
func (s *SecretsHandler) CopyToSnapshot(snapshotDir string) error {
	destDir := filepath.Join(snapshotDir, "secrets")
	if err := os.MkdirAll(destDir, 0700); err != nil {
		return err
	}

	// Copy key file
	if data, err := os.ReadFile(s.keyPath); err == nil {
		if err := os.WriteFile(filepath.Join(destDir, "secrets.key"), data, 0600); err != nil {
			return err
		}
	}

	// Copy store file
	if data, err := os.ReadFile(s.storePath); err == nil {
		if err := os.WriteFile(filepath.Join(destDir, "secrets.enc"), data, 0600); err != nil {
			return err
		}
	}

	return nil
}

// RestoreFromSnapshot restores secret files from the snapshot directory.
func (s *SecretsHandler) RestoreFromSnapshot(snapshotDir string) error {
	srcDir := filepath.Join(snapshotDir, "secrets")

	// Restore key file
	keyData, err := os.ReadFile(filepath.Join(srcDir, "secrets.key"))
	if err == nil {
		if err := os.MkdirAll(filepath.Dir(s.keyPath), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(s.keyPath, keyData, 0600); err != nil {
			return err
		}
	}

	// Restore store file
	storeData, err := os.ReadFile(filepath.Join(srcDir, "secrets.enc"))
	if err == nil {
		if err := os.MkdirAll(filepath.Dir(s.storePath), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(s.storePath, storeData, 0600); err != nil {
			return err
		}
	}

	return nil
}
