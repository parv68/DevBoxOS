package logging

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/devboxos/devboxos/shared/runtime"
)

const (
	maxLogSize    = 50 * 1024 * 1024 // 50MB
	maxLogFiles   = 5
	logDateFormat = "2006-01-02"
)

// Store manages persistent log storage.
type Store struct {
	baseDir string
	mu      sync.Mutex
}

// NewStore creates a new log store.
func NewStore(baseDir string) *Store {
	return &Store{
		baseDir: filepath.Join(baseDir, ".devbox", "logs"),
	}
}

// EnsureDir creates the log directory if needed.
func (s *Store) EnsureDir(projectName string) error {
	return os.MkdirAll(s.projectDir(projectName), 0755)
}

// Append writes log lines to the persistent store.
func (s *Store) Append(projectName, serviceName string, lines []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.EnsureDir(projectName); err != nil {
		return err
	}

	svcDir := filepath.Join(s.projectDir(projectName), serviceName)
	if err := os.MkdirAll(svcDir, 0755); err != nil {
		return err
	}

	today := time.Now().Format(logDateFormat)
	logFile := filepath.Join(svcDir, fmt.Sprintf("%s.log", today))

	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("write log line: %w", err)
		}
	}

	return nil
}

// Read returns log lines for a service.
func (s *Store) Read(projectName, serviceName string, since time.Time, limit int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	svcDir := filepath.Join(s.projectDir(projectName), serviceName)
	if _, err := os.Stat(svcDir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(svcDir)
	if err != nil {
		return nil, fmt.Errorf("read log dir: %w", err)
	}

	var allLines []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		fileDate, err := time.Parse(logDateFormat, strings.TrimSuffix(entry.Name(), ".log"))
		if err != nil {
			continue
		}

		if !since.IsZero() && fileDate.Before(since.Truncate(24*time.Hour)) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(svcDir, entry.Name()))
		if err != nil {
			continue
		}

		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			if line != "" {
				allLines = append(allLines, line)
			}
		}
	}

	if limit > 0 && len(allLines) > limit {
		allLines = allLines[len(allLines)-limit:]
	}

	return allLines, nil
}

// Search searches logs for a pattern.
func (s *Store) Search(projectName, serviceName string, pattern string, since time.Time) ([]string, error) {
	lines, err := s.Read(projectName, serviceName, since, 0)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile("(?i)" + pattern)
	if err != nil {
		return nil, fmt.Errorf("compile regex: %w", err)
	}

	var matches []string
	for _, line := range lines {
		if re.MatchString(line) {
			matches = append(matches, line)
		}
	}

	return matches, nil
}

// Rotate rotates log files that exceed the size limit.
func (s *Store) Rotate(projectName, serviceName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	svcDir := filepath.Join(s.projectDir(projectName), serviceName)
	if _, err := os.Stat(svcDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(svcDir)
	if err != nil {
		return fmt.Errorf("read log dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.Size() > maxLogSize {
			if err := s.rotateFile(filepath.Join(svcDir, entry.Name())); err != nil {
				return fmt.Errorf("rotate %s: %w", entry.Name(), err)
			}
		}
	}

	return s.cleanupOldLogs(projectName, serviceName)
}

// Export exports logs to a file.
func (s *Store) Export(projectName, serviceName, outputPath string) error {
	lines, err := s.Read(projectName, serviceName, time.Time{}, 0)
	if err != nil {
		return err
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("write line: %w", err)
		}
	}

	return nil
}

// Size returns the total size of logs for a project.
func (s *Store) Size(projectName string) (int64, error) {
	var total int64

	projectDir := s.projectDir(projectName)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return 0, nil
	}

	err := filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})

	return total, err
}

// Clear removes all logs for a project.
func (s *Store) Clear(projectName string) error {
	return os.RemoveAll(s.projectDir(projectName))
}

func (s *Store) projectDir(projectName string) string {
	return filepath.Join(s.baseDir, projectName)
}

func (s *Store) rotateFile(logPath string) error {
	rotatedPath := logPath + ".1"

	if _, err := os.Stat(rotatedPath); err == nil {
		os.Remove(rotatedPath)
	}

	return os.Rename(logPath, rotatedPath)
}

func (s *Store) cleanupOldLogs(projectName, serviceName string) error {
	svcDir := filepath.Join(s.projectDir(projectName), serviceName)

	entries, err := os.ReadDir(svcDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -30)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}

		fileDate, err := time.Parse(logDateFormat, strings.TrimSuffix(entry.Name(), ".log"))
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			os.Remove(filepath.Join(svcDir, entry.Name()))
		}
	}

	return nil
}

// Collector continuously collects logs from a stream.
type Collector struct {
	store   *Store
	project string
	service string
	done    chan struct{}
}

// NewCollector creates a new log collector.
func NewCollector(store *Store, project, service string) *Collector {
	return &Collector{
		store:   store,
		project: project,
		service: service,
		done:    make(chan struct{}),
	}
}

// Start starts collecting logs by polling the Docker runtime.
func (c *Collector) Start(ctx context.Context, rt runtime.Runtime, containerID string) {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		lastTimestamp := ""

		for {
			select {
			case <-c.done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.collectOnce(rt, containerID, &lastTimestamp)
			}
		}
	}()
}

func (c *Collector) collectOnce(rt runtime.Runtime, containerID string, lastTimestamp *string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reader, err := rt.StreamLogs(ctx, containerID, runtime.LogOptions{
		Follow: false,
		Tail:   50,
		Since:  *lastTimestamp,
	})
	if err != nil {
		return
	}
	defer reader.Close()

	var batch []string
	header := make([]byte, 8)

	for {
		_, err := io.ReadFull(reader, header)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			break
		}

		payloadLen := binary.BigEndian.Uint32(header[4:8])
		if payloadLen == 0 || payloadLen > 1024*1024 {
			continue
		}

		payload := make([]byte, payloadLen)
		_, err = io.ReadFull(reader, payload)
		if err != nil {
			break
		}

		line := strings.TrimSpace(string(payload))
		if line != "" {
			if len(line) > 30 && line[0] >= '0' && line[0] <= '9' {
				parts := strings.SplitN(line, " ", 2)
				if len(parts) == 2 {
					*lastTimestamp = parts[0]
					batch = append(batch, line)
				}
			} else {
				batch = append(batch, line)
			}
		}
	}

	if len(batch) > 0 {
		c.store.Append(c.project, c.service, batch)
	}
}

// Stop stops the collector.
func (c *Collector) Stop() {
	close(c.done)
}
