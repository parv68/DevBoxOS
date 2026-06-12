package scanner

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// fixtureDir returns the absolute path to the scanner test fixtures directory.
func fixtureDir() string {
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Dir(file)
	return filepath.Join(base, "..", "..", "tests", "fixtures", "scanner")
}

// copyFixture copies a fixture directory into dest.
func copyFixture(t *testing.T, name, dest string) {
	t.Helper()
	src := filepath.Join(fixtureDir(), name)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Fatalf("fixture %s not found at %s", name, src)
	}
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		dstFile, err := os.Create(target)
		if err != nil {
			return err
		}
		defer dstFile.Close()
		_, err = io.Copy(dstFile, srcFile)
		return err
	})
	if err != nil {
		t.Fatalf("copy fixture %s: %v", name, err)
	}
}
